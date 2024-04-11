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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// State defines the desired state of the component
// +kubebuilder:validation:Enum="Enabled";"Disabled"
type State string

const (
	// Enabled installs the component
	Enabled State = "Enabled"

	// Disabled removes the component
	Disabled State = "Disabled"
)

// ComponentState defines the desired state of the component
type ComponentState struct {
	// State defines the desired state of a component
	// +kubebuilder:default="Disabled"
	State State `json:"state"`
}

// ShipwrightBuildSpec defines the desired state of Shipwright Builds
type ShipwrightBuildSpec struct {
	// ComponentState defines the desired state of the Shipwright Build component
	ComponentState `json:",inline"`
}

// ShipwrightSpec defines the desired state of Shipwright components
type ShipwrightSpec struct {
	// Build defines the desired state of Shipwright Build component
	Build ShipwrightBuildSpec `json:"build,omitempty"`
}

// SharedResourceSpec defines the desired state of Shared Resources CSI Driver
type SharedResourceSpec struct {
	// ComponentState defines the desired state of SharedResource component
	ComponentState `json:",inline"`
}

// OpenShiftBuildSpec defines the desired state of OpenShiftBuild
type OpenShiftBuildSpec struct {
	// Shipwright defines the desired state of Shipwright components
	Shipwright ShipwrightSpec `json:"shipwright,omitempty"`

	// SharedResource defines the desired state of SharedResource component
	SharedResource SharedResourceSpec `json:"sharedResource,omitempty"`
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

	Spec   OpenShiftBuildSpec   `json:"spec,omitempty"`
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
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}
