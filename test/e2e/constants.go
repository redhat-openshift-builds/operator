package e2e

const (
	testingNamespace                = "builds-test"
	openshiftBuildsNamespace        = "rh-openshift-builds-tenant"
	entitlementPod                  = "etc-pki-entitlement-test"
	buildRunCrName                  = "entitled-br"
	buildCrName                     = "buildah-rhel"
	entitlementSecret               = "entitlement-key"
	entitlementTestPodFile          = "/test/data/entitlement-test-pod.yaml"
	entitledBuildRunFile            = "/test/data/entitled-buildrun.yaml"
	entitledBuildFile               = "/test/data/entitled-build.yaml"
	sharedSecretFile                = "/test/data/shared-secret.yaml"
	sharedSecretClusterRoleFile     = "/test/data/shared-secret-cluster-role.yaml"
	sharedSecretClusterRoleBindFile = "/test/data/shared-secret-cluster-role-bind.yaml"
	csiDriverSecretsRoleFile        = "/test/data/csi-driver-secrets-role.yaml"
	csiDriverSecretsRoleBindFile    = "/test/data/csi-driver-secrets-role-bind.yaml"
	pipelineBuilderRoleBindFile     = "/test/data/pipeline-builder-role-bind.yaml"
	imageStreamFile                 = "/test/data/image-stream.yaml"
)

var entitledPodResources = []string{
	sharedSecretFile,
	sharedSecretClusterRoleFile,
	sharedSecretClusterRoleBindFile,
	csiDriverSecretsRoleFile,
	csiDriverSecretsRoleBindFile,
}

var entitledBuildResources = []string{
	imageStreamFile,
	sharedSecretFile,
	sharedSecretClusterRoleFile,
	sharedSecretClusterRoleBindFile,
	csiDriverSecretsRoleFile,
	csiDriverSecretsRoleBindFile,
	pipelineBuilderRoleBindFile,
	entitledBuildFile,
}
