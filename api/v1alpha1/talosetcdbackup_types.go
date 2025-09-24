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

// TalosEtcdBackupSpec defines the desired state of TalosEtcdBackup
type TalosEtcdBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// TalosControlPlaneRef is a reference to the TalosControlPlane this backup is associated with
	// +kubebuilder:validation:required
	TalosControlPlaneRef *corev1.LocalObjectReference `json:"talosControlPlaneRef"`

	// BackupStorage specifies where to store the etcd backup
	// +kubebuilder:validation:required
	BackupStorage BackupStorage `json:"backupStorage"`
}

type BackupStorage struct {
	// S3 specifies the S3-compatible storage configuration for the etcd backup
	// +kubebuilder:validation:Required
	S3 *S3Storage `json:"s3,omitempty"`
}

type S3Storage struct {
	// Bucket is the name of the S3 bucket to store the etcd backup
	Bucket string `json:"bucket"`
	// Region is the AWS region where the S3 bucket is located
	Region string `json:"region"`
	// Endpoint is the S3 service endpoint (optional, for custom S3-compatible services)
	Endpoint string `json:"endpoint,omitempty"`
	// AccessKeyID is the access key ID for the S3 bucket
	AccessKeyID *corev1.SecretKeySelector `json:"accessKeyID"`
	// SecretAccessKey is the secret access key for the S3 bucket
	SecretAccessKey *corev1.SecretKeySelector `json:"secretAccessKey"`
	// InsecureSkipTLSVerify skips TLS verification for the S3 endpoint (optional)
	// +kubebuilder:default:=false
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// TalosEtcdBackupStatus defines the observed state of TalosEtcdBackup.
type TalosEtcdBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the TalosEtcdBackup resource.
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

// TalosEtcdBackup is the Schema for the talosetcdbackups API
type TalosEtcdBackup struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of TalosEtcdBackup
	// +required
	Spec TalosEtcdBackupSpec `json:"spec"`

	// status defines the observed state of TalosEtcdBackup
	// +optional
	Status TalosEtcdBackupStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// TalosEtcdBackupList contains a list of TalosEtcdBackup
type TalosEtcdBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosEtcdBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosEtcdBackup{}, &TalosEtcdBackupList{})
}
