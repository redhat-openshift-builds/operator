package common

import (
	"os"

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

// FetchCurrentNamespaceName returns namespace name by using information stored as file
// Returns default Openshift Builds namespace on error
// Refer: https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#without-using-a-proxy
func FetchCurrentNamespaceName() string {
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		CurrentNamespaceName = OpenShiftBuilNamespaceName
	} else {
		CurrentNamespaceName = string(namespace)
	}
	return CurrentNamespaceName
}
