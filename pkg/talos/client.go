// Package talos provides a programmatic interface to bootstrap Talos control plane nodes.
package talos

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"text/template"
	"time"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/configloader"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func NewClient(ctx context.Context, cfg *BundleConfig, insecure bool) (*TalosClient, error) {
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
func (tc *TalosClient) BootstrapNode(ctx context.Context) error {
	req := &machineapi.BootstrapRequest{
		RecoverEtcd:          false,
		RecoverSkipHashCheck: false,
	}
	if err := tc.Bootstrap(ctx, req); err != nil {
		return fmt.Errorf("failed to bootstrap node: %w", err)
	}
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
	_, err := tc.ApplyConfiguration(ctx, applyRequest)
	if err != nil {
		if isGracefulStop(err) {
			return nil
		}
		return fmt.Errorf("error applying new configuration: %w", err)
	}
	return nil
}

func (tc *TalosClient) GetTalosVersion(ctx context.Context) (string, error) {
	// Get the Talos version
	resp, err := tc.Version(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting Talos version: %w", err)
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no version information found")
	}
	if resp.Messages[0].Version == nil {
		return "", fmt.Errorf("version information is nil")
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
	_, err := tc.MachineClient.Upgrade(ctx, upgradeRequest)
	if err != nil {
		return fmt.Errorf("error upgrading machine: %w", err)
	}
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
			return nil, fmt.Errorf("error getting disks: %w", err)
		}
		disks := resp.Messages
		if len(disks) == 0 {
			return nil, fmt.Errorf("no disks found to install Talos")
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
	return nil, fmt.Errorf("no suitable install disk found on machine %s", tm.Name)
}

func (tc *TalosClient) GetServiceStatus(ctx context.Context, svcName string) (*string, error) {
	svcsInfo, err := tc.ServiceInfo(ctx, svcName)
	if err != nil {
		return nil, fmt.Errorf("error getting service info for %s: %w", svcName, err)
	}
	if len(svcsInfo) == 0 {
		return nil, nil
	}
	return &svcsInfo[0].Service.State, nil
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

func GetSecretBundleFromConfig(ctx context.Context, machineConfig []byte) (*secrets.Bundle, error) {
	// Load cfg from machineConfig
	cfg, err := configloader.NewFromBytes(machineConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading config from machineConfig: %w", err)
	}
	// Create a SecretBundle from the configData
	return secrets.NewBundleFromConfig(secrets.NewFixedClock(time.Now()), cfg), nil
}

// isGracefulStop returns true if the error is a gRPC Unavailable error caused by
// the server performing a graceful shutdown.
func isGracefulStop(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.Unavailable && strings.Contains(st.Message(), "graceful_stop")
}
