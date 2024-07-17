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
	OpenShiftBuildResourceName  = "cluster"
	OpenShiftBuildNamespaceName = "openshift-builds"
)

var (
	ShipwrightBuildManifestPath = filepath.Join("config", "shipwright", "build")
)
