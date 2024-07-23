package common_test

import (
	"github.com/manifestival/manifestival"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-builds/operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

var _ = Describe("Transformer", Label("transformer"), func() {
	var object *unstructured.Unstructured

	Describe("Remove RunAsUser and RunAsGroup from security context", func() {
		BeforeEach(func() {
			object = &unstructured.Unstructured{}
			deployment := &appsv1.Deployment{}
			deployment.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			})
			deployment.SetName("test")
			deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  ptr.To(int64(1000)),
				RunAsGroup: ptr.To(int64(1000)),
			}
			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: "test",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  ptr.To(int64(1000)),
						RunAsGroup: ptr.To(int64(1000)),
					},
				},
			}
			err := scheme.Scheme.Convert(deployment, object, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
		When("runAsUser and runAsGroup are set", func() {
			It("should remove runAsUser and runAsGroup", func() {
				deployment := &appsv1.Deployment{}
				err := common.RemoveRunAsUserRunAsGroup(object)
				Expect(err).ShouldNot(HaveOccurred())
				err = scheme.Scheme.Convert(object, deployment, nil)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(deployment.Spec.Template.Spec.SecurityContext.RunAsUser).To(BeNil())
				Expect(deployment.Spec.Template.Spec.SecurityContext.RunAsGroup).To(BeNil())
				Expect(deployment.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser).To(BeNil())
				Expect(deployment.Spec.Template.Spec.Containers[0].SecurityContext.RunAsGroup).To(BeNil())
			})
		})
	})

	Describe("Inject annotations", func() {
		var manifest manifestival.Manifest
		var annotations map[string]string
		BeforeEach(func() {
			annotations = map[string]string{
				"test-key": "test-value",
			}
			object = &unstructured.Unstructured{}
			object.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Service",
			})
			object.SetName("test")
			manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
		})
		When("kind is provided and it matches object's kind", func() {
			It("should return a Manifestival transformer that inject given annotations", func() {
				manifest, err := manifest.Transform(
					common.InjectAnnotations([]string{"Service"}, nil, annotations),
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(manifest.Resources()[0].GetAnnotations()).To(Equal(annotations))
			})
		})
		When("name is provided and it matches object's name", func() {
			It("should return a Manifestival transformer that inject given annotations", func() {
				manifest, err := manifest.Transform(
					common.InjectAnnotations(nil, []string{"test"}, annotations),
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(manifest.Resources()[0].GetAnnotations()).To(Equal(annotations))
			})
		})
		When("name and kind is provided and it matches object's both name and kind", func() {
			It("should return a Manifestival transformer that inject given annotations", func() {
				manifest, err := manifest.Transform(
					common.InjectAnnotations([]string{"Service"}, []string{"test"}, annotations),
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(manifest.Resources()[0].GetAnnotations()).To(Equal(annotations))
			})
		})
		When("name and kind is provided and it doesn't match object's name or kind", func() {
			It("should return a Manifestival transformer that does not inject given annotations", func() {
				manifest, err := manifest.Transform(
					common.InjectAnnotations([]string{"Deployment"}, []string{"not-matching"}, annotations),
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(manifest.Resources()[0].GetAnnotations()).To(BeNil())
			})
		})
	})
})
