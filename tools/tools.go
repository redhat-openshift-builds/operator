//go:build tools
// +build tools

// This is required to track and prefetch dependencies during build
// TODO: Use "tool" directive in go.mod when moving to go 1.24
package tools

import (
	_ "sigs.k8s.io/controller-runtime/tools/setup-envtest"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	_ "sigs.k8s.io/kustomize/kustomize/v5"
)
