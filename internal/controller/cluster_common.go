package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func updatePxeBootStackConfig(ctx context.Context, r *TalosClusterReconciler, namespace string, machines map[*talosv1alpha1.Machine]string) error {
	logger := log.FromContext(ctx)

	// Retrieving all TalosCluster resources
	var tcList talosv1alpha1.TalosClusterList
	if err := r.List(ctx, &tcList, client.InNamespace(namespace)); err != nil {
		return err
	}

	// Generating dnsmasq and Matchbox configurations
	// Base dnsmasq configuration : enable TFTP, set PXE boot file name and tag iPXE clients
	var dnsmasqConfigString = `bind-interfaces

# TFTP server:
enable-tftp
tftp-root=/var/lib/tftp

# UEFI iPXE boot file:
dhcp-match=set:efi,option:client-arch,7
dhcp-boot=tag:efi,ipxe.efi

# Tagging iPXE clients:
dhcp-userclass=set:ipxe,iPXE`
	var matchboxGroupsMap = make(map[string]string)
	var matchboxProfilesMap = make(map[string]string)
	for _, tc := range tcList.Items {
		if tc.Spec.PxeServerSpec != nil {
			// Global cluster config : tag hosts depending on the interface receiving the request, enable static IP assignment set iPXE boot file name
			dnsmasqConfigString = fmt.Sprintf(`%s

# %s
tag-if=set:if_%s,tag:%s
dhcp-range=tag:if_%s,%s,static
dhcp-boot=tag:if_%s,tag:ipxe,http://%s:8080/boot.ipxe`,
				dnsmasqConfigString, tc.Name, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Address, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Address,
			)

			// Machines specific config
			// Combining control plane and worker machines in a single array and mapping machines with the version of Talos they require
			if tc.Spec.ControlPlane != nil && tc.Spec.ControlPlane.Mode == TalosModeMetal {
				for _, m := range tc.Spec.ControlPlane.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						machines[&m] = tc.Spec.ControlPlane.Version
					}
				}
			}
			if tc.Spec.Worker != nil && tc.Spec.Worker.Mode == TalosModeMetal {
				for _, m := range tc.Spec.Worker.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						machines[&m] = tc.Spec.Worker.Version
					}
				}
			}
			var machineIndex = 0
			for m, version := range machines {
				// dnsmasq config : static IP addresses assignment
				dnsmasqConfigString = fmt.Sprintf("%s\ndhcp-host=tag:if_%s,%s,%s",
					dnsmasqConfigString, *tc.Spec.PxeServerSpec.Interface, *m.PxeClientSpec.MacAddress, *m.Address,
				)
				// Matchbox config : matching hosts with their MAC address
				var matchboxId = fmt.Sprintf("%s-%d", tc.Name, machineIndex)
				var matchboxFileName = fmt.Sprintf("%s.json", matchboxId)
				matchboxGroupsMap[matchboxFileName] = fmt.Sprintf(`{
	"id": "%s",
	"name": "%s",
	"profile": "%s",
	"selector": {
		"mac": "%s"
	}
}`,
					matchboxId, matchboxId, matchboxId, *m.PxeClientSpec.MacAddress,
				)
				var kernelCmdline string // Additional kernel command line arguments
				if m.PxeClientSpec.KernelCmdlineArgs != nil {
					kernelCmdline = *m.PxeClientSpec.KernelCmdlineArgs
				} else {
					kernelCmdline = ""
				}
				matchboxProfilesMap[matchboxFileName] = fmt.Sprintf(`{
	"id": "%s",
	"name": "%s",
	"boot": {
		"kernel": "/assets/vmlinuz-%s-%s",
		"initrd": ["/assets/initramfs-%s-%s.xz"],
		"args": [
			"initrd=initramfs.xz",
			"slab_nomerge",
			"pti=on",
			"console=tty0",
			"printk.devkmsg=on",
			"talos.platform=metal",
			"%s"
		]
	}
}`,
					matchboxId, matchboxId, version, *m.PxeClientSpec.CpuArchitecture, version, *m.PxeClientSpec.CpuArchitecture, kernelCmdline,
				)
				machineIndex++
			}
		}
	}

	// Creating ConfigMaps
	// dnsmasq
	dnsmasqConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dnsmasq-config",
			Namespace: namespace,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, dnsmasqConfigMap, func() error {
		dnsmasqConfigMap.Data = map[string]string{
			"dnsmasq-config": dnsmasqConfigString,
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "failed to update PXE boot stack configuration", "operation", op, "namespace", namespace)
		return err
	}
	// Matchbox groups
	matchboxGroupsConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "matchbox-groups",
			Namespace: namespace,
		},
	}
	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, matchboxGroupsConfigMap, func() error {
		matchboxGroupsConfigMap.Data = matchboxGroupsMap
		return nil
	})
	if err != nil {
		logger.Error(err, "failed to update PXE boot stack configuration", "operation", op, "namespace", namespace)
		return err
	}
	// Matchbox profiles
	matchboxProfilesConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "matchbox-profiles",
			Namespace: namespace,
		},
	}
	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, matchboxProfilesConfigMap, func() error {
		matchboxProfilesConfigMap.Data = matchboxProfilesMap
		return nil
	})
	if err != nil {
		logger.Error(err, "failed to update PXE boot stack configuration", "operation", op, "namespace", namespace)
		return err
	}

	return nil
}

func downloadTalosBootImages(machines map[*talosv1alpha1.Machine]string) error {
	// Parsing the machines array to determine every combination of Talos version + CPU architecture to take into account when downloading boot images
	var downloadList = make(map[string]string) // Maps a download URL to a destination path
	for m, version := range machines {
		// Kernel
		downloadList[fmt.Sprintf("%s/%s/vmlinuz-%s", TalosBootImageBaseUrl, version, *m.PxeClientSpec.CpuArchitecture)] = fmt.Sprintf("%s/vmlinuz-%s-%s", MatchboxAssetsPath, version, *m.PxeClientSpec.CpuArchitecture)
		// initramfs
		downloadList[fmt.Sprintf("%s/%s/initramfs-%s.xz", TalosBootImageBaseUrl, version, *m.PxeClientSpec.CpuArchitecture)] = fmt.Sprintf("%s/initramfs-%s-%s.xz", MatchboxAssetsPath, version, *m.PxeClientSpec.CpuArchitecture)
	}

	// Downloading files
	for url, path := range downloadList {
		if err := downloadFile(url, path); err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(url string, path string) error {
	// Create destination file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// Write downloaded file
	if resp.StatusCode == 200 {
		if _, err := io.Copy(file, resp.Body); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unable to download '%s' (HTTP status code: %d)", url, resp.StatusCode)
	}

	return nil
}
