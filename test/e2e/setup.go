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

package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	. "github.com/onsi/gomega"    //nolint:golint,revive

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/test/setup"
	"github.com/redhat-openshift-builds/operator/test/utils"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	shpoperatorv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	kubeClient client.Client
	mgr        *setup.OperatorManager
	projectDir string
)

var _ = BeforeSuite(func(ctx SpecContext) {
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(extv1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(shpoperatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(buildv1beta1.AddToScheme(scheme)).To(Succeed(), "adding build to scheme")

	ctrl.SetLogger(GinkgoLogr)
	config, err := ctrl.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "getting KUBECONFIG")
	kubeClient, err = client.New(config, client.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred(), "setting up kubeClient")

	By("Setting up project directory value")
	projectDir, err = utils.GetProjectDir()
	Expect(err).NotTo(HaveOccurred(), "getting project directory")

	By("installing operators", func() {
		mgr = setup.NewOperatorManager(kubeClient)
		err = mgr.InstallBuildsForOpenShift(ctx)
		Expect(err).NotTo(HaveOccurred(), "ensure Builds for OpenShift installed")
	})

	By("Creating the builds-test namespace", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "builds-test",
			},
		}
		Expect(kubeClient.Create(ctx, ns)).To(Succeed())
	})

})

var _ = AfterSuite(func(ctx SpecContext) {
	By("Tearing down the builds-test namespace", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "builds-test",
			},
		}
		Expect(kubeClient.Delete(ctx, ns)).NotTo(HaveOccurred(), "failed to delete the builds-test namespace")
		By("Waiting for the builds-test namespace to be fully deleted")
		Eventually(func() error {
			err := kubeClient.Get(ctx, client.ObjectKey{Name: "builds-test"}, ns)
			if err != nil && client.IgnoreNotFound(err) == nil {
				// Namespace no longer exists
				return nil
			}
			return fmt.Errorf("namespace builds-test still exists")
		}, 4*time.Minute, 10*time.Second).Should(Succeed(), "timed out waiting for namespace builds-test to be deleted")
	})
})
