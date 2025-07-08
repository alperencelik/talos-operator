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
	"encoding/base64"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/talos"
	"github.com/alperencelik/talos-operator/pkg/utils"
)

// TalosWorkerReconciler reconciles a TalosWorker object
type TalosWorkerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosworkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosworkers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosworkers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosWorker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TalosWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tw talosv1alpha1.TalosWorker
	if err := r.Get(ctx, req.NamespacedName, &tw); err != nil {
		// If the resource is not found, we simply return and do not requeue.
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	// Finalizer
	var finErr error
	if tw.ObjectMeta.DeletionTimestamp.IsZero() {
		finErr = r.handleFinalizer(ctx, tw)
		if finErr != nil {
			logger.Error(finErr, "failed to handle finalizer for TalosWorker", "name", tw.Name)
			return ctrl.Result{}, finErr
		}
	} else {
		// If the TalosWorker is being deleted, we handle the deletion logic
		if controllerutil.ContainsFinalizer(&tw, talosv1alpha1.TalosWorkerFinalizer) {
			// Handle the deletion logic here
			var res ctrl.Result
			res, finErr = r.handleDeletion(ctx, &tw)
			if finErr != nil {
				logger.Error(finErr, "failed to handle deletion for TalosWorker", "name", tw.Name)
				return res, finErr
			}
		}
		// Stop reconciling if the TalosWorker is being deleted
		return ctrl.Result{}, client.IgnoreNotFound(finErr)
	}

	// Get the mode of the TalosWorker
	var result ctrl.Result
	var err error
	switch tw.Spec.Mode {
	case TalosModeContainer:
		result, err = r.reconcileContainerMode(ctx, &tw)
		if err != nil {
			logger.Error(err, "failed to reconcile TalosWorker in container mode", "name", tw.Name)
		}
		return result, nil
	case TalosModeMetal:
		result, err := r.reconcileMetalMode(ctx, &tw)
		if err != nil {
			logger.Error(err, "failed to reconcile TalosWorker in metal mode", "name", tw.Name)
		}
		return result, nil
	default:
		logger.Error(nil, "unsupported TalosWorker mode", "mode", tw.Spec.Mode)
		return ctrl.Result{}, nil // Unsupported mode, do not requeue
	}
}

func (r *TalosWorkerReconciler) reconcileContainerMode(ctx context.Context, tw *talosv1alpha1.TalosWorker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TalosWorker in container mode", "name", tw.Name)
	// Generate the worker configuration
	if err := r.GenerateConfig(ctx, tw); err != nil {
		// Error is due to ref not found so report it in status and return without requeueing
		logger.Error(err, "failed to generate worker config", "name", tw.Name)
		return ctrl.Result{}, nil
	}
	if err := r.reconcileService(ctx, tw); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile service for TalosWorker %s: %w", tw.Name, err)
	}
	if err := r.reconcileStatefulSet(ctx, tw); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile StatefulSet for TalosWorker %s: %w", tw.Name, err)
	}
	if err := r.Status().Update(ctx, tw); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosWorker status %s: %w", tw.Name, err)
	}
	// Return no error and no requeue
	return ctrl.Result{}, nil
}

func (r *TalosWorkerReconciler) reconcileMetalMode(ctx context.Context, tw *talosv1alpha1.TalosWorker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TalosWorker in metal mode", "name", tw.Name)
	// Generate the worker configuration
	if err := r.GenerateConfig(ctx, tw); err != nil {
		// Error is due to ref not found so report it in status and return without requeueing
		logger.Error(err, "failed to generate worker config", "name", tw.Name)
		return ctrl.Result{}, nil
	}
	// Reconcile TalosMachine objects
	if err := r.handleTalosMachines(ctx, tw); err != nil {
		logger.Error(err, "failed to handle TalosMachines for TalosWorker", "name", tw.Name)
		return ctrl.Result{}, err
	}
	// Return no error and no requeue
	return ctrl.Result{}, nil
}

func (r *TalosWorkerReconciler) handleTalosMachines(ctx context.Context, tw *talosv1alpha1.TalosWorker) error {
	logger := log.FromContext(ctx)
	// List existing ones
	existing := &talosv1alpha1.TalosMachineList{}
	if err := r.List(ctx, existing, client.InNamespace(tw.Namespace),
		client.MatchingFields{"spec.workerRef.name": tw.Name},
	); err != nil {
		return fmt.Errorf("failed to list TalosMachines: %w", err)
	}
	// Desired state
	desired := make(map[string]bool)
	for _, ep := range tw.Spec.MetalSpec.Machines {
		desired[fmt.Sprintf("%s-%s", tw.Name, ep)] = true
	}
	// Delete orphaned machines
	for _, m := range existing.Items {
		if m.Spec.WorkerRef != nil && m.Spec.WorkerRef.Name == tw.Name {
			if !desired[m.Name] {
				if err := r.Delete(ctx, &m); err != nil && !kerrors.IsNotFound(err) {
					logger.Error(err, "Failed to delete orphaned TalosMachine", "name", m.Name)
					return fmt.Errorf("failed to delete orphaned TalosMachine %s: %w", m.Name, err)
				}
			}
		}
	}
	// Create or update machines
	for _, machine := range tw.Spec.MetalSpec.Machines {
		name := fmt.Sprintf("%s-%s", tw.Name, machine)
		tm := &talosv1alpha1.TalosMachine{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tw.Namespace},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, tm, func() error {
			if err := controllerutil.SetControllerReference(tw, tm, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference for TalosMachine %s: %w", tm.Name, err)
			}
			tm.Spec = talosv1alpha1.TalosMachineSpec{
				WorkerRef: &corev1.ObjectReference{
					Kind:       talosv1alpha1.GroupKindWorker,
					Name:       tw.Name,
					Namespace:  tw.Namespace,
					APIVersion: talosv1alpha1.GroupVersion.String(),
				},
				Endpoint:    machine,
				InstallDisk: tw.Spec.MetalSpec.InstallDisk,
			}
			return nil
		})
		if err != nil {
			logger.Error(err, "Failed to create or update TalosMachine", "name", tm.Name)
			return fmt.Errorf("failed to create or update TalosMachine %s: %w", tm.Name, err)
		}
	}
	return nil
}

func (r *TalosWorkerReconciler) reconcileService(ctx context.Context, tw *talosv1alpha1.TalosWorker) error {
	// Handle the service for the each replica of the TalosWorker
	for i := int32(0); i < tw.Spec.Replicas; i++ {
		svcName := fmt.Sprintf("%s-%d", tw.Name, i)
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: tw.Namespace,
			},
		}
		// set the owner reference for the service
		if err := controllerutil.SetControllerReference(tw, svc, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for Service %s: %w", svcName, err)
		}
		// Create or update the service
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
			svc.Spec = BuildServiceSpec(tw.Name, &i)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to create or update Service %s: %w", svcName, err)
		}
	}
	return nil
}

func (r *TalosWorkerReconciler) reconcileStatefulSet(ctx context.Context, tw *talosv1alpha1.TalosWorker) error {
	// TODO: Implement the logic to reconcile the StatefulSet for the TalosWorker
	stsName := tw.Name

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: tw.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tw, sts, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for StatefulSet %s: %w", stsName, err)
	}
	extraEnvs := BuildUserDataEnvVar(tw.Spec.ConfigRef, tw.Name, TalosMachineTypeWorker)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Spec = BuildStsSpec(tw.Name, tw.Spec.Replicas, tw.Spec.Version, TalosMachineTypeWorker, extraEnvs, tw.Spec.StorageClassName)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update StatefulSet %s: %w", stsName, err)
	}
	return nil

}

func (r *TalosWorkerReconciler) handleResourceNotFound(ctx context.Context, err error) error {
	logger := log.FromContext(ctx)
	if kerrors.IsNotFound(err) {
		// Resource not found, return without error
		logger.Info("TalosWorker resource not found, skipping reconciliation")
		return nil
	}
	// If the error is not a "not found" error, return it for further handling
	logger.Error(err, "Error fetching TalosWorker resource")
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&talosv1alpha1.TalosMachine{},
		// Index by the worker reference name
		"spec.workerRef.name",
		func(rawObj client.Object) []string {
			tm := rawObj.(*talosv1alpha1.TalosMachine)
			if tm.Spec.ControlPlaneRef != nil {
				return []string{tm.Spec.ControlPlaneRef.Name}
			}
			return nil
		},
	); err != nil {
		return fmt.Errorf("failed to index TalosMachine by workerRef.name: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosWorker{}).
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(stsPredicate)).
		Owns(&corev1.Service{}, builder.WithPredicates(svcPredicate)).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Only reconcile if the generation of the object has changed
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
		}).
		Named("talosworker").
		Complete(r)
}

func (r *TalosWorkerReconciler) GenerateConfig(ctx context.Context, tw *talosv1alpha1.TalosWorker) error {
	bundleConfig, err := r.SetConfig(ctx, tw)
	if err != nil {
		return fmt.Errorf("failed to set configuration for TalosWorker %s: %w", tw.Name, err)
	}
	// Generate the Talos worker configuration
	wkConfig, err := talos.GenerateWorkerConfig(bundleConfig, nil)
	if err != nil {
		return err
	}
	// Update the status
	if tw.Status.Config == string(*wkConfig) {
		return nil // No change in config, nothing to do
	}
	tw.Status.Config = string(*wkConfig)
	// Update the status of the TalosWorker
	if err := r.Status().Update(ctx, tw); err != nil {
		return fmt.Errorf("failed to update TalosWorker status %s: %w", tw.Name, err)
	}
	// Wrtie the Talos Worker configuration to a ConfigMap
	if err := r.WriteWorkerConfig(ctx, tw, wkConfig); err != nil {
		return fmt.Errorf("failed to write worker config: %w", err)
	}
	if err := r.updateState(ctx, tw, talosv1alpha1.StatePending); err != nil {
		return fmt.Errorf("failed to update TalosWorker state %s: %w", tw.Name, err)
	}
	return nil
}

func (r *TalosWorkerReconciler) WriteWorkerConfig(ctx context.Context, tw *talosv1alpha1.TalosWorker, wkConfig *[]byte) error {
	wkConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", tw.Name),
			Namespace: tw.Namespace,
		},
	}
	// Set the owner ref for the CM
	if err := ctrl.SetControllerReference(tw, wkConfigMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for ConfigMap: %w", err)
	}
	// Create or update the ConfigMap with the worker configuration
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, wkConfigMap, func() error {
		wkConfigMap.Data = map[string]string{
			"worker.yaml": base64.StdEncoding.EncodeToString(*wkConfig),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update ConfigMap: %w", err)
	}
	return nil
}

func (r *TalosWorkerReconciler) GetControlPlaneRef(ctx context.Context, tw *talosv1alpha1.TalosWorker) (tcp *talosv1alpha1.TalosControlPlane, err error) {
	logger := log.FromContext(ctx)
	// Fetch the TalosControlPlane reference from the ControlPlaneRef field
	tcp = &talosv1alpha1.TalosControlPlane{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tw.Spec.ControlPlaneRef.Name,
		Namespace: tw.Namespace,
	}, tcp); err != nil {
		if kerrors.IsNotFound(err) {
			logger.Error(err, "TalosControlPlane not found", "name", tw.Spec.ControlPlaneRef.Name)
			return nil, err
		}
		// Fire an event if the TalosControlPlane is not found and maybe requeue the reconciliation after some time
		r.Recorder.Eventf(tw, corev1.EventTypeWarning, "ControlPlaneNotFound",
			"TalosControlPlane %s not found in namespace %s", tw.Spec.ControlPlaneRef.Name, tw.Namespace)
		logger.Error(err, "Failed to get TalosControlPlane", "name", tw.Spec.ControlPlaneRef.Name)
		return nil, err
	}
	return tcp, nil
}

func (r *TalosWorkerReconciler) SetConfig(ctx context.Context, tw *talosv1alpha1.TalosWorker) (*talos.BundleConfig, error) {
	// Get the TalosControlPlane reference
	tcp, err := r.GetControlPlaneRef(ctx, tw)
	if err != nil {
		return nil, err
	}
	var replicas int
	var sans []string
	if tw.Spec.Mode == "container" {
		replicas = int(tw.Spec.Replicas)
		sans = utils.GenSans(tw.Name, &replicas)
	}
	sb, err := utils.SecretBundleDecoder(tcp.Status.SecretBundle)
	if err != nil {
		return nil, err
	}
	var ClientEndpoint []string
	if tw.Spec.Mode == "metal" {
		// TODO: Handle multiple machines in metal mode
		ClientEndpoint = tw.Spec.MetalSpec.Machines
	}
	var endpoint string
	// Construct endpoint
	if tcp.Spec.Endpoint != "" {
		endpoint = tcp.Spec.Endpoint
	} else {
		// Default endpoint is the TalosControlPlane name
		endpoint = fmt.Sprintf("https://%s:6443", tcp.Name)
	}
	// Generate the worker configuration
	return &talos.BundleConfig{
		ClusterName:    tcp.Name,
		Endpoint:       endpoint,
		Version:        tcp.Spec.Version,
		KubeVersion:    tcp.Spec.KubeVersion,
		SecretsBundle:  sb,
		Sans:           sans,
		ServiceCIDR:    &tcp.Spec.ServiceCIDR,
		PodCIDR:        &tcp.Spec.PodCIDR,
		ClientEndpoint: &ClientEndpoint,
	}, nil

}

func (r *TalosWorkerReconciler) handleFinalizer(ctx context.Context, tw talosv1alpha1.TalosWorker) error {
	logger := log.FromContext(ctx)
	if !controllerutil.ContainsFinalizer(&tw, talosv1alpha1.TalosWorkerFinalizer) {
		// Add the finalizer if it doesn't exist
		controllerutil.AddFinalizer(&tw, talosv1alpha1.TalosWorkerFinalizer)
		if err := r.Update(ctx, &tw); err != nil {
			logger.Error(err, "failed to add finalizer to TalosWorker", "name", tw.Name)
			return err
		}
	}
	return nil
}

func (r *TalosWorkerReconciler) handleDeletion(ctx context.Context, tw *talosv1alpha1.TalosWorker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deleting TalosWorker", "name", tw.Name)

	// Update the conditions to reflect the deletion
	if !meta.IsStatusConditionPresentAndEqual(tw.Status.Conditions, talosv1alpha1.ConditionDeleting, metav1.ConditionUnknown) {
		meta.SetStatusCondition(&tw.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionDeleting,
			Status:  metav1.ConditionTrue,
			Reason:  "Deleting",
			Message: "TalosWorker is being deleted",
		})
		if err := r.Status().Update(ctx, tw); err != nil {
			logger.Error(err, "failed to update TalosWorker status during deletion", "name", tw.Name)
			return ctrl.Result{}, err
		}
	}
	// Based on the TalosWorker mode, we can handle the deletion logic
	switch tw.Spec.Mode {
	case TalosModeContainer:
		// Do nothing
	default:
		logger.Error(nil, "unsupported TalosWorker mode for deletion", "mode", tw.Spec.Mode)
	}
	// Remove the finalizer to allow the resource to be deleted
	controllerutil.RemoveFinalizer(tw, talosv1alpha1.TalosWorkerFinalizer)
	if err := r.Update(ctx, tw); err != nil {
		logger.Error(err, "failed to remove finalizer from TalosWorker", "name", tw.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TalosWorkerReconciler) updateState(ctx context.Context, tw *talosv1alpha1.TalosWorker, state string) error {
	if tw.Status.State == state {
		return nil
	}
	tw.Status.State = state
	if err := r.Status().Update(ctx, tw); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to %s: %w", tw.Name, state, err)

	}
	return nil
}
