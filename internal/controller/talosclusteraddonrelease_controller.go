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
	"github.com/alperencelik/talos-operator/pkg/helm"
)

// TalosClusterAddonReleaseReconciler reconciles a TalosClusterAddonRelease object
type TalosClusterAddonReleaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddonreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddonreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=talos.alperen.cloud,resources=talosclusteraddonreleases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TalosClusterAddonRelease object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *TalosClusterAddonReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var tcAddonRelease talosv1alpha1.TalosClusterAddonRelease
	if err := r.Get(ctx, req.NamespacedName, &tcAddonRelease); err != nil {
		logger.Error(err, "unable to fetch TalosClusterAddonRelease")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var delErr error
	if tcAddonRelease.DeletionTimestamp.IsZero() {
		err := r.handleFinalizer(ctx, tcAddonRelease)
		if err != nil {
			logger.Error(err, "failed to handle finalizer for TalosClusterAddonRelease", "name", tcAddonRelease.Name)
			return ctrl.Result{}, err
		}
	} else {
		// Handle deletion
		if controllerutil.ContainsFinalizer(&tcAddonRelease, talosv1alpha1.TalosClusterAddonReleaseFinalizer) {
			res, delErr := r.handleDelete(ctx, tcAddonRelease)
			if delErr != nil {
				logger.Error(delErr, "failed to handle deletion for TalosClusterAddonRelease", "name", tcAddonRelease.Name)
				return res, delErr
			}
			controllerutil.RemoveFinalizer(&tcAddonRelease, talosv1alpha1.TalosClusterAddonReleaseFinalizer)
			if err := r.Update(ctx, &tcAddonRelease); err != nil {
				logger.Error(err, "failed to remove finalizer for TalosClusterAddonRelease", "name", tcAddonRelease.Name)
				return ctrl.Result{}, err
			}
		}
		// Object is being deleted, no further reconciliation needed
		return ctrl.Result{}, client.IgnoreNotFound(delErr)
	}
	// Handle the helm chart installation or update logic here

	// Get the TalosControlPlane referenced by this addon release
	tcp := &talosv1alpha1.TalosControlPlane{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      tcAddonRelease.Spec.ClusterRef.Name,
		Namespace: tcAddonRelease.Spec.ClusterRef.Namespace,
	}, tcp)
	if err != nil {
		logger.Error(err, "failed to get TalosControlPlane for TalosClusterAddonRelease", "name", tcAddonRelease.Name)
		return ctrl.Result{}, err
	}
	// Get the kubeconfig secret for the TalosControlPlane
	kubeconfigSecretName := fmt.Sprintf("%s-kubeconfig", tcp.Name)
	kubeconfigSecret := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      kubeconfigSecretName,
		Namespace: tcp.Namespace,
	}, kubeconfigSecret)
	if err != nil {
		logger.Error(err, "failed to get kubeconfig secret for TalosControlPlane", "name", tcp.Name)
		return ctrl.Result{}, err
	}
	// Get the kubeconfig data
	kubeconfigData, ok := kubeconfigSecret.Data["kubeconfig"]
	if !ok {
		err := fmt.Errorf("kubeconfig data not found in secret %s", kubeconfigSecretName)
		logger.Error(err, "kubeconfig data missing")
		return ctrl.Result{}, err
	}
	// Create a helm client using the kubeconfig data
	helmClient, err := helm.NewClient(kubeconfigData, tcAddonRelease.Spec.HelmSpec.ReleaseNamespace)
	if err != nil {
		logger.Error(err, "failed to create helm client")
		return ctrl.Result{}, err
	}
	_, err = helmClient.InstallOrUpgradeChart(ctx, tcAddonRelease.Spec.HelmSpec)
	if err != nil {
		logger.Error(err, "failed to install or upgrade helm chart")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosClusterAddonReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosClusterAddonRelease{}).
		Named("talosclusteraddonrelease").
		Complete(r)
}

func (r *TalosClusterAddonReleaseReconciler) handleFinalizer(ctx context.Context, tcAddonRelease talosv1alpha1.TalosClusterAddonRelease) error {
	if !controllerutil.ContainsFinalizer(&tcAddonRelease, talosv1alpha1.TalosClusterAddonReleaseFinalizer) {
		controllerutil.AddFinalizer(&tcAddonRelease, talosv1alpha1.TalosClusterAddonReleaseFinalizer)
		if err := r.Update(ctx, &tcAddonRelease); err != nil {
			return fmt.Errorf("failed to add finalizer to TalosClusterAddonRelease %s: %w", tcAddonRelease.Name, err)
		}
	}
	return nil
}

func (r *TalosClusterAddonReleaseReconciler) handleDelete(ctx context.Context, tcAddonRelease talosv1alpha1.TalosClusterAddonRelease) (ctrl.Result, error) {
	// TODO: Remove helm chart from the cluster
	_, _ = ctx, tcAddonRelease
	return ctrl.Result{}, nil
}
