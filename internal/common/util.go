package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	mfc "github.com/manifestival/controller-runtime-client"
	"github.com/manifestival/manifestival"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupManifestival instantiates a Manifestival instance for the provided file or directory
func SetupManifestival(client client.Client, fileOrDir string, recurse bool, logger logr.Logger) (manifestival.Manifest, error) {
	mfclient := mfc.NewClient(client)

	dataPath, err := KoDataPath()
	if err != nil {
		return manifestival.Manifest{}, err
	}
	manifest := filepath.Join(dataPath, fileOrDir)
	var src manifestival.Source
	if recurse {
		src = manifestival.Recursive(manifest)
	} else {
		src = manifestival.Path(manifest)
	}
	return manifestival.ManifestFrom(src, manifestival.UseClient(mfclient), manifestival.UseLogger(logger))
}

// KoDataPath retrieves the data path environment variable, returning error when not found.
func KoDataPath() (string, error) {
	dataPath, exists := os.LookupEnv(koDataPathEnv)
	if !exists {
		return "", fmt.Errorf("'%s' is not set", koDataPathEnv)
	}
	return dataPath, nil
}
