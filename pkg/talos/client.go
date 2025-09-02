// Package talos provides a programmatic interface to bootstrap Talos control plane nodes.
package talos

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"text/template"
	"time"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

type TalosClient struct {
	*client.Client
	Endpoint string
}

const (
	KUBELET_SERVICE_NAME   = "kubelet"
	KUBELET_STATUS_RUNNING = "Running"
)

// NewClient constructs a Talos API client using the default talosconfig file
// and context name. It returns an initialized client or an error.
func NewClient(cfg *BundleConfig, insecure bool) (*TalosClient, error) {

	ctx := context.Background()
	bundle, err := NewCPBundle(cfg, nil)
	if err != nil {
		return nil, err
	}
	var endpoints []string
	if cfg.ClientEndpoint != nil && len(*cfg.ClientEndpoint) > 0 {
		endpoints = *cfg.ClientEndpoint
	} else {
		endpoints = []string{cfg.ClusterName}
	}
	if insecure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // For testing purposes, skip TLS verification
		}
		c, err := client.New(ctx,
			client.WithEndpoints(endpoints...),
			client.WithConfig(bundle.TalosConfig()),
			client.WithTLSConfig(tlsConfig),
		)
		if err != nil {
			return nil, err
		}
		return &TalosClient{Client: c}, nil
	}
	c, err := client.New(ctx,
		client.WithEndpoints(endpoints...),
		client.WithConfig(bundle.TalosConfig()),
	)
	if err != nil {
		return nil, err
	}
	return &TalosClient{Client: c}, nil
}

// BootstrapNode replicates `talosctl bootstrap` by invoking the Bootstrap RPC on the node.
func (tc *TalosClient) BootstrapNode(cfg *BundleConfig) error {
	// Create a BootstrapRequest (no etcd recovery by default)
	req := &machineapi.BootstrapRequest{
		RecoverEtcd:          false,
		RecoverSkipHashCheck: false,
	}
	// TODO: Fix this one
	_ = tc.Bootstrap(context.Background(), req)
	return nil
}

// ApplyConfig applies the config to the machine by using talos client
func (tc *TalosClient) ApplyConfig(ctx context.Context, machineConfig []byte) error {
	applyRequest := &machineapi.ApplyConfigurationRequest{
		Data:           machineConfig,
		Mode:           machineapi.ApplyConfigurationRequest_AUTO,
		DryRun:         false,
		TryModeTimeout: durationpb.New(60 * time.Second),
	}
	resp, err := tc.ApplyConfiguration(ctx, applyRequest)
	if err != nil {
		return fmt.Errorf("error applying new configuration %s", err)
	}
	// TODO: Use FilterMessages over resp
	// Parse the response
	fmt.Printf("ApplyConfiguration response: %s\n", resp.Messages[0].String())
	return nil
}

func (tc *TalosClient) GetTalosVersion(ctx context.Context) (string, error) {
	// Get the Talos version
	resp, err := tc.Version(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting Talos version: %s", err)
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no version information found")
	}
	version := resp.Messages[0].Version.Tag
	return version, nil
}

func (tc *TalosClient) UpgradeTalosVersion(ctx context.Context, image string) error {
	// Decode machineConfig to get the image, stage, and force values
	// yaml.Marshal(machineConfig, &machineConfigStruct)
	// Create an UpgradeRequest
	upgradeRequest := &machineapi.UpgradeRequest{
		Image:      image,
		Stage:      false,                             // stage is to perform upgrade it after a reboot
		Force:      true,                              // force etcd checks etc
		RebootMode: machineapi.UpgradeRequest_DEFAULT, // DEFAULT or POWERCYCLE
	}
	resp, err := tc.MachineClient.Upgrade(ctx, upgradeRequest)
	if err != nil {
		return fmt.Errorf("error upgrading machine: %s", err)
	}
	// Parse the response
	fmt.Printf("Upgrade response: %s\n", resp.Messages[0].String())
	return nil
}

func (tc *TalosClient) GetInstallDisk(ctx context.Context, tm *talosv1alpha1.TalosMachine) (*string, error) {
	// Check if installDisk is provided
	if tm.Spec.MachineSpec != nil && tm.Spec.MachineSpec.InstallDisk != nil {
		// If installDisk is provided, return it
		return tm.Spec.MachineSpec.InstallDisk, nil
	} else {
		// Try to get it from the disks
		resp, err := tc.Disks(ctx)
		if err != nil {
			fmt.Printf("Error getting disks: %s\n", err)
			return nil, err
		}
		disks := resp.Messages
		if len(disks) == 0 {
			fmt.Println("No disks found to install Talos")
			return nil, err
		}
		// Get disks and remove the readonly ones
		for _, disk := range disks {
			for _, part := range disk.Disks {
				if part.Readonly {
					continue
				}
				// Look for nvme or sda disks
				switch part.DeviceName {
				case "/dev/nvme0n1", "/dev/sda":
					return &part.DeviceName, nil
				default:
					// If no specific disk is found, return the first writable disk
					if !part.Readonly {
						return &part.DeviceName, nil
					}
				}
			}
		}
	}
	return nil, nil
}

func (tc *TalosClient) GetServiceStatus(ctx context.Context, svcName string) string {
	// Get the service status
	svcsInfo, err := tc.ServiceInfo(ctx, svcName)
	if err != nil {
		fmt.Printf("Error getting service status: %s\n", err)
		return ""
	}
	svc := svcsInfo[0]
	// fmt.Printf("Service %s status: %s\n", svc, svc.Service.State)
	fmt.Printf("Service status: %s\n", svc.Service.State)
	return svc.Service.State
	// state := "running" // Placeholder for actual service state
	// return state
}

func (tc *TalosClient) ApplyMetaKey(ctx context.Context, endpoint string, meta *talosv1alpha1.META) error {
	// Set the meta key
	var key uint8 = 0x0a
	// Create a new template and parse the template string
	t, err := template.New("metakey").Parse(metaKeyTemplate)
	if err != nil {
		return err
	}

	// Create a new buffer to store the generated YAML
	var tpl bytes.Buffer

	data := struct {
		*talosv1alpha1.META
		Endpoint string
	}{
		META:     meta,
		Endpoint: endpoint,
	}

	// Execute the template with the meta data
	if err := t.Execute(&tpl, data); err != nil {
		return err
	}

	return tc.MetaWrite(ctx, key, tpl.Bytes())
}

// func (tc *TalosClient) GetMachineStatus(ctx context.Context) (*string, error) {
// // Get the machine status
// res, err := tc.COSI.Get(ctx,
// resource.NewMetadata("runtime", "machinestatuses", "", resource.VersionUndefined),
// state.WithGetUnmarshalOptions(state.WithSkipProtobufUnmarshal()),
// )
// if err != nil {
// return nil, fmt.Errorf("error getting machine status: %w", err)
// }
// fmt.Printf("Machine status: %v\n", res)

// // tc.COSI.List(ctx, resourceType, resourceID)
// return nil, nil
// }
