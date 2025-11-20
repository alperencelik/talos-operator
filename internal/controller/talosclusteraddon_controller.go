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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TalosClusterAddonReconciler reconciles a TalosClusterAddon object
type TalosClusterAddonReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddons,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddons/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddons/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosClusterAddon object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *TalosClusterAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var tcAddon talosv1alpha1.TalosClusterAddon
	if err := r.Get(ctx, req.NamespacedName, &tcAddon); err != nil {
		logger.Error(err, "unable to fetch TalosClusterAddon")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Finalizer
	var delErr error
	if tcAddon.DeletionTimestamp.IsZero() {
		delErr = r.handleFinalizer(ctx, tcAddon)
		if delErr != nil {
			logger.Error(delErr, "error while handling finalizer")
			r.Recorder.Event(&tcAddon, corev1.EventTypeWarning, "FinalizerFailed", delErr.Error())
			return ctrl.Result{}, fmt.Errorf("failed to handle finalizer: %w", delErr)
		}
	} else {
		// Handle deletion logic here
		if controllerutil.ContainsFinalizer(&tcAddon, talosv1alpha1.TalosControlPlaneFinalizer) {
			// Run delete operations
			var res ctrl.Result
			res, delErr = r.handleDelete(ctx, &tcAddon)
			if delErr != nil {
				logger.Error(delErr, "failed to handle delete for TalosControlPlane", "name", tcAddon.Name)
				r.Recorder.Event(&tcAddon, corev1.EventTypeWarning, "DeleteFailed", "Failed to handle delete for TalosControlPlane")

				return res, fmt.Errorf("failed to handle delete for TalosControlPlane %s: %w", tcAddon.Name, delErr)
			}
		}
		// Stop reconciliation as the object is being deleted
		return ctrl.Result{}, client.IgnoreNotFound(delErr)
	}
	// Handle reconciliation logic here
	logger.Info("Reconciling TalosClusterAddon", "name", tcAddon.Name)
	// 1. Find the matching clusters based on the ClusterSelector
	clusterList, err := r.listMatchingClusters(ctx, tcAddon)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list matching clusters: %w", err)
	}
	// 2. delete TalosClusterAddonRelease resources for clusters that no longer match
	// This part is optional and depends on whether you want to clean up releases for unmatched clusters
	err = r.cleanupUnmatchedReleases(ctx, tcAddon, clusterList)
	if err != nil {
		// TODO: Maybe cleanup should not fail the reconciliation
		return ctrl.Result{}, fmt.Errorf("failed to cleanup unmatched TalosClusterAddonRelease resources: %w", err)
	}
	// 3. For each matching cluster, ensure the TalosClusterAddonRelease resource is created/updated
	for _, cluster := range clusterList {
		// Implement the logic to create or update TalosClusterAddonRelease resources
		// Create the TalosClusterAddonRelease resource
		if !cluster.DeletionTimestamp.IsZero() {
			continue // Skip deleted clusters
		}
		// Create or update the TalosClusterAddonRelease resource
		err := r.createOrUpdateAddonRelease(ctx, tcAddon, cluster)
		if err != nil {
			logger.Error(err, "failed to create or update TalosClusterAddonRelease", "cluster", cluster.Name)
			r.Recorder.Event(&tcAddon, corev1.EventTypeWarning, "ReconcileFailed", fmt.Sprintf("Failed to create or update TalosClusterAddonRelease for cluster %s: %v", cluster.Name, err))
			return ctrl.Result{}, fmt.Errorf("failed to create or update TalosClusterAddonRelease for cluster %s: %w", cluster.Name, err)
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosClusterAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosClusterAddon{}).
		Named("talosclusteraddon").
		Complete(r)
}

func (r *TalosClusterAddonReconciler) handleFinalizer(ctx context.Context, tcAddon talosv1alpha1.TalosClusterAddon) error {
	if !controllerutil.ContainsFinalizer(&tcAddon, talosv1alpha1.TalosControlPlaneFinalizer) {
		controllerutil.AddFinalizer(&tcAddon, talosv1alpha1.TalosControlPlaneFinalizer)
		if err := r.Update(ctx, &tcAddon); err != nil {
			return fmt.Errorf("failed to add finalizer to TalosControlPlane %s: %w", tcAddon.Name, err)
		}
	}
	return nil
}

func (r *TalosClusterAddonReconciler) handleDelete(ctx context.Context, tcAddon *talosv1alpha1.TalosClusterAddon) (ctrl.Result, error) {
	// Perform any necessary cleanup operations here
	_, _ = ctx, tcAddon
	return ctrl.Result{}, nil
}

func (r *TalosClusterAddonReconciler) listMatchingClusters(ctx context.Context, tcAddon talosv1alpha1.TalosClusterAddon) ([]talosv1alpha1.TalosControlPlane, error) {
	logger := logf.FromContext(ctx)

	var clusterList talosv1alpha1.TalosControlPlaneList

	labelSelector, err := metav1.LabelSelectorAsSelector(&tcAddon.Spec.ClusterSelector)
	if err != nil {
		logger.Error(err, "failed to convert LabelSelector to Selector")
		return nil, fmt.Errorf("failed to convert LabelSelector to Selector: %w", err)
	}

	if err := r.List(ctx, &clusterList, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		logger.Error(err, "failed to list TalosControlPlanes")
		return nil, fmt.Errorf("failed to list TalosControlPlanes: %w", err)
	}

	return clusterList.Items, nil

}

func (r *TalosClusterAddonReconciler) createOrUpdateAddonRelease(ctx context.Context, tcAddon talosv1alpha1.TalosClusterAddon, cluster talosv1alpha1.TalosControlPlane) error {
	tcAddonRelease := &talosv1alpha1.TalosClusterAddonRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-addonrelease", cluster.Name, tcAddon.Name),
			Namespace: cluster.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, tcAddonRelease, func() error {
		// Set the spec fields based on the TalosClusterAddon and Cluster
		tcAddonRelease.Spec.HelmSpec = tcAddon.Spec.HelmSpec
		tcAddonRelease.Spec.ClusterRef = corev1.ObjectReference{
			Kind:       cluster.Kind,
			APIVersion: cluster.APIVersion,
			Name:       cluster.Name,
			Namespace:  cluster.Namespace,
		}
		return controllerutil.SetOwnerReference(&tcAddon, tcAddonRelease, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to create or update TalosClusterAddonRelease %s: %w", tcAddonRelease.Name, err)
	}
	return nil
}

func (r *TalosClusterAddonReconciler) cleanupUnmatchedReleases(ctx context.Context, tcAddon talosv1alpha1.TalosClusterAddon, matchedClusters []talosv1alpha1.TalosControlPlane) error {
	logger := logf.FromContext(ctx)

	var existingReleases talosv1alpha1.TalosClusterAddonReleaseList
	if err := r.List(ctx, &existingReleases, &client.ListOptions{
		Namespace: tcAddon.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list TalosClusterAddonReleases: %w", err)
	}
	for _, release := range existingReleases.Items {
		// Check if the release corresponds to a matched cluster
		clusterRefName := release.Spec.ClusterRef.Name
		if !isClusterInList(clusterRefName, matchedClusters) {
			// Delete the TalosClusterAddonRelease resource
			if err := r.Delete(ctx, &release); err != nil {
				logger.Error(err, "failed to delete TalosClusterAddonRelease", "name", release.Name)
				return fmt.Errorf("failed to delete TalosClusterAddonRelease %s: %w", release.Name, err)
			}
			logger.Info("deleted TalosClusterAddonRelease for unmatched cluster", "name", release.Name)
		}
	}
	return nil
}

func isClusterInList(clusterName string, clusterList []talosv1alpha1.TalosControlPlane) bool {
	for _, cluster := range clusterList {
		if cluster.Name == clusterName {
			return true
		}
	}
	return false
}
