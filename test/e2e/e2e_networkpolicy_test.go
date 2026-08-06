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
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	openshiftBuildsNS  = "openshift-builds"
	netpolMonitoringNS = "netpol-test-monitoring"
	netpolMonPodName   = "netpol-mon-probe"
	netpolDenyPodName  = "netpol-deny-probe"

	// tcpProbeTimeout is the max seconds to wait for a TCP connection attempt.
	// OVN-Kubernetes DROPs blocked packets, so an allowed connection gets a fast
	// response (exit 0 or 1, < 1 s), while a blocked one hangs until timeout (exit 124).
	tcpProbeTimeout = 4
)

var _ = Describe("NetworkPolicy enforcement test", Ordered, Label("e2e"), Label("networkpolicy"), func() {

	BeforeAll(func(ctx SpecContext) {
		By("Creating a namespace labeled as monitoring so the allow-side of monitoring NetworkPolicies can be tested")
		monNS := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: netpolMonitoringNS,
				Labels: map[string]string{
					"network.openshift.io/policy-group": "monitoring",
				},
			},
		}
		if err := kubeClient.Create(ctx, monNS); client.IgnoreAlreadyExists(err) != nil {
			Fail(fmt.Sprintf("failed to create monitoring test namespace: %v", err))
		}

		By("Creating probe pods for enforcement tests")
		pods := []struct {
			pod  *corev1.Pod
			desc string
		}{
			{netpolProbePod(netpolMonPodName, netpolMonitoringNS), "monitoring-namespace probe"},
			{netpolProbePod(netpolDenyPodName, testNamespace), "deny-namespace probe"},
		}
		for _, p := range pods {
			if err := kubeClient.Create(ctx, p.pod); client.IgnoreAlreadyExists(err) != nil {
				Fail(fmt.Sprintf("failed to create %s: %v", p.desc, err))
			}
		}

		By("Waiting for probe pods to be Running")
		for _, p := range pods {
			Eventually(func() bool {
				pod := &corev1.Pod{}
				if err := kubeClient.Get(ctx, client.ObjectKey{Name: p.pod.Name, Namespace: p.pod.Namespace}, pod); err != nil {
					return false
				}
				return pod.Status.Phase == corev1.PodRunning
			}, 3*time.Minute, 5*time.Second).Should(BeTrue(), "%s should be Running", p.desc)
		}
	})

	AfterAll(func(ctx SpecContext) {
		By("Cleaning up probe pods and monitoring namespace")
		_ = kubeClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: netpolMonPodName, Namespace: netpolMonitoringNS}})
		_ = kubeClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: netpolDenyPodName, Namespace: testNamespace}})
		_ = kubeClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: netpolMonitoringNS}})
	})

	Context("NetworkPolicy deployment", func() {
		It("should have all 5 ingress policies deployed in openshift-builds", func(ctx SpecContext) {
			for _, policyName := range []string{
				"default-deny-ingress",
				"csidriver-webhook-ingress",
				"shipwright-webhook-ingress",
				"monitoring-metrics-ingress-csi",
				"monitoring-metrics-ingress-shipwright",
			} {
				np := &networkingv1.NetworkPolicy{}
				Expect(kubeClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: openshiftBuildsNS}, np)).
					To(Succeed(), "NetworkPolicy %s should exist", policyName)
			}
		})

		It("should have default-deny-ingress covering all pods with no allow rules", func(ctx SpecContext) {
			np := &networkingv1.NetworkPolicy{}
			Expect(kubeClient.Get(ctx, types.NamespacedName{Name: "default-deny-ingress", Namespace: openshiftBuildsNS}, np)).To(Succeed())
			Expect(np.Spec.PodSelector.MatchLabels).To(BeEmpty(), "must apply to all pods (empty podSelector)")
			Expect(np.Spec.PodSelector.MatchExpressions).To(BeEmpty(), "must apply to all pods (empty podSelector)")
			Expect(np.Spec.Ingress).To(BeEmpty(), "deny-all means no ingress rules")
			Expect(np.Spec.PolicyTypes).To(ContainElement(networkingv1.PolicyTypeIngress))
		})

		It("should restrict webhook ingress to kube-apiserver namespace on port 8443", func(ctx SpecContext) {
			for _, policyName := range []string{"csidriver-webhook-ingress", "shipwright-webhook-ingress"} {
				np := &networkingv1.NetworkPolicy{}
				Expect(kubeClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: openshiftBuildsNS}, np)).To(Succeed())
				Expect(np.Spec.Ingress).To(HaveLen(1), "%s: expected one ingress rule", policyName)
				Expect(np.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels).To(
					HaveKeyWithValue("kubernetes.io/metadata.name", "openshift-kube-apiserver"),
					"%s: wrong source namespace", policyName)
				Expect(np.Spec.Ingress[0].Ports[0].Port.IntValue()).To(Equal(8443),
					"%s: wrong port", policyName)
			}
		})

		It("should restrict monitoring ingress to monitoring-labeled namespaces", func(ctx SpecContext) {
			for _, policyName := range []string{"monitoring-metrics-ingress-csi", "monitoring-metrics-ingress-shipwright"} {
				np := &networkingv1.NetworkPolicy{}
				Expect(kubeClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: openshiftBuildsNS}, np)).To(Succeed())
				Expect(np.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels).To(
					HaveKeyWithValue("network.openshift.io/policy-group", "monitoring"),
					"%s: wrong source namespace selector", policyName)
			}
		})

		It("should have monitoring-metrics-ingress-csi targeting CSI pods on ports 6000 and 9898", func(ctx SpecContext) {
			np := &networkingv1.NetworkPolicy{}
			Expect(kubeClient.Get(ctx, types.NamespacedName{Name: "monitoring-metrics-ingress-csi", Namespace: openshiftBuildsNS}, np)).To(Succeed())
			Expect(np.Spec.PodSelector.MatchLabels).To(HaveKeyWithValue("app", "shared-resource-csi-driver-node"))
			ports := make([]int, 0, len(np.Spec.Ingress[0].Ports))
			for _, p := range np.Spec.Ingress[0].Ports {
				ports = append(ports, p.Port.IntValue())
			}
			Expect(ports).To(ConsistOf(6000, 9898), "CSI metrics policy should allow ports 6000 (provisioner) and 9898 (healthz)")
		})

		It("should have monitoring-metrics-ingress-shipwright targeting Shipwright pods on port 8383", func(ctx SpecContext) {
			np := &networkingv1.NetworkPolicy{}
			Expect(kubeClient.Get(ctx, types.NamespacedName{Name: "monitoring-metrics-ingress-shipwright", Namespace: openshiftBuildsNS}, np)).To(Succeed())
			Expect(np.Spec.PodSelector.MatchLabels).To(HaveKeyWithValue("name", "shipwright-build"))
			ports := make([]int, 0, len(np.Spec.Ingress[0].Ports))
			for _, p := range np.Spec.Ingress[0].Ports {
				ports = append(ports, p.Port.IntValue())
			}
			Expect(ports).To(ConsistOf(8383), "Shipwright metrics policy should allow exactly port 8383")
		})

		It("should have webhook policies targeting the correct pod labels", func(ctx SpecContext) {
			for policyName, expectedLabel := range map[string]map[string]string{
				"csidriver-webhook-ingress":  {"name": "shared-resource-csi-driver-webhook"},
				"shipwright-webhook-ingress": {"name": "shp-build-webhook"},
			} {
				np := &networkingv1.NetworkPolicy{}
				Expect(kubeClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: openshiftBuildsNS}, np)).To(Succeed())
				for k, v := range expectedLabel {
					Expect(np.Spec.PodSelector.MatchLabels).To(HaveKeyWithValue(k, v),
						"%s should target pods with %s=%s", policyName, k, v)
				}
			}
		})
	})

	Context("default-deny-ingress", func() {
		It("should BLOCK arbitrary traffic from an external namespace to openshift-builds pods", func(ctx SpecContext) {
			operatorPods := &corev1.PodList{}
			Expect(kubeClient.List(ctx, operatorPods, client.InNamespace(openshiftBuildsNS),
				client.MatchingLabels{"control-plane": "controller-manager"})).To(Succeed())
			Expect(operatorPods.Items).NotTo(BeEmpty(), "operator pod must exist")
			ip := operatorPods.Items[0].Status.PodIP
			Expect(ip).NotTo(BeEmpty(), "operator pod must have an IP")

			By(fmt.Sprintf("Probing operator pod %s:8443 from builds-test (expect BLOCKED)", ip))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, ip, 8443)).To(
				BeFalse(),
				"default-deny-ingress should DROP traffic from builds-test to operator pod (no specific allow policy covers it)")
		})
	})

	Context("monitoring-metrics-ingress-csi", func() {
		var csiPodIP string

		BeforeEach(func(ctx SpecContext) {
			pods := &corev1.PodList{}
			Expect(kubeClient.List(ctx, pods, client.InNamespace(openshiftBuildsNS),
				client.MatchingLabels{"app": "shared-resource-csi-driver-node"})).To(Succeed())
			if len(pods.Items) == 0 || pods.Items[0].Status.PodIP == "" {
				Skip("CSI DaemonSet pod has no IP yet — skipping enforcement test")
			}
			csiPodIP = pods.Items[0].Status.PodIP
		})

		It("should ALLOW a monitoring-labeled namespace to reach CSI metrics port 9898", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing CSI pod %s:9898 from monitoring namespace (expect ALLOWED)", csiPodIP))
			Expect(netpolIsAllowed(netpolMonitoringNS, netpolMonPodName, csiPodIP, 9898)).To(
				BeTrue(),
				"monitoring-metrics-ingress-csi should ALLOW monitoring namespace → CSI pod port 9898")
		})

		It("should ALLOW a monitoring-labeled namespace to reach CSI metrics port 6000", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing CSI pod %s:6000 from monitoring namespace (expect ALLOWED)", csiPodIP))
			Expect(netpolIsAllowed(netpolMonitoringNS, netpolMonPodName, csiPodIP, 6000)).To(
				BeTrue(),
				"monitoring-metrics-ingress-csi should ALLOW monitoring namespace → CSI pod port 6000")
		})

		It("should BLOCK a non-monitoring namespace from reaching CSI metrics port 9898", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing CSI pod %s:9898 from builds-test (expect BLOCKED)", csiPodIP))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, csiPodIP, 9898)).To(
				BeFalse(),
				"default-deny-ingress should DROP traffic from non-monitoring namespace → CSI pod port 9898")
		})

		It("should BLOCK a non-monitoring namespace from reaching CSI metrics port 6000", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing CSI pod %s:6000 from builds-test (expect BLOCKED)", csiPodIP))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, csiPodIP, 6000)).To(
				BeFalse(),
				"default-deny-ingress should DROP traffic from non-monitoring namespace → CSI pod port 6000")
		})
	})

	Context("monitoring-metrics-ingress-shipwright", func() {
		var shipwrightPodIP string

		BeforeEach(func(ctx SpecContext) {
			pods := &corev1.PodList{}
			Expect(kubeClient.List(ctx, pods, client.InNamespace(openshiftBuildsNS),
				client.MatchingLabels{"name": "shipwright-build"})).To(Succeed())
			if len(pods.Items) == 0 || pods.Items[0].Status.PodIP == "" {
				Skip("Shipwright build pod has no IP yet — skipping enforcement test")
			}
			shipwrightPodIP = pods.Items[0].Status.PodIP
		})

		It("should ALLOW a monitoring-labeled namespace to reach Shipwright metrics port 8383", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing Shipwright pod %s:8383 from monitoring namespace (expect ALLOWED)", shipwrightPodIP))
			Expect(netpolIsAllowed(netpolMonitoringNS, netpolMonPodName, shipwrightPodIP, 8383)).To(
				BeTrue(),
				"monitoring-metrics-ingress-shipwright should ALLOW monitoring namespace → Shipwright pod port 8383")
		})

		It("should BLOCK a non-monitoring namespace from reaching Shipwright metrics port 8383", func(ctx SpecContext) {
			By(fmt.Sprintf("Probing Shipwright pod %s:8383 from builds-test (expect BLOCKED)", shipwrightPodIP))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, shipwrightPodIP, 8383)).To(
				BeFalse(),
				"default-deny-ingress should DROP traffic from non-monitoring namespace → Shipwright pod port 8383")
		})
	})

	Context("webhook ingress policies", func() {
		It("should BLOCK non-kube-apiserver namespace from reaching CSI webhook port 8443", func(ctx SpecContext) {
			pods := &corev1.PodList{}
			Expect(kubeClient.List(ctx, pods, client.InNamespace(openshiftBuildsNS),
				client.MatchingLabels{"name": "shared-resource-csi-driver-webhook"})).To(Succeed())
			if len(pods.Items) == 0 || pods.Items[0].Status.PodIP == "" {
				Skip("CSI webhook pod has no IP yet — skipping enforcement test")
			}
			ip := pods.Items[0].Status.PodIP

			By(fmt.Sprintf("Probing CSI webhook pod %s:8443 from builds-test (expect BLOCKED)", ip))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, ip, 8443)).To(
				BeFalse(),
				"csidriver-webhook-ingress should only allow kube-apiserver; builds-test traffic must be DROPPED")
		})

		It("should BLOCK non-kube-apiserver namespace from reaching Shipwright webhook port 8443", func(ctx SpecContext) {
			pods := &corev1.PodList{}
			Expect(kubeClient.List(ctx, pods, client.InNamespace(openshiftBuildsNS),
				client.MatchingLabels{"name": "shp-build-webhook"})).To(Succeed())
			if len(pods.Items) == 0 || pods.Items[0].Status.PodIP == "" {
				Skip("Shipwright webhook pod has no IP yet — skipping enforcement test")
			}
			ip := pods.Items[0].Status.PodIP

			By(fmt.Sprintf("Probing Shipwright webhook pod %s:8443 from builds-test (expect BLOCKED)", ip))
			Expect(netpolIsAllowed(testNamespace, netpolDenyPodName, ip, 8443)).To(
				BeFalse(),
				"shipwright-webhook-ingress should only allow kube-apiserver; builds-test traffic must be DROPPED")
		})
	})

	Context("kube-apiserver to Shipwright webhook", func() {
		It("should allow kube-apiserver to reach the Shipwright admission webhook", func(ctx SpecContext) {
			build := &buildv1beta1.Build{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "netpol-webhook-probe-",
					Namespace:    testNamespace,
				},
				Spec: buildv1beta1.BuildSpec{
					Source: &buildv1beta1.Source{
						Type: buildv1beta1.GitType,
						Git:  &buildv1beta1.Git{URL: "https://github.com/example/nonexistent"},
					},
					Strategy: buildv1beta1.Strategy{Name: "buildpacks-extender"},
					Output:   buildv1beta1.Image{Image: "example.com/nonexistent:latest"},
				},
			}
			err := kubeClient.Create(ctx, build)
			defer func() { _ = kubeClient.Delete(ctx, build) }()

			if err != nil {
				Expect(err.Error()).NotTo(MatchRegexp("(context deadline exceeded|i/o timeout)"),
					"webhook timed out — shipwright-webhook-ingress may be blocking kube-apiserver → shp-build-webhook")
			}
		})
	})
})

// netpolProbePod returns a minimal pod that sleeps indefinitely, used as a source for TCP probes.
// The security context satisfies OpenShift's restricted PodSecurity profile.
func netpolProbePod(name, namespace string) *corev1.Pod {
	allowPrivEsc := false
	runAsNonRoot := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": "netpol-probe"},
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   &runAsNonRoot,
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{{
				Name:    "probe",
				Image:   "registry.access.redhat.com/ubi9/ubi:latest",
				Command: []string{"sh", "-c", "sleep infinity"},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &allowPrivEsc,
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
			}},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
}

// netpolIsAllowed attempts a TCP connection from podName in namespace to targetIP:targetPort
// and returns true if it completes within tcpProbeTimeout seconds 
func netpolIsAllowed(namespace, podName, targetIP string, targetPort int) bool {
	cmd := exec.Command("kubectl", "exec",
		"-n", namespace, podName,
		"-c", "probe", "--",
		"bash", "-c",
		fmt.Sprintf(
			"timeout %d bash -c 'echo > /dev/tcp/%s/%d' 2>/dev/null; code=$?; [ $code -ne 124 ]",
			tcpProbeTimeout, targetIP, targetPort,
		),
	)
	output, err := cmd.CombinedOutput()
	GinkgoWriter.Printf("netpolIsAllowed %s/%s → %s:%d err=%v out=%q\n",
		namespace, podName, targetIP, targetPort, err, strings.TrimSpace(string(output)))
	return err == nil
}
