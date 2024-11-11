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
	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	. "github.com/onsi/gomega"    //nolint:golint,revive

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/test/setup"
	shpoperatorv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
)

var (
	kubeClient client.Client
	mgr        *setup.OperatorManager
)

var _ = BeforeSuite(func(ctx SpecContext) {
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(extv1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")
	Expect(shpoperatorv1alpha1.AddToScheme(scheme)).To(Succeed(), "setting up kubeClient")

	ctrl.SetLogger(GinkgoLogr)
	config, err := ctrl.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "getting KUBECONFIG")
	kubeClient, err = client.New(config, client.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred(), "setting up kubeClient")

	By("installing operators", func() {
		mgr = setup.NewOperatorManager(kubeClient)
		err = mgr.InstallBuildsForOpenShift(ctx)
		Expect(err).NotTo(HaveOccurred(), "ensure Builds for OpenShift installed")
	})

})
