package common

import (
	"slices"
	"strings"

	"github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RemoveRunAsUserRunAsGroup is a Manifestival transformer function that removes runAsUser and runAsGroup
// from a Deployment container's security context
func RemoveRunAsUserRunAsGroup(object *unstructured.Unstructured) error {
	if object.GetKind() != "Deployment" {
		return nil
	}

	deployment := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(object, deployment, nil); err != nil {
		return err
	}

	deployment.Spec.Template.Spec.SecurityContext.RunAsUser = nil
	deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = nil

	for _, container := range deployment.Spec.Template.Spec.Containers {
		container.SecurityContext.RunAsUser = nil
		container.SecurityContext.RunAsGroup = nil
	}

	return scheme.Scheme.Convert(deployment, object, nil)
}

// InjectAnnotations is a Manifestival transformer to add given annotations in resources of provided Kinds.
func InjectAnnotations(kinds, names []string, annotations map[string]string) manifestival.Transformer {
	return func(object *unstructured.Unstructured) error {
		if len(kinds) > 0 && !slices.Contains(kinds, object.GetKind()) {
			return nil
		}
		if len(names) > 0 && !slices.Contains(names, object.GetName()) {
			return nil
		}
		object.SetAnnotations(annotations)

		return nil
	}
}

// InjectFinalizer appends finalizer to the passed resources metadata.
func InjectFinalizer(finalizer string) manifestival.Transformer {
	return func(u *unstructured.Unstructured) error {
		finalizers := u.GetFinalizers()
		if !controllerutil.ContainsFinalizer(u, finalizer) {
			finalizers = append(finalizers, finalizer)
			u.SetFinalizers(finalizers)
		}
		return nil
	}
}

// InjectTLSArgsIntoContainer injects --tls-min-version and --tls-cipher-suites flags into a named container.
// Removes any existing TLS args before injecting new ones.
// Works on any Deployment that has a container with the specified name.
func InjectTLSArgsIntoContainer(containerName, minVersion, cipherSuites string) manifestival.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" {
			return nil
		}

		deploy := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(u, deploy, nil); err != nil {
			return err
		}

		for i, container := range deploy.Spec.Template.Spec.Containers {
			if container.Name == containerName {
				// Remove existing TLS args
				args := filterOutTLSArgs(container.Args)

				// Inject new TLS args if provided
				if minVersion != "" {
					args = append(args, "--tls-min-version="+minVersion)
				}
				if cipherSuites != "" {
					args = append(args, "--tls-cipher-suites="+cipherSuites)
				}

				deploy.Spec.Template.Spec.Containers[i].Args = args
				break
			}
		}

		return scheme.Scheme.Convert(deploy, u, nil)
	}
}

// filterOutTLSArgs removes --tls-min-version and --tls-cipher-suites from args list
func filterOutTLSArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--tls-min-version=") && !strings.HasPrefix(arg, "--tls-cipher-suites=") {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}
