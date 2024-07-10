package build_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-builds/operator/internal/common"
	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	_ "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/redhat-openshift-builds/operator/internal/shipwright/build"
)

var _ = Describe("Build", Label("shipwright", "build"), func() {
	var (
		ctx             context.Context
		err             error
		result          controllerutil.OperationResult
		shipwrightBuild *build.ShipwrightBuild
		list            *shipwrightv1alpha1.ShipwrightBuildList
		object          *shipwrightv1alpha1.ShipwrightBuild
	)

	BeforeEach(OncePerOrdered, func() {
		ctx = context.Background()
		shipwrightBuild = build.New(fake.NewClientBuilder().WithScheme(scheme).Build())
	})

	JustBeforeEach(OncePerOrdered, func() {
		list = &shipwrightv1alpha1.ShipwrightBuildList{}
		object = &shipwrightv1alpha1.ShipwrightBuild{}
		result, err = shipwrightBuild.CreateOrUpdate(ctx, owner)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(shipwrightBuild.Client.List(ctx, list)).To(Succeed())
		Expect(list.Items).ShouldNot(BeEmpty())
		object = &list.Items[0]
	})

	Describe("Getting resource", Label("get"), func() {
		When("there are no existing resources", func() {
			JustBeforeEach(func() {
				Expect(shipwrightBuild.Delete(ctx, owner)).To(Succeed())
			})
			It("should throw NotFound error", func() {
				_, err := shipwrightBuild.Get(ctx, owner)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})
		When("there is an existing resource", func() {
			It("should successfully fetch the resource ", func() {
				fetchedObject, err := shipwrightBuild.Get(ctx, owner)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(fetchedObject).To(Equal(object))
			})
		})
	})

	Describe("Creating resource", Label("create", "update"), func() {
		When("the resource doesn't exists", Ordered, func() {
			It("should create a new resource", func() {
				Expect(result).To(Equal(controllerutil.OperationResultCreated))
			})
			It("should should have one instance", func() {
				Expect(list.Items).To(HaveLen(1))
			})
			It("should prefix the owner name", func() {
				Expect(object.GetName()).To(ContainSubstring(owner.GetName()))
			})
			It("should have OpenshiftBuild finalizer", func() {
				Expect(object.GetFinalizers()).To(ContainElement(common.OpenShiftBuildFinalizerName))
			})
			It("should have the controller reference", func() {
				Expect(metav1.IsControlledBy(object, owner)).To(BeTrue())
			})
			It("should have target namespace set to openshift build", func() {
				Expect(object.Spec.TargetNamespace).To(Equal(common.OpenShiftBuildNamespaceName))
			})
		})
		When("there is an existing resource with same spec", Ordered, func() {
			BeforeAll(func() {
				result, err := shipwrightBuild.CreateOrUpdate(ctx, owner)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result).To(Equal(controllerutil.OperationResultCreated))
			})
			It("should not create any resource", func() {
				Expect(list.Items).To(HaveLen(1))
			})
			It("should not change any resource", func() {
				Expect(result).To(Equal(controllerutil.OperationResultNone))
			})
		})
		When("there is an existing resource different spec", Ordered, func() {
			BeforeAll(func() {
				object := &shipwrightv1alpha1.ShipwrightBuild{}
				object.SetName("test")
				object.Spec.TargetNamespace = "test"
				ctrl.SetControllerReference(owner, object, scheme)
				Expect(shipwrightBuild.Client.Create(ctx, object)).To(Succeed())
			})
			It("should not create any new resource", func() {
				Expect(list.Items).To(HaveLen(1))
			})
			It("should not create any resource", func() {
				Expect(result).To(Equal(controllerutil.OperationResultUpdated))
			})
			It("should update the specs to match expected", func() {
				Expect(object.Spec.TargetNamespace).To(Equal(common.OpenShiftBuildNamespaceName))
			})
		})
	})

	Describe("Deleting resource", Label("delete"), Ordered, func() {
		When("there is an existing resource", func() {
			It("should successfully delete the resource", func() {
				err := shipwrightBuild.Delete(ctx, owner)
				Expect(err).ShouldNot(HaveOccurred())
				_, err = shipwrightBuild.Get(ctx, owner)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})
		When("the resource doesn't exists", func() {
			It("should return NotFound error", func() {
				err := shipwrightBuild.Delete(ctx, owner)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})
})
