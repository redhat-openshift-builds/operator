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

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/redhat-openshift-builds/operator/internal/common"
	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/pkg/apis/v1alpha1"
)

// SharedResourceReconciler reconciles a SharedResource object
type SharedResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.openshift.io,resources=sharedresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.openshift.io,resources=sharedresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=sharedresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SharedResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *SharedResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)
	logger := log.FromContext(ctx).WithValues("name", req.Name)
	logger.Info("Starting reconciliation")

	// retrieve the SharedResource instance requested for reconcile
	sr := &operatorv1alpha1.SharedResource{}
	if err := r.Client.Get(ctx, req.NamespacedName, sr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found!")
			return ctrl.Result{Requeue: true}, r.CreateSharedResource(ctx)
		}
		logger.Error(err, "Failed to get resource")
		return ctrl.Result{}, err
	}

	// Initialize status
	if sr.Status.Conditions == nil {
		logger.Info("Initializing status")
		sr.Status.Conditions = []metav1.Condition{}
		apimeta.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
			Type:    operatorv1alpha1.ConditionReady,
			Status:  metav1.ConditionUnknown,
			Reason:  "Initializing",
			Message: "Initializing Shared Resource Object",
		})
		return ctrl.Result{Requeue: true}, r.Client.Status().Update(ctx, sr)
	}

	// Add finalizer
	if ok := controllerutil.AddFinalizer(sr, common.OpenShiftBuildFinalizerName); ok {
		logger.Info("Adding finalizer")
		return ctrl.Result{Requeue: true}, r.Client.Update(ctx, sr)
	}

	// Perform cleanup
	if !sr.GetDeletionTimestamp().IsZero() {
		logger.Info("Resource is marked for deletion")
		return ctrl.Result{Requeue: true}, r.Client.Delete(ctx, sr)
	}

	// Update status
	logger.Info("Updating status")
	apimeta.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
		Type:    operatorv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Success",
		Message: "Successfully reconciled SharedResource",
	})

	return ctrl.Result{}, nil
}

func (r *SharedResourceReconciler) CreateSharedResource(ctx context.Context) error {
	// TODO
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SharedResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SharedResource{}).
		Complete(r)
}
