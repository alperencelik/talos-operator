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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/helm"
)

// TalosAddonReconciler reconciles a TalosAddon object
type TalosAddonReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosaddons,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosaddons/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosaddons/finalizers,verbs=update
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TalosAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var addon talosv1alpha1.TalosAddon
	if err := r.Get(ctx, req.NamespacedName, &addon); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Finalizer logic
	if addon.DeletionTimestamp.IsZero() {
		// Add finalizer for this CR
		err := r.handleFinalizer(ctx, &addon)
		if err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&addon, talosv1alpha1.TalosAddonFinalizer) {
			err := r.handleDelete(ctx, &addon)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to handle delete: %w", err)
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(&addon, talosv1alpha1.TalosAddonFinalizer)
			if err := r.Update(ctx, &addon); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling TalosAddon", "TalosAddon", req.NamespacedName)

	// Check if already installed with matching generation
	if addon.Status.ObservedGeneration == addon.Generation &&
		meta.IsStatusConditionTrue(addon.Status.Conditions, talosv1alpha1.ConditionReady) {
		logger.Info("Addon is already installed and up to date. Skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Get the TalosCluster
	cluster := &talosv1alpha1.TalosCluster{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      addon.Spec.ClusterRef.Name,
		Namespace: addon.Namespace,
	}, cluster); err != nil {
		logger.Error(err, "Failed to get TalosCluster")
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "ClusterNotFound",
			Message: fmt.Sprintf("Failed to get TalosCluster: %v", err),
		})
		addon.Status.State = talosv1alpha1.StateFailed
		if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Set progressing condition
	meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionProgressing,
		Status:  metav1.ConditionTrue,
		Reason:  "InstallingAddon",
		Message: "Installing Helm chart addon",
	})
	addon.Status.State = talosv1alpha1.StateInstalling
	if err := r.Status().Update(ctx, &addon); err != nil {
		logger.Error(err, "Failed to update status to progressing")
	}

	// Get kubeconfig for the cluster
	kubeconfigSecret := &corev1.Secret{}
	kubeconfigSecretName := fmt.Sprintf("%s-kubeconfig", cluster.Name)
	if err := r.Get(ctx, types.NamespacedName{
		Name:      kubeconfigSecretName,
		Namespace: cluster.Namespace,
	}, kubeconfigSecret); err != nil {
		logger.Error(err, "Failed to get kubeconfig secret")
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "KubeconfigNotFound",
			Message: fmt.Sprintf("Failed to get kubeconfig secret: %v", err),
		})
		addon.Status.State = talosv1alpha1.StateFailed
		if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	kubeconfig := kubeconfigSecret.Data["kubeconfig"]
	if len(kubeconfig) == 0 {
		err := fmt.Errorf("kubeconfig data is empty")
		logger.Error(err, "Invalid kubeconfig secret")
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "InvalidKubeconfig",
			Message: "Kubeconfig data is empty",
		})
		addon.Status.State = talosv1alpha1.StateFailed
		if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Collect values from references
	values := make(map[string]interface{})
	for k, v := range addon.Spec.HelmRelease.Values {
		values[k] = v
	}

	for _, ref := range addon.Spec.HelmRelease.ValuesFrom {
		refValues, err := r.getValuesFromReference(ctx, addon.Namespace, ref)
		if err != nil {
			logger.Error(err, "Failed to get values from reference", "kind", ref.Kind, "name", ref.Name)
			meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
				Type:    talosv1alpha1.ConditionFailed,
				Status:  metav1.ConditionTrue,
				Reason:  "ValuesReferenceError",
				Message: fmt.Sprintf("Failed to get values from %s/%s: %v", ref.Kind, ref.Name, err),
			})
			addon.Status.State = talosv1alpha1.StateFailed
			if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
				logger.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{}, err
		}
		for k, v := range refValues {
			values[k] = v
		}
	}

	// Install or upgrade the Helm chart
	helmClient, err := helm.NewClient(kubeconfig, addon.Spec.HelmRelease.TargetNamespace)
	if err != nil {
		logger.Error(err, "Failed to create Helm client")
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "HelmClientError",
			Message: fmt.Sprintf("Failed to create Helm client: %v", err),
		})
		addon.Status.State = talosv1alpha1.StateFailed
		if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	releaseName := addon.Spec.HelmRelease.ReleaseName
	if releaseName == "" {
		releaseName = addon.Name
	}

	revision, err := helmClient.InstallOrUpgrade(
		releaseName,
		addon.Spec.HelmRelease.RepoURL,
		addon.Spec.HelmRelease.ChartName,
		addon.Spec.HelmRelease.Version,
		values,
	)
	if err != nil {
		logger.Error(err, "Failed to install or upgrade Helm chart")
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    talosv1alpha1.ConditionFailed,
			Status:  metav1.ConditionTrue,
			Reason:  "HelmInstallFailed",
			Message: fmt.Sprintf("Failed to install/upgrade Helm chart: %v", err),
		})
		addon.Status.State = talosv1alpha1.StateFailed
		if statusErr := r.Status().Update(ctx, &addon); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Set ready condition
	meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
		Type:    talosv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "AddonInstalled",
		Message: "Helm chart addon installed successfully",
	})
	addon.Status.State = talosv1alpha1.StateReady
	addon.Status.LastAppliedRevision = fmt.Sprintf("%d", revision)
	addon.Status.ObservedGeneration = addon.Generation

	if err := r.Status().Update(ctx, &addon); err != nil {
		logger.Error(err, "Failed to update status to ready")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully installed or upgraded addon", "TalosAddon", req.NamespacedName)
	return ctrl.Result{}, nil
}

// handleFinalizer adds the finalizer to the resource
func (r *TalosAddonReconciler) handleFinalizer(ctx context.Context, addon *talosv1alpha1.TalosAddon) error {
	if !controllerutil.ContainsFinalizer(addon, talosv1alpha1.TalosAddonFinalizer) {
		controllerutil.AddFinalizer(addon, talosv1alpha1.TalosAddonFinalizer)
		if err := r.Update(ctx, addon); err != nil {
			return err
		}
	}
	return nil
}

// handleDelete handles the deletion of the addon
func (r *TalosAddonReconciler) handleDelete(ctx context.Context, addon *talosv1alpha1.TalosAddon) error {
	logger := logf.FromContext(ctx)
	logger.Info("Handling deletion of TalosAddon", "name", addon.Name)

	// Get the TalosCluster
	cluster := &talosv1alpha1.TalosCluster{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      addon.Spec.ClusterRef.Name,
		Namespace: addon.Namespace,
	}, cluster); err != nil {
		// If cluster is not found, we can skip uninstalling
		logger.Info("TalosCluster not found, skipping Helm uninstall", "cluster", addon.Spec.ClusterRef.Name)
		return nil
	}

	// Get kubeconfig for the cluster
	kubeconfigSecret := &corev1.Secret{}
	kubeconfigSecretName := fmt.Sprintf("%s-kubeconfig", cluster.Name)
	if err := r.Get(ctx, types.NamespacedName{
		Name:      kubeconfigSecretName,
		Namespace: cluster.Namespace,
	}, kubeconfigSecret); err != nil {
		// If kubeconfig is not found, we can skip uninstalling
		logger.Info("Kubeconfig secret not found, skipping Helm uninstall", "secret", kubeconfigSecretName)
		return nil
	}

	kubeconfig := kubeconfigSecret.Data["kubeconfig"]
	if len(kubeconfig) == 0 {
		logger.Info("Kubeconfig data is empty, skipping Helm uninstall")
		return nil
	}

	// Uninstall the Helm release
	helmClient, err := helm.NewClient(kubeconfig, addon.Spec.HelmRelease.TargetNamespace)
	if err != nil {
		logger.Error(err, "Failed to create Helm client for deletion")
		// Continue with deletion even if we can't create the client
		return nil
	}

	releaseName := addon.Spec.HelmRelease.ReleaseName
	if releaseName == "" {
		releaseName = addon.Name
	}

	if err := helmClient.Uninstall(releaseName); err != nil {
		logger.Error(err, "Failed to uninstall Helm release", "release", releaseName)
		// We don't return error here to allow the resource to be deleted
	} else {
		logger.Info("Successfully uninstalled Helm release", "release", releaseName)
	}

	return nil
}

// getValuesFromReference retrieves values from a ConfigMap or Secret
func (r *TalosAddonReconciler) getValuesFromReference(ctx context.Context, namespace string, ref talosv1alpha1.ValueReference) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	switch ref.Kind {
	case "ConfigMap":
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: namespace,
		}, cm); err != nil {
			return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
		}

		if ref.Key != "" {
			if val, ok := cm.Data[ref.Key]; ok {
				values[ref.Key] = val
			}
		} else {
			for k, v := range cm.Data {
				values[k] = v
			}
		}

	case "Secret":
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: namespace,
		}, secret); err != nil {
			return nil, fmt.Errorf("failed to get Secret: %w", err)
		}

		if ref.Key != "" {
			if val, ok := secret.Data[ref.Key]; ok {
				values[ref.Key] = string(val)
			}
		} else {
			for k, v := range secret.Data {
				values[k] = string(v)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported reference kind: %s", ref.Kind)
	}

	return values, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosAddon{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
