// Package talos provides a programmatic interface to bootstrap Talos control plane nodes.
package talos

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

type TalosClient struct {
	*client.Client
}

// NewClient constructs a Talos API client using the default talosconfig file
// and context name. It returns an initialized client or an error.
func NewClient(cfg *BundleConfig, insecure bool) (*TalosClient, error) {

	ctx := context.Background()
	bundle, err := NewCPBundle(cfg, nil)
	if err != nil {
		return nil, err
	}
	var endpoint string
	if cfg.ClientEndpoint != nil {
		endpoint = *cfg.ClientEndpoint
	} else {
		endpoint = fmt.Sprintf("%s", cfg.ClusterName)
	}
	if insecure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // For testing purposes, skip TLS verification
		}
		c, err := client.New(ctx,
			client.WithEndpoints(endpoint),
			client.WithConfig(bundle.TalosConfig()),
			client.WithTLSConfig(tlsConfig),
		)
		if err != nil {
			return nil, err
		}
		return &TalosClient{Client: c}, nil
	}
	c, err := client.New(ctx,
		client.WithEndpoints(endpoint),
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
	// TODO: Fix here
	// Perform the bootstrap operation
	// 	if err := tc.Client.Bootstrap(context.Background(), req); err != nil {
	// return err
	// }
	tc.Client.Bootstrap(context.Background(), req)
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
	resp, err := tc.Client.ApplyConfiguration(ctx, applyRequest)
	if err != nil {
		return fmt.Errorf("Error applying new configration %s", err)
	}
	// TODO: Use FilterMessages over resp
	_ = resp

	return nil
}

func (tc *TalosClient) GetInstallDisk(ctx context.Context, tcp *talosv1alpha1.TalosControlPlane) (*string, error) {
	// Check if the TalosControlPlane is in metal mode
	if tcp.Spec.Mode != "metal" {
		return nil, nil
	}
	// Check if installDisk is provided
	if tcp.Spec.MetalSpec.InstallDisk != nil {
		// If installDisk is provided, return it
		return tcp.Spec.MetalSpec.InstallDisk, nil
	} else {
		// Try to get it from the disks
		resp, err := tc.Client.Disks(ctx)
		if err != nil {
			fmt.Printf("Error getting disks: %s\n", err)
			return nil, err
		}
		disks := resp.Messages
		if len(disks) == 0 {
			fmt.Println("No disks found to install Talos")
			return nil, err
		}
		var diskName string
		// Get disks and remove the readonly ones
		for _, disk := range disks {
			for _, part := range disk.Disks {
				if part.Readonly {
					continue
				}
				// DEBUG
				fmt.Printf("Disk: %s, Readonly: %t\n", part.Name, part.Readonly)
				time.Sleep(20 * time.Second)
				// Look for nvme or sda disks
				switch part.Name {
				case "nvme0n1", "sda":
					diskName = part.Name
				}
				// If we found a disk, break the loop
				if diskName != "" {
					break
				}
			}
		}
		diskName = fmt.Sprintf("/dev/%s", diskName)
		return &diskName, nil
	}
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
