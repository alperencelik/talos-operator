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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiaddons "sigs.k8s.io/cluster-api-addon-provider-helm/api/v1alpha1"
)

// TalosClusterAddonSpec defines the desired state of TalosClusterAddon.
type TalosClusterAddonSpec struct {
	// clusterSelector is a label selector that matches the TalosCluster resources
	// to which this TalosClusterAddon should be applied.
	// It allows the addon to be associated with specific clusters based on their labels.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// helmSpec contains the configuration for the Helm chart that defines the addon.
	HelmSpec HelmSpec `json:"helmSpec,omitempty"`
}

type HelmSpec struct {
	// chartName is the name of the Helm chart in the repository.
	// e.g. chart-path oci://repo-url/chart-name as chartName: chart-name and https://repo-url/chart-name as chartName: chart-name
	ChartName string `json:"chartName"`

	// repoURL is the URL of the Helm chart repository.
	// e.g. chart-path oci://repo-url/chart-name as repoURL: oci://repo-url and https://repo-url/chart-name as repoURL: https://repo-url
	RepoURL string `json:"repoURL"`

	// releaseName is the release name of the installed Helm chart. If it is not specified, a name will be generated.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// namespace is the namespace the Helm release will be installed on each selected
	// Cluster. If it is not specified, it will be set to the default namespace.
	// +optional
	// +kubebuilder:default:=default
	ReleaseNamespace string `json:"namespace,omitempty"`

	// version is the version of the Helm chart. If it is not specified, the chart will use
	// and be kept up to date with the latest version.
	// +optional
	Version string `json:"version,omitempty"`

	// valuesTemplate is an inline YAML representing the values for the Helm chart. This YAML supports Go templating to reference
	// fields from each selected workload Cluster and programatically create and set values.
	// +optional
	ValuesTemplate string `json:"valuesTemplate,omitempty"`

	// options represents CLI flags passed to Helm operations (i.e. install, upgrade, delete)
	// include options such as wait, skipCRDs, timeout, waitForJobs, etc.
	// Inherited from Cluster API Addons API.
	// +optional
	Options *capiaddons.HelmOptions `json:"options,omitempty"`

	// credentials is a reference to an object containing the OCI credentials. If it is not specified, no credentials will be used.
	// Inherited from Cluster API Addons API.
	// +optional
	Credentials *capiaddons.Credentials `json:"credentials,omitempty"`

	// tlsConfig contains the TLS configuration for a HelmChartProxy.
	// Inherited from Cluster API Addons API.
	// +optional
	TLSConfig *capiaddons.TLSConfig `json:"tlsConfig,omitempty"`
}

// TalosClusterAddonStatus defines the observed state of TalosClusterAddon.
type TalosClusterAddonStatus struct {
	// conditions represent the latest available observations of the TalosClusterAddon's current state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tca
// +kubebuilder:printcolumn:name="Chart",type=string,JSONPath=`.spec.helmSpec.chartName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.helmSpec.version`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TalosClusterAddon is the Schema for the talosclusteraddons API.
type TalosClusterAddon struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of TalosClusterAddon
	// +required
	Spec TalosClusterAddonSpec `json:"spec"`

	// status defines the observed state of TalosClusterAddon
	// +optional
	Status TalosClusterAddonStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// TalosClusterAddonList contains a list of TalosClusterAddon
type TalosClusterAddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosClusterAddon `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosClusterAddon{}, &TalosClusterAddonList{})
}
