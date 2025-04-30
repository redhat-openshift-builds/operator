package build_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shipwrightv1alpha1 "github.com/shipwright-io/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var scheme *runtime.Scheme
var owner *unstructured.Unstructured

func TestBuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Build Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	shipwrightv1alpha1.AddToScheme(scheme)

	// create an owner object
	gvk := schema.GroupVersionKind{
		Group:   "shipwright.io",
		Version: "v1",
		Kind:    "Owner",
	}
	owner = &unstructured.Unstructured{}
	owner.SetName("owner")
	owner.SetGroupVersionKind(gvk)
	owner.SetUID(uuid.NewUUID())
	scheme.AddKnownTypeWithName(gvk, owner)
})
