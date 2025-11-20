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

// TalosAddonSpec defines the desired state of TalosAddon.
type TalosAddonSpec struct {
	// ClusterRef references the TalosCluster this addon should be installed on
	// +kubebuilder:validation:Required
	ClusterRef corev1.LocalObjectReference `json:"clusterRef"`

	// HelmRelease defines the Helm chart to be installed
	// +kubebuilder:validation:Required
	HelmRelease HelmReleaseSpec `json:"helmRelease"`
}

// HelmReleaseSpec defines the Helm chart installation details.
type HelmReleaseSpec struct {
	// ChartName is the name of the Helm chart
	// +kubebuilder:validation:Required
	ChartName string `json:"chartName"`

	// RepoURL is the URL of the Helm repository
	// +kubebuilder:validation:Required
	RepoURL string `json:"repoURL"`

	// Version is the version of the Helm chart
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`

	// ReleaseName is the name of the Helm release
	// +kubebuilder:validation:Optional
	ReleaseName string `json:"releaseName,omitempty"`

	// TargetNamespace is the namespace where the Helm chart will be installed
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="default"
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Values is a map of values to pass to the Helm chart
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values map[string]string `json:"values,omitempty"`

	// ValuesFrom references a ConfigMap or Secret containing values
	// +kubebuilder:validation:Optional
	ValuesFrom []ValueReference `json:"valuesFrom,omitempty"`
}

// ValueReference contains a reference to a resource containing values.
type ValueReference struct {
	// Kind of the values reference (ConfigMap or Secret)
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name of the ConfigMap or Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key in the ConfigMap or Secret
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
}

// TalosAddonStatus defines the observed state of TalosAddon.
type TalosAddonStatus struct {
	// State represents the current state of the addon
	State string `json:"state,omitempty"`

	// Conditions is a list of conditions for the addon
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAppliedRevision is the revision of the last successfully applied Helm chart
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`

	// ObservedGeneration is the most recent generation observed for this addon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ta
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
// +kubebuilder:printcolumn:name="Chart",type=string,JSONPath=`.spec.helmRelease.chartName`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TalosAddon is the Schema for the talosaddons API.
type TalosAddon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TalosAddonSpec   `json:"spec,omitempty"`
	Status TalosAddonStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TalosAddonList contains a list of TalosAddon.
type TalosAddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosAddon `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosAddon{}, &TalosAddonList{})
}
