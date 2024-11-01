/*
Copyright 2024 Red Hat, Inc.

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

package setup

import (
	"context"
	"embed"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

//go:embed manifests/*
var manifests embed.FS

type OperatorManager struct {
	kubeClient client.Client
}

func NewOperatorManager(kubeClient client.Client) *OperatorManager {
	return &OperatorManager{
		kubeClient: kubeClient,
	}
}

func (m *OperatorManager) createFromManifest(ctx context.Context, file string, obj *unstructured.Unstructured) error {
	manifest, err := manifests.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(manifest, obj)
	if err != nil {
		return err
	}
	return m.kubeClient.Create(ctx, obj)
}

// InstallBuildsForOpenShift ensures that Builds for OpenShift and its operands have been installed
// and deployed on the cluster.
func (m *OperatorManager) InstallBuildsForOpenShift(ctx context.Context) error {
	subscription := m.buildsSubscriptionStub()
	opLog := crlog.FromContext(ctx).WithValues("namespace", subscription.GetNamespace(), "group", "operators.coreos.com")
	subLog := opLog.WithValues("kind", "Subscription", "name", subscription.GetName())
	err := m.kubeClient.Get(ctx, client.ObjectKeyFromObject(subscription), subscription)
	if kerrors.IsNotFound(err) {
		if err := m.createBuildsSubscription(ctx, subscription); err != nil {
			return fmt.Errorf("failed to create Builds for OpenShift subscription: %v", err)
		}
		subLog.Info("created object")
	}
	subLog.Info("found object")
	installedCSV, err := m.waitForInstalledCSV(ctx, subscription, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("could not determine Builds for OpenShift installedCSV: %v", err)
	}
	csvLog := opLog.WithValues("kind", "ClusterServiceVersion", "name", installedCSV)
	csvLog.Info("determined installedCSV")
	phase, err := m.waitForCSVSucceeded(ctx, installedCSV, 10*time.Minute)
	csvLog.WithValues("phase", phase).Info("CSV phase determined")
	if err != nil {
		return fmt.Errorf("CSV %s install did not succeed, reached phase %s: %v", installedCSV, phase, err)
	}
	return nil
}

// waitForInstalledCSV polls the provided Subscription object until `status.installedCSV` returns a
// value.
func (m *OperatorManager) waitForInstalledCSV(ctx context.Context, subscription *unstructured.Unstructured,
	timeout time.Duration) (string, error) {
	var installedCSV string
	// Poll up to the provided timeout
	waitErr := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = m.kubeClient.Get(ctx, client.ObjectKeyFromObject(subscription), subscription)
			if kerrors.IsNotFound(err) {
				// Keep polling if the subscription object is not found.
				done = false
				return
			}
			if err != nil {
				done = true
				return
			}
			// done is `true` if the value is found for .status.installedCSV
			installedCSV, done, err = unstructured.NestedString(subscription.Object, "status", "installedCSV")
			// Continue polling if installedCSV is the empty string
			if len(installedCSV) == 0 {
				done = false
			}
			return
		})
	return installedCSV, waitErr
}

// waitForCSVSucceeded polls the named CSV's object until `status.phase`
func (m *OperatorManager) waitForCSVSucceeded(ctx context.Context, installedCSV string, timeout time.Duration) (string,
	error) {
	csv := &unstructured.Unstructured{}
	csv.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	})
	csv.SetNamespace("openshift-builds")
	csv.SetName(installedCSV)

	var phase string
	waitErr := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = m.kubeClient.Get(ctx, client.ObjectKeyFromObject(csv), csv)
			if kerrors.IsNotFound(err) {
				// Keep polling if the subscription object is not found.
				done = false
				return
			}
			if err != nil {
				done = true
				return
			}
			// done is `true` if the value is found for .status.phase
			phase, done, err = unstructured.NestedString(csv.Object, "status", "phase")
			// Continue polling the phase is not "Succeeded" (includes empty value)
			if phase != "Succeeded" {
				done = false
			}
			return
		})
	return phase, waitErr
}

func (m *OperatorManager) buildsSubscriptionStub() *unstructured.Unstructured {
	subscription := &unstructured.Unstructured{}
	subscription.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "Subscription",
	})
	subscription.SetNamespace("openshift-builds")
	subscription.SetName("openshift-builds-operator")
	return subscription
}

func (m *OperatorManager) createBuildsSubscription(ctx context.Context, subscription *unstructured.Unstructured) error {
	return m.createFromManifest(ctx, "manifests/subscription-openshift-builds.yaml", subscription)
}
