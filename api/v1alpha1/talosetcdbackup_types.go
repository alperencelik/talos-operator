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

// TalosEtcdBackupSpec defines the desired state of TalosEtcdBackup.
type TalosEtcdBackupSpec struct {
	// talosControlPlaneRef is a reference to the TalosControlPlane this backup is associated with.
	// +kubebuilder:validation:required
	TalosControlPlaneRef *corev1.LocalObjectReference `json:"talosControlPlaneRef"`

	// backupStorage specifies where to store the etcd backup.
	// +kubebuilder:validation:required
	BackupStorage BackupStorage `json:"backupStorage"`
}

type BackupStorage struct {
	// s3 specifies the S3-compatible storage configuration for the etcd backup.
	// +kubebuilder:validation:Required
	S3 *S3Storage `json:"s3,omitempty"`
}

type S3Storage struct {
	// bucket is the name of the S3 bucket to store the etcd backup.
	Bucket string `json:"bucket"`
	// region is the AWS region where the S3 bucket is located.
	Region string `json:"region"`
	// endpoint is the S3 service endpoint (optional, for custom S3-compatible services).
	Endpoint string `json:"endpoint,omitempty"`
	// accessKeyID is the access key ID for the S3 bucket.
	AccessKeyID *corev1.SecretKeySelector `json:"accessKeyID"`
	// secretAccessKey is the secret access key for the S3 bucket.
	SecretAccessKey *corev1.SecretKeySelector `json:"secretAccessKey"`
	// insecureSkipTLSVerify skips TLS verification for the S3 endpoint (optional).
	// +kubebuilder:default:=false
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// TalosEtcdBackupStatus defines the observed state of TalosEtcdBackup.
type TalosEtcdBackupStatus struct {
	// filename is the name of the backup file in the storage backend.
	// +optional
	Filename string `json:"filename,omitempty"`
	// conditions represent the current state of the TalosEtcdBackup resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=teb
// +kubebuilder:printcolumn:name="Filename",type=string,JSONPath=`.status.filename`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TalosEtcdBackup is the Schema for the talosetcdbackups API.
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
