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

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
)

func waitForCRDExists(ctx context.Context, crdName string, timeout time.Duration) error {
	crd := &extv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
	}
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = kubeClient.Get(ctx, client.ObjectKeyFromObject(crd), crd)
			if errors.IsNotFound(err) {
				done = false
				return
			}
			// If error is not nil, then something went wrong.
			// Otherwise the CRD exists and we carry on.
			done = true
			return
		})
	return err
}

func waitForDeploymentReady(ctx context.Context, namespace string, name string, timeout time.Duration) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = kubeClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
			if errors.IsNotFound(err) {
				done = false
				return
			}
			if err != nil {
				done = true
				return
			}
			done = (deployment.Status.ReadyReplicas > 0)
			return
		})
	return err
}

func waitForDaemonSetReady(ctx context.Context, namespace string, name string, timeout time.Duration) error {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = kubeClient.Get(ctx, client.ObjectKeyFromObject(daemonSet), daemonSet)
			if errors.IsNotFound(err) {
				done = false
				return
			}
			if err != nil {
				done = true
				return
			}
			done = (daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled)
			return
		})
	return err
}

var _ = Describe("Builds for OpenShift operator", Label("e2e"), Label("operator"), func() {

	When("the operator has been deployed", func() {

		BeforeEach(func(ctx SpecContext) {
			err := waitForDeploymentReady(ctx, "openshift-builds", "openshift-builds-operator", 10*time.Minute)
			Expect(err).NotTo(HaveOccurred(), "checking if operator deployment is ready")
		})

		It("should create an OpenShiftBuild CR with defaults enabled", Label("shipwright"), Label("shared-resources"),
			func(ctx SpecContext) {
				obInstance := &operatorv1alpha1.OpenShiftBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
				}
				err := kubeClient.Get(ctx, client.ObjectKeyFromObject(obInstance), obInstance)
				Expect(err).NotTo(HaveOccurred(), "getting OpenShiftBuild instance")
				Expect(obInstance.Spec.Shipwright.Build.State).To(Equal(operatorv1alpha1.Enabled))
				Expect(obInstance.Spec.SharedResource.State).To(Equal(operatorv1alpha1.Enabled))
			})

		It("should create a ShipwrightBuild CR with correct defaults", Label("shipwright"), Label("builds"),
			func(ctx SpecContext) {
				shpInstances := &v1alpha1.ShipwrightBuildList{}
				listOpts := &client.ListOptions{
					Limit: 10,
				}
				err := kubeClient.List(ctx, shpInstances, listOpts)
				Expect(err).NotTo(HaveOccurred(), "getting ShipwrightBuild instances")
				instancesCount := len(shpInstances.Items)
				Expect(instancesCount).To(Equal(1), "checking ShipwrightBuild instances")
				if instancesCount == 0 {
					return
				}
				shpInstance := shpInstances.Items[0]
				Expect(len(shpInstance.OwnerReferences)).To(Equal(1), "checking owner reference count")
				Expect(shpInstance.Spec.TargetNamespace).To(Equal("openshift-builds"), "checking target namespace")
			})

		It("should deploy the Shared Resource CSI Driver", Label("shared-resources"),
			func(ctx SpecContext) {
				err := waitForCRDExists(ctx, "sharedconfigmaps.sharedresource.openshift.io", 10*time.Minute)
				Expect(err).NotTo(HaveOccurred(), "checking Shared Resource CRDs")
				err = waitForCRDExists(ctx, "sharedsecrets.sharedresource.openshift.io", 10*time.Minute)
				Expect(err).NotTo(HaveOccurred(), "checking Shared Resource CRDs")
				err = waitForDaemonSetReady(ctx, "openshift-builds", "shared-resource-csi-driver-node", 10*time.Minute)
				Expect(err).NotTo(HaveOccurred(), "checking Shared Resource CSI Driver DaemonSet")
				err = waitForDeploymentReady(ctx, "openshift-builds", "shared-resource-csi-driver-webhook", 10*time.Minute)
				Expect(err).NotTo(HaveOccurred(), "checking Shared Resource CSI Driver webhook")
			})
	})
})
