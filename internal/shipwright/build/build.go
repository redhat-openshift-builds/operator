package build

import (
	"context"

	"github.com/redhat-openshift-builds/operator/internal/common"
	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ShipwrightBuild type defines methods to Get, Create, Delete v1alpha1.ShipwrightBuild resource
type ShipwrightBuild struct {
	Client client.Client
}

// New creates new instance of ShipwrightBuild type
func New(client client.Client) *ShipwrightBuild {
	return &ShipwrightBuild{
		Client: client,
	}
}

// Get fetches the current v1alpha1.ShipwrightBuild object owned by the controller
func (sb *ShipwrightBuild) Get(ctx context.Context, owner client.Object) (*shipwrightv1alpha1.ShipwrightBuild, error) {
	list := &shipwrightv1alpha1.ShipwrightBuildList{}
	if err := sb.Client.List(ctx, list); err != nil {
		return nil, err
	}

	if len(list.Items) != 0 {
		for _, item := range list.Items {
			if metav1.IsControlledBy(&item, owner) {
				return &item, nil
			}
		}
	}

	gvk, err := sb.Client.GroupVersionKindFor(&shipwrightv1alpha1.ShipwrightBuild{})
	if err != nil {
		return nil, err
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    gvk.Group,
		Resource: gvk.Kind,
	}, "")
}

// CreateOrUpdate creates v1alpha1.ShipwrightBuild object
func (sb *ShipwrightBuild) CreateOrUpdate(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error) {
	object, err := sb.Get(ctx, owner)
	if err != nil && !apierrors.IsNotFound(err) {
		return "", err
	}

	if object == nil {
		object = &shipwrightv1alpha1.ShipwrightBuild{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: owner.GetName() + "-",
			},
		}
	}

	return ctrl.CreateOrUpdate(ctx, sb.Client, object, func() error {
		object.Spec = shipwrightv1alpha1.ShipwrightBuildSpec{
			TargetNamespace: common.OpenShiftBuildNamespaceName,
		}
		controllerutil.AddFinalizer(object, common.OpenShiftBuildFinalizerName)
		if err := ctrl.SetControllerReference(owner, object, sb.Client.Scheme()); err != nil {
			return err
		}
		return nil
	})
}

// Delete deletes a v1alpha1.ShipwrightBuild objects
func (sb *ShipwrightBuild) Delete(ctx context.Context, owner client.Object) error {
	object, err := sb.Get(ctx, owner)
	if err != nil {
		return err
	}

	controllerutil.RemoveFinalizer(object, common.OpenShiftBuildFinalizerName)
	if err := sb.Client.Update(ctx, object); err != nil {
		return err
	}

	return sb.Client.Delete(ctx, object)
}
