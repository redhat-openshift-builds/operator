/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// ConditionReady object is providing service.
	ConditionReady = "Ready"
)

// State defines the desired state of the component
// +kubebuilder:validation:Enum="Enabled";"Disabled"
type State string

const (
	// Enabled will install the component
	Enabled State = "Enabled"

	// Disabled will remove the component
	Disabled State = "Disabled"
)

// ShipwrightBuild defines the desired state of Shipwright Builds
type ShipwrightBuild struct {
	// State defines the desired state of the Shipwright ShipwrightBuild component
	// +kubebuilder:default="Enabled"
	State `json:"state"`
}

// Shipwright defines the desired state of Shipwright components
type Shipwright struct {
	// Build defines the desired state of a Shipwright Build component
	// +kubebuilder:validation:Optional
	// +optional
	Build *ShipwrightBuild `json:"build,omitempty"`
}

// SharedResource defines the desired state of Shared Resources CSI Driver
type SharedResource struct {
	// State defines the desired state of SharedResource component
	// +kubebuilder:default="Enabled"
	State `json:"state"`
}

// DeepCopyObject implements client.Object.
func (in *SharedResource) DeepCopyObject() runtime.Object {
	panic("unimplemented")
}

// GetAnnotations implements client.Object.
func (in *SharedResource) GetAnnotations() map[string]string {
	panic("unimplemented")
}

// GetCreationTimestamp implements client.Object.
func (in *SharedResource) GetCreationTimestamp() metav1.Time {
	panic("unimplemented")
}

// GetDeletionGracePeriodSeconds implements client.Object.
func (in *SharedResource) GetDeletionGracePeriodSeconds() *int64 {
	panic("unimplemented")
}

// GetDeletionTimestamp implements client.Object.
func (in *SharedResource) GetDeletionTimestamp() *metav1.Time {
	panic("unimplemented")
}

// GetFinalizers implements client.Object.
func (in *SharedResource) GetFinalizers() []string {
	panic("unimplemented")
}

// GetGenerateName implements client.Object.
func (in *SharedResource) GetGenerateName() string {
	panic("unimplemented")
}

// GetGeneration implements client.Object.
func (in *SharedResource) GetGeneration() int64 {
	panic("unimplemented")
}

// GetLabels implements client.Object.
func (in *SharedResource) GetLabels() map[string]string {
	panic("unimplemented")
}

// GetManagedFields implements client.Object.
func (in *SharedResource) GetManagedFields() []metav1.ManagedFieldsEntry {
	panic("unimplemented")
}

// GetName implements client.Object.
func (in *SharedResource) GetName() string {
	panic("unimplemented")
}

// GetNamespace implements client.Object.
func (in *SharedResource) GetNamespace() string {
	panic("unimplemented")
}

// GetObjectKind implements client.Object.
func (in *SharedResource) GetObjectKind() schema.ObjectKind {
	panic("unimplemented")
}

// GetOwnerReferences implements client.Object.
func (in *SharedResource) GetOwnerReferences() []metav1.OwnerReference {
	panic("unimplemented")
}

// GetResourceVersion implements client.Object.
func (in *SharedResource) GetResourceVersion() string {
	panic("unimplemented")
}

// GetSelfLink implements client.Object.
func (in *SharedResource) GetSelfLink() string {
	panic("unimplemented")
}

// GetUID implements client.Object.
func (in *SharedResource) GetUID() types.UID {
	panic("unimplemented")
}

// SetAnnotations implements client.Object.
func (in *SharedResource) SetAnnotations(annotations map[string]string) {
	panic("unimplemented")
}

// SetCreationTimestamp implements client.Object.
func (in *SharedResource) SetCreationTimestamp(timestamp metav1.Time) {
	panic("unimplemented")
}

// SetDeletionGracePeriodSeconds implements client.Object.
func (in *SharedResource) SetDeletionGracePeriodSeconds(*int64) {
	panic("unimplemented")
}

// SetDeletionTimestamp implements client.Object.
func (in *SharedResource) SetDeletionTimestamp(timestamp *metav1.Time) {
	panic("unimplemented")
}

// SetFinalizers implements client.Object.
func (in *SharedResource) SetFinalizers(finalizers []string) {
	panic("unimplemented")
}

// SetGenerateName implements client.Object.
func (in *SharedResource) SetGenerateName(name string) {
	panic("unimplemented")
}

// SetGeneration implements client.Object.
func (in *SharedResource) SetGeneration(generation int64) {
	panic("unimplemented")
}

// SetLabels implements client.Object.
func (in *SharedResource) SetLabels(labels map[string]string) {
	panic("unimplemented")
}

// SetManagedFields implements client.Object.
func (in *SharedResource) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {
	panic("unimplemented")
}

// SetName implements client.Object.
func (in *SharedResource) SetName(name string) {
	panic("unimplemented")
}

// SetNamespace implements client.Object.
func (in *SharedResource) SetNamespace(namespace string) {
	panic("unimplemented")
}

// SetOwnerReferences implements client.Object.
func (in *SharedResource) SetOwnerReferences([]metav1.OwnerReference) {
	panic("unimplemented")
}

// SetResourceVersion implements client.Object.
func (in *SharedResource) SetResourceVersion(version string) {
	panic("unimplemented")
}

// SetSelfLink implements client.Object.
func (in *SharedResource) SetSelfLink(selfLink string) {
	panic("unimplemented")
}

// SetUID implements client.Object.
func (in *SharedResource) SetUID(uid types.UID) {
	panic("unimplemented")
}

// OpenShiftBuildSpec defines the desired state of OpenShiftBuild
type OpenShiftBuildSpec struct {
	// Shipwright defines the desired state of Shipwright components
	// +kubebuilder:validation:Optional
	// +optional
	Shipwright *Shipwright `json:"shipwright,omitempty"`

	// SharedResource defines the desired state of SharedResource component
	// +kubebuilder:validation:Optional
	// +optional
	SharedResource *SharedResource `json:"sharedResource,omitempty"`
}

// OpenShiftBuildStatus defines the observed state of OpenShiftBuild
type OpenShiftBuildStatus struct {

	// Conditions holds the latest available observations of a resource's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// OpenShiftBuild is the Schema for the openshiftbuilds API
type OpenShiftBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenShiftBuildSpec   `json:"spec"`
	Status OpenShiftBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenShiftBuildList contains a list of OpenShiftBuild
type OpenShiftBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenShiftBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenShiftBuild{}, &OpenShiftBuildList{})
}

// IsReady returns true the Ready condition status is True
func (status *OpenShiftBuildStatus) IsReady() bool {
	for _, condition := range status.Conditions {
		if condition.Type == ConditionReady && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}
