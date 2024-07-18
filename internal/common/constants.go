package common

import (
	"path/filepath"
)

const (
	OpenShiftBuildFinalizerName = "operator.openshift.io/openshiftbuilds"
	OpenShiftBuildCRDName       = "openshiftbuilds.operator.openshift.io"
	OpenShiftBuildResourceName  = "cluster"
	OpenShiftBuilNamespaceName  = "openshift-builds"
)

const (
	ShipwrightBuildCRDName                 = "shipwrightbuilds.operator.shipwright.io"
	ShipwrightBuildManifestPathEnv         = "SHIPWRIGHT_BUILD_MANIFEST_PATH"
	ShipwrightBuildStrategyManifestPathEnv = "SHIPWRIGHT_BUILD_STRATEGY_MANIFEST_PATH"
)

var (
	ShipwrightBuildManifestPath         = filepath.Join("config", "shipwright", "build", "release")
	ShipwrightBuildStrategyManifestPath = filepath.Join("config", "shipwright", "build", "strategy")
	CurrentNamespaceName                string
)
