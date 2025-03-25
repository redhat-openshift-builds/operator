package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/manifestival/manifestival"

	mfc "github.com/manifestival/controller-runtime-client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultConfigPath is the base path where manifests are stored in the container
	defaultConfigPath = "/config"
)

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

// SetupManifestival instantiates a Manifestival instance for the provided file or directory
// fileOrDir should be relative to the config directory
func SetupManifestival(client client.Client, fileOrDir string, recurse bool, logger logr.Logger) (manifestival.Manifest, error) {
	mfclient := mfc.NewClient(client)
	currentPath, err := os.Getwd()
	if err != nil {
		return manifestival.Manifest{}, fmt.Errorf("failed to get base path: %w", err)
	}

	basePath := currentPath
	for {
		if filepath.Base(basePath) == "operator" {
			break
		}
		parent := filepath.Dir(basePath)
		if parent == basePath {
			return manifestival.Manifest{}, fmt.Errorf("operator directory not found in path")
		}
		basePath = parent
	}

	manifest := filepath.Join(basePath, defaultConfigPath, fileOrDir)
	var src manifestival.Source
	if recurse {
		src = manifestival.Recursive(manifest)
	} else {
		src = manifestival.Path(manifest)
	}

	return manifestival.ManifestFrom(src,
		manifestival.UseClient(mfclient),
		manifestival.UseLogger(logger))
}

// GetConfigPath returns the full path for a given manifest file or directory
func GetConfigPath(relativePath string) string {
	return filepath.Join(defaultConfigPath, relativePath)
}
