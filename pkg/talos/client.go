// Package talos provides a programmatic interface to bootstrap Talos control plane nodes.
package talos

import (
	"context"
	"fmt"

	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
)

type TalosClient struct {
	*client.Client
}

// NewClient constructs a Talos API client using the default talosconfig file
// and context name. It returns an initialized client or an error.
func NewClient(cfg *BundleConfig) (*TalosClient, error) {

	ctx := context.Background()
	bundle, err := NewCPBundle(cfg)
	if err != nil {
		return nil, err
	}
	c, err := client.New(ctx,
		client.WithEndpoints(fmt.Sprintf("%s", cfg.ClusterName)),
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
