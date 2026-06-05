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

	"io"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/utils"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
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

// ApplyConfig applies the config to the machine by using talos client. If dryRun is set,
// the config is not applied and the returned string contains the change details (diff)
// reported by the node.
func (tc *TalosClient) ApplyConfig(ctx context.Context, machineConfig []byte, dryRun bool) (string, error) {
	applyRequest := &machineapi.ApplyConfigurationRequest{
		Data:           machineConfig,
		Mode:           machineapi.ApplyConfigurationRequest_AUTO,
		DryRun:         dryRun,
		TryModeTimeout: durationpb.New(60 * time.Second),
	}
	resp, err := tc.ApplyConfiguration(ctx, applyRequest)
	if err != nil {
		if isGracefulStop(err) {
			return "", nil
		}
		return "", fmt.Errorf("error applying new configuration: %w", err)
	}
	if !dryRun {
		return "", nil
	}
	var sb strings.Builder
	for _, m := range resp.GetMessages() {
		if details := m.GetModeDetails(); details != "" {
			sb.WriteString(details)
			sb.WriteString("\n")
		}
		for _, w := range m.GetWarnings() {
			sb.WriteString("WARNING: ")
			sb.WriteString(w)
			sb.WriteString("\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
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

// UpgradeTalosVersion upgrades the Talos installation to the given installer image.
func (tc *TalosClient) UpgradeTalosVersion(ctx context.Context, currentVersion, image string) error {
	if utils.SupportsLifecycleService(currentVersion) {
		return tc.upgradeViaLifecycleService(ctx, image)
	}
	return tc.upgradeViaMachineService(ctx, image)
}

func (tc *TalosClient) upgradeViaLifecycleService(ctx context.Context, image string) error {
	containerd := &common.ContainerdInstance{
		Driver:    common.ContainerDriver_CRI,
		Namespace: common.ContainerdNamespace_NS_SYSTEM,
	}

	if err := tc.pullInstallerImage(ctx, containerd, image); err != nil {
		return fmt.Errorf("error pulling Talos installer image: %w", err)
	}

	stream, err := tc.LifecycleClient.Upgrade(ctx, &machineapi.LifecycleServiceUpgradeRequest{
		Containerd: containerd,
		Source: &machineapi.InstallArtifactsSource{
			ImageName: image,
		},
	})
	if err != nil {
		return fmt.Errorf("error starting Talos upgrade: %w", err)
	}

	for {
		resp, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			if isGracefulStop(recvErr) {
				break
			}
			return fmt.Errorf("error during Talos upgrade: %w", recvErr)
		}
		exit, ok := resp.GetProgress().GetResponse().(*machineapi.LifecycleServiceInstallProgress_ExitCode)
		if ok && exit.ExitCode != 0 {
			return fmt.Errorf("talos upgrade installer exited with code %d", exit.ExitCode)
		}
	}
	// Reboot the machine after upgrade
	if _, err := tc.MachineClient.Reboot(ctx, &machineapi.RebootRequest{
		Mode: machineapi.RebootRequest_DEFAULT,
	}); err != nil && !isGracefulStop(err) {
		return fmt.Errorf("error triggering reboot after Talos upgrade: %w", err)
	}
	return nil
}

func (tc *TalosClient) upgradeViaMachineService(ctx context.Context, image string) error {
	req := &machineapi.UpgradeRequest{
		Image:      image,
		Stage:      false,
		Force:      true,
		RebootMode: machineapi.UpgradeRequest_DEFAULT,
	}
	// Deprecated on 1.13 and should be removed at 1.18
	if _, err := tc.MachineClient.Upgrade(ctx, req); err != nil { //nolint:staticcheck
		if isGracefulStop(err) {
			return nil
		}
		return fmt.Errorf("error upgrading machine: %w", err)
	}
	return nil
}

// pullInstallerImage pulls the specificied image as pre-upgrade step.
func (tc *TalosClient) pullInstallerImage(
	ctx context.Context, containerd *common.ContainerdInstance, image string,
) error {
	stream, err := tc.ImageClient.Pull(ctx, &machineapi.ImageServicePullRequest{
		Containerd: containerd,
		ImageRef:   image,
	})
	if err != nil {
		return err
	}
	for {
		_, recvErr := stream.Recv()
		if recvErr == io.EOF {
			return nil
		}
		if recvErr != nil {
			if isGracefulStop(recvErr) {
				return nil
			}
			return recvErr
		}
	}
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
