package common

import (
	"slices"

	"github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RemoveRunAsUserRunAsGroup is a Manifestival transformer function that removes runAsUser and runAsGroup
// from a Deployment container's security context
func RemoveRunAsUserRunAsGroup(object *unstructured.Unstructured) error {
	if object.GetKind() != "Deployment" {
		return nil
	}

	deployment := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(object, deployment, nil); err != nil {
		return err
	}

	deployment.Spec.Template.Spec.SecurityContext.RunAsUser = nil
	deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = nil

	for _, container := range deployment.Spec.Template.Spec.Containers {
		container.SecurityContext.RunAsUser = nil
		container.SecurityContext.RunAsGroup = nil
	}

	return scheme.Scheme.Convert(deployment, object, nil)
}

// InjectAnnotations is a Manifestival transformer to add given annotations in resources of provided Kinds.
func InjectAnnotations(kinds, names []string, annotations map[string]string) manifestival.Transformer {
	return func(object *unstructured.Unstructured) error {
		if len(kinds) > 0 && !slices.Contains(kinds, object.GetKind()) {
			return nil
		}
		if len(names) > 0 && !slices.Contains(names, object.GetName()) {
			return nil
		}
		object.SetAnnotations(annotations)

		return nil
	}
}

// InjectFinalizer appends finalizer to the passed resources metadata.
func InjectFinalizer(finalizer string) manifestival.Transformer {
	return func(u *unstructured.Unstructured) error {
		finalizers := u.GetFinalizers()
		if !controllerutil.ContainsFinalizer(u, finalizer) {
			finalizers = append(finalizers, finalizer)
			u.SetFinalizers(finalizers)
		}
		return nil
	}
}
