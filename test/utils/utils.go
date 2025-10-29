/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"bytes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	prometheusOperatorVersion = "v0.68.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.5.3"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}

// ApplyResourceFromFile reads a YAML file, decodes it into a Kubernetes resource, and applies it to the cluster.
func ApplyResourceFromFile(ctx context.Context, kubeClient client.Client, filePath string) error {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	yamlDocs := strings.Split(string(data), "\n---\n")

	for _, docString := range yamlDocs {
		docString = strings.TrimSpace(docString)
		if docString == "" {
			continue
		}

		// Decode YAML into an unstructured object
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(docString), obj); err != nil {
			return fmt.Errorf("failed to decode YAML document from %s: %w", filePath, err)
		}

		// Apply the object. If creation fails because it already exists, try an update.
		if err := kubeClient.Create(ctx, obj); err != nil {
			if errors.IsAlreadyExists(err) {
				// Get the existing object to retrieve its resourceVersion for the update.
				existingObj := &unstructured.Unstructured{}
				existingObj.SetGroupVersionKind(obj.GroupVersionKind())
				if getErr := kubeClient.Get(ctx, client.ObjectKeyFromObject(obj), existingObj); getErr != nil {
					return fmt.Errorf("failed to get existing resource for update from %s: %w", filePath, getErr)
				}
				obj.SetResourceVersion(existingObj.GetResourceVersion())

				if updateErr := kubeClient.Update(ctx, obj); updateErr != nil {
					return fmt.Errorf("failed to update resource from %s: %w", filePath, updateErr)
				}
			} else {
				return fmt.Errorf("failed to create resource from %s: %w", filePath, err)
			}
		}
	}
	return nil
}

// DeleteResourceFromFile reads a YAML file, decodes it into a Kubernetes resource, and deletes it from the cluster.
func DeleteResourceFromFile(ctx context.Context, kubeClient client.Client, filePath string) error {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	// Decode YAML into an unstructured object
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to decode YAML for file %q: %w", filePath, err)
	}

	// Delete the resource
	if err := kubeClient.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete resource %s/%s (%s): %w",
			obj.GetNamespace(), obj.GetName(), obj.GetKind(), err)
	}

	return nil
}

// ExecInPod runs a command inside a specific container and returns the output.
func ExecInPod(clientset kubernetes.Interface, kubeConfig *rest.Config, podName, namespace string, command ...string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: "test-container", Command: command,
		Stdin: false, Stdout: true, Stderr: true,
	}, scheme.ParameterCodec)

	// Use the kubeConfig that was passed in
	exec, err := remotecommand.NewSPDYExecutor(kubeConfig, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), nil
}
