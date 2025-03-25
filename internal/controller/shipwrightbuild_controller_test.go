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
	"strings"
	"testing"

	o "github.com/onsi/gomega"

	"github.com/shipwright-io/operator/api/v1alpha1"
	shipwrightoperator "github.com/shipwright-io/operator/controllers"
	swocommon "github.com/shipwright-io/operator/pkg/common"
	tektonoperatorv1alpha1 "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	tektonoperatorv1alpha1client "github.com/tektoncd/operator/pkg/client/clientset/versioned/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var allowedRegistries = []string{
	"registry.redhat.io",
	"registry.access.redhat.com",
	"brew.registry.redhat.io",
	"registry.stage.redhat.io",
}

const (
	defaultTargetNamespace = "shipwright-build"
)

// bootstrapShipwrightBuildReconciler start up a new instance of ShipwrightBuildReconciler which is
// ready to interact with Manifestival, returning the Manifestival instance and the client.
func bootstrapShipwrightBuildReconciler(
	t *testing.T,
	b *v1alpha1.ShipwrightBuild,
	tcfg *tektonoperatorv1alpha1.TektonConfig,
	tcrds []*crdv1.CustomResourceDefinition,
) (client.Client, *crdclientv1.Clientset, *tektonoperatorv1alpha1client.Clientset, *shipwrightoperator.ShipwrightBuildReconciler) {
	g := o.NewGomegaWithT(t)

	s := runtime.NewScheme()
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Namespace{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.ShipwrightBuild{})
	s.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.ClusterRoleBinding{})
	s.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.ClusterRole{})

	logger := zap.New()

	c := fake.NewClientBuilder().WithScheme(s).WithObjects(b).Build()
	var crdClient *crdclientv1.Clientset
	var toClient *tektonoperatorv1alpha1client.Clientset
	if len(tcrds) > 0 {
		objs := []runtime.Object{}
		for _, obj := range tcrds {
			objs = append(objs, obj)
		}
		crdClient = crdclientv1.NewSimpleClientset(objs...)
	} else {
		crdClient = crdclientv1.NewSimpleClientset()
	}
	if tcfg == nil {
		toClient = tektonoperatorv1alpha1client.NewSimpleClientset()
	} else {
		toClient = tektonoperatorv1alpha1client.NewSimpleClientset(tcfg)
	}
	r := &ShipwrightBuildReconciler{CRDClient: crdClient.ApiextensionsV1(), TektonOperatorClient: toClient.OperatorV1alpha1(), Client: c, Scheme: s, Logger: logger}

	// creating targetNamespace on which Shipwright-Build will be deployed against, before the other
	// tests takes place
	if b.Spec.TargetNamespace != "" {
		t.Logf("Creating test namespace '%s'", b.Spec.TargetNamespace)
		t.Run("create-test-namespace", func(t *testing.T) {
			err := c.Create(
				context.TODO(),
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: b.Spec.TargetNamespace}},
				&client.CreateOptions{},
			)
			g.Expect(err).To(o.BeNil())
		})
	}

	// manifestival instance is setup as part of controller-=runtime's SetupWithManager, thus calling
	// the setup before all other methods
	t.Run("setupManifestival", func(t *testing.T) {
		err := r.setupManifestival()
		g.Expect(err).To(o.BeNil())
	})

	reconciler := shipwrightoperator.ShipwrightBuildReconciler(*r)
	reconciler.Client = c

	return c, crdClient, toClient, &reconciler
}

// testShipwrightBuildManifestReplacement simulates the reconciliation process to test
// the manifest replacement process.
func testShipwrightBuildManifestReplacement(t *testing.T, targetNamespace string) {
	g := o.NewGomegaWithT(t)
	ctx := context.TODO()

	namespacedName := types.NamespacedName{Namespace: "default", Name: "name"}
	req := reconcile.Request{NamespacedName: namespacedName}

	b := &v1alpha1.ShipwrightBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: v1alpha1.ShipwrightBuildSpec{
			TargetNamespace: targetNamespace,
		},
	}
	crd1 := &crdv1.CustomResourceDefinition{}
	crd1.Name = "taskruns.tekton.dev"
	crd2 := &crdv1.CustomResourceDefinition{}
	crd2.Name = "tektonconfigs.operator.tekton.dev"
	crd2.Labels = map[string]string{"operator.tekton.dev/release": swocommon.TektonOpMinSupportedVersion}
	crds := []*crdv1.CustomResourceDefinition{crd1, crd2}
	_, _, _, r := bootstrapShipwrightBuildReconciler(t, b, nil, crds)

	t.Run("manifest-has-only-whitelisted-registries", func(t *testing.T) {
		res, err := r.Reconcile(ctx, req)
		g.Expect(err).To(o.BeNil())
		g.Expect(res.Requeue).To(o.BeTrue())

		manifest := r.Manifest.Resources()
		images := getAllImagesFromManifest(manifest)
		for _, image := range images {
			match := false
			for _, registry := range allowedRegistries {
				if strings.Contains(image, registry) {
					match = true
					break
				}
			}
			g.Expect(match).To(o.BeTrue())
		}
	})
}

func getAllImagesFromManifest(manifest []unstructured.Unstructured) []string {
	images := []string{}

	for _, resource := range manifest {
		// Check Deployments
		if resource.GetKind() == "Deployment" {
			deployment := &appsv1.Deployment{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, deployment)
			if err != nil {
				continue
			}
			// Get container images from pods
			for _, container := range deployment.Spec.Template.Spec.Containers {
				images = append(images, container.Image)
			}
			for _, container := range deployment.Spec.Template.Spec.InitContainers {
				images = append(images, container.Image)
			}
		}

		// [TODO] check other resource types that could contain images
		// For example: DaemonSets, StatefulSets, CronJobs, etc.
	}

	return images
}

func TestShipwrightBuildManifestReplacement(t *testing.T) {
	tests := []struct {
		testName        string
		targetNamespace string
	}{{
		testName:        "target namespace is not informed",
		targetNamespace: defaultTargetNamespace,
	}}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			testShipwrightBuildManifestReplacement(t, tt.targetNamespace)
		})
	}
}
