package e2e

const (
	testingNamespace             = "builds-test"
	openshiftconfigNamespace     = "openshift-config-managed"
	entitlementPod               = "etc-pki-entitlement-test"
	buildRunCrName               = "entitled-br"
	entitlementSecret            = "etc-pki-entitlement"
	entitlementTestPodFile       = "/test/data/entitlement-test-pod.yaml"
	entitledBuildRunFile         = "/test/data/entitled-buildrun.yaml"
	entitledBuildFile            = "/test/data/entitled-build.yaml"
	sharedSecretFile             = "/test/data/shared-secret.yaml"
	sharedSecretClusterRoleFile  = "/test/data/shared-secret-cluster-role.yaml"
	sharedSecretCSIRoleFile      = "/test/data/shared-secret-csi-role.yaml"
	csiDriverRoleBindFile        = "/test/data/csi-driver-role-bind.yaml"
	pipelineBuilderRoleBindFile  = "/test/data/pipeline-builder-role-bind.yaml"
	imageStreamFile              = "/test/data/image-stream.yaml"
	entitlementPodServiceAccount = "/test/data/entitlement-pod-sa-bind.yaml"
)

var entitledPodResources = []string{
	sharedSecretFile,
	sharedSecretClusterRoleFile,
	sharedSecretCSIRoleFile,
	csiDriverRoleBindFile,
	entitlementPodServiceAccount,
}

var entitledBuildResources = []string{
	imageStreamFile,
	sharedSecretFile,
	sharedSecretClusterRoleFile,
	sharedSecretCSIRoleFile,
	csiDriverRoleBindFile,
	pipelineBuilderRoleBindFile,
	entitledBuildFile,
}
