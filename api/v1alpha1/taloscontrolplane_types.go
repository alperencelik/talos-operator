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

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.clusterDomain) || has(self.clusterDomain)", message="ClusterDomain is immutable"
// +kubebuilder:validadtion:XValidation:rule="!has(oldSelf.mode) || has(self.mode)", message="Mode is immutable"
// +kubebuilder:validation:XValidation:rule="self.mode!='metal' || has(self.metalSpec)", message="MetalSpec is required when mode 'metal'"

// TalosControlPlaneSpec defines the desired state of TalosControlPlane.
type TalosControlPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of Talos to use for the control plane(controller-manager, scheduler, kube-apiserver, etcd) -- e.g "v1.33.1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.10.3"
	Version string `json:"version,omitempty"`

	// TODO: Add support for other modes like metal, cloud, etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=container;metal;cloud
	Mode string `json:"mode,omitempty"`

	// Number of control-plane machines to maintain
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// Metal Spec is required when mode is 'metal'
	MetalSpec MetalSpec `json:"metalSpec,omitempty"`

	// Endpoint for the Kubernetes API Server
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9.-]+(:\d+)?$`
	Endpoint string `json:"endpoint,omitempty"`

	// KubeVersion is the version of Kubernetes to use for the control plane
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.33.1"
	KubeVersion string `json:"kubeVersion,omitempty"`

	// ClusterDomain is the domain for the Kubernetes cluster
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^([a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?\.)+[a-z]{2,}$`
	// +kubebuilder:default="cluster.local"
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// StorageClassName is the name of the storage class to use for persistent volumes
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$`
	StorageClassName *string `json:"storageClassName,omitempty"`

	// PodCIDRs is the list of CIDR ranges for pod IPs in the cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Items=pattern=`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`
	PodCIDR []string `json:"podCIDR,omitempty"`

	// ServiceCIDRs is the list of CIDR ranges for service IPs in the cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Items=pattern=`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`
	ServiceCIDR []string `json:"serviceCIDR,omitempty"`

	// +kubebuilder:validation:Optional
	// Reference to a ConfigMap containing the Talos cluster configuration
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`
}

type MetalSpec struct {
	// Machines is a list of machine specifications for the Talos control plane.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Machines []string `json:"machines,omitempty"`
	// MachineSpec defines the specifications for each Talos control plane machine.
	// +kubebuilder:validation:Optional
	MachineSpec *MachineSpec `json:"machineSpec,omitempty"`
}

type MachineSpec struct {
	// InstallDisk is the disk to use for installing Talos on the control plane machines.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^/dev/[a-z]+[0-9]*$`
	InstallDisk *string `json:"installDisk,omitempty"`
	// Wipe indicates whether to wipe the disk before installation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Wipe bool `json:"wipe,omitempty"`
	// Image is the Talos image to use for this machine.
	// +kubebuilder:validation:Optional
	Image *string `json:"image,omitempty"`
}

// TalosControlPlaneStatus defines the observed state of TalosControlPlane.
type TalosControlPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions is a list of conditions for the Talos control plane
	State        string             `json:"state,omitempty"` // Current state of the control plane
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
	Config       string             `json:"config,omitempty"`       // Reference to the Talos configuration used for the control plane
	SecretBundle string             `json:"secretBundle,omitempty"` // Reference to the secrets bundle used for the control plane
	BundleConfig string             `json:"bundleConfig,omitempty"` // Reference to the bundle configuration used for the control plane
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosControlPlane is the Schema for the taloscontrolplanes API.
type TalosControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TalosControlPlaneSpec   `json:"spec,omitempty"`
	Status TalosControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TalosControlPlaneList contains a list of TalosControlPlane.
type TalosControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosControlPlane{}, &TalosControlPlaneList{})
}
