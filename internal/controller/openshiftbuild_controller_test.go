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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
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

		It("enables Shared Resources", func(ctx SpecContext) {
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
		}, SpecTimeout(1*time.Minute))
	})
})

// 	When("bootstrapping the OpenShiftBuild resource", func() {

// 		AfterEach(func() {
// 			// Cleanup the created OpenShiftBuild resource
// 			obj := &operatorv1alpha1.OpenShiftBuild{}
// 			err := k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, obj)
// 			Expect(err).NotTo(HaveOccurred(), "get created OpenShiftBuild resource")

// 			controllerutil.RemoveFinalizer(obj, finalizerName)
// 			err = k8sClient.Update(ctx, obj)
// 			Expect(err).NotTo(HaveOccurred(), "remove finalizer from OpenShiftBuild resource")

// 			err = k8sClient.Delete(ctx, obj)
// 			Expect(err).NotTo(HaveOccurred(), "delete OpenShiftBuild resource")
// 		})

// 		It("should create the OpenShiftBuild resource if not present", func() {
// 			err := reconciler.BootstrapOpenShiftBuild(ctx, k8sClient)
// 			Expect(err).NotTo(HaveOccurred(), "boostrap OpenShiftBuild resource")

// 			resultObj := &operatorv1alpha1.OpenShiftBuild{}
// 			err = k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, resultObj)
// 			Expect(err).NotTo(HaveOccurred(), "get created OpenShiftBuild object")
// 			validateDefaults(resultObj)
// 		})

// 		It("should set required fields with default values", func() {
// 			// Create empty spec object
// 			buildObj := &operatorv1alpha1.OpenShiftBuild{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: common.OpenShiftBuildResourceName,
// 				},
// 				Spec: operatorv1alpha1.OpenShiftBuildSpec{},
// 			}
// 			err := k8sClient.Create(ctx, buildObj)
// 			Expect(err).NotTo(HaveOccurred(), "create empty OpenShiftBuild resource")

// 			err = reconciler.BootstrapOpenShiftBuild(ctx, k8sClient)
// 			Expect(err).NotTo(HaveOccurred(), "boostrap OpenShiftBuild resource")

// 			// Re-fetch an object to get updated values
// 			resultObj := &operatorv1alpha1.OpenShiftBuild{}
// 			err = k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, resultObj)
// 			Expect(err).NotTo(HaveOccurred(), "get updated OpenShiftBuild object")
// 			validateDefaults(resultObj)
// 		})

// 	})
// })

// func validateDefaults(obj *operatorv1alpha1.OpenShiftBuild) {
// 	Expect(obj.Spec.Shipwright).NotTo(BeNil())
// 	Expect(obj.Spec.Shipwright.Build).NotTo(BeNil())
// 	Expect(obj.Spec.Shipwright.Build.State).To(Equal(operatorv1alpha1.Enabled))
// 	Expect(controllerutil.ContainsFinalizer(obj, common.OpenShiftBuildFinalizerName)).To(BeTrue(), "checking for finalizer %q", common.OpenShiftBuildFinalizerName)

// 	Expect(obj.Spec.SharedResource).NotTo(BeNil())
// 	Expect(obj.Spec.SharedResource.State).To(Equal(operatorv1alpha1.Enabled))
// }
