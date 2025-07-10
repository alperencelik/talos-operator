package talos

import (
	"fmt"

	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
)

// type ControlPlaneConfig struct {
// ClusterName   string
// Endpoint      string
// Version       string
// KubeVersion   string
// SecretsBundle *secrets.Bundle
// Sans          []string  // Additional Subject Alternative Names for the API server
// PodCIDR       *[]string // Pod CIDR ranges
// ServiceCIDR   *[]string // Service CIDR ranges
// }

func GenerateControlPlaneConfig(cfg *BundleConfig, patches *[]string) (*[]byte, error) {

	bundle, err := NewCPBundle(cfg, patches)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config bundle: %w", err)
	}
	// DEBUG
	// err = WriteControlPlaneConfig(bundle, "./")
	// if err != nil {
	// return nil, fmt.Errorf("failed to write control plane config: %w", err)
	// }
	// err = WriteTalosConfig(bundle, "./")
	// if err != nil {
	// return nil, fmt.Errorf("failed to write talos config: %w", err)
	// }
	// DEBUG END

	bytes, err := bundle.Serialize(encoder.CommentsDisabled, machine.TypeControlPlane)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config bundle: %w", err)
	}
	return &bytes, nil
}
