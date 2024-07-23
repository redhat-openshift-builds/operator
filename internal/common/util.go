package common

import "os"

// FetchCurrentNamespaceName returns namespace name by using information stored as file
// Returns default Openshift Builds namespace on error
// Refer: https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#without-using-a-proxy
func FetchCurrentNamespaceName() string {
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		CurrentNamespaceName = OpenShiftBuildNamespaceName
	} else {
		CurrentNamespaceName = string(namespace)
	}
	return CurrentNamespaceName
}
