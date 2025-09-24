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

	"github.com/alperencelik/talos-operator/pkg/talos"
	"k8s.io/apimachinery/pkg/runtime"
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

	// Get the controlplane ref
	var tcp talosv1alpha1.TalosControlPlane
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: teb.Spec.TalosControlPlaneRef.Name}, &tcp); err != nil {
		logger.Error(err, "Failed to get TalosControlPlane", "TalosControlPlane", teb.Spec.TalosControlPlaneRef.Name)
		return ctrl.Result{}, err
	}
	bcString := tcp.Status.BundleConfig
	if bcString == "" {
		return ctrl.Result{}, fmt.Errorf("TalosControlPlane %s bundleConfig is empty", tcp.Name)
	}
	// Parse the bundle config
	bc, err := talos.ParseBundleConfig(bcString)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse bundle config: %w", err)
	}
	tc, err := talos.NewClient(bc, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create talos client: %w", err)
	}
	_, err = tc.TakeSnapshot(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to take etcd snapshot: %w", err)
	}
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
