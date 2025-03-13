package registry

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var allowedRegistries = []string{
	"registry.redhat.io",
	"registry.access.redhat.com",
	"brew.registry.redhat.io",
	"registry.stage.redhat.io",
}

type validationError struct {
	File  string
	Image string
}

func TestRegistryValidation(t *testing.T) {
	t.Run("should fail if image references are not from Red Hat approved registries", func(t *testing.T) {
		// Get project root
		projectRoot, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		if strings.HasSuffix(projectRoot, "test/integration/registry") {
			projectRoot = filepath.Join(projectRoot, "../../..")
		}

		// Find all YAML files
		yamlFiles, err := findYAMLFiles(projectRoot)
		if err != nil {
			t.Fatalf("Failed to find YAML files: %v", err)
		}

		var validationErrors []validationError

		// Check each YAML file
		for _, file := range yamlFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", file, err)
				continue
			}

			var node yaml.Node
			err = yaml.Unmarshal(content, &node)
			if err != nil {
				t.Errorf("Failed to parse YAML in file %s: %v", file, err)
				continue
			}

			var images []string
			findImageReferences(&node, &images)

			// Validate each image reference
			for _, image := range images {
				if image != "" && !validateRegistry(image) {
					validationErrors = append(validationErrors, validationError{
						File:  file,
						Image: image,
					})
				}
			}
		}

		// Print summary and fail test if there are validation errors
		if len(validationErrors) > 0 {
			t.Errorf("\nTotal invalid references Found: %d", len(validationErrors))
			for i, err := range validationErrors {
				t.Errorf("%d: File: %s\nInvalid image reference: %s\n", i+1, err.File, err.Image)
			}

			t.Fatal("Test failed due to invalid registry references")
		}
	})
}

func findYAMLFiles(root string) ([]string, error) {
	var files []string
	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}

func findImageReferences(node *yaml.Node, images *[]string) {
	if node == nil {
		return
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			// Check for common image-related keys
			if key.Value == "image" || key.Value == "from" || strings.Contains(key.Value, "Image") {
				if value.Kind == yaml.ScalarNode {
					*images = append(*images, value.Value)
				}
			}
		}
	}

	// Recursively check all child nodes
	for _, child := range node.Content {
		findImageReferences(child, images)
	}
}

// validateRegistry checks if the image reference uses an approved registry
func validateRegistry(image string) bool {
	for _, registry := range allowedRegistries {
		if strings.HasPrefix(image, registry+"/") {
			return true
		}
	}
	return false
}
