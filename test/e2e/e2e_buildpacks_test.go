package e2e

import (
	"fmt"
	"io"

	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	. "github.com/onsi/gomega"    //nolint:golint,revive

	"sigs.k8s.io/yaml"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Buildpacks e2e tests", Label("buildpacks"), func() {

	Context("When using the buildpacks ClusterBuildStrategy with a sample Node.js app", func() {
		var (
			build           *buildv1beta1.Build
			createdBuildRun *buildv1beta1.BuildRun
		)

		AfterEach(func(ctx SpecContext) {
			if CurrentSpecReport().Failed() && createdBuildRun != nil {
				By(fmt.Sprintf("Dumping logs for failed BuildRun: %s", createdBuildRun.Name))
				podList := &corev1.PodList{}
				err := kubeClient.List(ctx, podList, client.InNamespace(testNamespace), client.MatchingLabels{
					"buildrun.shipwright.io/name": createdBuildRun.Name,
				})
				if err != nil {
					GinkgoWriter.Printf("Error listing pods for BuildRun %s: %v\n", createdBuildRun.Name, err)
				} else if len(podList.Items) > 0 {
					buildPod := &podList.Items[0]
					for _, container := range buildPod.Spec.Containers {
						GinkgoWriter.Printf("\n----- Logs from container: %s (Pod: %s)-----\n", container.Name, buildPod.Name)
						req := clientset.CoreV1().Pods(testNamespace).GetLogs(buildPod.Name, &corev1.PodLogOptions{Container: container.Name})
						logStream, streamErr := req.Stream(ctx)
						if streamErr != nil {
							GinkgoWriter.Printf("Error streaming logs for container %s in pod %s: %v\n", container.Name, buildPod.Name, streamErr)
							continue
						}
						defer logStream.Close()
						if _, copyErr := io.Copy(GinkgoWriter, logStream); copyErr != nil {
							GinkgoWriter.Printf("Error copying log stream for container %s in pod %s: %v\n", container.Name, buildPod.Name, copyErr)
						}
					}
					GinkgoWriter.Println("\n----- End of logs -----")
				}
			}

			By("Cleaning up test-specific Build and BuildRun resources")
			if createdBuildRun != nil {
				GinkgoWriter.Printf("Attempting to delete BuildRun '%s'\n", createdBuildRun.Name)
				if err := kubeClient.Delete(ctx, createdBuildRun); err != nil && !errors.IsNotFound(err) {
					GinkgoWriter.Printf("Error deleting BuildRun '%s': %v\n", createdBuildRun.Name, err)
				}
				Eventually(func() bool {
					err := kubeClient.Get(ctx, client.ObjectKeyFromObject(createdBuildRun), &buildv1beta1.BuildRun{})
					return errors.IsNotFound(err)
				}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "BuildRun not deleted within timeout")
			}
			if build != nil {
				GinkgoWriter.Printf("Attempting to delete Build '%s'\n", build.Name)
				if err := kubeClient.Delete(ctx, build); err != nil && !errors.IsNotFound(err) {
					GinkgoWriter.Printf("Error deleting Build '%s': %v\n", build.Name, err)
				}
				Eventually(func() bool {
					err := kubeClient.Get(ctx, client.ObjectKeyFromObject(build), &buildv1beta1.Build{})
					return errors.IsNotFound(err)
				}, 30*time.Second, 2*time.Second).Should(BeTrue(), "Build not deleted within timeout")
			}
		})

		It("Should successfully trigger and complete a BuildRun", func(ctx SpecContext) {
			buildPath := filepath.Join(projectDir, "test", "data", "buildpack-nodejs-build.yaml")
			By(fmt.Sprintf("Loading Build resource from file: %s", buildPath))
			buildPayload, err := os.ReadFile(buildPath)
			Expect(err).NotTo(HaveOccurred(), "failed to read build definition YAML file")

			// Replace the placeholder in the output image with the correct testing namespace.
			buildYAML := strings.Replace(string(buildPayload), "##NAMESPACE##", testNamespace, 1)

			build = &buildv1beta1.Build{}
			err = yaml.Unmarshal([]byte(buildYAML), build)
			Expect(err).NotTo(HaveOccurred(), "failed to unmarshal build YAML into object")

			build.Namespace = testNamespace

			By(fmt.Sprintf("Ensuring Build resource '%s' doesn't exist before creation", build.Name))
			existingBuild := &buildv1beta1.Build{}
			err = kubeClient.Get(ctx, client.ObjectKey{Name: build.Name, Namespace: testNamespace}, existingBuild)
			if err == nil {
				GinkgoWriter.Printf("Build '%s' already exists, deleting it before recreation.\n", build.Name)
				err = kubeClient.Delete(ctx, existingBuild)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to delete existing Build '%s'", build.Name))
				Eventually(func() bool {
					err := kubeClient.Get(ctx, client.ObjectKey{Name: build.Name, Namespace: testNamespace}, &buildv1beta1.Build{})
					return errors.IsNotFound(err)
				}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "existing Build %s not deleted within timeout", build.Name)
			} else if !errors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to check existence of Build '%s'", build.Name))
			}

			By(fmt.Sprintf("Creating Build resource '%s'", build.Name))
			Expect(kubeClient.Create(ctx, build)).To(Succeed(), "failed to create Build resource")

			buildRunTemplate := &buildv1beta1.BuildRun{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-run-", build.Name),
					Namespace:    testNamespace,
				},
				Spec: buildv1beta1.BuildRunSpec{
					Build: buildv1beta1.ReferencedBuild{
						Name: &build.Name,
					},
				},
			}
			By(fmt.Sprintf("Creating BuildRun for Build '%s'", build.Name))
			Expect(kubeClient.Create(ctx, buildRunTemplate)).To(Succeed(), "failed to create BuildRun")

			By("Waiting for BuildRun to come up with its generated name")
			Eventually(func() error {
				brList := &buildv1beta1.BuildRunList{}
				err := kubeClient.List(ctx, brList, client.InNamespace(testNamespace))
				if err != nil {
					return err
				}
				for i := range brList.Items {
					br := &brList.Items[i]
					if strings.HasPrefix(br.Name, build.Name) && br.Spec.Build.Name != nil && *br.Spec.Build.Name == build.Name {
						createdBuildRun = br
						return nil
					}
				}
				return fmt.Errorf("buildrun for build %s not found yet", build.Name)
			}, "1m", "2s").Should(Succeed(), "failed to fetch created BuildRun")

			By(fmt.Sprintf("Waiting for BuildRun '%s' to complete successfully", createdBuildRun.Name))
			Eventually(func(g Gomega) {
				br := &buildv1beta1.BuildRun{}
				err := kubeClient.Get(ctx, client.ObjectKey{Name: createdBuildRun.Name, Namespace: testNamespace}, br)
				g.Expect(err).NotTo(HaveOccurred())

				createdBuildRun = br

				succeededCondition := br.Status.GetCondition(buildv1beta1.Succeeded)
				g.Expect(succeededCondition).NotTo(BeNil(), "Succeeded condition should not be nil")
				g.Expect(succeededCondition.Status).To(Equal(corev1.ConditionTrue), "BuildRun should have succeeded")

			}, 2*time.Minute, 10*time.Second).Should(Succeed(), "BuildRun %s did not succeed within timeout.", createdBuildRun.Name)

			GinkgoWriter.Printf("BuildRun %s completed successfully.\n", createdBuildRun.Name)
		})
	})
})
