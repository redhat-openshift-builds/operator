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
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// ConditionReady object is providing service.
	ConditionReady = "Ready"
)

// State defines the desired state of a component
// +kubebuilder:validation:Enum="Enabled";"Disabled"
type State string

const (
	// Enabled will install the component, including any additional custom resource definitions.
	Enabled State = "Enabled"

	// Disabled will remove the component, but may leave behind any custom resource definitions.
	Disabled State = "Disabled"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// OpenShiftBuild describes the desired state of Builds for OpenShift, and the status of
// all deployed components.
type OpenShiftBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenShiftBuildSpec   `json:"spec,omitempty"`
	Status OpenShiftBuildStatus `json:"status,omitempty"`
}

// OpenShiftBuildSpec defines the desired state of Builds for OpenShift components.
type OpenShiftBuildSpec struct {

	// Shipwright defines the desired state of Shipwright components.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Shipwright *Shipwright `json:"shipwright,omitempty"`

	// SharedResource defines the desired state of the Shared Resource CSI Driver components.
	//
	// +kubebuilder:validation:Optional
	// +optional
	SharedResource *SharedResource `json:"sharedResource,omitempty"`
}

// Shipwright defines the desired state of Shipwright components
type Shipwright struct {

	// Build defines the desired state of Shipwright Build APIs, controllers, and related components.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Build *ShipwrightBuild `json:"build,omitempty"`
}

// ShipwrightBuild defines the desired state of Shipwright Builds
type ShipwrightBuild struct {

	// State defines the desired state of the Shipwright Build controller, APIs, and related
	// components. Must be one of Enabled or Disabled.
	//
	// +kubebuilder:default="Enabled"
	State `json:"state"`
}

// SharedResource defines the desired state of Shared Resource CSI Driver and components.
type SharedResource struct {

	// State defines the desired state of SharedResource CSI Driver, APIs, and related components.
	// Must be one of Enabled or Disabled.
	//
	// +kubebuilder:default="Enabled"
	State `json:"state"`
}

// OpenShiftBuildStatus defines the observed state of OpenShiftBuild
type OpenShiftBuildStatus struct {

	// Conditions holds the latest available observations of a resource's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
