package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsControlledBy returns true if the Controller Reference matches the same group, version and kind.
func IsControlledBy(object client.Object, owner *metav1.OwnerReference) bool {
	controller := metav1.GetControllerOf(object)
	return controller != nil && *controller.Controller &&
		controller.APIVersion == owner.APIVersion &&
		controller.Kind == owner.Kind
}
