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

	"github.com/alperencelik/talos-operator/pkg/storage"
	"github.com/alperencelik/talos-operator/pkg/talos"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

// TalosEtcdBackupReconciler reconciles a TalosEtcdBackup object
type TalosEtcdBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosetcdbackups/finalizers,verbs=update
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosEtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *TalosEtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var teb talosv1alpha1.TalosEtcdBackup
	if err := r.Get(ctx, req.NamespacedName, &teb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Finalizer logic
	if teb.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer for this CR
		err := r.handleFinalizer(ctx, &teb)
		if err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&teb, talosv1alpha1.TalosEtcdBackupFinalizer) {
			res, err := r.handleDelete(ctx, &teb)
			if err != nil {
				return res, fmt.Errorf("failed to handle delete: %w", err)
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(&teb, talosv1alpha1.TalosEtcdBackupFinalizer)
			if err := r.Update(ctx, &teb); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	logger.Info("Reconciling TalosEtcdBackup", "TalosEtcdBackup", req.NamespacedName)

	// Set progressing condition
	meta.SetStatusCondition(&teb.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionProgressing,
		Status:  metav1.ConditionTrue,
		Reason:  "BackupInProgress",
		Message: "Etcd backup is in progress",
	})
	if err := r.Status().Update(ctx, &teb); err != nil {
		logger.Error(err, "Failed to update status to progressing")
	}

	// Perform the backup
	if err := r.performBackup(ctx, &teb); err != nil {
		logger.Error(err, "Failed to perform backup")

		// Set failed condition
		meta.SetStatusCondition(&teb.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "BackupFailed",
			Message: fmt.Sprintf("Failed to backup etcd: %v", err),
		})
		meta.SetStatusCondition(&teb.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionProgressing,
			Status:  metav1.ConditionFalse,
			Reason:  "BackupFailed",
			Message: "Backup failed",
		})

		if statusErr := r.Status().Update(ctx, &teb); statusErr != nil {
			logger.Error(statusErr, "Failed to update status after backup failure")
		}

		return ctrl.Result{}, err
	}

	// Set ready condition
	meta.SetStatusCondition(&teb.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "BackupSucceeded",
		Message: "Etcd backup completed successfully",
	})
	meta.SetStatusCondition(&teb.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionProgressing,
		Status:  metav1.ConditionFalse,
		Reason:  "BackupCompleted",
		Message: "Backup completed",
	})
	if err := r.Status().Update(ctx, &teb); err != nil {
		logger.Error(err, "Failed to update status to ready")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully completed etcd backup", "TalosEtcdBackup", req.NamespacedName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosEtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosEtcdBackup{}).
		Named("talosetcdbackup").
		Complete(r)
}

func (r *TalosEtcdBackupReconciler) handleFinalizer(ctx context.Context, teb *talosv1alpha1.TalosEtcdBackup) error {
	if !controllerutil.ContainsFinalizer(teb, talosv1alpha1.TalosEtcdBackupFinalizer) {
		controllerutil.AddFinalizer(teb, talosv1alpha1.TalosEtcdBackupFinalizer)
		if err := r.Update(ctx, teb); err != nil {
			return fmt.Errorf("failed to update TalosEtcdBackup with finalizer: %w", err)
		}
	}
	return nil
}

func (r *TalosEtcdBackupReconciler) handleDelete(ctx context.Context, teb *talosv1alpha1.TalosEtcdBackup) (ctrl.Result, error) {
	// TODO: Remove backup from external storage if needed
	return ctrl.Result{}, nil
}

// performBackup executes the etcd backup by streaming from Talos to S3
func (r *TalosEtcdBackupReconciler) performBackup(ctx context.Context, teb *talosv1alpha1.TalosEtcdBackup) error {
	logger := logf.FromContext(ctx)

	// Get the TalosControlPlane
	var tcp talosv1alpha1.TalosControlPlane
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: teb.Namespace,
		Name:      teb.Spec.TalosControlPlaneRef.Name,
	}, &tcp); err != nil {
		return fmt.Errorf("failed to get TalosControlPlane: %w", err)
	}

	// Validate bundle config
	bcString := tcp.Status.BundleConfig
	if bcString == "" {
		return fmt.Errorf("TalosControlPlane %s bundleConfig is empty", tcp.Name)
	}

	// Parse the bundle config
	bc, err := talos.ParseBundleConfig(bcString)
	if err != nil {
		return fmt.Errorf("failed to parse bundle config: %w", err)
	}

	// Create Talos client
	talosClient, err := talos.NewClient(bc, false)
	if err != nil {
		return fmt.Errorf("failed to create talos client: %w", err)
	}

	// Get S3 configuration and credentials from secrets
	s3Config, err := r.getS3Config(ctx, teb)
	if err != nil {
		return fmt.Errorf("failed to get S3 configuration: %w", err)
	}

	// Create S3 client
	s3Client, err := storage.NewS3Client(ctx, s3Config)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Get etcd snapshot reader (streaming)
	logger.Info("Starting etcd snapshot from Talos API")
	snapshotReader, err := talosClient.EtcdSnapshotReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get etcd snapshot reader: %w", err)
	}
	defer snapshotReader.Close()

	// Generate backup key
	backupKey := storage.GenerateBackupKey(tcp.Name)
	logger.Info("Uploading etcd snapshot to S3", "bucket", s3Config.Bucket, "key", backupKey)

	// Stream directly to S3
	if err := s3Client.Upload(ctx, backupKey, snapshotReader); err != nil {
		return fmt.Errorf("failed to upload snapshot to S3: %w", err)
	}

	logger.Info("Successfully uploaded etcd snapshot to S3", "bucket", s3Config.Bucket, "key", backupKey)
	return nil
}

// getS3Config retrieves S3 configuration from the TalosEtcdBackup spec and resolves credentials from secrets
func (r *TalosEtcdBackupReconciler) getS3Config(ctx context.Context, teb *talosv1alpha1.TalosEtcdBackup) (*storage.S3Config, error) {
	if teb.Spec.BackupStorage.S3 == nil {
		return nil, fmt.Errorf("S3 storage configuration is not specified")
	}

	s3Spec := teb.Spec.BackupStorage.S3

	// Retrieve access key ID from secret
	accessKeyID, err := r.getSecretValue(ctx, teb.Namespace, s3Spec.AccessKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get access key ID: %w", err)
	}

	// Retrieve secret access key from secret
	secretAccessKey, err := r.getSecretValue(ctx, teb.Namespace, s3Spec.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret access key: %w", err)
	}

	return &storage.S3Config{
		Bucket:             s3Spec.Bucket,
		Region:             s3Spec.Region,
		Endpoint:           s3Spec.Endpoint,
		AccessKeyID:        accessKeyID,
		SecretAccessKey:    secretAccessKey,
		InsecureSkipVerify: s3Spec.InsecureSkipTLSVerify,
	}, nil
}

// getSecretValue retrieves a value from a Kubernetes secret
func (r *TalosEtcdBackupReconciler) getSecretValue(ctx context.Context, namespace string, selector *corev1.SecretKeySelector) (string, error) {
	if selector == nil {
		return "", fmt.Errorf("secret selector is nil")
	}

	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      selector.Name,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", selector.Name, err)
	}

	value, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)
	}

	return string(value), nil
}
