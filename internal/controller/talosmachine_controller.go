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
	"sigs.k8s.io/controller-runtime/pkg/log"

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
		logger.Error(err, "unable to fetch TalosMachine")
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	logger.Info("Reconciling TalosMachine", "name", talosMachine.Name, "namespace", talosMachine.Namespace)

	// Check for the machine type and handle accordingly
	switch {
	case talosMachine.Spec.ControlPlaneRef != nil:
		logger.Info("Processing TalosMachine as Control Plane", "name", talosMachine.Name)
		// Handle control plane specific logic here
		res, err := r.handleControlPlaneMachine(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error handling Control Plane machine", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		return res, nil
	case talosMachine.Spec.WorkerRef != nil:
		logger.Info("Processing TalosMachine as Worker", "name", talosMachine.Name)
		// Handle control plane specific logic here
		res, err := r.handleWorkerMachine(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error handling Control Plane machine", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		return res, nil
	default:
		logger.Info("TalosMachine is neither Control Plane nor Worker", "name", talosMachine.Name)
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosMachine{}).
		Named("talosmachine").
		Complete(r)
}

func (r *TalosMachineReconciler) handleControlPlaneMachine(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Get config from ControlPlaneRef
	tcp := &talosv1alpha1.TalosControlPlane{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tm.Spec.ControlPlaneRef.Name,
		Namespace: tm.Namespace,
	}, tcp); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	if tcp.Status.Config == "" {
		logger.Info("TalosControlPlane config is not set, waiting for it to be ready", "name", tcp.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}
	// Get bundleConfig from TalosControlPlane status
	bcString := tcp.Status.BundleConfig
	if bcString == "" {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tcp.Name)
		return ctrl.Result{}, nil
	}
	// Parse the bundleConfig
	bc, err := talos.ParseBundleConfig(bcString)
	if err != nil {
		logger.Error(err, "Failed to parse Talos bundle config", "name", tcp.Name)
		return ctrl.Result{}, fmt.Errorf("failed to parse Talos bundle config for Control Plane %s: %w", tcp.Name, err)
	}
	secretBundle, err := utils.SecretBundleDecoder(tcp.Status.SecretBundle)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to decode secret bundle for Control Plane %s: %w", tcp.Name, err)
	}
	secretBundle.Clock = talos.NewClock()
	bc.SecretsBundle = secretBundle
	var insecure bool
	if tm.Status.State == talosv1alpha1.StatePending || tm.Status.State == "" {
		insecure = true // If the machine is pending, we allow insecure TLS
	} else {
		insecure = false // Otherwise, we use secure TLS
	}
	tc, err := talos.NewClient(bc, insecure) // true for insecure TLS
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Talos client for Control Plane %s: %w", tcp.Name, err)
	}

	err = tc.ApplyConfig(ctx, []byte(tcp.Status.Config))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply Talos config for Control Plane %s: %w", tcp.Name, err)
	}
	// Update state as Available
	if err := r.updateState(ctx, tm, talosv1alpha1.StateAvailable); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosMachine %s status to Available: %w", tm.Name, err)
	}
	//
	return ctrl.Result{}, nil
}

func (r *TalosMachineReconciler) handleWorkerMachine(ctx context.Context, machine *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Get config from WorkerRef
	tw := &talosv1alpha1.TalosWorker{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      machine.Spec.WorkerRef.Name,
		Namespace: machine.Namespace,
	}, tw); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	if tw.Status.Config == "" {
		logger.Info("TalosWorker config is not set, waiting for it to be ready", "name", tw.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}
	// TODO: Implement applyconfig structure here
	return ctrl.Result{}, nil
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
