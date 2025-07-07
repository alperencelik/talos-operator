/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TalosMachineSpec defines the desired state of TalosMachine.
type TalosMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Endpoint is the Talos API endpoint for this machine.
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint,omitempty"`
	// InstallDisk is the disk where Talos will be installed.
	// +kubebuilder:validation:Optional
	InstallDisk *string `json:"installDisk,omitempty"`

	// ControlPlaneRef is a reference to the TalosControlPlane this machine belongs to.
	ControlPlaneRef *corev1.ObjectReference `json:"controlPlaneRef,omitempty"`

	// WorkerRef is a reference to the TalosWorker this machine belongs to.
	WorkerRef *corev1.ObjectReference `json:"workerRef,omitempty"`
}

// TalosMachineStatus defines the observed state of TalosMachine.
type TalosMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Config string `json:"config,omitempty"` // Base64 encoded Talos configuration

	State string `json:"state,omitempty"` // e.g., "Ready", "Provisioning", "Failed"
	// Conditions represent the latest available observations of a TalosMachine's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosMachine is the Schema for the talosmachines API.
type TalosMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TalosMachineSpec   `json:"spec,omitempty"`
	Status TalosMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TalosMachineList contains a list of TalosMachine.
type TalosMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosMachine{}, &TalosMachineList{})
}
