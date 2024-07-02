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

// SharedResourceSpec defines the desired state of SharedResource
type SharedResourceSpec struct {
	// TargetNamespace is the target namespace where SharedResource controller will be deployed.
	TargetNamespace string `json:"targetNamespace,omitempty"`
}

// SharedResourceStatus defines the observed state of SharedResource
type SharedResourceStatus struct {
	// Conditions holds the latest available observations of a resource's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SharedResource is the Schema for the sharedresources API
type SharedResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SharedResourceSpec   `json:"spec,omitempty"`
	Status SharedResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SharedResourceList contains a list of SharedResource
type SharedResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SharedResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SharedResource{}, &SharedResourceList{})
}

// IsReady returns true the Ready condition status is True
func (status SharedResourceStatus) IsReady() bool {
	for _, condition := range status.Conditions {
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}