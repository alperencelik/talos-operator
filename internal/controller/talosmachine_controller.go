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
	"os"
	"strings"
	"time"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/talos"
	"github.com/alperencelik/talos-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"
)

// TalosMachineReconciler reconciles a TalosMachine object
type TalosMachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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
				r.Recorder.Event(&talosMachine, corev1.EventTypeWarning, "DeleteFailed", "Failed to handle delete for TalosMachine")
				return res, err
			}
			// Remove the finalizer
			controllerutil.RemoveFinalizer(&talosMachine, talosv1alpha1.TalosMachineFinalizer)
			if err := r.Update(ctx, &talosMachine); err != nil {
				logger.Error(err, "Failed to remove finalizer for TalosMachine", "name", talosMachine.Name)
				r.Recorder.Event(&talosMachine, corev1.EventTypeWarning, "FinalizerRemoveFailed", "Failed to remove finalizer for TalosMachine")
				return ctrl.Result{}, err
			}
		}
		// Stop the reconciliation if the finalizer is not present
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}
	// Get the reconcile mode from the annotation
	reconcileMode := r.getReconciliationMode(ctx, &talosMachine)
	switch reconcileMode {
	case ReconcileModeDisable:
		logger.Info("Reconciliation is disabled for this TalosWorker", "name", talosMachine.Name, "namespace", talosMachine.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeDryRun:
		logger.Info("Dry run mode is not implemented yet, skipping reconciliation", "name", talosMachine.Name, "namespace", talosMachine.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeNormal:
		// Do nothing, proceed with reconciliation
	}
	// Check whether we should wait for machine to be ready
	if talosMachine.Status.State == talosv1alpha1.StateInstalling || talosMachine.Status.State == talosv1alpha1.StateUpgrading {
		// If the machine is in the installing state, we should wait for it to be ready
		res, err := r.CheckMachineReady(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Error checking machine readiness", "name", talosMachine.Name)
			return ctrl.Result{}, err
		}
		if res.Requeue {
			logger.Info("Requeuing reconciliation to check machine readiness", "name", talosMachine.Name)
			r.Recorder.Event(&talosMachine, corev1.EventTypeNormal, "Requeuing", "Requeuing reconciliation to check machine readiness")
			return res, nil // Requeue the reconciliation to check the machine status again
		}
	}
	// Re-get the machine object to ensure we have the latest state
	if err := r.Get(ctx, req.NamespacedName, &talosMachine); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Check if feature flag for meta key is enabled and handle it
	if os.Getenv("ENABLE_META_KEY") == "true" {
		// Handle the meta key if there is any entry to pass
		err := r.handleMetaKey(ctx, &talosMachine)
		if err != nil {
			logger.Error(err, "Failed to handle meta key for TalosMachine", "name", talosMachine.Name)
			r.Recorder.Event(&talosMachine, corev1.EventTypeWarning, "MetaKeyFailed", "Failed to handle meta key for TalosMachine")
			return ctrl.Result{}, err
		}
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
	r.Recorder.Event(tm, corev1.EventTypeNormal, "Reconciling", "Handling control plane machine")
	// Get the bundle config from TalosControlPlane
	bc, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		logger.Error(err, "Failed to get BundleConfig for TalosMachine", "name", tm.Name)
		r.Recorder.Event(tm, corev1.EventTypeWarning, "BundleConfigFailed", "Failed to get BundleConfig for TalosMachine")
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	if bc == nil {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tm.Name)
		r.Recorder.Event(tm, corev1.EventTypeNormal, "BundleConfigNotSet", "TalosControlPlane bundleConfig is not set, waiting for it to be ready")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}

	var cpConfig *[]byte
	// If the TalosMachine has a configRef, get the config from there. Else generate the config from the bundleConfig
	if tm.Spec.ConfigRef != nil {
		// Get the config from the ConfigMap
		data, err := r.GetConfigMapData(ctx, tm)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get configRef for TalosMachine %s: %w", tm.Name, err)
		}
		cpConfig = utils.StringToBytePtr(strings.TrimSpace(*data))
	} else {
		// Apply patches to config before applying it
		patches, err := r.metalConfigPatches(ctx, tm, bc)
		if err != nil {
			r.Recorder.Event(tm, corev1.EventTypeWarning, "MetalConfigPatchFailed", "Failed to get metal config patches for TalosMachine")
			return ctrl.Result{}, fmt.Errorf("failed to get metal config patches for TalosMachine %s: %w", tm.Name, err)
		}
		cpConfig, err = talos.GenerateControlPlaneConfig(bc, patches)
		if err != nil {
			r.Recorder.Event(tm, corev1.EventTypeWarning, "ConfigGenerationFailed", "Failed to generate Control Plane config for TalosMachine")
			return ctrl.Result{}, fmt.Errorf("failed to generate Control Plane config for TalosMachine %s: %w", tm.Name, err)
		}
		if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.ImageCache {
			*cpConfig = append(*cpConfig, []byte(talos.ImageCacheVolumeConfig)...)
		}
		// If there is additionalConfig, merge it with the generated config
		if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.AdditionalConfig != nil {
			// Add --- delimeter and then the additional config
			*cpConfig = append(*cpConfig, []byte("\n---\n")...)
			*cpConfig = append(*cpConfig, tm.Spec.MachineSpec.AdditionalConfig.Raw...)
		}
	}
	// Check if the current config is the same as the one in status
	if tm.Status.Config == string(*cpConfig) && tm.Status.ObservedVersion == tm.Spec.Version {
		// Return since the machine is in desired state
		return ctrl.Result{}, nil
	}
	err = r.UpgradeOrApplyConfig(ctx, tm, bc, cpConfig)
	if err != nil {
		logger.Error(err, "Failed to apply or upgrade Talos config for TalosMachine", "name", tm.Name)
		r.Recorder.Event(tm, corev1.EventTypeWarning, "ConfigApplyFailed", "Failed to apply or upgrade Talos config for TalosMachine")
		return ctrl.Result{}, fmt.Errorf("failed to apply or upgrade Talos config for TalosMachine %s: %w", tm.Name, err)
	}
	// TODO: Review here to make it more event driven -- maybe implement watcher, etc.
	return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check the machine status again
}

// TODO: Fix this one
func (r *TalosMachineReconciler) handleWorkerMachine(ctx context.Context, tm *talosv1alpha1.TalosMachine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	r.Recorder.Event(tm, corev1.EventTypeNormal, "Reconciling", "Handling worker machine")
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
		r.Recorder.Event(tm, corev1.EventTypeWarning, "BundleConfigFailed", "Failed to get BundleConfig for TalosMachine")
		return ctrl.Result{}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	if bc == nil {
		logger.Info("TalosControlPlane bundleConfig is not set, waiting for it to be ready", "name", tm.Name)
		r.Recorder.Event(tm, corev1.EventTypeNormal, "BundleConfigNotSet", "TalosControlPlane bundleConfig is not set, waiting for it to be ready")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Requeue after 30 seconds to check again
	}

	var workerConfig *[]byte

	// If the TalosMachine has a configRef, get the config from there. Else generate the config from the bundleConfig
	if tm.Spec.ConfigRef != nil {
		// Get the config from the ConfigMap
		data, err := r.GetConfigMapData(ctx, tm)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get configRef for TalosMachine %s: %w", tm.Name, err)
		}
		workerConfig = utils.StringToBytePtr(strings.TrimSpace(*data))
	} else {
		// Apply patches to config before applying it
		patches, err := r.metalConfigPatches(ctx, tm, bc)
		if err != nil {
			r.Recorder.Event(tm, corev1.EventTypeWarning, "MetalConfigPatchFailed", "Failed to get metal config patches for TalosMachine")
			return ctrl.Result{}, fmt.Errorf("failed to get metal config patches for TalosMachine %s: %w", tm.Name, err)
		}
		// Generate the worker config
		workerConfig, err = talos.GenerateWorkerConfig(bc, patches)
		if err != nil {
			r.Recorder.Event(tm, corev1.EventTypeWarning, "ConfigGenerationFailed", "Failed to generate Worker config for TalosMachine")
			return ctrl.Result{}, fmt.Errorf("failed to generate Worker config for TalosMachine %s: %w", tm.Name, err)
		}
		if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.ImageCache {
			*workerConfig = append(*workerConfig, []byte(talos.ImageCacheVolumeConfig)...)
		}
	}

	// Check if the current config is the same as the one in status
	if tm.Status.Config == string(*workerConfig) && tm.Status.ObservedVersion == tm.Spec.Version {
		// Return since the machine is in desired state
		return ctrl.Result{}, nil
	}
	err = r.UpgradeOrApplyConfig(ctx, tm, bc, workerConfig)
	if err != nil {
		logger.Error(err, "Failed to apply or upgrade Talos config for TalosMachine", "name", tm.Name)
		r.Recorder.Event(tm, corev1.EventTypeWarning, "ConfigApplyFailed", "Failed to apply or upgrade Talos config for TalosMachine")
		return ctrl.Result{}, fmt.Errorf("failed to apply or upgrade Talos config for TalosMachine %s: %w", tm.Name, err)
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
	// If the TalosControlPlane reference is nil, the machine is orphaned
	if tcp == nil {
		logger.Info("TalosControlPlane reference is nil, waiting for it to be ready", "name", tm.Name)
		// Update the staus to Orphaned and don't reconcile
		if err := r.updateState(ctx, tm, talosv1alpha1.StateOrphaned); err != nil {
			return nil, fmt.Errorf("failed to update TalosMachine %s status to Orphaned: %w", tm.Name, err)
		}
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
	// If machine is orphaned, we don't need to do anything
	if tm.Status.State == talosv1alpha1.StateOrphaned {
		return ctrl.Result{}, nil
	}
	// Run talosctl reset command to reset the machine
	config, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	// Make the client for the machine
	config.ClientEndpoint = &[]string{tm.Spec.Endpoint}
	tc, err := talos.NewClient(config, false)
	if err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
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
	// Disk Patches
	diskNamePtr, err := talosclient.GetInstallDisk(ctx, tm)
	if err != nil {
		return nil, fmt.Errorf("failed to get install disk for TalosMachine %s: %w", tm.Name, err)
	}
	diskName := utils.PtrToString(diskNamePtr)
	diskPatch := fmt.Sprintf(talos.InstallDisk, diskName)

	// Wipe Disk Patch
	var wipeDiskPatch string
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.Wipe {
		wipeDiskPatch = fmt.Sprintf(talos.WipeDisk, tm.Spec.MachineSpec.Wipe)
	}

	// Install Image Patch
	var imagePatch string
	// If the .machineSpec.image is set, use it
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.Image != nil && *tm.Spec.MachineSpec.Image != "" {
		// if the .machineSpec.image has version suffix, directly use it if not append the version to the image
		var imageWithVersion string
		if utils.HasVersionSuffix(*tm.Spec.MachineSpec.Image) {
			imageWithVersion = *tm.Spec.MachineSpec.Image
		} else {
			imageWithVersion = fmt.Sprintf("%s:%s", *tm.Spec.MachineSpec.Image, config.Version)
		}
		imagePatch = fmt.Sprintf(talos.InstallImage, imageWithVersion)
	} else {
		// If the .machineSpec.image is not set, use the default image from the version
		defaultImageWithVersion := fmt.Sprintf("%s:%s", talos.DefaultTalosImage, config.Version)
		imagePatch = fmt.Sprintf(talos.InstallImage, defaultImageWithVersion)
	}
	// patches
	var patches []string
	patches = append(patches, diskPatch)
	patches = append(patches, wipeDiskPatch)
	patches = append(patches, imagePatch)
	// Air gapped patch
	var airGappedPatch string
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.AirGap {
		airGappedPatch = talos.AirGapp
		patches = append(patches, airGappedPatch)
	}

	var imageCachePatch string
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.ImageCache {
		imageCachePatch = talos.ImageCache
		patches = append(patches, imageCachePatch)
	}

	var allowSchedulingPatch string
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.AllowSchedulingOnControlPlanes {
		allowSchedulingPatch = talos.AllowSchedulingOnControlPlanes
		patches = append(patches, allowSchedulingPatch)
	}

	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.Registries != nil {
		jsonBytes, err := yaml.YAMLToJSON(tm.Spec.MachineSpec.Registries.Raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert registries to json: %w", err)
		}
		path := "/machine/registries"
		patch := fmt.Sprintf(`[{"op": "add", "path": "%s", "value": %s}]`, path, string(jsonBytes))
		patches = append(patches, patch)
	}

	return &patches, nil
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
	svcState := tc.GetServiceStatus(ctx, talos.KUBELET_SERVICE_NAME)
	if svcState == "" {
		logger.Info("Kubelet service state is empty, requeuing reconciliation", "name", tm.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
	}
	if svcState != talos.KUBELET_STATUS_RUNNING {
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

func (r *TalosMachineReconciler) UpgradeOrApplyConfig(ctx context.Context, tm *talosv1alpha1.TalosMachine, bc *talos.BundleConfig, config *[]byte) error {
	// Check whether we need to construct maintenance mode or not
	insecure := tm.Status.State == "" || tm.Status.State == talosv1alpha1.StatePending
	// Create Talos client
	tc, err := talos.NewClient(bc, insecure) // true for insecure TLS
	if err != nil {
		return fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	configDrift := tm.Status.Config != string(*config)
	applyConfigurationFunc := func() error {
		if err := tc.ApplyConfig(ctx, *config); err != nil {
			return fmt.Errorf("failed to apply Talos config for TalosMachine %s: %w", tm.Name, err)
		}
		// Prepare a merge patch to update only our status fields
		orig := tm.DeepCopy()
		tm.Status.Config = string(*config)
		tm.Status.ObservedVersion = tm.Spec.Version
		if tm.Status.State != talosv1alpha1.StateInstalling {
			tm.Status.State = talosv1alpha1.StateInstalling
		}
		if err := r.Status().Patch(ctx, tm, client.MergeFrom(orig)); err != nil {
			return fmt.Errorf("failed to patch TalosMachine %s status with config: %w", tm.Name, err)
		}
		return nil
	}
	// If insecure we can only apply the config, otherwise we can upgrade the Talos version
	// I think if it's insecure I don't need to check whether config drift or not, I can just apply the config
	if insecure {
		return applyConfigurationFunc()
	}
	// If not insecure then we can check the Talos version and upgrade if necessary
	// Get current Talos version
	actualVersion, err := tc.GetTalosVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Talos version for TalosMachine %s: %w", tm.Name, err)
	}
	// Make sure that actual version complies with the version format: vX.Y.Z
	if !utils.IsValidTalosVersion(actualVersion) {
		return fmt.Errorf("invalid Talos version format for TalosMachine %s: %s", tm.Name, actualVersion)
	}

	// If the version is the same, we can apply the config
	if actualVersion == tm.Spec.Version {
		if configDrift {
			// Apply the config
			return applyConfigurationFunc()
		}
	} else {
		// If the version is different, we need to upgrade
		image := fmt.Sprintf("ghcr.io/siderolabs/installer:%s", tm.Spec.Version)
		if err := tc.UpgradeTalosVersion(ctx, image); err != nil {
			return fmt.Errorf("failed to upgrade Talos version for TalosMachine %s: %w", tm.Name, err)
		}
		// Update it to Upgrading state
		orig := tm.DeepCopy()
		tm.Status.ObservedVersion = tm.Spec.Version
		if tm.Status.State != talosv1alpha1.StateUpgrading {
			tm.Status.State = talosv1alpha1.StateUpgrading
		}
		if err := r.Status().Patch(ctx, tm, client.MergeFrom(orig)); err != nil {
			return fmt.Errorf("failed to patch TalosMachine %s status with config: %w", tm.Name, err)
		}
	}
	return nil
}

func (r *TalosMachineReconciler) getReconciliationMode(ctx context.Context, tm *talosv1alpha1.TalosMachine) string {
	logger := log.FromContext(ctx)
	// Check if the annotation exists
	mode, exists := tm.Annotations[ReconcileModeAnnotation]
	if !exists {
		return ReconcileModeNormal
	}
	switch strings.ToLower(mode) {
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

func (r *TalosMachineReconciler) handleMetaKey(ctx context.Context, tm *talosv1alpha1.TalosMachine) error {
	if tm.Spec.MachineSpec == nil {
		return nil // No machine spec, early return
	} else {
		if tm.Spec.MachineSpec.Meta == nil {
			return nil // No meta key is set
		}
	}
	bc, err := r.GetBundleConfig(ctx, tm)
	if err != nil {
		return fmt.Errorf("failed to get BundleConfig for TalosMachine %s: %w", tm.Name, err)
	}
	// Check whether we need to construct maintenance mode or not
	insecure := tm.Status.State == "" || tm.Status.State == talosv1alpha1.StatePending
	// Create Talos client
	tc, err := talos.NewClient(bc, insecure) // true for insecure TLS
	if err != nil {
		return fmt.Errorf("failed to create Talos client for TalosMachine %s: %w", tm.Name, err)
	}
	// Apply the meta key
	if err := tc.ApplyMetaKey(ctx, tm.Spec.Endpoint, tm.Spec.MachineSpec.Meta); err != nil {
		return fmt.Errorf("failed to apply meta key for TalosMachine %s: %w", tm.Name, err)
	}

	return nil
}

func (r *TalosMachineReconciler) GetConfigMapData(ctx context.Context, tm *talosv1alpha1.TalosMachine) (*string, error) {
	if tm.Spec.ConfigRef == nil {
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tm.Spec.ConfigRef.Name,
			Namespace: tm.Namespace,
		}, cm); err != nil {
			return nil, r.handleResourceNotFound(ctx, err)
		}
		data, ok := cm.Data[tm.Spec.ConfigRef.Key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in ConfigMap %s for TalosMachine %s", tm.Spec.ConfigRef.Key, tm.Spec.ConfigRef.Name, tm.Name)
		}
		return &data, nil
	}
	return nil, nil
}
