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

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

// TalosEtcdBackupScheduleReconciler reconciles a TalosEtcdBackupSchedule object
type TalosEtcdBackupScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackupschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackupschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackupschedules/finalizers,verbs=update
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackups,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TalosEtcdBackupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var schedule talosv1alpha1.TalosEtcdBackupSchedule
	if err := r.Get(ctx, req.NamespacedName, &schedule); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Finalizer logic
	if schedule.DeletionTimestamp.IsZero() {
		// Add finalizer for this CR
		if err := r.handleFinalizer(ctx, &schedule); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&schedule, talosv1alpha1.TalosEtcdBackupScheduleFinalizer) {
			if err := r.handleDelete(ctx, &schedule); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to handle delete: %w", err)
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(&schedule, talosv1alpha1.TalosEtcdBackupScheduleFinalizer)
			if err := r.Update(ctx, &schedule); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling TalosEtcdBackupSchedule", "schedule", req.NamespacedName)

	// If paused, skip scheduling
	if schedule.Spec.Paused {
		logger.Info("Backup schedule is paused, skipping")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Validate cron schedule
	cronSchedule, err := cron.ParseStandard(schedule.Spec.Schedule)
	if err != nil {
		logger.Error(err, "Invalid cron schedule", "schedule", schedule.Spec.Schedule)
		meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "InvalidSchedule",
			Message: fmt.Sprintf("Invalid cron schedule: %v", err),
		})
		if statusErr := r.Status().Update(ctx, &schedule); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Get the next scheduled time
	now := time.Now()
	nextSchedule := cronSchedule.Next(now)

	// Determine if we should create a backup
	shouldCreateBackup := false
	lastScheduleTime := schedule.Status.LastScheduleTime

	if lastScheduleTime == nil {
		// First time running, create a backup
		shouldCreateBackup = true
	} else {
		// Calculate when the next backup should have been created after the last one
		lastScheduledNext := cronSchedule.Next(lastScheduleTime.Time)
		// If we're past the next scheduled time, create a backup
		if now.After(lastScheduledNext) || now.Equal(lastScheduledNext) {
			shouldCreateBackup = true
		}
	}

	// Check if we need to create a new backup
	if shouldCreateBackup {
		// Time to create a new backup
		if err := r.createBackup(ctx, &schedule); err != nil {
			logger.Error(err, "Failed to create backup")
			meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
				Type:    talosv1alpha1.ConditionFailed,
				Status:  metav1.ConditionTrue,
				Reason:  "BackupCreationFailed",
				Message: fmt.Sprintf("Failed to create backup: %v", err),
			})
			if statusErr := r.Status().Update(ctx, &schedule); statusErr != nil {
				logger.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{}, err
		}

		// Update last schedule time
		schedule.Status.LastScheduleTime = &metav1.Time{Time: now}
	}

	// Clean up old backups based on retention policy
	if err := r.cleanupOldBackups(ctx, &schedule); err != nil {
		logger.Error(err, "Failed to cleanup old backups")
		// Don't fail the reconciliation for cleanup errors
	}

	// Update status with next schedule time
	schedule.Status.NextScheduleTime = &metav1.Time{Time: nextSchedule}

	// Set ready condition
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "ScheduleActive",
		Message: "Backup schedule is active",
	})

	if err := r.Status().Update(ctx, &schedule); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Requeue to check again at the next scheduled time
	requeueAfter := time.Until(nextSchedule)
	if requeueAfter < 0 {
		requeueAfter = time.Minute
	}

	logger.Info("Next backup scheduled", "time", nextSchedule, "requeueAfter", requeueAfter)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *TalosEtcdBackupScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosEtcdBackupSchedule{}).
		Owns(&talosv1alpha1.TalosEtcdBackup{}).
		Named("talosetcdbackupschedule").
		Complete(r)
}

func (r *TalosEtcdBackupScheduleReconciler) handleFinalizer(ctx context.Context, schedule *talosv1alpha1.TalosEtcdBackupSchedule) error {
	if !controllerutil.ContainsFinalizer(schedule, talosv1alpha1.TalosEtcdBackupScheduleFinalizer) {
		controllerutil.AddFinalizer(schedule, talosv1alpha1.TalosEtcdBackupScheduleFinalizer)
		if err := r.Update(ctx, schedule); err != nil {
			return fmt.Errorf("failed to update TalosEtcdBackupSchedule with finalizer: %w", err)
		}
	}
	return nil
}

func (r *TalosEtcdBackupScheduleReconciler) handleDelete(ctx context.Context, schedule *talosv1alpha1.TalosEtcdBackupSchedule) error {
	logger := logf.FromContext(ctx)
	logger.Info("Deleting TalosEtcdBackupSchedule and its managed backups", "schedule", schedule.Name)

	// List all backups owned by this schedule
	var backupList talosv1alpha1.TalosEtcdBackupList
	if err := r.List(ctx, &backupList, client.InNamespace(schedule.Namespace), client.MatchingLabels{
		"talos.alperen.cloud/backup-schedule": schedule.Name,
	}); err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Delete all owned backups
	for _, backup := range backupList.Items {
		if err := r.Delete(ctx, &backup); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete backup %s: %w", backup.Name, err)
		}
	}

	return nil
}

// createBackup creates a new TalosEtcdBackup from the schedule template
func (r *TalosEtcdBackupScheduleReconciler) createBackup(ctx context.Context, schedule *talosv1alpha1.TalosEtcdBackupSchedule) error {
	logger := logf.FromContext(ctx)

	// Generate backup name with timestamp
	backupName := fmt.Sprintf("%s-%d", schedule.Name, time.Now().Unix())

	backup := &talosv1alpha1.TalosEtcdBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: schedule.Namespace,
			Labels: map[string]string{
				"talos.alperen.cloud/backup-schedule": schedule.Name,
			},
		},
		Spec: schedule.Spec.BackupTemplate.Spec,
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(schedule, backup, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	logger.Info("Creating backup", "backup", backupName)
	if err := r.Create(ctx, backup); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// cleanupOldBackups removes backups that exceed the retention policy
func (r *TalosEtcdBackupScheduleReconciler) cleanupOldBackups(ctx context.Context, schedule *talosv1alpha1.TalosEtcdBackupSchedule) error {
	logger := logf.FromContext(ctx)

	retention := int32(5) // default
	if schedule.Spec.Retention != nil {
		retention = *schedule.Spec.Retention
	}

	// List all backups owned by this schedule
	var backupList talosv1alpha1.TalosEtcdBackupList
	if err := r.List(ctx, &backupList, client.InNamespace(schedule.Namespace), client.MatchingLabels{
		"talos.alperen.cloud/backup-schedule": schedule.Name,
	}); err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Filter successful backups
	var successfulBackups []talosv1alpha1.TalosEtcdBackup
	for _, backup := range backupList.Items {
		if meta.IsStatusConditionTrue(backup.Status.Conditions, talosv1alpha1.ConditionReady) {
			successfulBackups = append(successfulBackups, backup)
		}
	}

	// Sort by creation timestamp (newest first)
	sort.Slice(successfulBackups, func(i, j int) bool {
		return successfulBackups[i].CreationTimestamp.After(successfulBackups[j].CreationTimestamp.Time)
	})

	// Delete backups exceeding retention
	if len(successfulBackups) > int(retention) {
		backupsToDelete := successfulBackups[retention:]
		for _, backup := range backupsToDelete {
			logger.Info("Deleting old backup due to retention policy", "backup", backup.Name)
			if err := r.Delete(ctx, &backup); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete backup %s: %w", backup.Name, err)
			}
		}
	}

	return nil
}
