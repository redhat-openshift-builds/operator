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

	Describe("Inject TLS Args into Container", func() {
		var deployment *appsv1.Deployment
		var manifest manifestival.Manifest

		BeforeEach(func() {
			deployment = &appsv1.Deployment{}
			deployment.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			})
			deployment.SetName("test-webhook")
			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "shipwright-build-webhook",
					Image: "test:latest",
					Args:  []string{"--some-arg=value"},
				},
			}
			object = &unstructured.Unstructured{}
			err := scheme.Scheme.Convert(deployment, object, nil)
			Expect(err).ShouldNot(HaveOccurred())
			manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
		})

		When("TLS args are provided for shipwright-build-webhook container", func() {
			It("should inject TLS flags into the container", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "VersionTLS12", "TLS_AES_128_GCM_SHA256"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(ContainElement("--tls-min-version=VersionTLS12"))
				Expect(container.Args).To(ContainElement("--tls-cipher-suites=TLS_AES_128_GCM_SHA256"))
				Expect(container.Args).To(ContainElement("--some-arg=value"))
			})
		})

		When("TLS args are provided for shared-resource-csi-driver-webhook container", func() {
			BeforeEach(func() {
				deployment.Spec.Template.Spec.Containers[0].Name = "shared-resource-csi-driver-webhook"
				object = &unstructured.Unstructured{}
				err := scheme.Scheme.Convert(deployment, object, nil)
				Expect(err).ShouldNot(HaveOccurred())
				manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
			})

			It("should inject TLS flags into the CSI driver webhook container", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shared-resource-csi-driver-webhook", "VersionTLS13", "TLS_CHACHA20_POLY1305_SHA256"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(ContainElement("--tls-min-version=VersionTLS13"))
				Expect(container.Args).To(ContainElement("--tls-cipher-suites=TLS_CHACHA20_POLY1305_SHA256"))
			})
		})

		When("container already has existing TLS args", func() {
			BeforeEach(func() {
				deployment.Spec.Template.Spec.Containers[0].Args = []string{
					"--some-arg=value",
					"--tls-min-version=VersionTLS10",
					"--tls-cipher-suites=OLD_CIPHER",
					"--another-arg=test",
				}
				object = &unstructured.Unstructured{}
				err := scheme.Scheme.Convert(deployment, object, nil)
				Expect(err).ShouldNot(HaveOccurred())
				manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
			})

			It("should replace existing TLS args without duplicating", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "VersionTLS12", "NEW_CIPHER"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(ContainElement("--tls-min-version=VersionTLS12"))
				Expect(container.Args).To(ContainElement("--tls-cipher-suites=NEW_CIPHER"))
				Expect(container.Args).NotTo(ContainElement("--tls-min-version=VersionTLS10"))
				Expect(container.Args).NotTo(ContainElement("--tls-cipher-suites=OLD_CIPHER"))
				Expect(container.Args).To(ContainElement("--some-arg=value"))
				Expect(container.Args).To(ContainElement("--another-arg=test"))
			})
		})

		When("empty TLS values are provided", func() {
			It("should not inject empty TLS flags", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "", ""),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(Equal([]string{"--some-arg=value"}))
			})
		})

		When("only minVersion is provided", func() {
			It("should inject only tls-min-version flag", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "VersionTLS12", ""),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(ContainElement("--tls-min-version=VersionTLS12"))
				Expect(container.Args).NotTo(ContainElement(HavePrefix("--tls-cipher-suites=")))
			})
		})

		When("only cipherSuites is provided", func() {
			It("should inject only tls-cipher-suites flag", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "", "TLS_AES_256_GCM_SHA384"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(ContainElement("--tls-cipher-suites=TLS_AES_256_GCM_SHA384"))
				Expect(container.Args).NotTo(ContainElement(HavePrefix("--tls-min-version=")))
			})
		})

		When("deployment has multiple containers", func() {
			BeforeEach(func() {
				deployment.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Name:  "other-container",
						Image: "other:latest",
						Args:  []string{"--other-arg=value"},
					},
					{
						Name:  "shipwright-build-webhook",
						Image: "webhook:latest",
						Args:  []string{"--webhook-arg=value"},
					},
					{
						Name:  "sidecar",
						Image: "sidecar:latest",
						Args:  []string{"--sidecar-arg=value"},
					},
				}
				object = &unstructured.Unstructured{}
				err := scheme.Scheme.Convert(deployment, object, nil)
				Expect(err).ShouldNot(HaveOccurred())
				manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
			})

			It("should inject TLS flags only into the named container", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "VersionTLS12", "TLS_AES_128_GCM_SHA256"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				// Other container should not have TLS args
				Expect(resultDeployment.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{"--other-arg=value"}))

				// Target container should have TLS args
				webhookContainer := resultDeployment.Spec.Template.Spec.Containers[1]
				Expect(webhookContainer.Args).To(ContainElement("--tls-min-version=VersionTLS12"))
				Expect(webhookContainer.Args).To(ContainElement("--tls-cipher-suites=TLS_AES_128_GCM_SHA256"))
				Expect(webhookContainer.Args).To(ContainElement("--webhook-arg=value"))

				// Sidecar should not have TLS args
				Expect(resultDeployment.Spec.Template.Spec.Containers[2].Args).To(Equal([]string{"--sidecar-arg=value"}))
			})
		})

		When("container name does not match", func() {
			It("should not modify any container", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("non-existent-container", "VersionTLS12", "TLS_AES_128_GCM_SHA256"),
				)
				Expect(err).ShouldNot(HaveOccurred())

				resultDeployment := &appsv1.Deployment{}
				err = scheme.Scheme.Convert(&manifest.Resources()[0], resultDeployment, nil)
				Expect(err).ShouldNot(HaveOccurred())

				container := resultDeployment.Spec.Template.Spec.Containers[0]
				Expect(container.Args).To(Equal([]string{"--some-arg=value"}))
			})
		})

		When("resource is not a Deployment", func() {
			BeforeEach(func() {
				service := &corev1.Service{}
				service.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Service",
				})
				service.SetName("test-service")
				object = &unstructured.Unstructured{}
				err := scheme.Scheme.Convert(service, object, nil)
				Expect(err).ShouldNot(HaveOccurred())
				manifest, _ = manifestival.ManifestFrom(manifestival.Slice{*object})
			})

			It("should skip non-Deployment resources", func() {
				manifest, err := manifest.Transform(
					common.InjectTLSArgsIntoContainer("shipwright-build-webhook", "VersionTLS12", "TLS_AES_128_GCM_SHA256"),
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(manifest.Resources()[0].GetKind()).To(Equal("Service"))
			})
		})

	})
})
