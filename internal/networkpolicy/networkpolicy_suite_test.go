package networkpolicy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	operatorv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	"github.com/redhat-openshift-builds/operator/internal/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var scheme *runtime.Scheme
var owner *operatorv1alpha1.OpenShiftBuild

func TestNetworkPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NetworkPolicy Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(networkingv1.AddToScheme(scheme)).To(Succeed())

	owner = &operatorv1alpha1.OpenShiftBuild{}
	owner.SetName("cluster")
	owner.SetUID(uuid.NewUUID())
	owner.SetGroupVersionKind(operatorv1alpha1.GroupVersion.WithKind("OpenShiftBuild"))
})

func newUnstructuredNetworkPolicy(name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("networking.k8s.io/v1")
	obj.SetKind("NetworkPolicy")
	obj.SetName(name)
	obj.SetNamespace(common.OpenShiftBuildNamespaceName)
	return obj
}
