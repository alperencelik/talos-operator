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

// TalosMachineSpec defines the desired state of TalosMachine.
type TalosMachineSpec struct {

	// endpoint is the Talos API endpoint for this machine.
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// version is the desired version of Talos to run on this machine.
	// +kubebuilder:validation:Required
	Version string `json:"version"`

	// machineSpec is the machine specification for this TalosMachine.
	MachineSpec *MachineSpec `json:"machineSpec,omitempty"`

	// controlPlaneRef is a reference to the TalosControlPlane this machine belongs to.
	ControlPlaneRef *corev1.ObjectReference `json:"controlPlaneRef,omitempty"`

	// workerRef is a reference to the TalosWorker this machine belongs to.
	WorkerRef *corev1.ObjectReference `json:"workerRef,omitempty"`

	// configRef is a reference to a ConfigMap containing the Talos cluster configuration.
	// +kubebuilder:validation:Optional
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`

	// deletionPolicy specifies the deletion policy for the machine when deleting this Kubernetes resource (reset or preserve).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=reset;preserve
	// +kubebuilder:default=reset
	DeletionPolicy string `json:"deletionPolicy"`
}

type MachineSpec struct {
	// installDisk is the disk to use for installing Talos on the control plane machines.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^/dev/(sd[a-z][0-9]*|vd[a-z][0-9]*|nvme[0-9]+n[0-9]+(p[0-9]+)?)$`
	InstallDisk *string `json:"installDisk,omitempty"`
	// wipe indicates whether to wipe the disk before installation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Wipe bool `json:"wipe,omitempty"`
	// image is the Talos image to use for this machine.
	// +kubebuilder:validation:Optional
	Image *string `json:"image,omitempty"`
	// meta is the meta partition used by Talos.
	// +kubebuilder:validation:Optional
	Meta *META `json:"meta,omitempty"`
	// airGap indicates whether the machine is in an air-gapped environment.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AirGap bool `json:"airGap,omitempty"`
	// imageCache indicates whether to enable local image caching on the machine.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ImageCache bool `json:"imageCache,omitempty"`
	// allowSchedulingOnControlPlanes indicates whether to allow scheduling workloads on control plane nodes.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AllowSchedulingOnControlPlanes bool `json:"allowSchedulingOnControlPlanes,omitempty"`
	// registries is the path to a custom registries configuration file.
	// +kubebuilder:validation:Optional
	Registries *runtime.RawExtension `json:"registries,omitempty"`
	// additionalConfig is a list of additional Talos configuration documents to append to the
	// generated config, each separated by "---". Entries are appended in order: global first,
	// then machine-specific.
	// +kubebuilder:validation:Optional
	AdditionalConfig []runtime.RawExtension `json:"additionalConfig,omitempty"`
	// configPatches is a list of strategic merge patches applied to the generated Talos machine config.
	// Unlike additionalConfig (which appends a separate YAML document), each patch is merged into
	// the main machine config, allowing you to override or extend any field (e.g. machine.network).
	// +kubebuilder:validation:Optional
	ConfigPatches []runtime.RawExtension `json:"configPatches,omitempty"`
}

// TalosMachineStatus defines the observed state of TalosMachine.
type TalosMachineStatus struct {
	// observedVersion is the version of Talos running on this machine.
	ObservedVersion string `json:"observedVersion,omitempty"`
	// config is the base64 encoded Talos configuration.
	Config string `json:"config,omitempty"`
	// imported is only valid when ReconcileMode is 'import' and indicates whether the Talos machine has been imported.
	Imported *bool `json:"imported,omitempty"`
	// state is the current state of the machine (e.g., "Ready", "Provisioning", "Failed").
	State string `json:"state,omitempty"`
	// conditions represent the latest available observations of a TalosMachine's current state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tm
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.spec.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TalosMachine is the Schema for the talosmachines API.
type TalosMachine struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of TalosMachine.
	// +optional
	Spec TalosMachineSpec `json:"spec,omitempty"`

	// status defines the observed state of TalosMachine.
	// +optional
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
