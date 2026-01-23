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

	"strings"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
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

// TalosControlPlaneReconciler reconciles a TalosControlPlane object
type TalosControlPlaneReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	TalosImage = "ghcr.io/siderolabs/talos"
)

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=taloscontrolplanes/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TalosControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tcp talosv1alpha1.TalosControlPlane
	if err := r.Get(ctx, req.NamespacedName, &tcp); err != nil {
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Initialize ObservedKubeVersion if it's empty
	if tcp.Status.ObservedKubeVersion == "" && tcp.Spec.KubeVersion != "" {
		tcp.Status.ObservedKubeVersion = tcp.Spec.KubeVersion
		if err := r.Status().Update(ctx, &tcp); err != nil {
			logger.Error(err, "failed to initialize ObservedKubeVersion")
			r.Recorder.Event(&tcp, corev1.EventTypeWarning, "StatusUpdateFailed", "Failed to initialize ObservedKubeVersion")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	// Finalizer
	var delErr error
	if tcp.DeletionTimestamp.IsZero() {
		delErr = r.handleFinalizer(ctx, tcp)
		if delErr != nil {
			logger.Error(delErr, "failed to handle finalizer for TalosControlPlane", "name", tcp.Name)
			r.Recorder.Event(&tcp, corev1.EventTypeWarning, "FinalizerFailed", "Failed to handle finalizer for TalosControlPlane")
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
				r.Recorder.Event(&tcp, corev1.EventTypeWarning, "DeleteFailed", "Failed to handle delete for TalosControlPlane")
				return res, fmt.Errorf("failed to handle delete for TalosControlPlane %s: %w", tcp.Name, delErr)
			}
		}
		// Stop reconciliation as the object is being deleted
		return ctrl.Result{}, client.IgnoreNotFound(delErr)
	}

	// Get the reconcile mode from the annotation
	reconcileMode := r.getReconciliationMode(ctx, &tcp)
	switch strings.ToLower(reconcileMode) {
	case ReconcileModeDisable:
		logger.Info("Reconciliation is disabled for this TalosControlplane", "name", tcp.Name, "namespace", tcp.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeDryRun:
		logger.Info("Dry run mode is not implemented yet, skipping reconciliation", "name", tcp.Name, "namespace", tcp.Namespace)
		return ctrl.Result{}, nil
	case ReconcileModeImport:
		// If user would like to import existing TalosControlPlane run special logic and then continue normal reconciliation by setting Imported to true
		if tcp.Status.Imported == nil || !*tcp.Status.Imported {
			return r.ImportExistingTalosControlPlane(ctx, &tcp)
		}
	case ReconcileModeNormal:
		// Do nothing, proceed with reconciliation
	}

	// Get the mode of the TalosControlPlane
	var result ctrl.Result
	var err error
	switch tcp.Spec.Mode {
	case TalosModeContainer:
		r.Recorder.Event(&tcp, corev1.EventTypeNormal, "Reconciling", "Reconciling TalosControlPlane in container mode")
		result, err = r.reconcileContainerMode(ctx, &tcp)
		if err != nil {
			return result, fmt.Errorf("failed to reconcile TalosControlPlane in container mode: %w", err)
		}
	case TalosModeMetal:
		r.Recorder.Event(&tcp, corev1.EventTypeNormal, "Reconciling", "Reconciling TalosControlPlane in metal mode")
		result, err = r.reconcileMetalMode(ctx, &tcp)
		if err != nil {
			return result, fmt.Errorf("failed to reconcile TalosControlPlane in metal mode: %w", err)
		}
	default:
		logger.Info("Unsupported mode for TalosControlPlane", "mode", tcp.Spec.Mode)
		return ctrl.Result{}, nil
	}

	// KubeVersion reconciliation
	result, err = r.reconcileKubeVersion(ctx, &tcp)
	if err != nil {
		logger.Error(err, "failed to reconcile kube version", "name", tcp.Name, "namespace", tcp.Namespace)
		r.Recorder.Event(&tcp, corev1.EventTypeWarning, "KubeVersionReconciliationFailed", "Failed to reconcile kube version")
		return result, err
	}
	return result, nil
}

// reconcileKubeVersion reconciles the KubeVersion of the TalosControlPlane.
func (r *TalosControlPlaneReconciler) reconcileKubeVersion(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if tcp.Spec.KubeVersion == "" || tcp.Spec.KubeVersion == tcp.Status.ObservedKubeVersion {
		return ctrl.Result{}, nil
	}

	logger.Info("KubeVersion changed, starting upgrade", "old", tcp.Status.ObservedKubeVersion, "new", tcp.Spec.KubeVersion)
	r.Recorder.Event(tcp, corev1.EventTypeNormal, "KubeVersionUpgrade", fmt.Sprintf("KubeVersion changed from %s to %s, starting upgrade", tcp.Status.ObservedKubeVersion, tcp.Spec.KubeVersion))

	// Define the jobName
	var jobNamePrefix string
	if tcp.GetOwnerReferences() != nil && len(tcp.GetOwnerReferences()) > 0 {
		jobNamePrefix = tcp.ObjectMeta.OwnerReferences[0].Name
	} else {
		jobNamePrefix = tcp.Name
	}
	jobName := fmt.Sprintf("%s-upgrade-%s", jobNamePrefix, tcp.Spec.KubeVersion)
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: tcp.Namespace}, job)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Job does not exist, create it.
			logger.Info("creating upgrade job", "job", jobName)
			job = &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: tcp.Namespace,
				},
			}
			_, err := controllerutil.CreateOrUpdate(ctx, r.Client, job, func() error {
				image := utils.GetEnv("TALOS_OPERATOR_IMAGE", "alperencelik/talos-operator:latest")
				serviceAccount := utils.GetEnv("TALOS_OPERATOR_SERVICE_ACCOUNT", "talos-operator")
				job.Spec = BuildK8sUpgradeJobSpec(tcp, image, serviceAccount)
				return controllerutil.SetControllerReference(tcp, job, r.Scheme)
			})

			if err != nil {
				logger.Error(err, "failed to create or update upgrade job")
				r.Recorder.Event(tcp, corev1.EventTypeWarning, "UpgradeJobFailed", "Failed to create or update upgrade job")
				return ctrl.Result{Requeue: true}, err
			}
			tcp.Status.State = talosv1alpha1.StateUpgradingKubernetes
			if err := r.Status().Update(ctx, tcp); err != nil {
				logger.Error(err, "failed to update TalosControlPlane status after creating upgrade job")
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if job.Status.Failed > 0 {
		logger.Error(nil, "upgrade job failed")
		r.Recorder.Event(tcp, corev1.EventTypeWarning, "UpgradeJobFailed", "Upgrade job failed")
		tcp.Status.State = talosv1alpha1.StateKubernetesUpgradeFailed
		if err := r.Status().Update(ctx, tcp); err != nil {
			logger.Error(err, "failed to update TalosControlPlane status after upgrade job failed")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if job.Status.Succeeded > 0 {
		logger.Info("upgrade job succeeded", "job", jobName)
	} else {
		logger.Info("upgrade job is running", "job", jobName)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		// TODO: Look into this, for some reason it doesn't trigger reconciliation when the job is updated
		// Owns(&batchv1.Job{}, builder.WithPredicates(jobPredicate)).
		Owns(&batchv1.Job{}, builder.WithPredicates(jobPredicate)).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Only reconcile if the generation of the object has changed
				oldTcp, ok1 := e.ObjectOld.(*talosv1alpha1.TalosControlPlane)
				newTcp, ok2 := e.ObjectNew.(*talosv1alpha1.TalosControlPlane)
				if !ok1 || !ok2 {
					return false
				}
				// Check if the generation has changed
				condition1 := oldTcp.GetGeneration() != newTcp.GetGeneration()
				// Check if the observed kubeVersion has changed
				condition2 := oldTcp.Status.ObservedKubeVersion != newTcp.Status.ObservedKubeVersion
				return condition1 || condition2
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
		return ctrl.Result{Requeue: true}, nil
	}
	// Get the object again since the status might have been updated
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		logger.Error(err, "Failed to get TalosControlPlane after reconciling Service", "name", tcp.Name)
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Get the statefulset for the TalosControlPlane
	result, err := r.reconcileStatefulSet(ctx, tcp)
	if err != nil {
		logger.Error(err, "Failed to reconcile StatefulSet for TalosControlPlane", "name", tcp.Name, "error", err)
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to reconcile StatefulSet for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// If the previous op was create then requeue so that we refresh the cache and check the status
	if result.Requeue {
		return result, nil
	}
	if _, err := r.CheckControlPlaneReady(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if TalosControlPlane %s is ready: %w", tcp.Name, err)
	}

	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap Talos ControlPlane cluster for %s: %w", tcp.Name, err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}

	if err := r.WriteTalosConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write Talos config for %s: %w", tcp.Name, err)
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
		return ctrl.Result{Requeue: true}, err
	}

	// Wait for all TalosMachines to be created and status Available
	for {
		ready, err := r.CheckControlPlaneReady(ctx, tcp)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check if TalosControlPlane %s is ready: %w", tcp.Name, err)
		}
		if ready {
			break
		}
		time.Sleep(10 * time.Second)
	}

	// Send a bootstrap req to the TalosControlPlane
	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap Talos ControlPlane cluster for %s: %w", tcp.Name, err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}

	if err := r.WriteTalosConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write Talos config for %s: %w", tcp.Name, err)
	}

	return ctrl.Result{}, nil
}

func (r *TalosControlPlaneReconciler) handleTalosMachines(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)

	machineIPAddresses, err := getMachinesIPAddresses(ctx, r.Client, &tcp.Spec.MetalSpec.Machines)
	if err != nil {
		return fmt.Errorf("failed to get machine IP addresses for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// List existing ones
	existing := &talosv1alpha1.TalosMachineList{}
	if err := r.List(ctx, existing, client.InNamespace(tcp.Namespace),
		client.MatchingFields{"spec.controlPlaneRef.name": tcp.Name},
	); err != nil {
		return fmt.Errorf("failed to list TalosMachines: %w", err)
	}
	// Desired state
	desired := make(map[string]bool)
	for _, ip := range machineIPAddresses {
		desired[fmt.Sprintf("%s-%s", tcp.Name, ip)] = true
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
	// Create or update TalosMachines
	for _, ip := range machineIPAddresses {
		name := fmt.Sprintf("%s-%s", tcp.Name, ip)
		// If the talosContolPlane is imported then add an annotation to the TalosMachine to indicate that it's imported
		var annotations map[string]string
		if tcp.Status.Imported != nil && *tcp.Status.Imported {
			annotations = map[string]string{
				ReconcileModeAnnotation: ReconcileModeImport,
			}
		} else {
			annotations = nil
		}
		tm := &talosv1alpha1.TalosMachine{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tcp.Namespace, Annotations: annotations},
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
				Endpoint:    ip,
				Version:     tcp.Spec.Version,
				MachineSpec: tcp.Spec.MetalSpec.MachineSpec,
				ConfigRef:   tcp.Spec.ConfigRef,
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

func (r *TalosControlPlaneReconciler) CheckControlPlaneReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (bool, error) {
	// Check if all replicas of the StatefulSet are ready
	switch tcp.Spec.Mode {
	case TalosModeContainer:
		return r.checkContainerModeReady(ctx, tcp)
	case TalosModeMetal:
		return r.checkMetalModeReady(ctx, tcp)
	default:
		return false, fmt.Errorf("unsupported mode for TalosControlPlane %s: %s", tcp.Name, tcp.Spec.Mode)
	}
}

func (r *TalosControlPlaneReconciler) checkMetalModeReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (bool, error) {

	maxRetries := 10
	retryInterval := 10 * time.Second
	machines := &talosv1alpha1.TalosMachineList{}
	opts := []client.ListOption{
		client.InNamespace(tcp.Namespace),
		client.MatchingFields{"spec.controlPlaneRef.name": tcp.Name},
	}
	var allavailable bool
	for i := 0; i < maxRetries; i++ {
		// Need to re-list machines everytime because the update is not reflected in the cache immediately
		// Get the machines associated with the TalosControlPlane
		err := r.List(ctx, machines, opts...)
		if err != nil {
			return false, fmt.Errorf("error: %w", err)
		}
		// Remove the deletion timestamped machines from the list
		for i := len(machines.Items) - 1; i >= 0; i-- {
			if !machines.Items[i].DeletionTimestamp.IsZero() {
				// Keep only the machines that are not being deleted
				machines.Items = append(machines.Items[:i], machines.Items[i+1:]...)
			}
		}
		// Check if all machines are available
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
				return allAvailable, nil
			}
			// All machines are available, update the TalosControlPlane status to Available
			// If it's not ready or available, update the status to Available
			if tcp.Status.State != talosv1alpha1.StateReady {
				// Get the object once again before update it
				if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
					return allAvailable, fmt.Errorf("failed to get TalosControlPlane %s after checking machines: %w", tcp.Name, err)
				}
				// Update the status to Available
				if err := r.updateState(ctx, tcp, talosv1alpha1.StateAvailable); err != nil {
					return allAvailable, fmt.Errorf("failed to update TalosControlPlane %s status to Available: %w", tcp.Name, err)
				}
			}
			return allAvailable, nil
		}
		time.Sleep(retryInterval)
	}
	return allavailable, nil
}

func (r *TalosControlPlaneReconciler) checkContainerModeReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (bool, error) {
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
			return false, fmt.Errorf("failed to get StatefulSet %s: %w", sts.Name, err)
		}
		// Check if the number of ready replicas matches the desired replicas
		if sts.Status.ReadyReplicas < tcp.Spec.Replicas {
			logger.Info("Waiting for all replicas to be ready", "readyReplicas", sts.Status.ReadyReplicas, "desiredReplicas", tcp.Spec.Replicas)
			time.Sleep(retryInterval)
			continue // Retry after waiting
		}
	}
	// When all replicas are ready, update the TalosControlPlane status to ready
	if err := r.Get(ctx, client.ObjectKeyFromObject(tcp), tcp); err != nil {
		return false, fmt.Errorf("failed to get TalosControlPlane %s after reconciling StatefulSet: %w", tcp.Name, err)
	}
	if err := r.updateState(ctx, tcp, talosv1alpha1.StateReady); err != nil {
		return false, fmt.Errorf("failed to update TalosControlPlane %s status to Available: %w", tcp.Name, err)
	}
	return true, nil
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
	return nil
}

func (r *TalosControlPlaneReconciler) reconcileStatefulSet(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	stsName := tcp.Name

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, sts, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to set controller reference for StatefulSet %s: %w", stsName, err)
	}

	extraEnvs := BuildUserDataEnvVar(tcp.Spec.ConfigRef, tcp.Name, TalosMachineTypeControlPlane)

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Spec = BuildStsSpec(tcp.Name, tcp.Spec.Replicas, tcp.Spec.Version, TalosMachineTypeControlPlane, extraEnvs, tcp.Spec.StorageClassName)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create or update StatefulSet %s: %w", stsName, err)
	}
	if op == controllerutil.OperationResultCreated {
		// If the StatefulSet was created, we need to requeue to check the status
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
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
	// If the state is empty update it to Pending
	if tcp.Status.State == "" {
		// Update .status.state to Pending
		if err := r.updateState(ctx, tcp, talosv1alpha1.StatePending); err != nil {
			return fmt.Errorf("failed to update TalosControlPlane %s status to Pending: %w", tcp.Name, err)
		}
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
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cpConfigMap, func() error {
		cpConfigMap.Data = map[string]string{
			"controlplane.yaml": base64.StdEncoding.EncodeToString(*cpConfig),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update ConfigMap %s: %w", cpConfigMap.Name, err)
	}
	switch op {
	case controllerutil.OperationResultCreated:
		r.Recorder.Eventf(tcp, corev1.EventTypeNormal, "ConfigMapCreated", "Created ConfigMap %s for TalosControlPlane %s", cpConfigMap.Name, tcp.Name)
	case controllerutil.OperationResultUpdated:
		r.Recorder.Eventf(tcp, corev1.EventTypeNormal, "ConfigMapUpdated", "Updated ConfigMap %s for TalosControlPlane %s", cpConfigMap.Name, tcp.Name)
	}
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
	// If the controlplane is owned by a TalosCluster use it's name
	var secretName string
	if tcp.GetOwnerReferences() != nil && len(tcp.GetOwnerReferences()) > 0 {
		secretName = tcp.ObjectMeta.OwnerReferences[0].Name
	} else {
		// Fallback to the TalosControlPlane name
		secretName = tcp.Name
	}
	// Write the Talos config to a secret
	talosConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-talosconfig", secretName),
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
	return nil
}

func (r *TalosControlPlaneReconciler) BootstrapCluster(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)
	// Check if it's already bootstrapped
	if tcp.Status.State == talosv1alpha1.StateBootstrapped || tcp.Status.State == talosv1alpha1.StateReady {
		return nil
	}
	// Make sure that the .status.state is set to Available before bootstrapping
	if tcp.Status.State != talosv1alpha1.StateAvailable {
		return fmt.Errorf("TalosControlPlane %s is not in Available to bootstrap, current state: %s", tcp.Name, tcp.Status.State)
	}
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// If the mode is metal tweak the config to use the metal-specific endpoint to bootstrap
	if tcp.Spec.Mode == TalosModeMetal {
		// Use the first machine's endpoint for bootstrapping
		ip, err := getMachineIPAddress(ctx, r.Client, &tcp.Spec.MetalSpec.Machines[0])
		if err != nil {
			return fmt.Errorf("failed to get machine IP address for bootstrapping TalosControlPlane %s: %w", tcp.Name, err)
		}
		config.Endpoint = *ip
		config.ClientEndpoint = &[]string{*ip}
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
	// If it's installing skip writing the kubeconfig
	if tcp.Status.State == talosv1alpha1.StateInstalling {
		return nil
	}
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
	// If the controlplane is owned by a TalosCluster use it's name
	var secretName string
	if tcp.GetOwnerReferences() != nil && len(tcp.GetOwnerReferences()) > 0 {
		secretName = tcp.ObjectMeta.OwnerReferences[0].Name
	} else {
		// Fallback to the TalosControlPlane name
		secretName = tcp.Name
	}
	// Write the kubeconfig to a secret
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", secretName),
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, kubeconfigSecret, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for Kubeconfig Secret %s: %w", kubeconfigSecret.Name, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, kubeconfigSecret, func() error {
		kubeconfigSecret.Data = map[string][]byte{
			"kubeconfig": kubeconfig,
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
	logger := log.FromContext(ctx)
	// Genenrate the Subject Alternative Names (SANs) for the Talos ControlPlane
	var replicas int
	var sans []string
	if tcp.Spec.Mode == TalosModeContainer {
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
		ipAddresses, err := getMachinesIPAddresses(ctx, r.Client, &tcp.Spec.MetalSpec.Machines)
		if err != nil {
			logger.Error(err, "Failed to get machine IP addresses for TalosControlPlane", "name", tcp.Name)
			return nil, fmt.Errorf("failed to get machine IP addresses for TalosControlPlane %s: %w", tcp.Name, err)
		}
		ClientEndpoint = ipAddresses
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
		KubeVersion:    tcp.Status.ObservedKubeVersion,
		SecretsBundle:  *secretBundle,
		Sans:           sans,
		ServiceCIDR:    &tcp.Spec.ServiceCIDR,
		PodCIDR:        &tcp.Spec.PodCIDR,
		ClientEndpoint: &ClientEndpoint,
		CNI:            tcp.Spec.CNI,
	}, nil
}

func (r *TalosControlPlaneReconciler) SecretBundle(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*talos.SecretBundle, error) {
	logger := log.FromContext(ctx)
	var secretBundle talos.SecretBundle
	var err error
	// Get the secret bundle for the TalosControlPlane from .status.SecretBundle
	if tcp.Status.SecretBundle == "" {
		// Check if the configRef is set
		if tcp.Spec.ConfigRef != nil {
			// Get the config from the ConfigMap
			data, err := r.GetConfigMapData(ctx, tcp)
			if err != nil {
				return nil, fmt.Errorf("failed to get configRef for TalosControlPlane %s: %w", tcp.Name, err)
			}
			secretBundle, err = talos.GetSecretBundleFromConfig(ctx, []byte(*data))
			if err != nil {
				return nil, fmt.Errorf("failed to get SecretBundle from configRef for TalosControlPlane %s: %w", tcp.Name, err)
			}
		} else {
			logger.Info("SecretBundle is nil, generating new one")
			secretBundle, err = talos.NewSecretBundle()
			if err != nil {
				return nil, fmt.Errorf("failed to create new SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
			}
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
			logger.Error(err, "Error updating TalosControlPlane status")
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

func (r *TalosControlPlaneReconciler) getReconciliationMode(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) string {
	logger := log.FromContext(ctx)
	// Check if the annotation exists
	mode, exists := tcp.Annotations[ReconcileModeAnnotation]
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
	case ReconcileModeImport:
		logger.Info("Reconciliation mode is set to Import")
		return ReconcileModeImport
	default:
		logger.Info("Unknown reconciliation mode, defaulting to Normal")
		return ReconcileModeNormal
	}
}

func (r *TalosControlPlaneReconciler) GetConfigMapData(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*string, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tcp.Spec.ConfigRef.Name,
		Namespace: tcp.Namespace,
	}, cm); err != nil {
		return nil, r.handleResourceNotFound(ctx, err)
	}
	data, ok := cm.Data[tcp.Spec.ConfigRef.Key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in ConfigMap %s for TalosMachine %s", tcp.Spec.ConfigRef.Key, tcp.Spec.ConfigRef.Name, tcp.Name)
	}
	return &data, nil
}

func (r *TalosControlPlaneReconciler) ImportExistingTalosControlPlane(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Importing existing TalosControlPlane", "name", tcp.Name)
	// The container mode is not supported for import
	if tcp.Spec.Mode == TalosModeContainer {
		return ctrl.Result{}, fmt.Errorf("importing existing TalosControlPlane in container mode is not supported")
	}
	// ConfigRef must be set for import
	if tcp.Spec.ConfigRef == nil {
		return ctrl.Result{}, fmt.Errorf("configRef must be set to import existing TalosControlPlane %s", tcp.Name)
	}
	// Generate the Talos ControlPlane config
	if err := r.GenerateConfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// Set bootstrapped state since it's an existing cluster
	if err := r.updateState(ctx, tcp, talosv1alpha1.StateBootstrapped); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosControlPlane %s status to Bootstrapped: %w", tcp.Name, err)
	}
	// Update the imported condition with True status
	tcp.Status.Imported = ptr.To(true)
	if err := r.Status().Update(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosControlPlane %s status to Imported: %w", tcp.Name, err)
	}
	logger.Info("Successfully imported existing TalosControlPlane", "name", tcp.Name)
	// Fire an event
	r.Recorder.Eventf(tcp, corev1.EventTypeNormal, "Imported", "Imported existing TalosControlPlane %s", tcp.Name)
	// Requeue so that the normal reconciliation can proceed after import
	return ctrl.Result{Requeue: true}, nil
}
