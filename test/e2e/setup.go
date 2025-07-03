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
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	. "github.com/onsi/gomega"    //nolint:golint,revive

	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
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
	kubeClient    client.Client
	mgr           *setup.OperatorManager
	projectDir    string
	clientset     *kubernetes.Clientset
	testNamespace = "builds-test"
)

var _ = BeforeSuite(func(ctx SpecContext) {
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(extv1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(shpoperatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(buildv1beta1.AddToScheme(scheme)).To(Succeed(), "adding build to scheme")
	Expect(rbacv1.AddToScheme(scheme)).To(Succeed(), "adding rbacv1 to scheme")

	ctrl.SetLogger(GinkgoLogr)
	config, err := ctrl.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "getting KUBECONFIG")
	kubeClient, err = client.New(config, client.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred(), "setting up kubeClient")

	// clientset - to retrieve streaming logs from pods' containers for debugging purposes.
	clientset, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred(), "failed to create Kubernetes clientset")

	By("Setting up project directory value")
	projectDir, err = utils.GetProjectDir()
	Expect(err).NotTo(HaveOccurred(), "getting project directory")

	By("installing operators", func() {
		mgr = setup.NewOperatorManager(kubeClient)
		err = mgr.InstallBuildsForOpenShift(ctx)
		Expect(err).NotTo(HaveOccurred(), "ensure Builds for OpenShift installed")
	})

	By(fmt.Sprintf("Creating the %s namespace", testNamespace), func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		err := kubeClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create namespace %s", testNamespace))
		} else if errors.IsAlreadyExists(err) {
			GinkgoWriter.Printf("Namespace %s already exists, proceeding.\n", testNamespace)
		}
	})

	By("Ensuring 'buildpacks' ClusterBuildStrategy is present", func() {
		cbsName := "buildpacks-extender"
		cbs := &buildv1beta1.ClusterBuildStrategy{}

		// Check if the ClusterBuildStrategy already exists
		err := kubeClient.Get(ctx, client.ObjectKey{Name: cbsName}, cbs)
		if err != nil {
			if errors.IsNotFound(err) {
				GinkgoWriter.Printf("ClusterBuildStrategy '%s' not found. Applying it from remote URL.\n", cbsName)

				resp, fetchErr := http.Get("https://raw.githubusercontent.com/redhat-developer/openshift-builds-catalog/main/clusterBuildStrategy/buildpacks-extender/buildpacks-extender.yaml")
				Expect(fetchErr).NotTo(HaveOccurred(), "failed to fetch ClusterBuildStrategy YAML")
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("expected HTTP 200, got %d from URL", resp.StatusCode))

				yamlBytes, readErr := io.ReadAll(resp.Body)
				Expect(readErr).NotTo(HaveOccurred(), "failed to read ClusterBuildStrategy YAML")

				// Decode the YAML into a ClusterBuildStrategy object using the scheme
				deserializer := serializer.NewCodecFactory(scheme).UniversalDeserializer()

				newCBS := &buildv1beta1.ClusterBuildStrategy{}
				_, _, decodeErr := deserializer.Decode(yamlBytes, nil, newCBS)
				Expect(decodeErr).NotTo(HaveOccurred(), "failed to decode ClusterBuildStrategy YAML")

				if newCBS.Name == "" {
					newCBS.Name = cbsName
				}

				createErr := kubeClient.Create(ctx, newCBS)
				Expect(createErr).NotTo(HaveOccurred(), fmt.Sprintf("failed to create ClusterBuildStrategy %s", cbsName))
				GinkgoWriter.Printf("ClusterBuildStrategy '%s' applied successfully.\n", cbsName)
			} else {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get ClusterBuildStrategy '%s': %v", cbsName, err))
			}
		} else {
			GinkgoWriter.Printf("ClusterBuildStrategy '%s' already exists. No action needed.\n", cbsName)
		}
	})

	By(fmt.Sprintf("Granting default service account the ability to pull images in namespace %s", testNamespace), func() {
		defaultSaName := "default"
		Eventually(func() error {
			return kubeClient.Get(ctx, client.ObjectKey{Name: defaultSaName, Namespace: testNamespace}, &corev1.ServiceAccount{})
		}, 2*time.Minute, 5*time.Second).Should(Succeed(), "service account %s not found", defaultSaName)

		roleBindingName := fmt.Sprintf("%s-image-puller", defaultSaName)
		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: testNamespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      defaultSaName,
					Namespace: testNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "system:image-puller",
			},
		}

		// Check if the RoleBinding already exists
		getRoleBinding := &rbacv1.RoleBinding{}
		err := kubeClient.Get(ctx, client.ObjectKey{Name: roleBindingName, Namespace: testNamespace}, getRoleBinding)

		if err != nil {
			if errors.IsNotFound(err) {
				GinkgoWriter.Printf("RoleBinding '%s' not found. Creating it.\n", roleBindingName)
				createErr := kubeClient.Create(ctx, roleBinding)
				Expect(createErr).NotTo(HaveOccurred(), fmt.Sprintf("failed to create RoleBinding %s for service account %s: %v", roleBindingName, defaultSaName, createErr))
				GinkgoWriter.Printf("RoleBinding '%s' created successfully.\n", roleBindingName)
			} else {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get RoleBinding '%s': %v", roleBindingName, err))
			}
		} else {
			GinkgoWriter.Printf("RoleBinding '%s' already exists. No action needed.\n", roleBindingName)
		}
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
