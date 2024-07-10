package sharedresource

import (
	"path/filepath"
	"slices"

	"github.com/go-logr/logr"
	"github.com/manifestival/manifestival"
	"github.com/redhat-openshift-builds/operator/internal/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// FinalizerAnnotation annotation string appended to finalizer slice.
	FinalizerAnnotation = "finalizer.operator.sharedresource.io"
)

// SharedResource type defines methods to Get, Create v1alpha1.SharedResource resource
type SharedResource struct {
	Client   client.Client
	Logger   logr.Logger
	Manifest manifestival.Manifest
}

// New creates new instance of SharedResource type
func New(client client.Client) *SharedResource {
	return &SharedResource{
		Client: client,
	}
}

// InjectFinalizer appends finalizer to the passed resources metadata.
func InjectFinalizer() manifestival.Transformer {
	return func(u *unstructured.Unstructured) error {
		finalizers := u.GetFinalizers()
		if !slices.Contains(finalizers, FinalizerAnnotation) {
			finalizers = append(finalizers, FinalizerAnnotation)
			u.SetFinalizers(finalizers)
		}

		return nil
	}
}

// InjectOwner Should be OpenShiftBuildsController??
// could be done by existing manifestival transformer: https://github.com/manifestival/manifestival/blob/master/transform.go#L95

// func (r *SharedResource) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
// 	logger := r.Logger.WithValues("name", req.Name)
// 	logger.Info("Starting resource reconciliation...")

// 	// Applying transformers
// 	transformerfuncs := []manifestival.Transformer{}
// 	transformerfuncs = append(transformerfuncs, r.InjectFinalizer())
// 	transformerfuncs = append(transformerfuncs, manifestival.InjectNamespace(common.OpenShiftBuildNamespaceName))
// 	transformerfuncs = append(transformerfuncs, manifestival.InjectOwner(&openshiftv1alpha1.OpenShiftBuild{}))

// 	manifest, err := r.Manifest.Transform(transformerfuncs...)
// 	if err != nil {
// 		logger.Error(err, "transforming manifest")
// 		return RequeueWithError(err)
// 	}

// 	// TODO: Add deletion logic pertaining to the scenario when operator is being uninstalled.
// 	// i.e. if any CSI SharedResource object exists, the CRs shouldn't be deleted.

// 	// Rolling out the resources described on the manifests
// 	logger.Info("Applying manifests...")
// 	if err := manifest.Apply(); err != nil {
// 		logger.Error(err, "applying manifest")
// 		return RequeueWithError(err)
// 	}

// 	return NoRequeue()
// }

// setupManifestival instantiate manifestival with local controller attributes.
func (sr *SharedResource) setupManifestival() error {
	var err error
	sr.Manifest, err = common.SetupManifestival(sr.Client, filepath.Join("config", "sharedresource"), true, sr.Logger)
	if err != nil {
		return err
	}

	return nil
}

// RequeueWithError triggers a object requeue because the informed error happend.
func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, err
}

// NoRequeue all done, the object does not need reconciliation anymore.
func NoRequeue() (ctrl.Result, error) {
	return ctrl.Result{Requeue: false}, nil
}

// // Get fetches the current v1alpha1.SharedResource object owned by the controller
// func (sr *SharedResource) Get(ctx context.Context, owner client.Object) (*operatorv1alpha1.SharedResource, error) {
// 	list := &operatorv1alpha1.SharedResourceList{}
// 	if err := sr.Client.List(ctx, list); err != nil {
// 		return nil, err
// 	}

// 	if len(list.Items) != 0 {
// 		for _, item := range list.Items {
// 			if metav1.IsControlledBy(&item, owner) {
// 				return &item, nil
// 			}
// 		}
// 	}

// 	gvk, err := sr.Client.GroupVersionKindFor(&operatorv1alpha1.SharedResource{})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return nil, apierrors.NewNotFound(schema.GroupResource{
// 		Group:    gvk.Group,
// 		Resource: gvk.Kind,
// 	}, "")
// }

// // Create creates v1alpha1.SharedResource object
// func (sr *SharedResource) Create(ctx context.Context, owner client.Object) error {
// 	object := &operatorv1alpha1.SharedResource{
// 		ObjectMeta: metav1.ObjectMeta{
// 			GenerateName: owner.GetName() + "-",
// 			Finalizers:   []string{common.OpenShiftBuildFinalizerName},
// 		},
// 		Spec: operatorv1alpha1.SharedResourceSpec{
// 			TargetNamespace: common.OpenShiftBuildNamespaceName,
// 		},
// 	}
// 	if err := ctrl.SetControllerReference(owner, object, sr.Client.Scheme()); err != nil {
// 		return err
// 	}
// 	return sr.Client.Create(ctx, object)
// }

// // CreateOrUpdate updates existing v1alpha1.SharedResource object
// func (sr *SharedResource) CreateOrUpdate(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error) {
// 	object, err := sr.Get(ctx, owner)
// 	if err != nil && !apierrors.IsNotFound(err) {
// 		return "", err
// 	}

// 	if object == nil {
// 		object = &operatorv1alpha1.SharedResource{
// 			ObjectMeta: metav1.ObjectMeta{
// 				GenerateName: owner.GetName() + "-",
// 			},
// 		}
// 	}

// 	return ctrl.CreateOrUpdate(ctx, sr.Client, object, func() error {
// 		object.Spec = operatorv1alpha1.SharedResourceSpec{
// 			TargetNamespace: common.OpenShiftBuildNamespaceName,
// 		}
// 		controllerutil.AddFinalizer(object, common.OpenShiftBuildFinalizerName)
// 		if err := ctrl.SetControllerReference(owner, object, sr.Client.Scheme()); err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// }

// // Delete deletes v1alpha1.SharedResource object
// // func (sr *SharedResource) Delete(ctx context.Context, owner client.Object) error {
// // 	object, err := sr.Get(ctx, owner)
// // 	if err != nil {
// // 		return err
// // 	}

// // 	controllerutil.RemoveFinalizer(object, common.OpenShiftBuildFinalizerName)
// // 	if err := sr.Client.Update(ctx, object); err != nil {
// // 		return err
// // 	}

// // 	return sr.Client.Delete(ctx, object)
// // }
