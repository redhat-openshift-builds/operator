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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/redhat-openshift-builds/operator/internal/common"
	"github.com/redhat-openshift-builds/operator/internal/sharedresource"
	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/pkg/apis/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OpenShiftBuildReconciler reconciles a OpenShiftBuild object
type OpenShiftBuildReconciler struct {
	Client client.Client
	Scheme *apiruntime.Scheme
}

//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OpenShiftBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("name", req.Name)
	logger.Info("Starting reconciliation")

	// Get OpenShiftBuild object
	openShiftBuild := &openshiftv1alpha1.OpenShiftBuild{}
	if err := r.Client.Get(ctx, req.NamespacedName, openShiftBuild); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource not found")
			return ctrl.Result{Requeue: true}, r.CreateOpenShiftBuild(ctx)
		}
		logger.Error(err, "Failed to get resource")
		return ctrl.Result{}, err
	}

	// Initialize status
	if openShiftBuild.Status.Conditions == nil {
		logger.Info("Initializing status")
		openShiftBuild.Status.Conditions = []metav1.Condition{}
		apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
			Type:    openshiftv1alpha1.ConditionReady,
			Status:  metav1.ConditionUnknown,
			Reason:  "Initializing",
			Message: "Initializing Openshift Builds Operator",
		})
		return ctrl.Result{Requeue: true}, r.Client.Status().Update(ctx, openShiftBuild)
	}

	// Add finalizer
	if ok := controllerutil.AddFinalizer(openShiftBuild, common.OpenShiftBuildFinalizerName); ok {
		logger.Info("Adding finalizer")
		return ctrl.Result{Requeue: true}, r.Client.Update(ctx, openShiftBuild)
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

	// Perform cleanup
	if !openShiftBuild.GetDeletionTimestamp().IsZero() {
		logger.Info("Resource is marked for deletion")
		if openShiftBuild.Spec.SharedResource.State == openshiftv1alpha1.Enabled {
			logger.Info("Deleting SharedResource")
			openShiftBuild.Spec.SharedResource.State = openshiftv1alpha1.Disabled
			return ctrl.Result{Requeue: true}, r.Client.Update(ctx, openShiftBuild)
		}
		logger.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(openShiftBuild, common.OpenShiftBuildFinalizerName)
		return ctrl.Result{Requeue: true}, r.Client.Update(ctx, openShiftBuild)
	}

	// Update status
	logger.Info("Updating status")
	apimeta.SetStatusCondition(&openShiftBuild.Status.Conditions, metav1.Condition{
		Type:    openshiftv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Success",
		Message: "Successfully reconciled OpenShiftBuild",
	})

	return ctrl.Result{}, r.Client.Status().Update(ctx, openShiftBuild)
}

func (r *OpenShiftBuildReconciler) CreateOpenShiftBuild(ctx context.Context) error {
	openShiftBuild := &openshiftv1alpha1.OpenShiftBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:       common.OpenShiftBuildResourceName,
			Finalizers: []string{common.OpenShiftBuildFinalizerName},
		},
		Spec: openshiftv1alpha1.OpenShiftBuildSpec{
			SharedResource: &openshiftv1alpha1.SharedResourceState{
				State: openshiftv1alpha1.Disabled,
			},
		},
	}
	return r.Client.Create(ctx, openShiftBuild)
}

// ReconcileSharedResource creates or deletes SharedResource object
func (r *OpenShiftBuildReconciler) ReconcileSharedResource(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	logger := log.FromContext(ctx).WithValues("name", owner.Name)

	sr := sharedresource.New(r.Client)

	switch owner.Spec.SharedResource.State {
	case openshiftv1alpha1.Enabled:
		result, err := sr.CreateOrUpdate(ctx, owner)
		if err != nil {
			return err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			logger.Info("Creating SharedResources")
		case controllerutil.OperationResultUpdated:
			logger.Info("Updating SharedResources")
		}
	case openshiftv1alpha1.Disabled:
		// At present, we do not support deletion of existing SharedResource objects once enabled.
		logger.Info("SharedResouces is Disabled, Nothing to do")
	default:
		return errors.New("unknown component state")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenShiftBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Check if OpenShiftBuild CRD is present
	CRDClient, err := apiextensionsv1.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	_, err = CRDClient.CustomResourceDefinitions().Get(context.TODO(), common.OpenShiftBuildCRDName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Replace the client with a non-cached client to create resource before starting manager
	r.Client, err = client.New(mgr.GetConfig(), client.Options{Scheme: r.Scheme})
	if err != nil {
		return err
	}

	// Create OpenShitBuild CR if not present
	list := openshiftv1alpha1.OpenShiftBuildList{}
	if err := r.Client.List(context.TODO(), &list); err != nil {
		return err
	}
	if len(list.Items) == 0 {
		if err := r.CreateOpenShiftBuild(context.TODO()); err != nil {
			return err
		}
	}

	// Restore the client with cache client from manager
	r.Client = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&openshiftv1alpha1.OpenShiftBuild{}).
		Owns(&openshiftv1alpha1.SharedResource{}).
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
