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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TalosControlPlaneReconciler reconciles a TalosControlPlane object
type TalosControlPlaneReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	TalosImage = "ghcr.io/siderolabs/talos"

	// TalosContainer Vars
	TalosPlatformKey       = "PLATFORM"
	TalosPlatformContainer = "container"
)

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosControlPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TalosControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tcp talosv1alpha1.TalosControlPlane
	if err := r.Get(ctx, req.NamespacedName, &tcp); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}
	// Finalizer
	var delErr error
	if tcp.ObjectMeta.DeletionTimestamp.IsZero() {
		delErr = r.handleFinalizer(ctx, tcp)
		if delErr != nil {
			logger.Error(delErr, "failed to handle finalizer for TalosControlPlane", "name", tcp.Name)
			return ctrl.Result{}, fmt.Errorf("failed to handle finalizer for TalosControlPlane %s: %w", tcp.Name, delErr)
		}
	} else {
		// Handle deletion logic here
		if controllerutil.ContainsFinalizer(&tcp, talosv1alpha1.TalosControlPlaneFinalizer) {
			// Run delete operations
			var res ctrl.Result
			res, delErr = r.handleDelete(ctx, &tcp)
			if delErr != nil {
				logger.Error(delErr, "failed to handle delete for TalosControlPlane", "name", tcp.Name)
				return res, fmt.Errorf("failed to handle delete for TalosControlPlane %s: %w", tcp.Name, delErr)
			}
		}
		// Stop reconciliation as the object is being deleted
		return ctrl.Result{}, client.IgnoreNotFound(delErr)
	}

	// Get the mode of the TalosControlPlane
	var result ctrl.Result
	var err error
	switch tcp.Spec.Mode {
	case TalosModeContainer:
		result, err = r.reconcileContainerMode(ctx, &tcp)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile TalosControlPlane in container mode: %w", err)
		}
	case TalosModeMetal:
		result, err = r.reconcileMetalMode(ctx, &tcp)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile TalosControlPlane in metal mode: %w", err)
		}
	default:
		logger.Info("Unsupported mode for TalosControlPlane", "mode", tcp.Spec.Mode)
		return ctrl.Result{}, nil
	}
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: Interesting stuff, look further into this
	// I did implement since I had to use client.MatchingFields but look through the documentation to understand how it works
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&talosv1alpha1.TalosMachine{},
		// Index by the control plane reference name
		"spec.controlPlaneRef.name",
		func(rawObj client.Object) []string {
			tm := rawObj.(*talosv1alpha1.TalosMachine)
			if tm.Spec.ControlPlaneRef != nil {
				return []string{tm.Spec.ControlPlaneRef.Name}
			}
			return nil
		},
	); err != nil {
		return fmt.Errorf("failed to index TalosMachine by controlPlaneRef.name: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosControlPlane{}).
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(stsPredicate)).
		Owns(&corev1.Service{}, builder.WithPredicates(svcPredicate)).
		Owns(&talosv1alpha1.TalosMachine{}, builder.WithPredicates(talosMachinePredicate)).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Only reconcile if the generation of the object has changed
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
		}).
		Named("taloscontrolplane").
		Complete(r)
}

func (r *TalosControlPlaneReconciler) reconcileContainerMode(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling TalosControlPlane in container mode", "name", tcp.Name)
	// Generate the Talos ControlPlane config
	if err := r.GenerateConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate Talos ControlPlane config for %s: %w", tcp.Name, err)
	}
	// Get the object again since the status might have been updated
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		logger.Error(err, "Failed to get TalosControlPlane after generating config", "name", tcp.Name)
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	if err := r.reconcileService(ctx, tcp); err != nil {
		logger.Error(err, "Failed to reconcile Service for TalosControlPlane", "name", tcp.Name, "error", err)
		return ctrl.Result{}, nil
	}
	// Get the object again since the status might have been updated
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		logger.Error(err, "Failed to get TalosControlPlane after reconciling Service", "name", tcp.Name)
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Get the statefulset for the TalosControlPlane
	if err := r.reconcileStatefulSet(ctx, tcp); err != nil {
		logger.Error(err, "Failed to reconcile StatefulSet for TalosControlPlane", "name", tcp.Name, "error", err)
		return ctrl.Result{}, nil
	}

	// TODO: Implement something that waits for the StatefulSet to be ready before proceeding
	if err := r.CheckControlPlaneReady(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if TalosControlPlane %s is ready: %w", tcp.Name, err)
	}

	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap Talos ControlPlane cluster for %s: %w", tcp.Name, err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}

	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) reconcileMetalMode(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TalosControlPlane in metal mode", "name", tcp.Name)

	// Generate the Talos ControlPlane config
	if err := r.GenerateConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate Talos ControlPlane config for %s: %w", tcp.Name, err)
	}

	// Reconcile TalosMachine object
	if err := r.handleTalosMachines(ctx, tcp); err != nil {
		logger.Error(err, "Failed to reconcile TalosMachine objects", "name", tcp.Name)
		return ctrl.Result{}, err
	}

	// Wait for all TalosMachines to be created and status Available
	err := r.CheckControlPlaneReady(ctx, tcp)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if TalosControlPlane %s is ready: %w", tcp.Name, err)
	}
	// Send a bootstrap req to the TalosControlPlane
	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap Talos ControlPlane cluster for %s: %w", tcp.Name, err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}

	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) handleTalosMachines(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)
	// List existing ones
	existing := &talosv1alpha1.TalosMachineList{}
	if err := r.List(ctx, existing, client.InNamespace(tcp.Namespace),
		client.MatchingFields{"spec.controlPlaneRef.name": tcp.Name},
	); err != nil {
		return fmt.Errorf("failed to list TalosMachines: %w", err)
	}
	// Desired state
	desired := make(map[string]bool)
	for _, ep := range tcp.Spec.MetalSpec.Machines {
		desired[fmt.Sprintf("%s-%s", tcp.Name, ep)] = true
	}
	// Delete orphaned machines
	for _, m := range existing.Items {
		if m.Spec.ControlPlaneRef != nil && m.Spec.ControlPlaneRef.Name == tcp.Name {
			if !desired[m.Name] {
				if err := r.Delete(ctx, &m); err != nil && !kerrors.IsNotFound(err) {
					logger.Error(err, "Failed to delete orphaned TalosMachine", "name", m.Name)
					return fmt.Errorf("failed to delete orphaned TalosMachine %s: %w", m.Name, err)
				}
			}
		}
	}
	for _, machine := range tcp.Spec.MetalSpec.Machines {
		name := fmt.Sprintf("%s-%s", tcp.Name, machine)
		tm := &talosv1alpha1.TalosMachine{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tcp.Namespace},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, tm, func() error {
			if err := controllerutil.SetControllerReference(tcp, tm, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference for TalosMachine %s: %w", tm.Name, err)
			}
			tm.Spec = talosv1alpha1.TalosMachineSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					Kind:       talosv1alpha1.GroupKindControlPlane,
					Name:       tcp.Name,
					Namespace:  tcp.Namespace,
					APIVersion: talosv1alpha1.GroupVersion.String(),
				},
				Endpoint:    machine,
				InstallDisk: tcp.Spec.MetalSpec.InstallDisk,
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

func (r *TalosControlPlaneReconciler) CheckControlPlaneReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	// Check if all replicas of the StatefulSet are ready
	switch tcp.Spec.Mode {
	case "container":
		return r.checkContainerModeReady(ctx, tcp)
	case "metal":
		return r.checkMetalModeReady(ctx, tcp)
	default:
		return fmt.Errorf("unsupported mode for TalosControlPlane %s: %s", tcp.Name, tcp.Spec.Mode)
	}
}

//func (r *TalosControlPlaneReconciler) GetControlPlaneConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) ([]byte, error) {
//// MachineConfig is retrieved from the configMap
//cm := &corev1.ConfigMap{
//ObjectMeta: metav1.ObjectMeta{
//Name:      fmt.Sprintf("%s-config", tcp.Name),
//Namespace: tcp.Namespace,
//},
//}
//// Get the ConfigMap to retrieve the MachineConfig
//if err := r.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
//if kerrors.IsNotFound(err) {
//return nil, fmt.Errorf("ConfigMap %s for TalosControlPlane %s not found: %w", cm.Name, tcp.Name, err)
//}
//return nil, fmt.Errorf("failed to get ConfigMap %s for TalosControlPlane %s: %w", cm.Name, tcp.Name, err)
//}
//// Decode the MachineConfig from the ConfigMap data
//machineConfigData, exists := cm.Data["controlplane.yaml"]
//if !exists {
//return nil, fmt.Errorf("controlplane.yaml not found in ConfigMap %s for TalosControlPlane %s", cm.Name, tcp.Name)
//}
//// Decode the base64 encoded MachineConfig
//machineCfg, err := base64.StdEncoding.DecodeString(machineConfigData)
//if err != nil {
//return nil, fmt.Errorf("failed to decode controlplane.yaml from ConfigMap %s for TalosControlPlane %s: %w", cm.Name, tcp.Name, err)
//}
//return machineCfg, nil
//}

func (r *TalosControlPlaneReconciler) checkMetalModeReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {

	maxRetries := 10
	retryInterval := 5 * time.Second
	machines := &talosv1alpha1.TalosMachineList{}
	opts := []client.ListOption{
		client.InNamespace(tcp.Namespace),
		client.MatchingFields{"spec.controlPlaneRef.name": tcp.Name},
	}
	err := r.List(ctx, machines, opts...)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}
	for i := 0; i < maxRetries; i++ {
		// Get the machines associated with the TalosControlPlane
		allAvailable := true
		for _, machine := range machines.Items {
			if machine.Status.State != talosv1alpha1.StateAvailable {
				allAvailable = false
				log.FromContext(ctx).Info("Waiting for TalosMachine to be available", "machine", machine.Name, "state", machine.Status.State)
				break
			}
		}
		if allAvailable {
			// If the .state is Ready or Available don't try to update the status
			if tcp.Status.State == talosv1alpha1.StateReady || tcp.Status.State == talosv1alpha1.StateAvailable {
				return nil
			}
			// All machines are available, update the TalosControlPlane status to Available
			if err := r.updateState(ctx, tcp, talosv1alpha1.StateAvailable); err != nil {
				return fmt.Errorf("failed to update TalosControlPlane %s status to Available: %w", tcp.Name, err)
			}
			break // Exit the loop if all machines are available
		}
		time.Sleep(retryInterval)
	}
	// 		svcInfo, _ := talosClient.ServiceInfo(ctx, "kubelet")
	// if len(svcInfo) > 0 && svcInfo[0].Service.State == "running" {
	// return nil
	// }
	// // Wait and retry
	// time.Sleep(retryInterval)
	// if svcInfo[0].Service.Events != nil && len(svcInfo[0].Service.Events.Events) > 0 {
	// lastEvent = svcInfo[0].Service.Events.Events[len(svcInfo[0].Service.Events.Events)-1].Msg
	// }
	// }
	// // Update the Conditions with the kubelet's last event
	// if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
	// return fmt.Errorf("failed to get TalosControlPlane %s after checking kubelet service: %w", tcp.Name, err)
	// }
	// // Update the status condition to reflect the failure
	// condition := metav1.Condition{
	// Type:    "KubeletReady", // talosv1alpha1.ConditionKubeletReady,
	// Status:  metav1.ConditionFalse,
	// Reason:  "KubeletNotRunning",
	// Message: fmt.Sprintf("Kubelet is not healthy because of %s", lastEvent),
	// }
	// meta.SetStatusCondition(&tcp.Status.Conditions, condition)
	// if err := r.Status().Update(ctx, tcp); err != nil {
	// return fmt.Errorf("failed to update TalosControlPlane %s status with kubelet condition: %w", tcp.Name, err)
	// }
	return nil
}

func (r *TalosControlPlaneReconciler) checkContainerModeReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcp.Name,
			Namespace: tcp.Namespace,
		},
	}
	// Implement a retry mechanism to ensure the StatefulSet is ready
	maxRetries := 5
	retryInterval := 10 * time.Second
	for i := 0; i < maxRetries; i++ {
		// Get the StatefulSet to check its status
		if err := r.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
			//
		}
		// Check if the number of ready replicas matches the desired replicas
		if sts.Status.ReadyReplicas < tcp.Spec.Replicas {
			logger.Info("Waiting for all replicas to be ready", "readyReplicas", sts.Status.ReadyReplicas, "desiredReplicas", tcp.Spec.Replicas)
			time.Sleep(retryInterval)
			continue // Retry after waiting
		}
	}
	// When all replicas are ready, update the TalosControlPlane status to Available
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		return fmt.Errorf("failed to get TalosControlPlane %s after reconciling StatefulSet: %w", tcp.Name, err)
	}
	if err := r.updateState(ctx, tcp, talosv1alpha1.StateAvailable); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to Available: %w", tcp.Name, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) handleResourceNotFound(ctx context.Context, err error) error {
	logger := log.FromContext(ctx)
	if kerrors.IsNotFound(err) {
		logger.Info("TalosControlPlane resource not found. Ignoring since object must be deleted")
		return nil
	}
	return err
}

// reconcileService creates or updates a Service for a given replica index.
func (r *TalosControlPlaneReconciler) reconcileService(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	// Handle the services for each replica of the TalosControlPlane
	for i := int32(0); i < tcp.Spec.Replicas; i++ {
		// build the Service name
		svcName := fmt.Sprintf("%s-%d", tcp.Name, i)
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: tcp.Namespace,
			},
		}
		// set owner reference
		if err := controllerutil.SetControllerReference(tcp, svc, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for Service %s: %w", svcName, err)
		}

		// create or patch
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
			svc.Spec = BuildServiceSpec(tcp.Name, &i)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to create or update Service %s: %w", svcName, err)
		}
	}
	// Handle the control plane service which supposed to be exposed to the outside world
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcp.Name,
			Namespace: tcp.Namespace,
		},
	}
	// set owner reference
	if err := controllerutil.SetControllerReference(tcp, svc, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for Service %s: %w", tcp.Name, err)
	}
	// create or patch
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec = BuildServiceSpec(tcp.Name, nil) // No index for the control plane service
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update Service %s: %w", tcp.Name, err)
	}
	// TODO: Proper logging for the op
	return nil
}

func (r *TalosControlPlaneReconciler) reconcileStatefulSet(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	stsName := tcp.Name

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, sts, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for StatefulSet %s: %w", stsName, err)
	}

	extraEnvs := BuildUserDataEnvVar(tcp.Spec.ConfigRef, tcp.Name, TalosMachineTypeControlPlane)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Spec = BuildStsSpec(tcp.Name, tcp.Spec.Replicas, tcp.Spec.Version, TalosMachineTypeControlPlane, extraEnvs, tcp.Spec.StorageClassName)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update StatefulSet %s: %w", stsName, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) GenerateConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	bundleConfig, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	var patches *[]string
	// Generate the Talos ControlPlane config
	cpConfig, err := talos.GenerateControlPlaneConfig(bundleConfig, patches)
	if err != nil {
		return fmt.Errorf("failed to generate Talos ControlPlane config for %s: %w", tcp.Name, err)
	}
	bcBytes, err := json.Marshal(bundleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal BundleConfig for %s: %w", tcp.Name, err)
	}
	if tcp.Status.Config == string(*cpConfig) && tcp.Status.BundleConfig == string(bcBytes) {
		return nil // No changes in the config, skip update
	}
	tcp.Status.Config = string(*cpConfig)
	// store it in the status
	tcp.Status.BundleConfig = string(bcBytes)
	// Update the TalosControlPlane status with the config
	if err := r.Status().Update(ctx, tcp); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status with config: %w", tcp.Name, err)
	}
	// Write the Talos ControlPlane config to a ConfigMap
	err = r.WriteControlPlaneConfig(ctx, tcp, cpConfig)
	if err != nil {
		return fmt.Errorf("failed to write Talos ControlPlane config for %s: %w", tcp.Name, err)
	}
	// Write talosconfig to a Secret
	if err := r.WriteTalosConfig(ctx, tcp); err != nil {
		return fmt.Errorf("failed to write Talos config for %s: %w", tcp.Name, err)
	}
	// Update .status.state to Pending
	if err := r.updateState(ctx, tcp, talosv1alpha1.StatePending); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to Pending: %w", tcp.Name, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) WriteControlPlaneConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane, cpConfig *[]byte) error {
	// Set the configMap name and namespace
	cpConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", tcp.Name),
			Namespace: tcp.Namespace,
		},
	}
	// Set the ownerRef for the CM
	if err := controllerutil.SetControllerReference(tcp, cpConfigMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for ConfigMap %s: %w", cpConfigMap.Name, err)
	}
	// Create or update the ConfigMap
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cpConfigMap, func() error {
		cpConfigMap.Data = map[string]string{
			"controlplane.yaml": base64.StdEncoding.EncodeToString(*cpConfig),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update ConfigMap %s: %w", cpConfigMap.Name, err)
	}
	r.Recorder.Eventf(tcp, corev1.EventTypeNormal, "ConfigGenerated", "Generated Talos ControlPlane config for %s", tcp.Name)
	return nil
}

func (r *TalosControlPlaneReconciler) WriteTalosConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	// logger := log.FromContext(ctx)

	// Set the Talos ControlPlane config
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	bundle, err := talos.NewCPBundle(config, nil)
	if err != nil {
		return fmt.Errorf("failed to generate Talos ControlPlane bundle for %s: %w", tcp.Name, err)
	}
	// Generate the Talos config
	data, err := yaml.Marshal(talos.TalosConfig(bundle))
	if err != nil {
		return fmt.Errorf("failed to marshal Talos config for %s: %w", tcp.Name, err)
	}
	// Write the Talos config to a secret
	talosConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-talosconfig", tcp.Name),
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, talosConfigSecret, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for TalosConfig Secret %s: %w", talosConfigSecret.Name, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, talosConfigSecret, func() error {
		key := fmt.Sprintf("%s.talosconfig", tcp.Name)
		existing, exists := talosConfigSecret.Data[key]
		if exists && bytes.Equal(existing, data) {
			return nil // Skip update if content is identical
		}
		if talosConfigSecret.Data == nil {
			talosConfigSecret.Data = map[string][]byte{}
		}
		talosConfigSecret.Data[key] = data
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update TalosConfig Secret %s: %w", talosConfigSecret.Name, err)
	}
	r.Recorder.Eventf(tcp, corev1.EventTypeNormal, "TalosConfigWritten", "Wrote Talos config for %s", tcp.Name)
	return nil
}

func (r *TalosControlPlaneReconciler) BootstrapCluster(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)
	// Check if it's already bootstrapped
	if tcp.Status.State == talosv1alpha1.StateBootstrapped || tcp.Status.State == talosv1alpha1.StateReady {
		return nil
	}
	// Make sure that the .status.state is set to Available before bootstrapping
	// TODO: Implement properly
	if tcp.Status.State != talosv1alpha1.StateAvailable {
		return fmt.Errorf("TalosControlPlane %s is not in Available to bootstrap, current state: %s", tcp.Name, tcp.Status.State)
	}
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// If the mode is metal tweak the config to use the metal-specific endpoint to bootstrap
	if tcp.Spec.Mode == "metal" {
		// Use the first machine's endpoint for bootstrapping
		var newEndpoint string
		newEndpoint = tcp.Spec.MetalSpec.Machines[0]   // Use the first machine's endpoint
		config.Endpoint = newEndpoint                  // Set the Endpoint to the first machine's endpoint
		config.ClientEndpoint = &[]string{newEndpoint} // Set the ClientEndpoint to the same as Endpoint
	}

	// Create a Talos client
	talosClient, err := talos.NewClient(config, false)
	if err != nil {
		return fmt.Errorf("failed to create Talos client for ControlPlane %s: %w", tcp.Name, err)
	}
	//  Bootstrap the Talos node
	if err := talosClient.BootstrapNode(config); err != nil {
		return fmt.Errorf("failed to bootstrap Talos node for ControlPlane %s: %w", tcp.Name, err)
	}
	// Get the object again since the status might have been updated
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		logger.Error(err, "Failed to get TalosControlPlane after generating config", "name", tcp.Name)
		return fmt.Errorf("failed to get TalosControlPlane %s after bootstrapping: %w", tcp.Name, err)
	}
	// Update state as Bootstrapped
	if err := r.updateState(ctx, tcp, talosv1alpha1.StateBootstrapped); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to Bootstrapped: %w", tcp.Name, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) WriteKubeconfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	//
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	talosClient, err := talos.NewClient(config, false)
	if err != nil {
		return fmt.Errorf("failed to create Talos client for ControlPlane %s: %w", tcp.Name, err)
	}
	// Generate the kubeconfig for the Talos ControlPlane
	kubeconfig, err := talosClient.Kubeconfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// Write the kubeconfig to a secret
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", tcp.Name),
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, kubeconfigSecret, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for Kubeconfig Secret %s: %w", kubeconfigSecret.Name, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, kubeconfigSecret, func() error {
		kubeconfigSecret.Data = map[string][]byte{
			"kubeconfig": []byte(kubeconfig),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update Kubeconfig Secret %s: %w", kubeconfigSecret.Name, err)
	}
	// Get the TalosControlPlane object again to update the status
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		return fmt.Errorf("failed to get TalosControlPlane %s after writing kubeconfig: %w", tcp.Name, err)
	}

	if err := r.updateState(ctx, tcp, talosv1alpha1.StateReady); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to Ready: %w", tcp.Name, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) SetConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*talos.BundleConfig, error) {
	// Genenrate the Subject Alternative Names (SANs) for the Talos ControlPlane
	var replicas int
	var sans []string
	if tcp.Spec.Mode == "container" {
		replicas = int(tcp.Spec.Replicas)
		sans = utils.GenSans(tcp.Name, &replicas)
	}
	// Get the latest TalosControlPlane object
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("TalosControlPlane %s not found: %w", tcp.Name, err)
		}
		return nil, fmt.Errorf("failed to get TalosControlPlane %s: %w", tcp.Name, err)
	}
	// Get secret bundle
	secretBundle, err := r.SecretBundle(ctx, tcp)
	if err != nil {
		return nil, fmt.Errorf("failed to get SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
	}
	var ClientEndpoint []string
	if tcp.Spec.Mode == "metal" {
		// TODO: Handle multiple machines in metal mode
		ClientEndpoint = tcp.Spec.MetalSpec.Machines
	}
	var endpoint string
	// Construct endpoint
	if tcp.Spec.Endpoint != "" {
		endpoint = tcp.Spec.Endpoint
	} else {
		// Default endpoint is the TalosControlPlane name
		endpoint = fmt.Sprintf("https://%s:6443", tcp.Name)
	}

	// Generate the Talos ControlPlane config
	return &talos.BundleConfig{
		ClusterName:    tcp.Name,
		Endpoint:       endpoint,
		Version:        tcp.Spec.Version,
		KubeVersion:    tcp.Spec.KubeVersion,
		SecretsBundle:  talos.SecretBundle(*secretBundle),
		Sans:           sans,
		ServiceCIDR:    &tcp.Spec.ServiceCIDR,
		PodCIDR:        &tcp.Spec.PodCIDR,
		ClientEndpoint: &ClientEndpoint,
	}, nil
}

func (r *TalosControlPlaneReconciler) SecretBundle(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*talos.SecretBundle, error) {
	logger := log.FromContext(ctx)
	var secretBundle talos.SecretBundle
	var err error
	// Get the secret bundle for the TalosControlPlane from .status.SecretBundle
	if tcp.Status.SecretBundle == "" {
		logger.Info("SecretBundle is nil, generating new one")
		secretBundle, err = talos.NewSecretBundle()
		if err != nil {
			return nil, fmt.Errorf("failed to create new SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
		}
		// Update the TalosControlPlane status with the new SecretBundle
		secretBundleBytes, err := yaml.Marshal(secretBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
		}
		// Get the object again since the status might have been updated
		if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
			return nil, fmt.Errorf("failed to get TalosControlPlane %s after generating SecretBundle: %w", tcp.Name, err)
		}

		// Converts bytes to a string and sets it in the status
		tcp.Status.SecretBundle = string(secretBundleBytes)
		if err := r.Status().Update(ctx, tcp); err != nil {
			return nil, fmt.Errorf("failed to update TalosControlPlane %s status with SecretBundle: %w", tcp.Name, err)
		}
	} else {
		// Get the existing SecretBundle from the status
		// logger.Info("Using existing SecretBundle from status")
		secretBundle, err = utils.SecretBundleDecoder(tcp.Status.SecretBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to decode SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
		}
	}
	// DEBUG: SET Clock forcefully -- investigate later
	secretBundle.Clock = talos.NewClock()

	return &secretBundle, nil
}

func (r *TalosControlPlaneReconciler) handleFinalizer(ctx context.Context, tcp talosv1alpha1.TalosControlPlane) error {
	if !controllerutil.ContainsFinalizer(&tcp, talosv1alpha1.TalosControlPlaneFinalizer) {
		controllerutil.AddFinalizer(&tcp, talosv1alpha1.TalosControlPlaneFinalizer)
		if err := r.Update(ctx, &tcp); err != nil {
			return fmt.Errorf("failed to add finalizer to TalosControlPlane %s: %w", tcp.Name, err)
		}
	}
	return nil
}

func (r *TalosControlPlaneReconciler) handleDelete(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deleting TalosControlPlane", "name", tcp.Name)

	// Update the conditions to mark the TalosControlPlane as being deleted
	if !meta.IsStatusConditionPresentAndEqual(tcp.Status.Conditions, talosv1alpha1.ConditionDeleting, metav1.ConditionUnknown) {
		meta.SetStatusCondition(&tcp.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionDeleting,
			Status:  metav1.ConditionUnknown,
			Reason:  "Deleting",
			Message: "Deleting TalosControlPlane",
		})
		if err := r.Status().Update(ctx, tcp); err != nil {
			logger.Error(err, "Error updating VirtualMachine status")
			return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
		}
	}
	// Based on the TalosControlPlane mode, handle the deletion
	switch tcp.Spec.Mode {
	case TalosModeContainer:
		// In container mode, we need to delete the StatefulSet and Services
		res, err := r.handleContainerModeDelete(ctx, tcp)
		if err != nil {
			logger.Error(err, "Failed to handle container mode delete for TalosControlPlane", "name", tcp.Name)
			return res, fmt.Errorf("failed to handle container mode delete for TalosControlPlane %s: %w", tcp.Name, err)
		}
	case TalosModeMetal:
		// In metal mode, we need to delete the TalosMachines
		machines := &talosv1alpha1.TalosMachineList{}
		opts := []client.ListOption{
			client.InNamespace(tcp.Namespace),
			client.MatchingFields{"spec.controlPlaneRef.name": tcp.Name},
		}
		if err := r.List(ctx, machines, opts...); err != nil {
			logger.Error(err, "Failed to list TalosMachines for TalosControlPlane", "name", tcp.Name)
			return ctrl.Result{}, fmt.Errorf("failed to list TalosMachines for TalosControlPlane %s: %w", tcp.Name, err)
		}
		// Delete each TalosMachine associated with the TalosControlPlane
		for _, machine := range machines.Items {
			if err := r.Delete(ctx, &machine); err != nil && !kerrors.IsNotFound(err) {
				logger.Error(err, "Failed to delete TalosMachine", "name", machine.Name)
				return ctrl.Result{}, fmt.Errorf("failed to delete TalosMachine %s for TalosControlPlane %s: %w", machine.Name, tcp.Name, err)
			}
		}
		// Wait for the TalosMachines to be deleted
		remaining := &talosv1alpha1.TalosMachineList{}
		if err := r.List(ctx, remaining, opts...); err != nil {
			logger.Error(err, "Failed to list remaining TalosMachines for TalosControlPlane", "name", tcp.Name)
			return ctrl.Result{}, fmt.Errorf("failed to list remaining TalosMachines for TalosControlPlane %s: %w", tcp.Name, err)
		}
		if len(remaining.Items) > 0 {
			return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
		}
	default:
		logger.Info("Unsupported mode for TalosControlPlane during deletion, finalizer will be removed", "mode", tcp.Spec.Mode)
	}
	// Remove the finalizer from the TalosControlPlane
	controllerutil.RemoveFinalizer(tcp, talosv1alpha1.TalosControlPlaneFinalizer)
	if err := r.Update(ctx, tcp); err != nil {
		logger.Error(err, "Failed to remove finalizer from TalosControlPlane", "name", tcp.Name)
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer from TalosControlPlane %s: %w", tcp.Name, err)
	}
	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) handleContainerModeDelete(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {

	// sts := &appsv1.StatefulSet{
	// ObjectMeta: metav1.ObjectMeta{
	// Name:      tcp.Name,
	// Namespace: tcp.Namespace,
	// },
	// }
	// if err := r.Delete(ctx, sts); err != nil && !kerrors.IsNotFound(err) {
	// logger.Error(err, "Failed to delete StatefulSet for TalosControlPlane", "name", tcp.Name)
	// return ctrl.Result{}, fmt.Errorf("failed to delete StatefulSet for TalosControlPlane %s: %w", tcp.Name, err)
	// }
	// // Delete the Services associated with the TalosControlPlane
	// for i := int32(0); i < tcp.Spec.Replicas; i++ {
	// svcName := fmt.Sprintf("%s-%d", tcp.Name, i)
	// svc := &corev1.Service{
	// ObjectMeta: metav1.ObjectMeta{
	// Name:      svcName,
	// Namespace: tcp.Namespace,
	// },
	// }
	// if err := r.Delete(ctx, svc); err != nil && !kerrors.IsNotFound(err) {
	// logger.Error(err, "Failed to delete Service for TalosControlPlane", "name", svcName)
	// return ctrl.Result{}, fmt.Errorf("failed to delete Service %s for TalosControlPlane %s: %w", svcName, tcp.Name, err)
	// }
	// }
	// // Delete the control plane Service
	// controlPlaneSvc := &corev1.Service{
	// ObjectMeta: metav1.ObjectMeta{
	// Name:      tcp.Name,
	// Namespace: tcp.Namespace,
	// },
	// }
	// if err := r.Delete(ctx, controlPlaneSvc); err != nil && !kerrors.IsNotFound(err) {
	// logger.Error(err, "Failed to delete control plane Service for TalosControlPlane", "name", tcp.Name)
	// return ctrl.Result{}, fmt.Errorf("failed to delete control plane Service for TalosControlPlane %s: %w", tcp.Name, err)
	// }
	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) updateState(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane, state string) error {
	if tcp.Status.State == state {
		return nil
	}
	tcp.Status.State = state
	if err := r.Status().Update(ctx, tcp); err != nil {
		return fmt.Errorf("failed to update TalosControlPlane %s status to %s: %w", tcp.Name, state, err)

	}
	return nil
}
