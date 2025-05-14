package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"

	openshiftv1alpha1 "github.com/redhat-openshift-builds/operator/api/v1alpha1"
	"github.com/redhat-openshift-builds/operator/internal/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MockShipwrightManager implements the ShipwrightManager interface for testing
type MockShipwrightManager struct {
	CreateOrUpdateFunc func(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error)
	DeleteFunc         func(ctx context.Context, owner client.Object) error
}

func (m *MockShipwrightManager) CreateOrUpdate(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error) {
	if m.CreateOrUpdateFunc != nil {
		return m.CreateOrUpdateFunc(ctx, owner)
	}
	return controllerutil.OperationResultNone, nil
}

func (m *MockShipwrightManager) Delete(ctx context.Context, owner client.Object) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, owner)
	}
	return nil
}

// Ensure MockShipwrightManager satisfies the interface (compile-time check)
// This line acts as a compile-time assertion ensuring that the mock struct *MockShipwrightManager correctly implements all methods required by the ShipwrightManager interface.

var _ ShipwrightManager = (*MockShipwrightManager)(nil)

// MockSharedResourceManager implements the SharedResourceManager interface for testing
type MockSharedResourceManager struct {
	ReconcileFunc func(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error
}

func (m *MockSharedResourceManager) Reconcile(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
	if m.ReconcileFunc != nil {
		return m.ReconcileFunc(ctx, owner)
	}
	return nil
}

// Ensure MockSharedResourceManager satisfies the interface (compile-time check)
// This line acts as a compile-time assertion ensuring that the mock struct *MockSharedResourceManager correctly implements all methods required by the SharedResourceManager interface.
var _ SharedResourceManager = (*MockSharedResourceManager)(nil)

// TestOpenShiftBuildReconciler verifies the fix for the requeue bug
func TestOpenShiftBuildReconciler(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	// Create a new scheme and add relevant types (operator's API)
	scheme := runtime.NewScheme()
	err := openshiftv1alpha1.AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred(), "Scheme setup should succeed")

	// Define common names and the specific error to simulate
	testObjectName := common.OpenShiftBuildResourceName
	testObjectNsN := types.NamespacedName{Name: testObjectName} // Assuming cluster-scoped or default ns
	shipwrightReconcileError := errors.New("forced shipwright reconcile error for test")

	t.Run("FIXED: should return error when reconcile fails, triggering requeue", func(t *testing.T) {
		// Arrange
		// Reset Gomega for sub-test
		g := NewGomegaWithT(t)
		// 1. Initial Object State
		initialBuild := &openshiftv1alpha1.OpenShiftBuild{
			ObjectMeta: metav1.ObjectMeta{
				Name:       testObjectName,
				Generation: 1,
				Finalizers: []string{common.OpenShiftBuildFinalizerName},
			},
			Spec: openshiftv1alpha1.OpenShiftBuildSpec{
				Shipwright: &openshiftv1alpha1.Shipwright{
					Build: &openshiftv1alpha1.ShipwrightBuild{
						State: openshiftv1alpha1.Enabled,
					},
				},
				SharedResource: &openshiftv1alpha1.SharedResource{
					State: openshiftv1alpha1.Enabled,
				},
			},
			Status: openshiftv1alpha1.OpenShiftBuildStatus{
				// Provide an initial status to bypass the status init block if needed
				Conditions: []metav1.Condition{
					{
						Type:   openshiftv1alpha1.ConditionReady,
						Status: metav1.ConditionUnknown, // Start as Unknown
						Reason: "TestingInitialState",
					},
				},
			},
		}

		// 2. Fake Client Setup
		// The fake client's Patch/UpdateStatus defaults to success (returns nil error).
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(initialBuild).           // Add the initial object state
			WithStatusSubresource(initialBuild). // Enable status mocking
			Build()

		// 3. Mock Dependencies Setup
		mockShipwright := &MockShipwrightManager{
			CreateOrUpdateFunc: func(ctx context.Context, owner client.Object) (controllerutil.OperationResult, error) {
				return controllerutil.OperationResultNone, shipwrightReconcileError
			},
			DeleteFunc: func(ctx context.Context, owner client.Object) error {
				return nil
			},
		}
		mockSharedResource := &MockSharedResourceManager{
			ReconcileFunc: func(ctx context.Context, owner *openshiftv1alpha1.OpenShiftBuild) error {
				return nil
			},
		}

		// 4. Reconciler Instantiation
		// Uses interfaces defined in openshiftbuild_controller.go
		reconciler := &OpenShiftBuildReconciler{
			Client:         fakeClient,
			APIReader:      fakeClient,
			Scheme:         scheme,
			Logger:         logr.Discard(),     // Suppress logs during test
			Shipwright:     mockShipwright,     // Inject Shipwright mock
			SharedResource: mockSharedResource, // Inject SharedResource mock
		}

		// Act
		// 5. Perform Reconciliation
		req := ctrl.Request{NamespacedName: testObjectNsN}
		result, err := reconciler.Reconcile(ctx, req)

		// Assert
		g.Expect(err).To(HaveOccurred(), "FIX VERIFIED: Reconcile should now return the original error")
		g.Expect(err).To(MatchError(shipwrightReconcileError), "FIX VERIFIED: The returned error should be the one from the failed component")
		g.Expect(result).To(Equal(ctrl.Result{}), "FIX VERIFIED: Reconcile should return empty result when returning an error")
	})
}
