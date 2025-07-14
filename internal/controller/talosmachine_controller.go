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
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/talos"
	"github.com/alperencelik/talos-operator/pkg/utils"
)

// TalosMachineReconciler reconciles a TalosMachine object
type TalosMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TalosMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the machine object and decide whether it's a control plane or worker machine
	var talosMachine talosv1alpha1.TalosMachine
	if err := r.Get(ctx, req.NamespacedName, &talosMachine); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	logger.Info("Reconciling TalosMachine", "name", talosMachine.Name, "namespace", talosMachine.Namespace)
	// Finalizer
	if talosMachine.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so we add the finalizer if it's not already present
		err := r.handleFinalizer(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Failed to handle finalizer for TalosMachine", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted, so we handle the finalizer logic
		if controllerutil.ContainsFinalizer(&talosMachine, talosv1alpha1.TalosMachineFinalizer) {
			// Run delete operations
			res, err := r.handleDelete(ctx, &talosMachine)
			if err != nil {
				logger.Error(err, "Failed to handle delete for TalosMachine", "name", talosMachine.Name)
				return res, err
			}
			// Remove the finalizer
			controllerutil.RemoveFinalizer(&talosMachine, talosv1alpha1.TalosMachineFinalizer)
			if err := r.Update(ctx, &talosMachine); err != nil {
				logger.Error(err, "Failed to remove finalizer for TalosMachine", "name", talosMachine.Name)
				return ctrl.Result{}, err
			}
		}
		// Stop the reconciliation if the finalizer is not present
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}
	// Check whether we should wait for machine to be ready
	if talosMachine.Status.State == talosv1alpha1.StateInstalling {
		// If the machine is in the installing state, we should wait for it to be ready
		res, err := r.CheckMachineReady(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error checking machine readiness", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		if res.Requeue {
			logger.Info("Requeuing reconciliation to check machine readiness", "name", talosMachine.Name)
			return res, nil // Requeue the reconciliation to check the machine status again
		}
	}
	// Re-get the machine object to ensure we have the latest state
	if err := r.Get(ctx, req.NamespacedName, &talosMachine); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Check for the machine type and handle accordingly
	switch {
	case talosMachine.Spec.ControlPlaneRef != nil:
		// Handle control plane specific logic here
		res, err := r.handleControlPlaneMachine(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error handling Control Plane machine", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		return res, nil
	case talosMachine.Spec.WorkerRef != nil:
		// Handle control plane specific logic here
		res, err := r.handleWorkerMachine(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error handling Worker machine", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		return res, nil
	default:
		logger.Info("TalosMachine is neither Control Plane nor Worker", "name", talosMachine.Name)
		return ctrl.Result{}, nil
	}
}

func (r *TalosMachineReconciler) handleControlPlaneMachine(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Get the bundle config from TalosControlPlane
	bc, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		logger.Error(err, "Failed to get BundleConfig for TalosMachine", "name", tm.Name)
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	if bc == nil {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tm.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}
	// Apply patches to config before applying it
	patches, err := r.metalConfigPatches(ctx, tm, bc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get metal config patches for TalosMachine %s: %w", tm.Name, err)
	}
	cpConfig, err := talos.GenerateControlPlaneConfig(bc, patches)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate Control Plane config for TalosMachine %s: %w", tm.Name, err)
	}
	// Check if the current config is the same as the one in status
	if tm.Status.Config == string(*cpConfig) {
		// Return since the config has not changed
		return ctrl.Result{}, nil
	}
	// Write that cpConfig to status.config
	tm.Status.Config = string(*cpConfig)
	if err := r.Status().Update(ctx, tm); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status with config: %w", tm.Name, err)
	}
	var insecure bool
	if tm.Status.State == talosv1alpha1.StatePending || tm.Status.State == "" {
		insecure = true // If the machine is pending, we allow insecure TLS
	} else {
		insecure = false // Otherwise, we use secure TLS
	}
	// Create Talos client
	tc, err := talos.NewClient(bc, insecure) // true for insecure TLS
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	if err = tc.ApplyConfig(ctx, *cpConfig); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply Talos config for TalosMachine %s: %w", tm.Name, err)
	}
	// Update the state to Installing
	if err := r.updateState(ctx, tm, talosv1alpha1.StateInstalling); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status to Installing: %w", tm.Name, err)
	}
	// TODO: Review here to make it more event driven -- maybe implement watcher, etc.
	return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check the machine status again
}

// TODO: Fix this one
func (r *TalosMachineReconciler) handleWorkerMachine(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Get config from WorkerRef
	tw := &talosv1alpha1.TalosWorker{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tm.Spec.WorkerRef.Name,
		Namespace: tm.Namespace,
	}, tw); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	bc, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		logger.Error(err, "Failed to get BundleConfig for TalosMachine", "name", tm.Name)
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	if bc == nil {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tm.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}
	// Apply patches to config before applying it
	patches, err := r.metalConfigPatches(ctx, tm, bc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get metal config patches for TalosMachine %s: %w", tm.Name, err)
	}
	// Generate the worker config
	workerConfig, err := talos.GenerateWorkerConfig(bc, patches)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate Worker config for TalosMachine %s: %w", tm.Name, err)
	}
	// Check if the current config is the same as the one in status
	if tm.Status.Config == string(*workerConfig) {
		// Return since the config has not changed
		return ctrl.Result{}, nil
	}
	// Write that workerConfig to status.config
	tm.Status.Config = string(*workerConfig)
	if err := r.Status().Update(ctx, tm); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status with config: %w", tm.Name, err)
	}
	var insecure bool
	if tm.Status.State == talosv1alpha1.StatePending || tm.Status.State == "" {
		insecure = true // If the machine is pending, we allow insecure TLS
	} else {
		insecure = false // Otherwise, we use secure TLS
	}
	// Create Talos client
	tc, err := talos.NewClient(bc, insecure) // true for insecure TLS
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	// Apply the worker config
	if err := tc.ApplyConfig(ctx, *workerConfig); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply Talos config for TalosMachine %s: %w", tm.Name, err)
	}
	// Update the state to Installing
	if err := r.updateState(ctx, tm, talosv1alpha1.StateInstalling); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status to Installing: %w", tm.Name, err)
	}
	// TODO: Review here to make it more event driven -- maybe implement watcher, etc.
	return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check the machine status again
}

func (r *TalosMachineReconciler) handleResourceNotFound(ctx context.Context, err error) error {
	logger := log.FromContext(ctx)
	if kerrors.IsNotFound(err) {
		logger.Info("TalosMachine resource not found. Ignoring since object must be deleted")
		return nil
	}
	return err
}

func (r *TalosMachineReconciler) updateState(ctx context.Context, tm *talosv1alpha1.TalosMachine, state string) error {
	if tm.Status.State == state {
		return nil
	}
	tm.Status.State = state
	if err := r.Status().Update(ctx, tm); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to %s: %w", tm.Name, state, err)

	}
	return nil
}

func (r *TalosMachineReconciler) GetControlPlaneRef(ctx context.Context, tm *talosv1alpha1.TalosMachine) (*talosv1alpha1.TalosControlPlane, error) {
	tcp := &talosv1alpha1.TalosControlPlane{}
	// If it's a controlPlane machine get it from TalosMachine --> TalosWorker --> TalosControlPlane
	// Check if it's a worker machine
	if tm.Spec.ControlPlaneRef == nil {
		tw := &talosv1alpha1.TalosWorker{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tm.Spec.WorkerRef.Name,
			Namespace: tm.Namespace,
		}, tw); err != nil {
			return nil, r.handleResourceNotFound(ctx, err)
		}
		// TODO: Check the controlPlane reference in TalosWorker
		name := tw.Spec.ControlPlaneRef.Name
		if name == "" {
			return nil, fmt.Errorf("TalosWorker %s does not have a Control Plane reference", tw.Name)
		}
		// Get the TalosControlPlane reference from the TalosWorker
		if err := r.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: tm.Namespace,
		}, tcp); err != nil {
			return nil, r.handleResourceNotFound(ctx, err)
		}
	} else {
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tm.Spec.ControlPlaneRef.Name,
			Namespace: tm.Namespace,
		}, tcp); err != nil {
			return nil, r.handleResourceNotFound(ctx, err)
		}
	}
	return tcp, nil
}

func (r *TalosMachineReconciler) GetBundleConfig(ctx context.Context, tm *talosv1alpha1.TalosMachine) (*talos.BundleConfig, error) {
	logger := log.FromContext(ctx)
	// Get the TalosControlPlane reference
	tcp, err := r.GetControlPlaneRef(ctx, tm)
	if err != nil {
		logger.Error(err, "Failed to get Control Plane reference for TalosMachine", "name", tm.Name)
	}
	if tcp == nil {
		logger.Info("TalosControlPlane reference is nil, waiting for it to be ready", "name", tm.Name)
		// TODO: Requeue after some time to check again
		return nil, nil
	}
	// Get bundleConfig from TalosControlPlane status
	bcString := tcp.Status.BundleConfig
	if bcString == "" {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tcp.Name)
		return nil, nil
	}
	// Parse the bundleConfig
	bc, err := talos.ParseBundleConfig(bcString)
	if err != nil {
		logger.Error(err, "Failed to parse Talos bundle config", "name", tcp.Name)
		return nil, fmt.Errorf("failed to parse Talos bundle config for Control Plane %s: %w", tcp.Name, err)
	}
	secretBundle, err := utils.SecretBundleDecoder(tcp.Status.SecretBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret bundle for Control Plane %s: %w", tcp.Name, err)
	}
	secretBundle.Clock = talos.NewClock()
	bc.SecretsBundle = secretBundle
	// TODO: Review that one for worker machines
	if tm.Spec.WorkerRef != nil {
		// Get the TalosWorker reference
		tw := &talosv1alpha1.TalosWorker{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tm.Spec.WorkerRef.Name,
			Namespace: tm.Namespace,
		}, tw); err != nil {
			return nil, r.handleResourceNotFound(ctx, err)
		}
		bc.ClientEndpoint = &tw.Spec.MetalSpec.Machines
	}
	return bc, nil
}

func (r *TalosMachineReconciler) handleFinalizer(ctx context.Context, tm *talosv1alpha1.TalosMachine) error {
	if !controllerutil.ContainsFinalizer(tm, talosv1alpha1.TalosMachineFinalizer) {
		controllerutil.AddFinalizer(tm, talosv1alpha1.TalosMachineFinalizer)
		if err := r.Update(ctx, tm); err != nil {
			return err
		}
	}
	return nil
}

func (r *TalosMachineReconciler) handleDelete(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	// Run talosctl reset command to reset the machine
	config, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	// Make the client for the machine
	config.ClientEndpoint = &[]string{tm.Spec.Endpoint}
	tc, err := talos.NewClient(config, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	if err := tc.Reset(ctx, false, true); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reset TalosMachine %s: %w", tm.Name, err)
	}
	return ctrl.Result{}, nil
}

func (r *TalosMachineReconciler) metalConfigPatches(ctx context.Context, tm *talosv1alpha1.TalosMachine, config *talos.BundleConfig) (*[]string, error) {

	var insecure = false
	if tm.Status.State == talosv1alpha1.StatePending || tm.Status.State == "" {
		insecure = true // Use insecure mode for pending state
	}
	// If the mode is metal, we need to apply the metal-specific patches -- diskPatch
	talosclient, err := talos.NewClient(config, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	diskNamePtr, err := talosclient.GetInstallDisk(ctx, tm)
	if err != nil {
		return nil, fmt.Errorf("failed to get install disk for TalosMachine %s: %w", tm.Name, err)
	}
	diskName := utils.PtrToString(diskNamePtr)
	diskPatch := fmt.Sprintf(talos.InstallDisk, diskName)
	return &[]string{diskPatch, talos.WipeDisk}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosMachine{}).
		Named("talosmachine").
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Only reconcile if the generation of the object has changed
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
		}).
		Complete(r)
}

func (r *TalosMachineReconciler) CheckMachineReady(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	// To check a machine take a look for Kubelet status
	logger := log.FromContext(ctx)
	// Create Talos client
	config, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	// Anytime we check machine we should beb apply-config before already so create secureClient
	tc, err := talos.NewClient(config, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	// Check if the machine is ready
	svcState := tc.GetServiceStatus(ctx, "kubelet")
	if svcState == "" {
		logger.Info("Kubelet service state is empty, requeuing reconciliation", "name", tm.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
	}
	if svcState != "running" {
		logger.Info("Kubelet service is not running, requeuing reconciliation", "name", tm.Name, "state", svcState)
		return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
	}
	// If the machine is ready, update the state to Available
	if tm.Status.State != talosv1alpha1.StateAvailable {
		if err := r.updateState(ctx, tm, talosv1alpha1.StateAvailable); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status to Available: %w", tm.Name, err)
		}
	}
	return ctrl.Result{}, nil
}
