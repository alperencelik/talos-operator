package talos

import (
	"fmt"
	"os"

	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"gopkg.in/yaml.v2"
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
	dir := fmt.Sprintf("./")
	bundle.Write(dir, encoder.CommentsDisabled, machine.TypeControlPlane)
	// Save the talosconfig to a file
	data, err := yaml.Marshal(bundle.TalosConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Talos config: %w", err)
	}
	if err := os.WriteFile(dir+"talosconfig.yaml", data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write Talos config to file: %w", err)
	}
	// DEBUG END

	bytes, err := bundle.Serialize(encoder.CommentsDisabled, machine.TypeControlPlane)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config bundle: %w", err)
	}
	return &bytes, nil
}
