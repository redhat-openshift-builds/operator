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
	"fmt"
	"os"
	"time"

	manifestivalclient "github.com/manifestival/controller-runtime-client"

	"github.com/manifestival/manifestival"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/redhat-openshift-builds/operator/internal/common"
	"github.com/redhat-openshift-builds/operator/internal/sharedresource"
	shipwrightbuild "github.com/redhat-openshift-builds/operator/internal/shipwright/build"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("OpenShiftBuild controller", Label("integration", "openshiftbuild"), Serial, func() {

	When("an empty OpenShiftBuild is created", func() {

		var openshiftBuild *operatorv1alpha1.OpenShiftBuild

		BeforeEach(func(ctx SpecContext) {
			openshiftBuild = &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.OpenShiftBuildResourceName,
				},
				Spec:   operatorv1alpha1.OpenShiftBuildSpec{},
				Status: operatorv1alpha1.OpenShiftBuildStatus{},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(openshiftBuild), openshiftBuild)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, openshiftBuild)).To(Succeed())
			}
		})

		AfterEach(func(ctx SpecContext) {
			Expect(k8sClient.Delete(ctx, openshiftBuild)).To(Succeed())
		})

		It("enables Shipwright Builds", func(ctx SpecContext) {
			buildObj := &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.OpenShiftBuildResourceName,
				},
			}
			Eventually(func() operatorv1alpha1.State {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildObj), buildObj); err != nil {
					return ""
				}
				if buildObj.Spec.Shipwright == nil || buildObj.Spec.Shipwright.Build == nil {
					return ""
				}
				return buildObj.Spec.Shipwright.Build.State
			}).WithContext(ctx).Should(Equal(operatorv1alpha1.Enabled), "check spec.shipwright.build is enabled")
		}, SpecTimeout(1*time.Minute))

		It("enables and deploys Shared Resources", Label("shared-resources"), func(ctx SpecContext) {
			buildObj := &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.OpenShiftBuildResourceName,
				},
			}
			Eventually(func() operatorv1alpha1.State {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildObj), buildObj); err != nil {
					return ""
				}
				if buildObj.Spec.SharedResource == nil {
					return ""
				}
				return buildObj.Spec.SharedResource.State
			}).WithContext(ctx).Should(Equal(operatorv1alpha1.Enabled), "check spec.sharedResource is enabled")

			Eventually(func() error {
				sarRole := &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "csi-driver-shared-resource",
					},
				}
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(sarRole), sarRole)
			}).WithContext(ctx).Should(Succeed(), "deploy shared resources SAR Role")

			Eventually(func() error {
				sarRoleBinding := &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "csi-driver-shared-resource",
					},
				}
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(sarRoleBinding), sarRoleBinding)
			}).WithContext(ctx).Should(Succeed(), "deploy shared resources SAR Role")

		}, SpecTimeout(1*time.Minute))
	})
})

var _ = Describe("Main Operator Controller with Sub-Reconciler Failure", func() {
	const (
		CRName      = "test-shipwright-build"
		CRNamespace = "openshift-builds"
		timeout     = time.Second * 10
		interval    = time.Millisecond * 250
	)

	ctx := context.Background()

	var openShiftBuildReconciler *OpenShiftBuildReconciler

	Context("When ShipwrightBuild sub-component reconciliation fails", func() {
		It("Should requeue the main operator reconciliation", func() {
			By("Creating an OpenShiftBuild instance with invalid Shipwright Build state")

			openShiftBuildReconciler = &OpenShiftBuildReconciler{
				Client:         k8sClient,
				Scheme:         testEnv.Scheme,
				Logger:         ctrl.Log.WithName("test-openshiftbuild-reconciler"),
				APIReader:      k8sClient,
				SharedResource: &sharedresource.SharedResource{},
				Shipwright:     shipwrightbuild.New(k8sClient, CRNamespace),
			}

			cr := &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: CRName,
				},
				Spec: operatorv1alpha1.OpenShiftBuildSpec{
					Shipwright: &operatorv1alpha1.Shipwright{
						Build: &operatorv1alpha1.ShipwrightBuild{
							State: operatorv1alpha1.Enabled,
						},
					},
					SharedResource: &operatorv1alpha1.SharedResource{
						State: operatorv1alpha1.Disabled,
					},
				},
				Status: operatorv1alpha1.OpenShiftBuildStatus{
					Conditions: []metav1.Condition{
						{
							Type:   operatorv1alpha1.ConditionReady,
							Status: metav1.ConditionFalse,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Ensure the CR is retrievable before reconciling, so the Get in Reconcile works
			crKey := types.NamespacedName{Name: CRName, Namespace: CRNamespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, crKey, &operatorv1alpha1.OpenShiftBuild{})
			}, timeout, interval).Should(Succeed())

			By("Directly invoking the main Reconcile function")
			req := ctrl.Request{
				NamespacedName: crKey,
			}

			result, err := openShiftBuildReconciler.Reconcile(ctx, req)

			By("Asserting that an error was returned (triggering requeue)")
			Expect(err).To(HaveOccurred(), "Main Reconcile should return an error when sub-component 1 fails")

			By("Asserting that the result does not ask for RequeueAfter (default requeue-on-error)")
			Expect(result.Requeue).To(BeFalse(), "Result.Requeue should be false when an error is returned for default backoff requeue")
			Expect(result.RequeueAfter).To(BeZero(), "Result.RequeueAfter should be zero for default backoff requeue")

			// Cleanup
			By("Deleting the CR for SharedResource failure test")
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		})
	})

	Context("When SharedResource sub-component reconciliation fails", func() {
		It("Should requeue the main operator reconciliation", func() {
			By("Creating an OpenShiftBuild instance configured to make SharedResource reconciler fail")
			sharedManifestPath := common.SharedResourceManifestPath
			if path, ok := os.LookupEnv(common.SharedResourceManifestPath); ok {
				sharedManifestPath = path
			}
			sharedManifest, err := manifestival.NewManifest(sharedManifestPath, []manifestival.Option{
				manifestival.UseLogger(ctrl.Log.WithName("test-openshiftbuild-reconciler")),
				manifestival.UseClient(manifestivalclient.NewClient(k8sClient)),
			}...)
			if err != nil {
				fmt.Println("Error creating shared manifest", err)
			}

			openShiftBuildReconciler = &OpenShiftBuildReconciler{
				Client:         k8sClient,
				Scheme:         testEnv.Scheme,
				Logger:         ctrl.Log.WithName("test-openshiftbuild-reconciler"),
				APIReader:      k8sClient,
				SharedResource: sharedresource.New(k8sClient, sharedManifest),
				Shipwright:     shipwrightbuild.New(k8sClient, CRNamespace),
			}

			cr := &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: CRName,
				},
				Spec: operatorv1alpha1.OpenShiftBuildSpec{
					Shipwright: &operatorv1alpha1.Shipwright{
						Build: &operatorv1alpha1.ShipwrightBuild{
							State: operatorv1alpha1.Disabled,
						},
					},
					SharedResource: &operatorv1alpha1.SharedResource{
						State: operatorv1alpha1.Enabled,
					},
				},
				Status: operatorv1alpha1.OpenShiftBuildStatus{
					Conditions: []metav1.Condition{
						{
							Type:   operatorv1alpha1.ConditionReady,
							Status: metav1.ConditionFalse,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			crKey := types.NamespacedName{Name: CRName, Namespace: CRNamespace}
			Eventually(func() error {
				return k8sClient.Get(ctx, crKey, &operatorv1alpha1.OpenShiftBuild{})
			}, timeout, interval).Should(Succeed())

			By("Directly invoking the main Reconcile function")
			req := ctrl.Request{
				NamespacedName: crKey,
			}

			result, err := openShiftBuildReconciler.Reconcile(ctx, req)

			By("Asserting that an error was returned (triggering requeue)")
			Expect(err).To(HaveOccurred(), "Main Reconcile should return an error when SharedResource sub-component fails")

			Expect(err.Error()).To(ContainSubstring("SharedResources reconciliation failed"))

			By("Asserting that the result does not ask for RequeueAfter (default requeue-on-error)")
			Expect(result.Requeue).To(BeFalse(), "Result.Requeue should be false when an error is returned for default backoff requeue")
			Expect(result.RequeueAfter).To(BeZero(), "Result.RequeueAfter should be zero for default backoff requeue")

			// Cleanup
			By("Deleting the CR for SharedResource failure test")
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		})
	})
})
