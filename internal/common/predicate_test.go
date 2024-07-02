package common

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func TestIsControlledBy(t *testing.T) {
	RegisterFailHandler(Fail)
	owner := &metav1.OwnerReference{
		APIVersion:         "test/v1",
		Kind:               "Owner",
		BlockOwnerDeletion: ptr.To(true),
		Controller:         ptr.To(true),
	}
	t.Run("object is controlled by owner", func(t *testing.T) {
		object := &unstructured.Unstructured{}
		object.SetName("test")
		object.SetOwnerReferences([]metav1.OwnerReference{*owner})
		Expect(IsControlledBy(object, owner)).To(BeTrue())
	})
	t.Run("object is not controlled by owner", func(t *testing.T) {
		object := &unstructured.Unstructured{}
		object.SetName("test")
		object.SetOwnerReferences([]metav1.OwnerReference{{
			APIVersion: "test/v1",
			Kind:       "Test",
		}})
		Expect(IsControlledBy(object, owner)).To(BeFalse())
	})
}
