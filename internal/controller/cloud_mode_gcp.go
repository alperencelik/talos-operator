package controller

import (
	"context"
	"fmt"

	computev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/compute/v1beta1"
	kccv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/k8s/v1alpha1"
	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *TalosControlPlaneReconciler) reconcileGCPResources(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) error {

	// TODO:
	// Before proceeding, ensure GCP controller is installed and running in the cluster.
	// This can be done by checking for the existence of a known KCC resource or CRD.

	// 6. Create Address (Global)
	addrName := fmt.Sprintf("%s-lb-ip", tcp.Name)
	addr := &computev1beta1.ComputeAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addrName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, addr, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for address %s: %w", addrName, err)
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, addr, func() error {
		addr.Spec = computev1beta1.ComputeAddressSpec{
			// TODO: Review if I need to set explicitly it here
			ResourceID: ptr.To(addrName),
			Location:   "global",
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile address %s: %w", addrName, err)
	}

	gcpSpec := tcp.Spec.CloudSpec.GCP

	// 1. Create ComputeInstance for each replica
	for i := int32(0); i < tcp.Spec.Replicas; i++ {
		instanceName := fmt.Sprintf("%s-%d", tcp.Name, i)
		instance := &computev1beta1.ComputeInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: tcp.Namespace,
			},
		}

		if err := controllerutil.SetControllerReference(tcp, instance, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for instance %s: %w", instanceName, err)
		}

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, instance, func() error {
			instance.Spec = computev1beta1.ComputeInstanceSpec{
				ResourceID:  ptr.To(instanceName),
				Zone:        ptr.To(gcpSpec.Zone),
				MachineType: ptr.To(gcpSpec.InstanceType),
				BootDisk: &computev1beta1.InstanceBootDisk{
					InitializeParams: &computev1beta1.InstanceInitializeParams{
						SourceImageRef: &kccv1alpha1.ResourceRef{External: gcpSpec.Image},
					},
				},
				NetworkInterface: []computev1beta1.InstanceNetworkInterface{
					{
						NetworkRef: &kccv1alpha1.ResourceRef{External: gcpSpec.Network},
						// SubnetworkRef: &kccv1alpha1.ResourceRef{External: gcpSpec.Subnetwork},
					},
				},

				Tags: gcpSpec.Tags,
				Metadata: []computev1beta1.InstanceMetadata{
					{
						Key:   "user-data",
						Value: fmt.Sprintf("#!/bin/bash\n# Talos Control Plane %d", i),
					},
				},
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to reconcile instance %s: %w", instanceName, err)
		}
	}

	// 2. Create Instance Group (Unmanaged)
	igName := fmt.Sprintf("%s-ig", tcp.Name)
	ig := &computev1beta1.ComputeInstanceGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      igName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, ig, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for instance group %s: %w", igName, err)
	}

	// Collect instance refs
	var instances []kccv1alpha1.ResourceRef
	for i := int32(0); i < tcp.Spec.Replicas; i++ {
		instanceName := fmt.Sprintf("%s-%d", tcp.Name, i)
		instances = append(instances, kccv1alpha1.ResourceRef{Name: instanceName})
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ig, func() error {
		ig.Spec = computev1beta1.ComputeInstanceGroupSpec{
			ResourceID: ptr.To(igName),
			Zone:       gcpSpec.Zone,
			Instances:  instances,
			NamedPort: []computev1beta1.InstancegroupNamedPort{
				{
					Name: "tcp6443",
					Port: 6443,
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile instance group %s: %w", igName, err)
	}

	// 3. Create Health Check
	hcName := fmt.Sprintf("%s-health-check", tcp.Name)
	hc := &computev1beta1.ComputeHealthCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, hc, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for health check %s: %w", hcName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, hc, func() error {
		hc.Spec = computev1beta1.ComputeHealthCheckSpec{
			Location:   "global",
			ResourceID: ptr.To(hcName),
			TcpHealthCheck: &computev1beta1.HealthcheckTcpHealthCheck{
				Port: ptr.To(int64(6443)),
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile health check %s: %w", hcName, err)
	}

	// 4. Create Backend Service
	beName := fmt.Sprintf("%s-be", tcp.Name)
	be := &computev1beta1.ComputeBackendService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      beName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, be, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for backend service %s: %w", beName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, be, func() error {
		be.Spec = computev1beta1.ComputeBackendServiceSpec{
			ResourceID:   ptr.To(beName),
			Protocol:     ptr.To("TCP"),
			HealthChecks: []computev1beta1.BackendserviceHealthChecks{{HealthCheckRef: &kccv1alpha1.ResourceRef{Name: hcName}}},
			TimeoutSec:   ptr.To(int64(300)),
			PortName:     ptr.To("tcp6443"),
			Location:     "global",
			Backend: []computev1beta1.BackendserviceBackend{
				{
					Group: computev1beta1.BackendserviceGroup{
						InstanceGroupRef: &kccv1alpha1.ResourceRef{Name: igName},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile backend service %s: %w", beName, err)
	}

	// 5. Create TCP Proxy
	proxyName := fmt.Sprintf("%s-tcp-proxy", tcp.Name)
	proxy := &computev1beta1.ComputeTargetTCPProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxyName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, proxy, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for tcp proxy %s: %w", proxyName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, proxy, func() error {
		proxy.Spec = computev1beta1.ComputeTargetTCPProxySpec{
			ResourceID:        ptr.To(proxyName),
			BackendServiceRef: kccv1alpha1.ResourceRef{Name: beName},
			ProxyHeader:       ptr.To("NONE"),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile tcp proxy %s: %w", proxyName, err)
	}

	// 7. Create Forwarding Rule
	frName := fmt.Sprintf("%s-fwd-rule", tcp.Name)
	fr := &computev1beta1.ComputeForwardingRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, fr, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for forwarding rule %s: %w", frName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, fr, func() error {
		fr.Spec = computev1beta1.ComputeForwardingRuleSpec{
			ResourceID: ptr.To(frName),
			IpAddress:  &computev1beta1.ForwardingruleIpAddress{AddressRef: &kccv1alpha1.ResourceRef{Name: addrName}},
			Target:     &computev1beta1.ForwardingruleTarget{TargetTCPProxyRef: &kccv1alpha1.ResourceRef{Name: proxyName}},
			PortRange:  ptr.To("443"),
			Location:   "global",
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile forwarding rule %s: %w", frName, err)
	}

	// 8. Create Firewall Rules
	// Allow 6443 from LB ranges
	fwName := fmt.Sprintf("%s-controlplane-firewall", tcp.Name)
	fw := &computev1beta1.ComputeFirewall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fwName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, fw, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for firewall %s: %w", fwName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, fw, func() error {
		fw.Spec = computev1beta1.ComputeFirewallSpec{
			ResourceID:   ptr.To(fwName),
			NetworkRef:   kccv1alpha1.ResourceRef{External: gcpSpec.Network},
			SourceRanges: []string{"130.211.0.0/22", "35.191.0.0/16"},
			TargetTags:   gcpSpec.Tags,
			Allow: []computev1beta1.FirewallAllow{
				{
					Protocol: "tcp",
					Ports:    []string{"6443"},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile firewall %s: %w", fwName, err)
	}

	// Allow talosctl (50000) from everywhere (or restricted)
	fwTalosName := fmt.Sprintf("%s-controlplane-talosctl", tcp.Name)
	fwTalos := &computev1beta1.ComputeFirewall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fwTalosName,
			Namespace: tcp.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(tcp, fwTalos, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for firewall %s: %w", fwTalosName, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, fwTalos, func() error {
		fwTalos.Spec = computev1beta1.ComputeFirewallSpec{
			ResourceID:   ptr.To(fwTalosName),
			NetworkRef:   kccv1alpha1.ResourceRef{External: gcpSpec.Network},
			SourceRanges: []string{"0.0.0.0/0"},
			TargetTags:   gcpSpec.Tags,
			Allow: []computev1beta1.FirewallAllow{
				{
					Protocol: "tcp",
					Ports:    []string{"50000"},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile firewall %s: %w", fwTalosName, err)
	}

	return nil
}

func (r *TalosControlPlaneReconciler) checkCloudModeReady(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (bool, error) {
	// Check if all instances are running
	for i := int32(0); i < tcp.Spec.Replicas; i++ {
		instanceName := fmt.Sprintf("%s-%d", tcp.Name, i)
		instance := &computev1beta1.ComputeInstance{}
		if err := r.Get(ctx, types.NamespacedName{Name: instanceName, Namespace: tcp.Namespace}, instance); err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		// Check status (simplified)
		// KCC resources usually have a Ready condition
		// For now, assume if it exists, it's creating.
	}
	return true, nil
}

func (r *TalosControlPlaneReconciler) GetGCPLoadBalancerIP(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) ([]string, error) {
	// Get the LoadBalancer IP from the Address resource
	addrName := fmt.Sprintf("%s-lb-ip", tcp.Name)
	addr := &computev1beta1.ComputeAddress{}
	if err := r.Get(ctx, types.NamespacedName{Name: addrName, Namespace: tcp.Namespace}, addr); err != nil {
		return nil, fmt.Errorf("failed to get address %s: %w", addrName, err)
	}
	if addr.Status.ObservedState.Address == nil {
		return nil, fmt.Errorf("address %s does not have an assigned IP yet", addrName)
	}
	return []string{*addr.Status.ObservedState.Address}, nil
}
