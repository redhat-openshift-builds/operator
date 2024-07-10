package controller

import (
	"context"
	"os"

	manifestivalclient "github.com/manifestival/controller-runtime-client"
	"github.com/manifestival/manifestival"
	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
	shipwrightoperator "github.com/shipwright-io/operator/controllers"
	tektonoperatorv1alpha1 "github.com/tektoncd/operator/pkg/client/clientset/versioned/typed/operator/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ShipwrightBuildReconciler shipwrightoperator.ShipwrightBuildReconciler

func (r *ShipwrightBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create Owner Reference for filtering
	gvk, err := r.Client.GroupVersionKindFor(&openshiftv1alpha1.OpenShiftBuild{})
	if err != nil {
		return err
	}
	owner := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}

	// Initialize logger
	r.Logger = mgr.GetLogger()

	// Initialize CRD client from REST config
	if r.CRDClient, err = apiextensionsv1.NewForConfig(mgr.GetConfig()); err != nil {
		return err
	}

	// Initialize Tekton Operator client from REST config
	if r.TektonOperatorClient, err = tektonoperatorv1alpha1.NewForConfig(mgr.GetConfig()); err != nil {
		return err
	}

	// Initialize Manifest
	manifestivalOptions := []manifestival.Option{
		manifestival.UseLogger(r.Logger),
		manifestival.UseClient(manifestivalclient.NewClient(mgr.GetClient())),
	}

	// Shipwright Build release manifests
	manifestPath := common.ShipwrightBuildManifestPath
	if path, ok := os.LookupEnv(common.ShipwrightBuildManifestPathEnv); ok {
		manifestPath = path
	}
	if r.Manifest, err = manifestival.NewManifest(manifestPath, manifestivalOptions...); err != nil {
		return err
	}

	// Shipwright Build strategies manifests
	manifestPath = common.ShipwrightBuildStrategyManifestPath
	if path, ok := os.LookupEnv(common.ShipwrightBuildStrategyManifestPathEnv); ok {
		manifestPath = path
	}
	if r.BuildStrategyManifest, err = manifestival.NewManifest(manifestPath, manifestivalOptions...); err != nil {
		return err
	}

	// Check if Shipwright Build CRD exists
	_, err = r.CRDClient.CustomResourceDefinitions().Get(context.TODO(), common.ShipwrightBuildCRDName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	reconciler := shipwrightoperator.ShipwrightBuildReconciler(*r)

	return ctrl.NewControllerManagedBy(mgr).
		For(&shipwrightv1alpha1.ShipwrightBuild{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return common.IsControlledBy(e.Object, owner)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return common.IsControlledBy(e.ObjectOld, owner) &&
					common.IsControlledBy(e.ObjectNew, owner) &&
					!controllerutil.ContainsFinalizer(e.ObjectNew, common.OpenShiftBuildFinalizerName) &&
					!e.ObjectNew.GetDeletionTimestamp().IsZero()
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(&reconciler)
}
