package controller

import (
	"context"
	"fmt"
	"time"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *TalosControlPlaneReconciler) reconcileCloudMode(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TalosControlPlane in cloud mode", "name", tcp.Name)

	// Currently only GCP is supported
	if tcp.Spec.CloudSpec.GCP == nil {
		return ctrl.Result{}, fmt.Errorf("GCP spec is required for cloud mode")
	}
	// TODO: Switch case for other cloud providers when added
	// Reconcile GCP resources
	if err := r.reconcileGCPResources(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile GCP resources: %w", err)
	}

	// Generate the Talos ControlPlane config
	if err := r.GenerateConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate Talos ControlPlane config for %s: %w", tcp.Name, err)
	}

	// Check if ready
	ready, err := r.checkCloudModeReady(ctx, tcp)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if cloud mode is ready: %w", err)
	}

	if !ready {
		logger.Info("Cloud mode resources are not ready yet")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Bootstrap if needed
	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap cluster: %w", err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	if err := r.WriteTalosConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write talos config: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) GetLoadBalancerIP(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) ([]string, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting LoadBalancer IP for TalosControlPlane", "name", tcp.Name)

	// Currently only GCP is supported
	if tcp.Spec.CloudSpec.GCP == nil {
		return nil, fmt.Errorf("GCP spec is required for cloud mode")
	}

	// Get GCP LoadBalancer IP
	lbIPs, err := r.GetGCPLoadBalancerIP(ctx, tcp)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP LoadBalancer IP: %w", err)
	}

	return lbIPs, nil
}
