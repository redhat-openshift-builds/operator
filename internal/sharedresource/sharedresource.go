package sharedresource

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/manifestival/manifestival"
	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedResource type defines methods to Get, Create v1alpha1.SharedResource resource
type SharedResource struct {
	Client   client.Client
	Logger   logr.Logger
	Manifest manifestival.Manifest
	State    openshiftv1alpha1.State
}

// New creates new instance of SharedResource type
func New(client client.Client, manifest manifestival.Manifest) *SharedResource {
	return &SharedResource{
		Client:   client,
		Manifest: manifest,
	}
}

// UpdateSharedResource transforms the manifests, and applies or deletes them based on SharedResource.State.
func (sr *SharedResource) Reconcile(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	logger := sr.Logger.WithValues("name", owner.Name)

	if owner.Spec.SharedResource == nil {
		owner.Spec.SharedResource = &openshiftv1alpha1.SharedResource{
			State: openshiftv1alpha1.Enabled,
		}
		if err := sr.Client.Update(ctx, owner); err != nil {
			return fmt.Errorf("failed to update OpenShiftBuild with default values: %v", err)
		}
	}
	sr.State = owner.Spec.SharedResource.State

	// Applying transformers
	transformerfuncs := []manifestival.Transformer{}
	transformerfuncs = append(transformerfuncs, manifestival.InjectOwner(owner))
	transformerfuncs = append(transformerfuncs, manifestival.InjectNamespace(common.OpenShiftBuildNamespaceName))
	if sr.State == openshiftv1alpha1.Enabled && owner.DeletionTimestamp.IsZero() {
		transformerfuncs = append(transformerfuncs, common.InjectFinalizer(common.OpenShiftBuildFinalizerName))
	}

	manifest, err := sr.Manifest.Transform(transformerfuncs...)
	if err != nil {
		logger.Error(err, "transforming manifest")
		return err
	}

	// The deleteManifests is invoked if either SharedResource is disabled or
	// the owner is being deleted with enabled SharedResource
	if !owner.DeletionTimestamp.IsZero() || sr.State == openshiftv1alpha1.Disabled {
		return sr.deleteManifests(&manifest)
	}

	logger.Info("Applying manifests...")
	return manifest.Apply()
}

// deleteManifests removes the applied finalizer from all manifest.Resources &
// performs deletion of the resources if SharedResource.State is disabled.
func (sr *SharedResource) deleteManifests(manifest *manifestival.Manifest) error {
	mfc := sr.Manifest.Client
	for _, res := range manifest.Resources() {
		obj, err := mfc.Get(&res)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		// removes finalizers
		if len(obj.GetFinalizers()) > 0 {
			obj.SetFinalizers([]string{})
			if err := mfc.Update(obj); err != nil {
				return err
			}
		}

		// Perform explicit deletion of resources only when SharedResource is Disabled.
		// When owner is set for deletion, the deletion of resources will be performed by reconciler.
		if sr.State == openshiftv1alpha1.Disabled {
			sr.Logger.Info("Deleting SharedResources")
			mfc.Delete(&res)
		}
	}
	return nil
}
