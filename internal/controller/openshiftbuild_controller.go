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
	"fmt"

	"github.com/redhat-openshift-builds/operator/internal/common"
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
	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
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

	// Create OpenShiftBuild resource if missing
	openShiftBuild := &openshiftv1alpha1.OpenShiftBuild{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name},
	}
	if result, err := r.CreateOrUpdate(ctx, r.Client, openShiftBuild); err != nil {
		logger.Error(err, "Failed to create or update resource")
		return ctrl.Result{}, err
	} else if result != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("Resource %s", result))
		return ctrl.Result{}, nil
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

	// TODO: Create ShipwrightBuild CR

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
		if !controllerutil.ContainsFinalizer(object, common.OpenShiftBuildFinalizerName) {
			controllerutil.AddFinalizer(object, common.OpenShiftBuildFinalizerName)
		}
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

// SetupWithManager sets up the controller with the Manager.
func (r *OpenShiftBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {

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
