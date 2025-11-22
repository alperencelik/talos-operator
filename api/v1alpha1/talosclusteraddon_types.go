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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TalosClusterAddonSpec defines the desired state of TalosClusterAddon
type TalosClusterAddonSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// ClusterSelector is a label selector that matches the TalosCluster resources
	// to which this TalosClusterAddon should be applied.
	// It allows the addon to be associated with specific clusters based on their labels.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// Helm spec contains the configuration for the Helm chart that defines the addon.
	HelmSpec HelmSpec `json:"helmSpec,omitempty"`
}

type HelmSpec struct {
	// ChartName is the name of the Helm chart in the repository.
	// e.g. chart-path oci://repo-url/chart-name as chartName: chart-name and https://repo-url/chart-name as chartName: chart-name
	ChartName string `json:"chartName"`

	// RepoURL is the URL of the Helm chart repository.
	// e.g. chart-path oci://repo-url/chart-name as repoURL: oci://repo-url and https://repo-url/chart-name as repoURL: https://repo-url
	RepoURL string `json:"repoURL"`

	// ReleaseName is the release name of the installed Helm chart. If it is not specified, a name will be generated.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// ReleaseNamespace is the namespace the Helm release will be installed on each selected
	// Cluster. If it is not specified, it will be set to the default namespace.
	// +optional
	// +kubebuilder:default:=default
	ReleaseNamespace string `json:"namespace,omitempty"`

	// Version is the version of the Helm chart. If it is not specified, the chart will use
	// and be kept up to date with the latest version.
	// +optional
	Version string `json:"version,omitempty"`

	// ValuesTemplate is an inline YAML representing the values for the Helm chart. This YAML supports Go templating to reference
	// fields from each selected workload Cluster and programatically create and set values.
	// +optional
	ValuesTemplate string `json:"valuesTemplate,omitempty"`

	// Options represents CLI flags passed to Helm operations (i.e. install, upgrade, delete)
	// include options such as wait, skipCRDs, timeout, waitForJobs, etc.
	// Inherited from Cluster API Addons API.
	// +optional
	Options *capiaddons.HelmOptions `json:"options,omitempty"`

	// Credentials is a reference to an object containing the OCI credentials. If it is not specified, no credentials will be used.
	// Inherited from Cluster API Addons API.
	// +optional
	Credentials *capiaddons.Credentials `json:"credentials,omitempty"`

	// TLSConfig contains the TLS configuration for a HelmChartProxy.
	// Inherited from Cluster API Addons API.
	// +optional
	TLSConfig *capiaddons.TLSConfig `json:"tlsConfig,omitempty"`
}

// TalosClusterAddonStatus defines the observed state of TalosClusterAddon.
type TalosClusterAddonStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the TalosClusterAddon resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TalosClusterAddon is the Schema for the talosclusteraddons API
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
