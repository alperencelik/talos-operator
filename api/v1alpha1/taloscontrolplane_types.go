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

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.clusterDomain) || has(self.clusterDomain)", message="ClusterDomain is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.mode) || has(self.mode)", message="Mode is immutable"
// +kubebuilder:validation:XValidation:rule="self.mode != 'metal' || size(self.metalSpec.machines) > 0",message="Machines is required when mode is 'metal'"
// +kubebuilder:validation:XValidation:rule="self.mode != 'container' || self.replicas >= 1",message="replicas must be at least 1 when mode is 'container'"

// TalosControlPlaneSpec defines the desired state of TalosControlPlane.
type TalosControlPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// version of Talos to use for the control plane(controller-manager, scheduler, kube-apiserver, etcd) -- e.g "v1.12.1"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.12.1"
	Version string `json:"version,omitempty"`

	// mode specifies the deployment mode for the control plane (container, metal, or cloud).
	// TODO: Add support for cloud mode
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=container;metal;cloud
	Mode string `json:"mode,omitempty"`

	// replicas is the number of control-plane machines to maintain. Only applies when mode is 'container'.
	// +kubebuilder:validation:Optional
	Replicas int32 `json:"replicas,omitempty"`

	// metalSpec is required when mode is 'metal'.
	MetalSpec MetalSpec `json:"metalSpec,omitempty"`

	// endpoint is the endpoint for the Kubernetes API Server.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9.-]+(:\d+)?$`
	Endpoint string `json:"endpoint,omitempty"`

	// Can't force the decrease CEL validation as it would prevent downgrades in some scenarios, eventhough the operator doesn't support downgrades
	// Example: User created cluster with v1.33.0, then tried to upgrade v1.35.0 but the job failed since talos doesn't allow that so now user
	// need to re-attempt upgrade to v1.34.0 but CEL validation would prevent that.

	// kubeVersion is the version of Kubernetes to use for the control plane.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+(-\w+)?$`
	// +kubebuilder:default="v1.35.0"
	KubeVersion string `json:"kubeVersion,omitempty"`

	// clusterDomain is the domain for the Kubernetes cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^([a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?\.)+[a-z]{2,}$`
	// +kubebuilder:default="cluster.local"
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// storageClassName is the name of the storage class to use for persistent volumes.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$`
	StorageClassName *string `json:"storageClassName,omitempty"`

	// podCIDR is the list of CIDR ranges for pod IPs in the cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Items=pattern=`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`
	PodCIDR []string `json:"podCIDR,omitempty"`

	// serviceCIDR is the list of CIDR ranges for service IPs in the cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Items=pattern=`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`
	ServiceCIDR []string `json:"serviceCIDR,omitempty"`

	// configRef is a reference to a ConfigMap containing the Talos controlplane configuration.
	// +kubebuilder:validation:Optional
	ConfigRef *corev1.ConfigMapKeySelector `json:"configRef,omitempty"`

	// cni is the CNI configuration for the cluster.
	// +kubebuilder:validation:Optional
	CNI *CNIConfig `json:"cni,omitempty"`

	// resetOnDelete indicates whether deleting this Kubernetes resource also resets the machines
	// +kubebuilder:validation:Required
	ResetOnDelete bool `json:"resetOnDelete"`
}

// CNIConfig represents the CNI configuration options.
type CNIConfig struct {
	// name of CNI to use (flannel, custom, none).
	// +kubebuilder:validation:Enum=flannel;custom;none
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// urls containing manifests to apply for the CNI.
	// Should be present for "custom", must be empty for "flannel" and "none".
	// +kubebuilder:validation:Optional
	URLs []string `json:"urls,omitempty"`

	// flannel configuration options.
	// +kubebuilder:validation:Optional
	Flannel *FlannelCNIConfig `json:"flannel,omitempty"`
}

// FlannelCNIConfig represents the Flannel CNI configuration options.
type FlannelCNIConfig struct {
	// extraArgs are extra arguments for 'flanneld'.
	// +kubebuilder:validation:Optional
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type MetalSpec struct {
	// machines is a list of machine specifications for the Talos control plane.
	// +listType=atomic
	Machines []Machine `json:"machines,omitempty"`
	// machineSpec defines the specifications for each Talos control plane machine.
	// +kubebuilder:validation:Optional
	MachineSpec *MachineSpec `json:"machineSpec,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.address) != has(self.machineRef)",message="address and machineRef are mutually exclusive"
type Machine struct {
	// address is the IP address of the Talos machine.
	// +kubebuilder:validation:Pattern=`^(\d{1,3}\.){3}\d{1,3}$`
	Address *string `json:"address,omitempty"`
	// machineRef is a reference to a Kubernetes object from which the machine IP address can be extracted.
	// +kubebuilder:validation:Optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`
	// configPatches is a list of machine-specific config patches applied per machine.
	// Only define the machine level configPatches here because it appended after the root machineSpec.configPatches.
	// +kubebuilder:validation:Optional
	ConfigPatches []runtime.RawExtension `json:"configPatches,omitempty"`
	// additionalConfig is machine-specific additional Talos configuration to append to the generated config.
	// When set, it appends the root machineSpec.additionalConfig for this machine.
	// +kubebuilder:validation:Optional
	AdditionalConfig []runtime.RawExtension `json:"additionalConfig,omitempty"`
}

// META is network metadata for Talos machines
type META struct {
	// hostname is the hostname for the Talos machines.
	Hostname string `json:"hostname,omitempty"`
	// interface is the network interface name for Talos machines.
	Interface string `json:"interface,omitempty"`
	// subnet is the subnet prefix length for the Talos machines.
	Subnet int `json:"subnet,omitempty"`
	// gateway is the gateway for the Talos machines.
	Gateway string `json:"gateway,omitempty"`
	// dnsServers is a list of DNS servers for the Talos machines.
	DNSServers []string `json:"dnsServers,omitempty"`
}

// TalosControlPlaneStatus defines the observed state of TalosControlPlane.
type TalosControlPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// state is the current state of the control plane.
	State string `json:"state,omitempty"`
	// conditions is a list of conditions for the Talos control plane.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// config is the reference to the Talos configuration used for the control plane.
	Config string `json:"config,omitempty"`
	// secretBundle is the reference to the secrets bundle used for the control plane.
	SecretBundle string `json:"secretBundle,omitempty"`
	// bundleConfig is the reference to the bundle configuration used for the control plane.
	BundleConfig string `json:"bundleConfig,omitempty"`
	// imported is only valid when ReconcileMode is 'import' and indicates whether the Talos control plane has been imported.
	Imported *bool `json:"imported,omitempty"`
	// observedKubeVersion is the observed version of Kubernetes.
	// +kubebuilder:validation:Optional
	ObservedKubeVersion string `json:"observedKubeVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosControlPlane is the Schema for the taloscontrolplanes API.
type TalosControlPlane struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of TalosControlPlane.
	// +optional
	Spec TalosControlPlaneSpec `json:"spec,omitempty"`

	// status defines the observed state of TalosControlPlane.
	// +optional
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
