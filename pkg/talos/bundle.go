package talos

import (
	"fmt"
	"time"

	utils "github.com/alperencelik/talos-operator/pkg/utils"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/gen"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

var (
	removeAdmissionControl = `[{"op": "remove", "path": "/cluster/apiServer/admissionControl"}]`
	podSubnets             = `[{"op":"replace","path":"/cluster/network/podSubnets","value":%s}]`
	serviceSubnets         = `[{"op":"replace","path":"/cluster/network/serviceSubnets","value":%s}]`
)

type BundleConfig struct {
	ClusterName   string
	Endpoint      string
	Version       string
	KubeVersion   string
	SecretsBundle *secrets.Bundle
	Sans          []string  // Additional Subject Alternative Names for the API server
	PodCIDR       *[]string // Pod CIDR ranges
	ServiceCIDR   *[]string // Service CIDR ranges
}

type SecretBundle *secrets.Bundle

func NewCPBundle(cfg *BundleConfig) (*bundle.Bundle, error) {
	// Set up options for the Talos config generation
	var genOptions []generate.Option
	vc, err := versionContract(cfg.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version contract: %w", err)
	}

	genOptions = append(genOptions,
		generate.WithVersionContract(vc),
		generate.WithSecretsBundle(cfg.SecretsBundle),
		generate.WithAdditionalSubjectAltNames(cfg.Sans),
	)

	// Apply the CIDR patches
	cpPatches := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)
	// Apply the removeAdmissionControl patch
	cpPatches = append(cpPatches, removeAdmissionControl)

	b, err := gen.GenerateConfigBundle(genOptions,
		cfg.ClusterName, // Cluster name
		cfg.Endpoint,    // API endpoint
		cfg.KubeVersion, // Kubernetes version
		[]string{},
		cpPatches, // Control plane patches
		[]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config bundle: %w", err)
	}
	return b, nil
}

func NewWorkerBundle(cfg *BundleConfig) (*bundle.Bundle, error) {
	// Set up options for the Talos config generation
	var genOptions []generate.Option
	vc, err := versionContract(cfg.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version contract: %w", err)
	}
	// DEBUG: Set Clock forcefully
	cfg.SecretsBundle.Clock = NewClock()

	// Get the required info from the ControlPlaneConfig

	genOptions = append(genOptions,
		generate.WithVersionContract(vc),
		generate.WithSecretsBundle(cfg.SecretsBundle),
		generate.WithAdditionalSubjectAltNames(cfg.Sans),
	)

	workerPatches := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)

	b, err := gen.GenerateConfigBundle(genOptions,
		cfg.ClusterName, // Cluster name
		cfg.Endpoint,    // API endpoint
		cfg.KubeVersion, // Kubernetes version
		[]string{},
		[]string{},    // No control plane patches for worker nodes
		workerPatches, // Worker patches
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate worker config: %w", err)
	}
	return b, nil
}

func TalosConfig(b *bundle.Bundle) *clientconfig.Config {
	return b.TalosConfig()
}

func versionContract(version string) (*config.VersionContract, error) {
	contract, err := config.ParseContractFromVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version contract %q: %w", version, err)
	}
	return contract, nil
}

func NewSecretBundle() (SecretBundle, error) {
	bundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new secrets bundle: %w", err)
	}
	return bundle, nil
}

func NewClock() secrets.Clock {
	return secrets.NewClock()
}

func cidrPatches(podCIDR, serviceCIDR *[]string) []string {
	var cidrPatches []string

	if podCIDR != nil && len(*podCIDR) > 0 {
		podSubnets := fmt.Sprintf(podSubnets, utils.MarshalStringSlice(*podCIDR))
		cidrPatches = append(cidrPatches, podSubnets)
	}
	if serviceCIDR != nil && len(*serviceCIDR) > 0 {
		serviceSubnets := fmt.Sprintf(serviceSubnets, utils.MarshalStringSlice(*serviceCIDR))
		cidrPatches = append(cidrPatches, serviceSubnets)
	}
	return cidrPatches
}
