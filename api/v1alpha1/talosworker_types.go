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

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.mode) || self.mode == oldSelf.mode", message="Mode is immutable"
// +kubebuilder:validation:XValidation:rule="self.mode!='metal' || has(self.metalSpec)", message="MetalSpec is required when mode 'metal'"
// +kubebuilder:validation:XValidation:rule="self.mode != 'container' || self.replicas >= 1",message="replicas must be at least 1 when mode is 'container'"

// TalosWorkerSpec defines the desired state of TalosWorker.
type TalosWorkerSpec struct {

	// version of Talos to use for the worker nodes -- e.g "v1.12.1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.12.1"
	Version string `json:"version"`

	// mode specifies the deployment mode for the worker nodes (container, metal, or cloud).
	// TODO: Add support for cloud mode
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=container;metal;cloud
	Mode string `json:"mode"`

	// replicas is the number of worker machines to maintain. Only applies when mode is 'container'.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// metalSpec is required when mode is 'metal'.
	MetalSpec MetalSpec `json:"metalSpec,omitempty"`

	// kubeVersion is the version of Kubernetes to use for the worker nodes.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.35.0"
	KubeVersion string `json:"kubeVersion"`

	// storageClassName is the name of the storage class to use for persistent volumes.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="StorageClassName is immutable, you cannot change it after creation"
	StorageClassName *string `json:"storageClassName,omitempty"`

	// controlPlaneRef is a reference to the TalosControlPlane that this worker belongs to.
	// +kubebuilder:validation:Optional
	ControlPlaneRef corev1.LocalObjectReference `json:"controlPlaneRef"`

	// configRef is a reference to a ConfigMap containing the Talos cluster configuration.
	// +kubebuilder:validation:Optional
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`

	// deletionPolicy specifies the deletion policy for worker machines when deleting this Kubernetes resource (reset or preserve).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=reset;preserve
	// +kubebuilder:default=reset
	DeletionPolicy string `json:"deletionPolicy"`
}

// TalosWorkerStatus defines the observed state of TalosWorker.
type TalosWorkerStatus struct {
	// conditions represent the current state of the TalosWorker resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// config is the serialized Talos configuration for the worker.
	Config string `json:"config,omitempty"`
	// imported is only valid when ReconcileMode is 'import' and indicates whether the Talos worker has been imported.
	Imported *bool `json:"imported,omitempty"`
	// state represents the current state of the Talos worker (e.g., "Ready", "Provisioning", "Failed").
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tw
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TalosWorker is the Schema for the talosworkers API.
type TalosWorker struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of TalosWorker.
	// +optional
	Spec TalosWorkerSpec `json:"spec,omitempty"`

	// status defines the observed state of TalosWorker.
	// +optional
	Status TalosWorkerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TalosWorkerList contains a list of TalosWorker.
type TalosWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosWorker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosWorker{}, &TalosWorkerList{})
}
