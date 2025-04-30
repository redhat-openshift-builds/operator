package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-openshift-builds/operator/test/utils"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Entitlements for OpenShift builds operator", Label("e2e"), Label("entitlements"), func() {

	Context("When testing entitlement access via a pod", func() {

		AfterEach(func() {
			By("Cleaning up resources after pod-based entitlement access test")
			podToDel := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testingNamespace,
					Name:      entitlementPod,
				},
			}
			Expect(kubeClient.Delete(context.Background(), podToDel)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")

			for _, resource := range entitledPodResources {
				Expect(utils.DeleteResourceFromFile(context.Background(), kubeClient, projectDir+resource)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
			}
		})

		It("Should access entitled RHEL content", func(ctx SpecContext) {
			By("Ensuring etc-pki-entitlement secret exists")
			secretKey := client.ObjectKey{
				Name:      entitlementSecret,
				Namespace: openshiftconfigNamespace,
			}
			Eventually(func() error {
				return kubeClient.Get(ctx, secretKey, &corev1.Secret{})
			}, 10*time.Second).Should(Succeed())

			By("Applying RBAC and shared secret resources")
			for _, resource := range entitledPodResources {
				Expect(utils.ApplyResourceFromFile(ctx, kubeClient, projectDir+resource)).NotTo(HaveOccurred(), fmt.Sprintf("applying resource: %s", resource))
			}

			By("Creating a pod to use the entitlement secret")
			Expect(utils.ApplyResourceFromFile(ctx, kubeClient, projectDir+entitlementTestPodFile)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")

			By("Waiting for the pod to complete")
			Eventually(func() error {
				pod := &corev1.Pod{}
				err := kubeClient.Get(ctx, client.ObjectKey{Namespace: testingNamespace, Name: entitlementPod}, pod)
				if err != nil {
					return err
				}
				switch pod.Status.Phase {
				case corev1.PodSucceeded:
					By("Entitled access in pod completed successfully")
					return nil
				case corev1.PodFailed:
					return fmt.Errorf("pod %s failed", entitlementPod)
				default:
					return fmt.Errorf("pod %s is not yet completed, current phase: %s", entitlementPod, pod.Status.Phase)
				}
			}, 5*time.Minute, 10*time.Second).Should(Succeed(), fmt.Sprintf("waiting for pod %s to complete", entitlementPod))
		})
	})

	Context("When testing entitlement access with shared secret", func() {

		AfterEach(func(ctx SpecContext) {
			By("Cleaning up resources after shared secret-based entitlement access test")
			for _, resource := range entitledBuildResources {
				Expect(utils.DeleteResourceFromFile(ctx, kubeClient, projectDir+resource)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
			}
			err := utils.DeleteResourceFromFile(ctx, kubeClient, projectDir+entitledBuildRunFile)
			Expect(err).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
		})

		It("Should access entitled content using shared secret", func(ctx SpecContext) {
			By("Applying RBAC, shared secret, build resources")
			for _, resource := range entitledBuildResources {
				Expect(utils.ApplyResourceFromFile(ctx, kubeClient, projectDir+resource)).NotTo(HaveOccurred(), fmt.Sprintf("applying resource: %s", resource))
			}

			By("Applying buildrun")
			Expect(utils.ApplyResourceFromFile(ctx, kubeClient, projectDir+entitledBuildRunFile)).NotTo(HaveOccurred(), "applying entitled-buildrun.yaml")

			By("Waiting for buildrun to complete")
			Eventually(func() error {
				buildRun := &buildv1beta1.BuildRun{}
				err := kubeClient.Get(ctx, client.ObjectKey{Namespace: testingNamespace, Name: buildRunCrName}, buildRun)
				if err != nil {
					return fmt.Errorf("failed to get buildrun entitled-br: %w", err)
				}
				if !buildRun.IsDone() {
					return fmt.Errorf("BuildRun 'entitled-br' is not yet completed")
				}

				if buildRun.IsSuccessful() {
					By("BuildRun 'entitled-br' completed successfully")
					return nil
				}

				return fmt.Errorf("BuildRun '%s' failed with reason: %s, message: %s", buildRunCrName,
					buildRun.Status.FailureDetails.Reason, buildRun.Status.FailureDetails.Message)
			}, 5*time.Minute, 15*time.Second).Should(Succeed())
		})
	})
})
