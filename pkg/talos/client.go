// Package talos provides a programmatic interface to bootstrap Talos control plane nodes.
package talos

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

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
	bundle, err := NewCPBundle(cfg)
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
	_, err := tc.Client.ApplyConfiguration(ctx, applyRequest)
	if err != nil {
		return fmt.Errorf("Error applying new configration %s", err)
	}

	return nil
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
