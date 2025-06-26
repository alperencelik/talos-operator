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

// CEL validation rules for TalosClusterSpec fields

// +kubebuilder:validation:XValidation:rule="(has(self.controlPlane) && !has(self.controlPlaneRef)) || (!has(self.controlPlaneRef) && has(self.controlPlane)) || (!has(self.controlPlane) && !has(self.controlPlaneRef))",message="Specify either controlPlane or controlPlaneRef, but not both"
// +kubebuilder:validation:XValidation:rule="(has(self.worker) && !has(self.workerRef)) || (!has(self.workerRef) && has(self.worker)) || (!has(self.worker) && !has(self.workerRef))",message="Specify either worker or workerRef, but not both"

// TalosClusterSpec defines the desired state of TalosCluster.
type TalosClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ControlPlane defines the control plane configuration for the Talos cluster.
	// +kubebuilder:validation:Optional
	ControlPlane *TalosControlPlaneSpec `json:"controlPlane,omitempty"`

	// ControlPlaneRef references the TalosControlPlane resource that manages the control plane.
	// +kubebuilder:validation:Optional
	ControlPlaneRef *corev1.LocalObjectReference `json:"controlPlaneRef,omitempty"`

	// Worker defines the worker configuration for the Talos cluster.
	// +kubebuilder:validation:Optional
	Worker *TalosWorkerSpec `json:"worker,omitempty"`

	// WorkerRef references the TalosWorker resource that manages the worker nodes.
	// +kubebuilder:validation:Optional
	WorkerRef *corev1.LocalObjectReference `json:"workerRef,omitempty"`
}

// TalosClusterStatus defines the observed state of TalosCluster.
type TalosClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosCluster is the Schema for the talosclusters API.
type TalosCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TalosClusterSpec   `json:"spec,omitempty"`
	Status TalosClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TalosClusterList contains a list of TalosCluster.
type TalosClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosCluster{}, &TalosClusterList{})
}
