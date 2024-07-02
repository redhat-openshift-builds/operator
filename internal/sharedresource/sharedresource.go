package sharedresource

import (
	"context"

	"github.com/redhat-openshift-builds/operator/internal/common"
	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/pkg/apis/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SharedResource type defines methods to Get, Create v1alpha1.SharedResource resource
type SharedResource struct {
	Client client.Client
}

// New creates new instance of SharedResource type
func New(client client.Client) *SharedResource {
	return &SharedResource{
		Client: client,
	}
}

// Get fetches the current v1alpha1.SharedResource object owned by the controller
func (sr *SharedResource) Get(ctx context.Context, owner client.Object) (*operatorv1alpha1.SharedResource, error) {
	list := &operatorv1alpha1.SharedResourceList{}
	if err := sr.Client.List(ctx, list); err != nil {
		return nil, err
	}

	if len(list.Items) != 0 {
		for _, item := range list.Items {
			if metav1.IsControlledBy(&item, owner) {
				return &item, nil
			}
		}
	}

	gvk, err := sr.Client.GroupVersionKindFor(&operatorv1alpha1.SharedResource{})
	if err != nil {
		return nil, err
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    gvk.Group,
		Resource: gvk.Kind,
	}, "")
}

// Create creates v1alpha1.SharedResource object
func (sr *SharedResource) Create(ctx context.Context, owner client.Object) error {
	object := &operatorv1alpha1.SharedResource{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: owner.GetName() + "-",
			Finalizers:   []string{common.OpenShiftBuildFinalizerName},
		},
		Spec: operatorv1alpha1.SharedResourceSpec{
			TargetNamespace: common.OpenShiftBuildNamespaceName,
		},
	}
	if err := ctrl.SetControllerReference(owner, object, sr.Client.Scheme()); err != nil {
		return err
	}
	return sr.Client.Create(ctx, object)
}

// CreateOrUpdate updates existing v1alpha1.SharedResource object
func (sr *SharedResource) CreateOrUpdate(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error) {
	object, err := sr.Get(ctx, owner)
	if err != nil && !apierrors.IsNotFound(err) {
		return "", err
	}

	if object == nil {
		object = &operatorv1alpha1.SharedResource{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: owner.GetName() + "-",
			},
		}
	}

	return ctrl.CreateOrUpdate(ctx, sr.Client, object, func() error {
		object.Spec = operatorv1alpha1.SharedResourceSpec{
			TargetNamespace: common.OpenShiftBuildNamespaceName,
		}
		controllerutil.AddFinalizer(object, common.OpenShiftBuildFinalizerName)
		if err := ctrl.SetControllerReference(owner, object, sr.Client.Scheme()); err != nil {
			return err
		}
		return nil
	})
}

// Delete deletes v1alpha1.SharedResource object
// func (sr *SharedResource) Delete(ctx context.Context, owner client.Object) error {
// 	object, err := sr.Get(ctx, owner)
// 	if err != nil {
// 		return err
// 	}

// 	controllerutil.RemoveFinalizer(object, common.OpenShiftBuildFinalizerName)
// 	if err := sr.Client.Update(ctx, object); err != nil {
// 		return err
// 	}

// 	return sr.Client.Delete(ctx, object)
// }
