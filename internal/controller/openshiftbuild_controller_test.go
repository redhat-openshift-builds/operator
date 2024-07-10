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

	shipwrightbuild "github.com/redhat-openshift-builds/operator/internal/shipwright/build"

	"github.com/redhat-openshift-builds/operator/internal/common"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
)

const finalizerName = common.OpenShiftBuildFinalizerName

var _ = Describe("OpenShiftBuild Controller", Label("controller", "openshiftbuild"), func() {

	var (
		reconciler *OpenShiftBuildReconciler
		ctx        context.Context
	)

	BeforeEach(func() {
		reconciler = &OpenShiftBuildReconciler{
			APIReader:  k8sClient,
			Client:     k8sClient,
			Scheme:     k8sClient.Scheme(),
			Shipwright: shipwrightbuild.New(k8sClient),
		}
		ctx = context.Background()
	})

	When("reconciling an OpenShiftBuild resource", func() {

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
								State: operatorv1alpha1.Enabled,
							},
						},
						SharedResource: &operatorv1alpha1.SharedResource{
							State: operatorv1alpha1.Enabled,
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
			Expect(err).NotTo(HaveOccurred(), "get OpenShiftBuild resource")

			controllerutil.RemoveFinalizer(resource, finalizerName)
			Expect(k8sClient.Update(ctx, resource)).To(Succeed(), "remove finalizer from OpenShiftBuild resource")

			err = k8sClient.Delete(ctx, resource)
			Expect(err).NotTo(HaveOccurred(), "delete OpenShiftBuild resource")
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	When("bootstrapping the OpenShiftBuild resource", func() {

		AfterEach(func() {
			// Cleanup the created OpenShiftBuild resource
			obj := &operatorv1alpha1.OpenShiftBuild{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, obj)
			Expect(err).NotTo(HaveOccurred(), "get created OpenShiftBuild resource")

			controllerutil.RemoveFinalizer(obj, finalizerName)
			err = k8sClient.Update(ctx, obj)
			Expect(err).NotTo(HaveOccurred(), "remove finalizer from OpenShiftBuild resource")

			err = k8sClient.Delete(ctx, obj)
			Expect(err).NotTo(HaveOccurred(), "delete OpenShiftBuild resource")
		})

		It("should create the OpenShiftBuild resource if not present", func() {
			err := reconciler.BootstrapOpenShiftBuild(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred(), "boostrap OpenShiftBuild resource")

			resultObj := &operatorv1alpha1.OpenShiftBuild{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, resultObj)
			Expect(err).NotTo(HaveOccurred(), "get created OpenShiftBuild object")
			validateDefaults(resultObj)
		})

		It("should set required fields with default values", func() {
			// Create empty spec object
			buildObj := &operatorv1alpha1.OpenShiftBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.OpenShiftBuildResourceName,
				},
				Spec: operatorv1alpha1.OpenShiftBuildSpec{},
			}
			err := k8sClient.Create(ctx, buildObj)
			Expect(err).NotTo(HaveOccurred(), "create empty OpenShiftBuild resource")

			err = reconciler.BootstrapOpenShiftBuild(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred(), "boostrap OpenShiftBuild resource")

			// Re-fetch an object to get updated values
			resultObj := &operatorv1alpha1.OpenShiftBuild{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: common.OpenShiftBuildResourceName}, resultObj)
			Expect(err).NotTo(HaveOccurred(), "get updated OpenShiftBuild object")
			validateDefaults(resultObj)
		})

	})
})

func validateDefaults(obj *operatorv1alpha1.OpenShiftBuild) {
	Expect(obj.Spec.Shipwright).NotTo(BeNil())
	Expect(obj.Spec.Shipwright.Build).NotTo(BeNil())
	Expect(obj.Spec.Shipwright.Build.State).To(Equal(operatorv1alpha1.Enabled))
	Expect(controllerutil.ContainsFinalizer(obj, common.OpenShiftBuildFinalizerName)).To(BeTrue(), "checking for finalizer %q", common.OpenShiftBuildFinalizerName)

	Expect(obj.Spec.SharedResource).NotTo(BeNil())
	Expect(obj.Spec.SharedResource.State).To(Equal(operatorv1alpha1.Enabled))
}
