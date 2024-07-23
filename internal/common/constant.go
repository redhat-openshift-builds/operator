package common

import "path/filepath"

const (
	OpenShiftBuildFinalizerName   = "operator.openshift.io/openshiftbuilds"
	OpenShiftBuildOperatorCRDName = "openshiftbuilds.operator.openshift.io"
	OpenShiftBuildResourceName    = "cluster"
	OpenShiftBuildNamespaceName   = "openshift-builds"
)

const (
	ShipwrightBuildOperatorCRDName         = "shipwrightbuilds.operator.shipwright.io"
	ShipwrightBuildManifestPathEnv         = "SHIPWRIGHT_BUILD_MANIFEST_PATH"
	ShipwrightBuildStrategyManifestPathEnv = "SHIPWRIGHT_BUILD_STRATEGY_MANIFEST_PATH"
	ShipwrightWebhookServiceName           = "shp-build-webhook"
	ShipwrightWebhookCertSecretName        = "shipwright-build-webhook-cert"
)

var (
	ShipwrightBuildManifestPath         = filepath.Join("config", "shipwright", "build", "release")
	ShipwrightBuildStrategyManifestPath = filepath.Join("config", "shipwright", "build", "strategy")
	ShipwrightBuildCRDNames             = []string{
		"builds.shipwright.io",
		"buildruns.shipwright.io",
		"buildstrategies.shipwright.io",
		"clusterbuildstrategies.shipwright.io",
	}
)

var (
	CurrentNamespaceName string
)
