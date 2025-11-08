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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	operatormetrics "github.com/alperencelik/talos-operator/internal/metrics"
)

// TalosClusterReconciler reconciles a TalosCluster object
type TalosClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TalosClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	timer := operatormetrics.NewTimer()
	reconcileResult := "success"

	defer func() {
		timer.ObserveReconciliation("taloscluster", reconcileResult)
	}()

	var tc talosv1alpha1.TalosCluster
	if err := r.Get(ctx, req.NamespacedName, &tc); err != nil {
		if kerrors.IsNotFound(err) {
			reconcileResult = "not_found"
		} else {
			reconcileResult = "error"
		}
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	// Finalizer logic
	if tc.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		err := r.handleFinalizer(ctx, tc)
		if err != nil {
			logger.Error(err, "failed to handle finalizer for TalosCluster", "name", tc.Name)
			return ctrl.Result{}, err
		}
	} else {
		// Object is being deleted
		if controllerutil.ContainsFinalizer(&tc, talosv1alpha1.TalosClusterFinalizer) {

			res, err := r.handleDelete(ctx, tc)
			if err != nil {
				return res, fmt.Errorf("failed to handle delete for TalosCluster %s: %w", tc.Name, err)
			}
			// Remove finalizer and update
			controllerutil.RemoveFinalizer(&tc, talosv1alpha1.TalosClusterFinalizer)
			if err := r.Update(ctx, &tc); err != nil {
				logger.Error(err, "failed to remove finalizer from TalosCluster", "name", tc.Name)
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as object is under deletion
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling TalosCluster", "name", tc.Name, "namespace", tc.Namespace)
	// Get Reconciliation mode from annotation
	reconcileMode := r.getReconciliationMode(ctx, &tc)
	switch strings.ToLower(reconcileMode) {
	case ReconcileModeDisable:
		logger.Info("Reconciliation is disabled for this TalosCluster", "name", tc.Name, "namespace", tc.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeDryRun:
		logger.Info("Dry run mode is not implemented yet, skipping reconciliation", "name", tc.Name, "namespace", tc.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeNormal:
		// Do nothing, proceed with reconciliation
	}

	// Control Plane reconciliation
	res, err := r.reconcileControlPlane(ctx, &tc)
	if err != nil {
		logger.Error(err, "failed to reconcile control plane", "name", tc.Name, "namespace", tc.Namespace)
		reconcileResult = "error"
	}
	if res.Requeue {
		reconcileResult = "requeue"
		return res, nil
	}
	// Worker reconciliation
	res, err = r.reconcileWorker(ctx, &tc)
	if err != nil {
		logger.Error(err, "failed to reconcile worker", "name", tc.Name, "namespace", tc.Namespace)
		reconcileResult = "error"
	}

	// Update cluster health metric based on status
	healthy := meta.IsStatusConditionTrue(tc.Status.Conditions, talosv1alpha1.ConditionReady)
	operatormetrics.SetClusterHealth(tc.Namespace, tc.Name, healthy)

	// Update resource status metric
	status := "not_ready"
	if healthy {
		status = "ready"
	}
	operatormetrics.SetResourceStatus("taloscluster", tc.Namespace, tc.Name, status, 1.0)

	if res.Requeue {
		reconcileResult = "requeue"
	}
	return res, nil

}

// reconcileControlPlane handles the reconciliation logic for the Talos control plane.
func (r *TalosClusterReconciler) reconcileControlPlane(ctx context.Context, tc *talosv1alpha1.TalosCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if tc.Spec.ControlPlane == nil && tc.Spec.ControlPlaneRef == nil {
		logger.Info("No control plane configuration found, skipping reconciliation")
		return ctrl.Result{}, nil
	}
	if tc.Spec.ControlPlaneRef != nil {
		// Get the TalosControlPlane resource referenced by ControlPlaneRef
		var tcp talosv1alpha1.TalosControlPlane
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tc.Spec.ControlPlaneRef.Name,
			Namespace: tc.Namespace,
		}, &tcp); err != nil {
			logger.Error(err, "unable to fetch TalosControlPlane", "name", tc.Spec.ControlPlaneRef.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	// If ControlPlaneRef is not set, create a new TalosControlPlane resource
	// If inline ControlPlane spec is provided and no ControlPlaneRef, reconcile TalosControlPlane resource
	if tc.Spec.ControlPlane != nil && tc.Spec.ControlPlaneRef == nil {
		tcp := &talosv1alpha1.TalosControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tc.Name + "-controlplane",
				Namespace: tc.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, tcp, func() error {
			// set owner reference
			if err := controllerutil.SetOwnerReference(tc, tcp, r.Scheme); err != nil {
				return err
			}
			// set desired spec
			tcp.Spec = talosv1alpha1.TalosControlPlaneSpec{
				Version:          tc.Spec.ControlPlane.Version,
				Mode:             tc.Spec.ControlPlane.Mode,
				Replicas:         tc.Spec.ControlPlane.Replicas,
				Endpoint:         tc.Spec.ControlPlane.Endpoint,
				MetalSpec:        tc.Spec.ControlPlane.MetalSpec,
				KubeVersion:      tc.Spec.ControlPlane.KubeVersion,
				ClusterDomain:    tc.Spec.ControlPlane.ClusterDomain,
				StorageClassName: tc.Spec.ControlPlane.StorageClassName,
				PodCIDR:          tc.Spec.ControlPlane.PodCIDR,
				ServiceCIDR:      tc.Spec.ControlPlane.ServiceCIDR,
			}
			// Optionally set ConfigRef if provided
			if tc.Spec.ControlPlane.ConfigRef != nil {
				tcp.Spec.ConfigRef = &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tc.Spec.ControlPlane.ConfigRef.Name,
					},
					Key: tc.Spec.ControlPlane.ConfigRef.Key,
				}
			}
			return nil
		})
		if err != nil {
			logger.Error(err, "failed to create or update TalosControlPlane", "operation", op, "name", tcp.Name, "namespace", tcp.Namespace)
			return ctrl.Result{Requeue: true}, err
		}
		// Based on the op generate an event
		switch op {
		case controllerutil.OperationResultCreated:
			r.Recorder.Event(tc, corev1.EventTypeNormal, "Created", "TalosControlPlane created successfully")
			// Requeue the request to ensure that talosWorker is created after control plane is ready
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		case controllerutil.OperationResultUpdated:
			r.Recorder.Event(tc, corev1.EventTypeNormal, "Updated", "TalosControlPlane updated successfully")
			// If it's only a update then requeue immediately
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Info("Reconciled TalosControlPlane", "operation", op, "name", tcp.Name, "namespace", tcp.Namespace)
	}
	return ctrl.Result{}, nil
}

// reconcileWorker handles the reconciliation logic for the Talos worker nodes.
func (r *TalosClusterReconciler) reconcileWorker(ctx context.Context, tc *talosv1alpha1.TalosCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If neither inline worker spec nor reference is provided, skip reconciliation.
	if tc.Spec.Worker == nil && tc.Spec.WorkerRef == nil {
		logger.Info("No worker configuration found, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// If WorkerRef is set, ensure the referenced TalosWorker exists.
	if tc.Spec.WorkerRef != nil {
		var tw talosv1alpha1.TalosWorker
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tc.Spec.WorkerRef.Name,
			Namespace: tc.Namespace,
		}, &tw); err != nil {
			logger.Error(err, "unable to fetch TalosWorker", "name", tc.Spec.WorkerRef.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}
	var controlPlaneRefName string
	if tc.Spec.ControlPlaneRef != nil {
		controlPlaneRefName = tc.Spec.ControlPlaneRef.Name
	} else {
		controlPlaneRefName = tc.Name + "-controlplane"
	}

	// Inline Worker spec is provided and no WorkerRef, reconcile TalosWorker resource
	workerName := tc.Name + "-worker"
	tw := &talosv1alpha1.TalosWorker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workerName,
			Namespace: tc.Namespace,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, tw, func() error {
		// set owner reference
		if err := controllerutil.SetOwnerReference(tc, tw, r.Scheme); err != nil {
			return err
		}

		// set desired spec
		tw.Spec = talosv1alpha1.TalosWorkerSpec{
			Version:          tc.Spec.Worker.Version,
			Mode:             tc.Spec.Worker.Mode,
			Replicas:         tc.Spec.Worker.Replicas,
			MetalSpec:        tc.Spec.Worker.MetalSpec,
			KubeVersion:      tc.Spec.Worker.KubeVersion,
			StorageClassName: tc.Spec.Worker.StorageClassName,
			ControlPlaneRef: corev1.LocalObjectReference{
				Name: controlPlaneRefName,
			},
		}
		// Optionally set ConfigRef if provided
		if tc.Spec.Worker.ConfigRef != nil {
			tw.Spec.ConfigRef = &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tc.Spec.Worker.ConfigRef.Name,
				},
				Key: tc.Spec.Worker.ConfigRef.Key,
			}
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "failed to create or update TalosWorker", "operation", op, "name", tw.Name)
		return ctrl.Result{Requeue: true}, err
	}
	// Based on the op generate an event
	switch op {
	case controllerutil.OperationResultCreated:
		r.Recorder.Event(tc, corev1.EventTypeNormal, "Created", "TalosWorker created successfully")
	case controllerutil.OperationResultUpdated:
		r.Recorder.Event(tc, corev1.EventTypeNormal, "Updated", "TalosWorker updated successfully")
	}
	logger.Info("Reconciled TalosWorker", "operation", op, "name", tw.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosCluster{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&talosv1alpha1.TalosControlPlane{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&talosv1alpha1.TalosWorker{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("taloscluster").
		Complete(r)
}

func (r *TalosClusterReconciler) handleFinalizer(ctx context.Context, tc talosv1alpha1.TalosCluster) error {
	if !controllerutil.ContainsFinalizer(&tc, talosv1alpha1.TalosClusterFinalizer) {
		controllerutil.AddFinalizer(&tc, talosv1alpha1.TalosClusterFinalizer)
		if err := r.Update(ctx, &tc); err != nil {
			return fmt.Errorf("failed to add finalizer to TalosCluster: %w", err)
		}
	}
	return nil
}

func (r *TalosClusterReconciler) handleResourceNotFound(ctx context.Context, err error) error {
	logger := log.FromContext(ctx)
	if kerrors.IsNotFound(err) {
		logger.Info("TalosControlPlane resource not found. Ignoring since object must be deleted")
		return nil
	}
	return err
}

func (r *TalosClusterReconciler) handleDelete(ctx context.Context, tc talosv1alpha1.TalosCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update the conditions of the TalosCluster to reflect deletion
	if !meta.IsStatusConditionPresentAndEqual(tc.Status.Conditions, talosv1alpha1.ConditionDeleting, metav1.ConditionUnknown) {
		meta.SetStatusCondition(&tc.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionDeleting,
			Status:  metav1.ConditionUnknown,
			Reason:  "Deleting",
			Message: "TalosCluster is being deleted",
		})
		if err := r.Status().Update(ctx, &tc); err != nil {
			logger.Error(err, "failed to update TalosCluster status during deletion")
			return ctrl.Result{Requeue: true}, err
		}
	}
	// Delete the child resources if they exist
	if err := r.Delete(ctx, &talosv1alpha1.TalosWorker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tc.Name + "-worker",
			Namespace: tc.Namespace,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		logger.Error(err, "failed to delete TalosWorker during TalosCluster deletion", "name", tc.Name)
		return ctrl.Result{}, err
	}
	// Delete the TalosControlPlane
	if err := r.Delete(ctx, &talosv1alpha1.TalosControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tc.Name + "-controlplane",
			Namespace: tc.Namespace,
		},
	}); err != nil && !kerrors.IsNotFound(err) {
		logger.Error(err, "failed to delete TalosControlPlane during TalosCluster deletion", "name", tc.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TalosClusterReconciler) getReconciliationMode(ctx context.Context, tc *talosv1alpha1.TalosCluster) string {
	logger := log.FromContext(ctx)
	// Check if the annotation exists
	mode, exists := tc.Annotations[ReconcileModeAnnotation]
	if !exists {
		return ReconcileModeNormal
	}
	switch mode {
	case ReconcileModeNormal:
		logger.Info("Reconciliation mode is set to Normal")
		return ReconcileModeNormal
	case ReconcileModeDisable:
		logger.Info("Reconciliation mode is set to Disable")
		return ReconcileModeDisable
	case ReconcileModeDryRun:
		logger.Info("Reconciliation mode is set to DryRun")
		return ReconcileModeDryRun
	default:
		logger.Info("Unknown reconciliation mode, defaulting to Normal")
		return ReconcileModeNormal
	}
}
