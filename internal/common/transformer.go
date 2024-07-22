package common

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
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
