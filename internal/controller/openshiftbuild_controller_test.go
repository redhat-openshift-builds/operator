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

package controller

import (
	"context"

	"github.com/redhat-openshift-builds/operator/internal/common"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
)

var _ = Describe("OpenShiftBuild Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		namespacedName := types.NamespacedName{
			Name: common.OpenShiftBuildResourceName,
		}
		openShiftBuild := &operatorv1alpha1.OpenShiftBuild{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind OpenShiftBuild")
			err := k8sClient.Get(ctx, namespacedName, openShiftBuild)
			if err != nil && errors.IsNotFound(err) {
				resource := &operatorv1alpha1.OpenShiftBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name: common.OpenShiftBuildResourceName,
					},
					Spec: operatorv1alpha1.OpenShiftBuildSpec{
						Shipwright: &operatorv1alpha1.Shipwright{
							Build: &operatorv1alpha1.ShipwrightBuild{
								State: "Enabled",
							},
						},
						SharedResource: &operatorv1alpha1.SharedResource{
							State: "Enabled",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &operatorv1alpha1.OpenShiftBuild{}
			err := k8sClient.Get(ctx, namespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OpenShiftBuild")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			reconciler := &OpenShiftBuildReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
