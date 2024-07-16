/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	manifestivalclient "github.com/manifestival/controller-runtime-client"
	"github.com/manifestival/manifestival"
	"github.com/redhat-openshift-builds/operator/internal/sharedresource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	shipwrightbuild "github.com/redhat-openshift-builds/operator/internal/shipwright/build"
	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
)

// OpenShiftBuildReconciler reconciles a OpenShiftBuild object
type OpenShiftBuildReconciler struct {
	APIReader      client.Reader
	Client         client.Client
	Scheme         *apiruntime.Scheme
	Logger         logr.Logger
	SharedResource *sharedresource.SharedResource
	Shipwright     *shipwrightbuild.ShipwrightBuild
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OpenShiftBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("name", req.Name)
	logger.Info("Starting reconciliation")

	// Get OpenShiftBuild resource from cache
	openShiftBuild := &openshiftv1alpha1.OpenShiftBuild{}
	if err := r.Client.Get(ctx, req.NamespacedName, openShiftBuild); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource not found!")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get resource from cache")
		return ctrl.Result{}, err
	}

	// Initialize status
	if openShiftBuild.Status.Conditions == nil {
		openShiftBuild.Status.Conditions = []metav1.Condition{}
		apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
			Type:    openshiftv1alpha1.ConditionReady,
			Status:  metav1.ConditionUnknown,
			Reason:  "Initializing",
			Message: "Initializing Openshift Builds Operator",
		})
		if err := r.Client.Status().Update(ctx, openShiftBuild); err != nil {
			logger.Error(err, "Failed to initialize status")
			return ctrl.Result{}, err
		}
	}

	// TODO: Add any specific cleanup logic
	if !openShiftBuild.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.HandleDeletion(ctx, openShiftBuild)
	}

	// Reconcile Shipwright Build
	if err := r.ReconcileShipwrightBuild(ctx, openShiftBuild); err != nil {
		logger.Error(err, "Failed to reconcile ShipwrightBuild")
		apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
			Type:    openshiftv1alpha1.ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "Failed",
			Message: fmt.Sprintf("Failed to reconcile OpenShiftBuild: %v", err),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, openShiftBuild)
	}

	// Reconcile Shared Resources
	if err := r.ReconcileSharedResource(ctx, openShiftBuild); err != nil {
		logger.Error(err, "Failed to reconcile SharedResource")
		apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
			Type:    openshiftv1alpha1.ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "Failed",
			Message: fmt.Sprintf("Failed to reconcile OpenShiftBuild: %v", err),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, openShiftBuild)
	}

	// Update status
	apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
		Type:    openshiftv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Success",
		Message: "Successfully reconciled OpenShiftBuild",
	})
	if err := r.Client.Status().Update(ctx, openShiftBuild); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Finished reconciliation")
	return ctrl.Result{}, nil
}

// BootstrapOpenShiftBuild creates the default OpenShiftBuild instance ("cluster") if it is not
// present on the cluster.
func (r *OpenShiftBuildReconciler) BootstrapOpenShiftBuild(ctx context.Context, client client.Client) error {
	logger := log.FromContext(ctx).WithValues("name", common.OpenShiftBuildResourceName)
	bootstrapOpenShiftBuild := &openshiftv1alpha1.OpenShiftBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.OpenShiftBuildResourceName,
		},
	}
	if client == nil {
		client = r.Client
	}
	result, err := r.CreateOrUpdate(ctx, client, bootstrapOpenShiftBuild)
	if err != nil {
		logger.Error(err, "failed to boostrap OpenShiftBuild")
		return err
	}
	logger.Info("boostrap OpenShiftBuild reconciled", "result", result)
	return nil
}

// CreateOrUpdate will create or update v1alpha1.OpenShiftBuild resource
func (r *OpenShiftBuildReconciler) CreateOrUpdate(ctx context.Context, client client.Client, object *openshiftv1alpha1.OpenShiftBuild) (controllerutil.OperationResult, error) {
	return ctrl.CreateOrUpdate(ctx, client, object, func() error {
		controllerutil.AddFinalizer(object, common.OpenShiftBuildFinalizerName)
		if object.Spec.Shipwright == nil {
			object.Spec.Shipwright = &openshiftv1alpha1.Shipwright{
				Build: &openshiftv1alpha1.ShipwrightBuild{
					State: openshiftv1alpha1.Enabled,
				},
			}
		}
		if object.Spec.SharedResource == nil {
			object.Spec.SharedResource = &openshiftv1alpha1.SharedResource{
				State: openshiftv1alpha1.Enabled,
			}
		}
		return nil
	})
}

// ReconcileSharedResource creates and updates SharedResource objects
func (r *OpenShiftBuildReconciler) ReconcileSharedResource(ctx context.Context, openshiftBuild *openshiftv1alpha1.OpenShiftBuild) error {
	logger := log.FromContext(ctx).WithValues("name", openshiftBuild.ObjectMeta.Name)

	logger.Info("Reconciling SharedResource...")
	if err := r.SharedResource.Reconcile(openshiftBuild); err != nil {
		logger.Error(err, "Failed reconciling SharedResource...")
		return err
	}

	return nil
}

// BootStrapSharedResource initializes the manifestival to apply Shared Resources
func (r *OpenShiftBuildReconciler) setupSharedResource(mgr ctrl.Manager) error {
	// Initialize Manifestival
	manifestivalOptions := []manifestival.Option{
		manifestival.UseLogger(r.Logger),
		manifestival.UseClient(manifestivalclient.NewClient(mgr.GetClient())),
	}

	// Shared Resource manifests
	sharedManifestPath := common.SharedResourceManifestPath
	if path, ok := os.LookupEnv(common.SharedResourceManifestPath); ok {
		sharedManifestPath = path
	}
	sharedManifest, err := manifestival.NewManifest(sharedManifestPath, manifestivalOptions...)
	if err != nil {
		return err
	}

	// Initialize Shared Resource
	r.SharedResource = sharedresource.New(sharedManifest)
	return nil
}

// HandleDeletion deletes objects created by the controller
func (r *OpenShiftBuildReconciler) HandleDeletion(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	logger := log.FromContext(ctx).WithValues("name", owner.Name)
	if err := r.Shipwright.Delete(ctx, owner); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to delete Shipwright Build")
		return err
	}
	if err := r.SharedResource.Reconcile(owner); err != nil {
		logger.Error(err, "Failed to delete SharedResource")
		return err
	}
	if controllerutil.ContainsFinalizer(owner, common.OpenShiftBuildFinalizerName) {
		if ok := controllerutil.RemoveFinalizer(owner, common.OpenShiftBuildFinalizerName); ok {
			return r.Client.Update(ctx, owner)
		}
	}
	return nil
}

// ReconcileShipwrightBuild creates or deletes ShipwrightBuild object
func (r *OpenShiftBuildReconciler) ReconcileShipwrightBuild(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	logger := log.FromContext(ctx).WithValues("name", owner.Name)

	switch owner.Spec.Shipwright.Build.State {
	case openshiftv1alpha1.Enabled:
		result, err := r.Shipwright.CreateOrUpdate(ctx, owner)
		if err != nil {
			return err
		}
		logger.Info("ShipwrightBuild resource", "result", result)
	case openshiftv1alpha1.Disabled:
		if err := r.Shipwright.Delete(ctx, owner); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		logger.Info("ShipwrightBuild resource", "result", "deleted")
	default:
		return errors.New("unknown component state")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenShiftBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// bootstrap Shared Resources
	if err := r.setupSharedResource(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&openshiftv1alpha1.OpenShiftBuild{}).
		Owns(&shipwrightv1alpha1.ShipwrightBuild{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return !e.DeleteStateUnknown
			},
		}).
		Complete(r)
}
