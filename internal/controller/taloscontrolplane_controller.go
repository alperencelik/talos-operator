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
	"fmt"

	"gopkg.in/yaml.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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
		logger.Error(err, "unable to fetch TalosControlPlane")
		return ctrl.Result{}, r.handleResourceNotFound(ctx, err)
	}

	// Get the mode of the TalosControlPlane
	var result ctrl.Result
	var err error
	switch tcp.Spec.Mode {
	case "container":
		result, err = r.reconcileContainerMode(ctx, &tcp)
		if err != nil {
			logger.Error(err, "failed to reconcile TalosControlPlane in container mode")
		}
	default:
		logger.Info("Unsupported mode for TalosControlPlane", "mode", tcp.Spec.Mode)
		return ctrl.Result{}, nil
	}
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TalosControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&talosv1alpha1.TalosControlPlane{}).
		// Owns(&appsv1.StatefulSet{}).
		// Owns(&corev1.Service{}).
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

	if err := r.reconcileService(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile Service for TalosControlPlane %s: %w", tcp.Name, err)
	}

	// Get the statefulset for the TalosControlPlane
	if err := r.reconcileStatefulSet(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile StatefulSet for TalosControlPlane %s: %w", tcp.Name, err)
	}

	if err := r.BootstrapCluster(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to bootstrap Talos ControlPlane cluster for %s: %w", tcp.Name, err)
	}

	if err := r.WriteKubeconfig(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to write kubeconfig for TalosControlPlane %s: %w", tcp.Name, err)
	}

	if err := r.Status().Update(ctx, tcp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update TalosControlPlane %s status: %w", tcp.Name, err)
	}

	return ctrl.Result{}, nil
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
	logger := log.FromContext(ctx)

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
	// TODO: Proper logging
	logger.Info("Reconciled Service", "service", tcp.Name)
	return nil
}

func (r *TalosControlPlaneReconciler) reconcileStatefulSet(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)
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

	extraEnvs := BuildUserDataEnvVar(&tcp.Spec.ConfigRef, tcp.Name, TalosMachineTypeControlPlane)

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Spec = BuildStsSpec(tcp.Name, tcp.Spec.Replicas, tcp.Spec.Version, TalosMachineTypeControlPlane, extraEnvs)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update StatefulSet %s: %w", stsName, err)
	}
	logger.Info("Reconciled StatefulSet", "statefulset", stsName, "operation", op)
	return nil
}

func (r *TalosControlPlaneReconciler) GenerateConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	logger := log.FromContext(ctx)

	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// Generate the Talos ControlPlane config
	cpConfig, err := talos.GenerateControlPlaneConfig(config)
	if err != nil {
		return fmt.Errorf("failed to generate Talos ControlPlane config for %s: %w", tcp.Name, err)
	}
	tcp.Status.Config = string(*cpConfig)
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
	logger.Info("Generated Talos ControlPlane config", "name", tcp.Name)
	// Update the status condition to indicate the config is generated
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
	return nil
}

func (r *TalosControlPlaneReconciler) WriteTalosConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	// logger := log.FromContext(ctx)

	// Set the Talos ControlPlane config
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}

	bundle, err := talos.NewCPBundle(config)
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
	return nil
}

func (r *TalosControlPlaneReconciler) BootstrapCluster(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	// TODO: Implement
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	// Create a Talos client
	client, err := talos.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create Talos client for ControlPlane %s: %w", tcp.Name, err)
	}
	//  Bootstrap the Talos node
	if err := client.BootstrapNode(config); err != nil {
		return fmt.Errorf("failed to bootstrap Talos node for ControlPlane %s: %w", tcp.Name, err)
	}
	return nil
}

func (r *TalosControlPlaneReconciler) WriteKubeconfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {
	//
	config, err := r.SetConfig(ctx, tcp)
	if err != nil {
		return fmt.Errorf("failed to set config for TalosControlPlane %s: %w", tcp.Name, err)
	}
	client, err := talos.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create Talos client for ControlPlane %s: %w", tcp.Name, err)
	}
	// Generate the kubeconfig for the Talos ControlPlane
	kubeconfig, err := client.Kubeconfig(ctx)
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
	return nil
}

func (r *TalosControlPlaneReconciler) SetConfig(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*talos.BundleConfig, error) {
	// Genenrate the Subject Alternative Names (SANs) for the Talos ControlPlane
	sans := utils.GenSans(tcp.Name, int(tcp.Spec.Replicas))
	// Get secret bundle
	secretBundle, err := r.SecretBundle(ctx, tcp)
	if err != nil {
		return nil, fmt.Errorf("failed to get SecretBundle for TalosControlPlane %s: %w", tcp.Name, err)
	}

	// Generate the Talos ControlPlane config
	return &talos.BundleConfig{
		ClusterName:   tcp.Name,
		Endpoint:      fmt.Sprintf("https://%s:6443", tcp.Name),
		Version:       tcp.Spec.Version,
		KubeVersion:   tcp.Spec.KubeVersion,
		SecretsBundle: talos.SecretBundle(*secretBundle),
		Sans:          sans,
		ServiceCIDR:   &tcp.Spec.ServiceCIDR,
		PodCIDR:       &tcp.Spec.PodCIDR,
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
