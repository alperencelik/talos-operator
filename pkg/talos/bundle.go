package talos

import (
	"encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	utils "github.com/alperencelik/talos-operator/pkg/utils"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/gen"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	taloscni "github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
)

const (
	DefaultTalosImage = "ghcr.io/siderolabs/installer"
)

var (
	removeAdmissionControl         = `[{"op": "remove", "path": "/cluster/apiServer/admissionControl"}]`
	podSubnets                     = `[{"op":"replace","path":"/cluster/network/podSubnets","value":%s}]`
	serviceSubnets                 = `[{"op":"replace","path":"/cluster/network/serviceSubnets","value":%s}]`
	InstallDisk                    = `[{"op": "replace", "path": "/machine/install/disk", "value": "%s"}]`
	InstallImage                   = `[{"op": "replace", "path": "/machine/install/image", "value": "%s"}]`
	WipeDisk                       = `[{"op": "replace", "path": "/machine/install/wipe", "value": "%t"}]`
	AirGapp                        = `[{"op": "add", "path": "/machine/time", "value": {"disabled": true}}, {"op": "replace", "path": "/cluster/discovery/enabled", "value": false}]` // nolint:lll
	AllowSchedulingOnControlPlanes = `[{"op": "add", "path": "/cluster/allowSchedulingOnControlPlanes", "value": true}]`
	ImageCache                     = `[{"op": "add", "path": "/machine/features/imageCache", "value": {"localEnabled": true}}]` // nolint:lll
	ImageCacheVolumeConfig         = `
---
apiVersion: v1alpha1
kind: VolumeConfig
name: IMAGECACHE
provisioning:
  diskSelector:
    match: 'system_disk'
`
)

const (
	MaintenanceMode = true
)

type BundleConfig struct {
	ClusterName   string          `json:"clusterName"`    // Name of the Talos cluster
	Endpoint      string          `json:"endpoint"`       // Control plane endpoint for the Talos cluster
	Version       string          `json:"version"`        // Talos version to use
	KubeVersion   string          `json:"kubeVersion"`    // Kubernetes version to use
	SecretsBundle *secrets.Bundle `json:"-"`              // Secrets bundle for the Talos cluster
	Sans          []string        `json:"sans,omitempty"` // Additional Subject Alternative Names for the API server
	//nolint:lll // Description is long
	PodCIDR        *[]string           `json:"podCIDR,omitempty"`        // Pod CIDR ranges
	ServiceCIDR    *[]string           `json:"serviceCIDR,omitempty"`    // Service CIDR ranges
	ClientEndpoint *[]string           `json:"clientEndpoint,omitempty"` // Optional client endpoint for Talos API
	CNI            *v1alpha1.CNIConfig `json:"cni,omitempty"`            // CNI configuration
}

type SecretBundle *secrets.Bundle

func NewCPBundle(cfg *BundleConfig, patches *[]string) (*bundle.Bundle, error) {
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

	// Add CNI configuration if provided
	if cfg.CNI != nil {
		cniConfig := convertCNIConfig(cfg.CNI)
		genOptions = append(genOptions, generate.WithClusterCNIConfig(cniConfig))
	}

	// Apply the CIDR patches
	cpPatches := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)
	// Apply the removeAdmissionControl patch
	cpPatches = append(cpPatches, removeAdmissionControl)

	// If patches are provided, append them to the control plane patches
	if patches != nil && len(*patches) > 0 {
		cpPatches = append(cpPatches, *patches...)
	}

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

func NewWorkerBundle(cfg *BundleConfig, patches *[]string) (*bundle.Bundle, error) {
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

	// Add CNI configuration if provided
	if cfg.CNI != nil {
		cniConfig := convertCNIConfig(cfg.CNI)
		genOptions = append(genOptions, generate.WithClusterCNIConfig(cniConfig))
	}

	workerPatches := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)

	// If patches are provided, append them to the worker patches
	if patches != nil && len(*patches) > 0 {
		workerPatches = append(workerPatches, *patches...)
	}

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

func ParseBundleConfig(bc string) (*BundleConfig, error) {
	// Unmarshal the string into a BundleConfig struct
	var cfg BundleConfig
	err := json.Unmarshal([]byte(bc), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bundle config: %w", err)
	}
	if cfg.ClusterName == "" || cfg.Endpoint == "" || cfg.KubeVersion == "" {
		return nil, fmt.Errorf("invalid bundle config: missing required fields")
	}
	return &cfg, nil
}

// convertCNIConfig converts our CNI config to Talos CNI config
func convertCNIConfig(cni *v1alpha1.CNIConfig) *taloscni.CNIConfig {
	if cni == nil {
		return nil
	}
	talosCNI := &taloscni.CNIConfig{
		CNIName: cni.Name,
		CNIUrls: cni.URLs,
	}
	if cni.Flannel != nil {
		talosCNI.CNIFlannel = &taloscni.FlannelCNIConfig{
			FlanneldExtraArgs: cni.Flannel.ExtraArgs,
		}
	}
	return talosCNI
}
