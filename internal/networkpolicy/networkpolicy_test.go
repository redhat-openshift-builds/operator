package networkpolicy_test

import (
	"context"
	"path/filepath"

	manifestivalclient "github.com/manifestival/controller-runtime-client"
	"github.com/manifestival/manifestival"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	"github.com/redhat-openshift-builds/operator/internal/networkpolicy"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("NetworkPolicy", Label("networkpolicy"), func() {
	var (
		ctx       context.Context
		np        *networkpolicy.NetworkPolicy
		k8sClient client.Client
		manifest  manifestival.Manifest
	)

	BeforeEach(OncePerOrdered, func() {
		ctx = context.Background()
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		resources := []unstructured.Unstructured{
			*newUnstructuredNetworkPolicy("default-deny-ingress"),
			*newUnstructuredNetworkPolicy("webhook-ingress"),
		}
		var err error
		manifest, err = manifestival.ManifestFrom(
			manifestival.Slice(resources),
			manifestival.UseClient(manifestivalclient.NewClient(k8sClient)),
		)
		Expect(err).NotTo(HaveOccurred())

		np = networkpolicy.New(k8sClient, manifest, log.Log.WithName("test"))
	})

	Describe("Reconcile", Label("reconcile"), func() {

		When("the owner is not being deleted", Ordered, func() {
			var reconcileOwner *operatorv1alpha1.OpenShiftBuild

			BeforeAll(func() {
				reconcileOwner = owner.DeepCopy()
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())
			})

			It("should create all NetworkPolicy resources", func() {
				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				Expect(netpolList.Items).To(HaveLen(2))
			})

			It("should set the owner reference", func() {
				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				for _, item := range netpolList.Items {
					Expect(metav1.IsControlledBy(&item, reconcileOwner)).To(BeTrue())
				}
			})

			It("should set the finalizer", func() {
				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				for _, item := range netpolList.Items {
					Expect(item.GetFinalizers()).To(ContainElement(common.OpenShiftBuildFinalizerName))
				}
			})
		})

		When("the owner is being deleted", Ordered, func() {
			var reconcileOwner *operatorv1alpha1.OpenShiftBuild

			BeforeAll(func() {
				reconcileOwner = owner.DeepCopy()
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())

				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				Expect(netpolList.Items).To(HaveLen(2))

				now := metav1.Now()
				reconcileOwner.DeletionTimestamp = &now
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())
			})

			It("should delete all NetworkPolicy resources", func() {
				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				Expect(netpolList.Items).To(BeEmpty())
			})
		})

		When("reconcile is called multiple times", func() {
			It("should be idempotent", func() {
				reconcileOwner := owner.DeepCopy()
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())

				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				Expect(netpolList.Items).To(HaveLen(2))
			})
		})

		When("no resources exist during deletion", func() {
			It("should not error and should leave no resources behind", func() {
				reconcileOwner := owner.DeepCopy()
				now := metav1.Now()
				reconcileOwner.DeletionTimestamp = &now
				Expect(np.Reconcile(ctx, reconcileOwner)).To(Succeed())

				netpolList := &networkingv1.NetworkPolicyList{}
				Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
				Expect(netpolList.Items).To(BeEmpty())
			})
		})
	})

	Describe("Reconcile from manifest files", Label("manifest"), func() {
		var fileNp *networkpolicy.NetworkPolicy

		BeforeEach(func() {
			manifestPath := filepath.Join("..", "..", "config", "networkpolicies")
			Expect(manifestPath).To(BeADirectory(),
				"NetworkPolicy manifest directory must exist at %s", manifestPath)
			fileManifest, err := manifestival.NewManifest(
				manifestPath,
				manifestival.UseClient(manifestivalclient.NewClient(k8sClient)),
			)
			Expect(err).NotTo(HaveOccurred())
			fileNp = networkpolicy.New(k8sClient, fileManifest, log.Log.WithName("test-file"))
		})

		It("should apply all 5 NetworkPolicy resources from the manifest", func() {
			reconcileOwner := owner.DeepCopy()
			Expect(fileNp.Reconcile(ctx, reconcileOwner)).To(Succeed())

			netpolList := &networkingv1.NetworkPolicyList{}
			Expect(k8sClient.List(ctx, netpolList, client.InNamespace(common.OpenShiftBuildNamespaceName))).To(Succeed())
			Expect(netpolList.Items).To(HaveLen(5))

			names := make([]string, len(netpolList.Items))
			for i, item := range netpolList.Items {
				names[i] = item.Name
			}
			Expect(names).To(ContainElements(
				"default-deny-ingress",
				"csidriver-webhook-ingress",
				"shipwright-webhook-ingress",
				"monitoring-metrics-ingress-csi",
				"monitoring-metrics-ingress-shipwright",
			))
		})
	})
})
