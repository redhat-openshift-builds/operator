package networkpolicy

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/manifestival/manifestival"
	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkPolicy struct {
	Client   client.Client
	Logger   logr.Logger
	Manifest manifestival.Manifest
}

func New(client client.Client, manifest manifestival.Manifest, logger logr.Logger) *NetworkPolicy {
	return &NetworkPolicy{
		Client:   client,
		Manifest: manifest,
		Logger:   logger,
	}
}

func (np *NetworkPolicy) Reconcile(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	logger := np.Logger.WithValues("name", owner.Name)

	transformerfuncs := []manifestival.Transformer{
		manifestival.InjectOwner(owner),
		manifestival.InjectNamespace(common.OpenShiftBuildNamespaceName),
	}

	if owner.DeletionTimestamp.IsZero() {
		transformerfuncs = append(transformerfuncs, common.InjectFinalizer(common.OpenShiftBuildFinalizerName))
	}

	manifest, err := np.Manifest.Transform(transformerfuncs...)
	if err != nil {
		logger.Error(err, "Failed to transform NetworkPolicy manifests")
		return err
	}

	if !owner.DeletionTimestamp.IsZero() {
		logger.Info("OpenShiftBuild is being deleted, cleaning up NetworkPolicy resources")
		return np.deleteManifests(&manifest)
	}

	logger.Info("Applying NetworkPolicy manifests for zero-trust security")
	return manifest.Apply()
}

func (np *NetworkPolicy) deleteManifests(manifest *manifestival.Manifest) error {
	mfc := np.Manifest.Client
	for _, res := range manifest.Resources() {
		obj, err := mfc.Get(&res)
		if err != nil {
			if errors.IsNotFound(err) {
				continue 
			}
			return err
		}

		if len(obj.GetFinalizers()) > 0 {
			obj.SetFinalizers([]string{})
			if err := mfc.Update(obj); err != nil {
				return err
			}
		}

		np.Logger.Info("Deleting NetworkPolicy resource", "name", res.GetName())
		if err := mfc.Delete(&res); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
