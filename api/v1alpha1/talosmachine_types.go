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
	"k8s.io/apimachinery/pkg/runtime"
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

	// Version is the desired version of Talos to run on this machine.
	// +kubebuilder:validation:Required
	Version string `json:"version,omitempty"`

	// MachineSpec is the machine specification for this TalosMachine.
	MachineSpec *MachineSpec `json:"machineSpec,omitempty"`

	// ControlPlaneRef is a reference to the TalosControlPlane this machine belongs to.
	ControlPlaneRef *corev1.ObjectReference `json:"controlPlaneRef,omitempty"`

	// WorkerRef is a reference to the TalosWorker this machine belongs to.
	WorkerRef *corev1.ObjectReference `json:"workerRef,omitempty"`

	// +kubebuilder:validation:Optional
	// Reference to a ConfigMap containing the Talos cluster configuration
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`
}

type MachineSpec struct {
	// InstallDisk is the disk to use for installing Talos on the control plane machines.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^/dev/(sd[a-z][0-9]*|vd[a-z][0-9]*|nvme[0-9]+n[0-9]+(p[0-9]+)?)$`
	InstallDisk *string `json:"installDisk,omitempty"`
	// Wipe indicates whether to wipe the disk before installation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Wipe bool `json:"wipe,omitempty"`
	// Image is the Talos image to use for this machine.
	// +kubebuilder:validation:Optional
	Image *string `json:"image,omitempty"`
	// Meta is the meta partition that used by Talos.
	// +kubebuilder:validation:Optional
	Meta *META `json:"meta,omitempty"`
	// AirGap indicates whether the machine is in an air-gapped environment.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AirGap bool `json:"airGap,omitempty"`
	// ImageCache indicates whether to enable local image caching on the machine.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ImageCache bool `json:"imageCache,omitempty"`
	// AllowSchedulingOnControlPlanes indicates whether to allow scheduling workloads on control plane nodes.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AllowSchedulingOnControlPlanes bool `json:"allowSchedulingOnControlPlanes,omitempty"`
	// Registries is the path to a custom registries configuration file.
	// +kubebuilder:validation:Optional
	Registries *runtime.RawExtension `json:"registries,omitempty"`
	// AdditionalConfig is additional Talos configuration to append to the generated config.
	// +kubebuilder:validation:Optional
	AdditionalConfig *runtime.RawExtension `json:"additionalConfig,omitempty"`
}

// TalosMachineStatus defines the observed state of TalosMachine.
type TalosMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	ObservedVersion string `json:"observedVersion,omitempty"` // The version of Talos running on this machine
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
