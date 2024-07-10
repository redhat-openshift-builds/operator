package common

import (
	"path/filepath"
)

const (
	ShipwrightManifestPathEnv = "SHIPWRIGHT_MANIFEST_PATH"
)

const (
	OpenShiftBuildFinalizerName = "operator.openshift.io/openshiftbuilds"
	OpenShiftBuildCRDName       = "openshiftbuilds.operator.openshift.io"
	OpenShiftBuildResourceName  = "openshiftbuild"
	OpenShiftBuildNamespaceName = "openshift-builds"
	koDataPathEnv               = "KO_DATA_PATH"
)

var (
	ShipwrightBuildManifestPath = filepath.Join("config", "shipwright", "build")
)
