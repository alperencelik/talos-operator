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
)

// TalosEtcdBackupScheduleSpec defines the desired state of TalosEtcdBackupSchedule
type TalosEtcdBackupScheduleSpec struct {
	// Schedule is a cron expression defining when to run backups
	// For example: "0 2 * * *" for daily backups at 2 AM
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Schedule string `json:"schedule"`

	// BackupTemplate is the template for creating TalosEtcdBackup resources
	// +kubebuilder:validation:Required
	BackupTemplate TalosEtcdBackupTemplateSpec `json:"backupTemplate"`

	// Retention specifies how many successful backups to keep
	// Older backups will be automatically deleted
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=5
	// +optional
	Retention *int32 `json:"retention,omitempty"`

	// Paused can be set to true to pause the backup schedule
	// +kubebuilder:default:=false
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// TalosEtcdBackupTemplateSpec defines the template for creating TalosEtcdBackup resources
type TalosEtcdBackupTemplateSpec struct {
	// Spec is the specification of the TalosEtcdBackup to be created
	// +kubebuilder:validation:Required
	Spec TalosEtcdBackupSpec `json:"spec"`
}

// TalosEtcdBackupScheduleStatus defines the observed state of TalosEtcdBackupSchedule
type TalosEtcdBackupScheduleStatus struct {
	// LastScheduleTime is the last time a backup was scheduled
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// LastSuccessfulBackupTime is the last time a backup completed successfully
	// +optional
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`

	// NextScheduleTime is the next time a backup will be scheduled
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// ActiveBackups is the list of currently active backup jobs
	// +optional
	ActiveBackups []string `json:"activeBackups,omitempty"`

	// conditions represent the current state of the TalosEtcdBackupSchedule resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Last Backup",type=date,JSONPath=`.status.lastSuccessfulBackupTime`
// +kubebuilder:printcolumn:name="Next Backup",type=date,JSONPath=`.status.nextScheduleTime`
// +kubebuilder:printcolumn:name="Paused",type=boolean,JSONPath=`.spec.paused`

// TalosEtcdBackupSchedule is the Schema for the talosetcdbackupschedules API
type TalosEtcdBackupSchedule struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of TalosEtcdBackupSchedule
	// +required
	Spec TalosEtcdBackupScheduleSpec `json:"spec"`

	// status defines the observed state of TalosEtcdBackupSchedule
	// +optional
	Status TalosEtcdBackupScheduleStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// TalosEtcdBackupScheduleList contains a list of TalosEtcdBackupSchedule
type TalosEtcdBackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TalosEtcdBackupSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TalosEtcdBackupSchedule{}, &TalosEtcdBackupScheduleList{})
}
