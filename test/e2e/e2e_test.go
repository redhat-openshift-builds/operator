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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

var _ = Describe("Builds for OpenShift operator", Label("e2e"), func() {

	When("the operator has been deployed", func() {

		BeforeEach(func(ctx SpecContext) {
			err := waitForDeploymentReady(ctx, "openshift-builds", "openshift-builds-operator", 10*time.Minute)
			Expect(err).NotTo(HaveOccurred(), "checking if operator deployment is ready")
		})

		It("should create an OpenShiftBuild CRD with defaults enabled", func() {
			Fail("not implemented yet")
		})
	})
})
