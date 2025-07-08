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

// +kubebuilder:validadtion:XValidation:rule="!has(oldSelf.mode) || has(self.mode)", message="Mode is immutable"
// +kubebuilder:validation:XValidation:rule="self.mode!='metal' || has(self.metalSpec)", message="MetalSpec is required when mode 'metal'"

// TalosWorkerSpec defines the desired state of TalosWorker.
type TalosWorkerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of Talos to use for the control plane(controller-manager, scheduler, kube-apiserver, etcd) -- e.g "v1.33.1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.33.1"
	Version string `json:"version,omitempty"`

	// TODO: Add support for other modes like metal, cloud, etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=container;metal;cloud
	Mode string `json:"mode,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// Number of control-plane machines to maintain
	Replicas int32 `json:"replicas"`

	// Metal Spec is required when mode is 'metal'
	MetalSpec MetalSpec `json:"metalSpec,omitempty"`

	// KubeVersion is the version of Kubernetes to use for the control plane
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.33.1"
	KubeVersion string `json:"kubeVersion,omitempty"`

	// StorageClassName is the name of the storage class to use for persistent volumes
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="StorageClassName is immutable, you cannot change it after creation"
	StorageClassName *string `json:"storageClassName,omitempty"`

	// ControlPlaneRef is a reference to the TalosControlPlane that this worker belongs to
	// TODO:
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=objectreference
	ControlPlaneRef corev1.LocalObjectReference `json:"controlPlaneRef"`

	// +kubebuilder:validation:Optional
	// Reference to a ConfigMap containing the Talos cluster configuration
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`
}

// TalosWorkerStatus defines the observed state of TalosWorker.
type TalosWorkerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Config     string             `json:"config,omitempty"` // Serialized Talos configuration for the worker
	// State represents the current state of the Talos worker
	State string `json:"state,omitempty"` // e.g., "Ready", "Provisioning", "Failed"
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosWorker is the Schema for the talosworkers API.
type TalosWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TalosWorkerSpec   `json:"spec,omitempty"`
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
