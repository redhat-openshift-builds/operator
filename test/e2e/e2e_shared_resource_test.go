package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-builds/operator/test/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kubeConfig *rest.Config

const (
	sourceNS       = "default"
	targetNS       = "e2e-test-ns"
	secretName     = "e2e-test-secret"
	configMapName  = "e2e-test-configmap"
	podName        = "e2e-consumer-pod"
	csiSaNamespace = "openshift-builds"
)

var _ = Describe("Shared Resource CSI Driver test", Ordered, Label("e2e"), Label("shared-resources"), func() {

	// BeforeAll loads the kubeconfig and performs pre-flight checks.
	BeforeAll(func(ctx SpecContext) {
		var err error
		By("Getting the cluster config")
		kubeConfig, err = ctrl.GetConfig()
		Expect(err).NotTo(HaveOccurred(), "getting KUBECONFIG")

		By("Verifying the CSI driver components are configured with the fix for read-only root filesystem")
		daemonSetKey := client.ObjectKey{Name: "shared-resource-csi-driver-node", Namespace: csiSaNamespace}
		ds := &appsv1.DaemonSet{}
		Eventually(func() error {
			return kubeClient.Get(ctx, daemonSetKey, ds)
		}, "1m", "2s").Should(Succeed())

		foundHostpath := false
		for _, c := range ds.Spec.Template.Spec.Containers {
			if c.Name == "hostpath" {
				Expect(c.SecurityContext).NotTo(BeNil())
				Expect(c.SecurityContext.ReadOnlyRootFilesystem).To(Equal(boolPtr(true)))
				Expect(c.WorkingDir).To(Equal("/run/csi-data-dir"))
				foundHostpath = true
			}
		}
		Expect(foundHostpath).To(BeTrue())

		webhookKey := client.ObjectKey{Name: "shared-resource-csi-driver-webhook", Namespace: csiSaNamespace}
		webhookDeployment := &appsv1.Deployment{}
		Eventually(func() error {
			return kubeClient.Get(ctx, webhookKey, webhookDeployment)
		}, "1m", "2s").Should(Succeed())

		Expect(webhookDeployment.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		webhookContainer := webhookDeployment.Spec.Template.Spec.Containers[0]
		Expect(webhookContainer.SecurityContext).NotTo(BeNil())
		Expect(webhookContainer.SecurityContext.ReadOnlyRootFilesystem).To(Equal(boolPtr(true)))
	})

	Context("when sharing a Secret and a ConfigMap", func() {

		var testResources = []string{
			"/test/data/rbac.yaml",
			"/test/data/resources.yaml",
		}

		BeforeEach(func(ctx SpecContext) {
			By("Setting up resources for the test")

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNS}}
			Expect(client.IgnoreAlreadyExists(kubeClient.Create(ctx, ns))).To(Succeed())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: sourceNS},
				StringData: map[string]string{"username": "admin", "password": "s3cr3tPa$$wOrd"},
			}
			Expect(client.IgnoreAlreadyExists(kubeClient.Create(ctx, secret))).To(Succeed())

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: sourceNS},
				Data:       map[string]string{"database.url": "db.example.com", "color.theme": "dark"},
			}
			Expect(client.IgnoreAlreadyExists(kubeClient.Create(ctx, configMap))).To(Succeed())

			for _, resource := range testResources {
				Expect(utils.ApplyResourceFromFile(ctx, kubeClient, projectDir+resource)).To(Succeed())
			}
		})

		AfterEach(func(ctx SpecContext) {
			By("Cleaning up all the test resources")

			for _, resource := range testResources {
				_ = utils.DeleteResourceFromFile(ctx, kubeClient, projectDir+resource)
			}

			_ = kubeClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: targetNS}})
			_ = kubeClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNS}})
			_ = kubeClient.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: sourceNS}})
			_ = kubeClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: sourceNS}})
		})

		It("should mount both a SharedSecret and a SharedConfigMap into a pod", func(ctx SpecContext) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: targetNS},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "test-container",
						Image:   "registry.access.redhat.com/ubi8/ubi-minimal:latest",
						Command: []string{"sh", "-c", "sleep 1d"},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "secret-vol", MountPath: "/etc/secret", ReadOnly: true},
							{Name: "configmap-vol", MountPath: "/etc/configmap", ReadOnly: true},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "secret-vol",
							VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{
								Driver: "csi.sharedresource.openshift.io", ReadOnly: boolPtr(true),
								VolumeAttributes: map[string]string{"sharedSecret": "e2e-shared-secret"},
							}},
						},
						{
							Name: "configmap-vol",
							VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{
								Driver: "csi.sharedresource.openshift.io", ReadOnly: boolPtr(true),
								VolumeAttributes: map[string]string{"sharedConfigMap": "e2e-shared-configmap"},
							}},
						},
					},
				},
			}
			Expect(kubeClient.Create(ctx, pod)).To(Succeed())

			By("Waiting for the pod to become ready")
			Eventually(func() error {
				p := &corev1.Pod{}
				err := kubeClient.Get(ctx, client.ObjectKey{Name: podName, Namespace: targetNS}, p)
				if err != nil {
					return err
				}
				for _, cond := range p.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
				return fmt.Errorf("pod not yet ready")
			}, time.Minute*2, time.Second*2).Should(Succeed())

			By("Verifying the mounted content")
			username, err := utils.ExecInPod(clientset, kubeConfig, podName, targetNS, "cat", "/etc/secret/username")
			Expect(err).NotTo(HaveOccurred())
			Expect(username).To(Equal("admin"))

			password, err := utils.ExecInPod(clientset, kubeConfig, podName, targetNS, "cat", "/etc/secret/password")
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(Equal("s3cr3tPa$$wOrd"))

			dbURL, err := utils.ExecInPod(clientset, kubeConfig, podName, targetNS, "cat", "/etc/configmap/database.url")
			Expect(err).NotTo(HaveOccurred())
			Expect(dbURL).To(Equal("db.example.com"))

			theme, err := utils.ExecInPod(clientset, kubeConfig, podName, targetNS, "cat", "/etc/configmap/color.theme")
			Expect(err).NotTo(HaveOccurred())
			Expect(theme).To(Equal("dark"))
		})
	})
})

func boolPtr(b bool) *bool { return &b }
