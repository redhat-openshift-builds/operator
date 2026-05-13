package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	openshiftBuildsNS = "openshift-builds"
	kubeAPIServerNS   = "openshift-kube-apiserver"
	monitoringNS      = "openshift-monitoring"
	defaultNS         = "default"
)

var (
	netTestPodBuilds      = "nettest-builds"         
	netTestPodDefault     = "nettest-default"        
	netTestPodKubeAPI     = "nettest-kube-apiserver"  
	netTestPodMonitoring  = "nettest-monitoring"     
)

var _ = Describe("NetworkPolicy enforcement test", Ordered, Label("e2e"), Label("networkpolicy"), func() {

	BeforeAll(func(ctx SpecContext) {
		By("Creating network test pods for connectivity tests")
		testPods := []struct {
			pod         *corev1.Pod
			description string
		}{
			{createTestPod(netTestPodBuilds, openshiftBuildsNS, false), "openshift-builds"},
			{createTestPod(netTestPodDefault, defaultNS, false), "default"},
			{createTestPod(netTestPodKubeAPI, kubeAPIServerNS, true), "openshift-kube-apiserver"},
			{createTestPod(netTestPodMonitoring, monitoringNS, true), "openshift-monitoring"},
		}

		for _, tp := range testPods {
			err := kubeClient.Create(ctx, tp.pod)
			if client.IgnoreAlreadyExists(err) != nil {
				Fail(fmt.Sprintf("failed to create test pod %s: %v", tp.pod.Name, err))
			}
		}

		for _, tp := range testPods {
			Eventually(func() bool {
				pod := &corev1.Pod{}
				err := kubeClient.Get(ctx, client.ObjectKey{Name: tp.pod.Name, Namespace: tp.pod.Namespace}, pod)
				return err == nil && pod.Status.Phase == corev1.PodRunning
			}, 2*time.Minute, 5*time.Second).Should(BeTrue(), fmt.Sprintf("Test pod in %s should be running", tp.description))
		}

		fmt.Println("Network test pods ready")
	})

	AfterAll(func(ctx SpecContext) {
		By("Cleaning up network test pods")
		testPodsToDelete := []struct {
			name      string
			namespace string
		}{
			{netTestPodBuilds, openshiftBuildsNS},
			{netTestPodDefault, defaultNS},
			{netTestPodKubeAPI, kubeAPIServerNS},
			{netTestPodMonitoring, monitoringNS},
		}

		for _, tp := range testPodsToDelete {
			err := kubeClient.Delete(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tp.name,
					Namespace: tp.namespace,
				},
			})
			if client.IgnoreNotFound(err) != nil {
				fmt.Printf("Warning: failed to delete test pod %s: %v\n", tp.name, err)
			}
		}

		fmt.Println("NetworkPolicies Testing Done")
	})

	Context("NetworkPolicy Deployment", func() {
		BeforeAll(func() {
			fmt.Println("\n--Testing NetworkPolicy Deployment--")
		})

		AfterAll(func() {
			fmt.Println("\n--NetworkPolicy Deployment Tested--")
		})

		It("should have ingress policies deployed", func(ctx SpecContext) {
			policies := []string{
				"csidriver-webhook-ingress",
				"shipwright-webhook-ingress",
				"monitoring-metrics-ingress-csi",
				"monitoring-metrics-ingress-shipwright",
			}

			for _, policyName := range policies {
				np := &networkingv1.NetworkPolicy{}
				err := kubeClient.Get(ctx, types.NamespacedName{
					Name:      policyName,
					Namespace: openshiftBuildsNS,
				}, np)
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("%s deployed\n", policyName)
			}
		})
	})

	Context("Ingress to Webhook Services", Ordered, func() {
		BeforeAll(func() {
			fmt.Println("\n--Testing Ingress to Webhook Services--")
		})

		AfterAll(func() {
			fmt.Println("\n--Ingress to Webhook Services Tested--")
		})

		webhookTests := []struct {
			name        string
			serviceName string
			description string
		}{
			{"CSI webhook", "shared-resource-csi-driver-webhook", "CSI webhook"},
			{"Shipwright webhook", "shp-build-webhook", "Shipwright webhook"},
		}

		for _, wt := range webhookTests {
			It(fmt.Sprintf("should allow ingress to %s and block unauthorized access", wt.name), func(ctx SpecContext) {
				svc := &corev1.Service{}
				err := kubeClient.Get(ctx, types.NamespacedName{
					Name:      wt.serviceName,
					Namespace: openshiftBuildsNS,
				}, svc)
			Expect(err).NotTo(HaveOccurred())
				Expect(svc.Spec.ClusterIP).NotTo(BeEmpty())

				webhookIP := svc.Spec.ClusterIP
				fmt.Printf("Testing %s at %s:443\n", wt.description, webhookIP)

				By("Testing authorized access from openshift-kube-apiserver")
				succeeded := testNetworkConnectivityFromNamespace(kubeAPIServerNS, webhookIP, "443")
				Expect(succeeded).To(BeTrue(), "openshift-kube-apiserver SHOULD access %s", wt.description)

				By("Testing unauthorized access is blocked")
				succeeded = testNetworkConnectivityFromNamespace(defaultNS, webhookIP, "443")
				Expect(succeeded).To(BeFalse(), "default namespace should NOT access %s", wt.description)
				fmt.Printf("%s ingress verified\n", wt.description)
			})
		}
	})

	Context("Ingress to Metrics Endpoints", func() {
		BeforeAll(func() {
			fmt.Println("\n--Testing Ingress to Metrics Endpoints--")
		})

		AfterAll(func() {
			fmt.Println("\n--Ingress to Metrics Endpoints Tested--")
		})

		It("should allow metrics scraping from monitoring namespace", func(ctx SpecContext) {
			svc := &corev1.Service{}
			err := kubeClient.Get(ctx, types.NamespacedName{
				Name:      "shared-resource-csi-driver-node-metrics",
				Namespace: openshiftBuildsNS,
			}, svc)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Spec.ClusterIP).NotTo(BeEmpty())

			metricsIP := svc.Spec.ClusterIP
			fmt.Printf("Testing CSI metrics at %s:443\n", metricsIP)

			By("Testing authorized access from openshift-monitoring")
			succeeded := testNetworkConnectivityFromNamespace(monitoringNS, metricsIP, "443")
			Expect(succeeded).To(BeTrue(), "openshift-monitoring SHOULD access CSI metrics")

			By("Testing unauthorized access is blocked")
			succeeded = testNetworkConnectivityFromNamespace(defaultNS, metricsIP, "443")
			Expect(succeeded).To(BeFalse(), "default namespace should NOT access CSI metrics")
			fmt.Println("CSI metrics ingress verified")
		})
	})
})

// creates a test pod with security context for restricted namespaces
func createTestPod(name, namespace string, requireSecurityContext bool) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "nettest",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "nettest",
					Image:   "quay.io/curl/curl:latest",
					Command: []string{"sh", "-c", "sleep infinity"},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	if requireSecurityContext {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsNonRoot: ptr.To(true),
			RunAsUser:    ptr.To(int64(1000)),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		}
	}

	return pod
}

// executes a command in a pod using kubectl exec
func execInPod(namespace, podName, containerName string, command []string) (string, error) {
	args := []string{"exec", "-n", namespace, podName, "-c", containerName, "--"}
	args = append(args, command...)

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()

	return strings.TrimSpace(string(output)), err
}

// tests connectivity from a specific namespace
func testNetworkConnectivityFromNamespace(namespace, targetIP, targetPort string) bool {
	var podName string
	switch namespace {
	case defaultNS:
		podName = netTestPodDefault
	case openshiftBuildsNS:
		podName = netTestPodBuilds
	case kubeAPIServerNS:
		podName = netTestPodKubeAPI
	case monitoringNS:
		podName = netTestPodMonitoring
	default:
		Fail(fmt.Sprintf("No test pod available for namespace %s", namespace))
		return false
	}

	_, err := execInPod(namespace, podName, "nettest",
		[]string{"sh", "-c", fmt.Sprintf("nc -zv -w 3 %s %s", targetIP, targetPort)})

	return err == nil
}