package talos

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	utils "github.com/alperencelik/talos-operator/pkg/utils"
	"github.com/siderolabs/talos/pkg/images"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	taloscni "github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"gopkg.in/yaml.v2"
)

const (
	DefaultTalosImage = "ghcr.io/siderolabs/installer"
)

var (
	// maxVersionContract is the newest config contract the operator generates. Newer contracts move
	// most of the v1alpha1 fields into multi-document configs, which conflict with the v1alpha1 config
	// patches used by the operator and its users. Talos still accepts configs of older contracts.
	maxVersionContract = config.TalosVersion1_13

	removeAdmissionControl = `
cluster:
  apiServer:
    admissionControl:
      $patch: delete
`
	podSubnets = `
cluster:
  network:
    podSubnets: %s
`
	serviceSubnets = `
cluster:
  network:
    serviceSubnets: %s
`
	InstallDisk = `
machine:
  install:
    disk: %s
`
	InstallImage = `
machine:
  install:
    image: %s
`
	WipeDisk = `
machine:
  install:
    wipe: %t
`
	AirGapp = `
machine:
  time:
    disabled: true
cluster:
  discovery:
    enabled: false
`
	AllowSchedulingOnControlPlanes = `
cluster:
  allowSchedulingOnControlPlanes: true
`
	ImageCache = `
machine:
  features:
    imageCache:
      localEnabled: true
`
	ImageCacheVolumeConfig = `
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
	vc, err := versionContract(cfg.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version contract: %w", err)
	}

	genOptions := []generate.Option{
		generate.WithVersionContract(vc),
		generate.WithSecretsBundle(cfg.SecretsBundle),
		generate.WithAdditionalSubjectAltNames(cfg.Sans),
	}

	// Apply the CIDR patches
	cpPatches, err := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CIDR patches: %w", err)
	}
	// Add CNI configuration if provided
	if cfg.CNI != nil {
		patch, err := cniPatch(cfg.CNI)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CNI patch: %w", err)
		}
		cpPatches = append(cpPatches, patch)
	}
	// Apply the removeAdmissionControl patch
	cpPatches = append(cpPatches, removeAdmissionControl)

	// If patches are provided, append them to the control plane patches
	if patches != nil && len(*patches) > 0 {
		cpPatches = append(cpPatches, *patches...)
	}

	b, err := newConfigBundle(cfg, genOptions, cpPatches, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config bundle: %w", err)
	}
	return b, nil
}

func NewWorkerBundle(cfg *BundleConfig, patches *[]string) (*bundle.Bundle, error) {
	// Set up options for the Talos config generation
	vc, err := versionContract(cfg.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version contract: %w", err)
	}
	// DEBUG: Set Clock forcefully
	cfg.SecretsBundle.Clock = NewClock()

	// Get the required info from the ControlPlaneConfig

	genOptions := []generate.Option{
		generate.WithVersionContract(vc),
		generate.WithSecretsBundle(cfg.SecretsBundle),
		generate.WithAdditionalSubjectAltNames(cfg.Sans),
	}

	workerPatches, err := cidrPatches(cfg.PodCIDR, cfg.ServiceCIDR)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CIDR patches: %w", err)
	}
	// Add CNI configuration if provided
	if cfg.CNI != nil {
		patch, err := cniPatch(cfg.CNI)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CNI patch: %w", err)
		}
		workerPatches = append(workerPatches, patch)
	}

	// If patches are provided, append them to the worker patches
	if patches != nil && len(*patches) > 0 {
		workerPatches = append(workerPatches, *patches...)
	}

	b, err := newConfigBundle(cfg, genOptions, nil, workerPatches)
	if err != nil {
		return nil, fmt.Errorf("failed to generate worker config: %w", err)
	}
	return b, nil
}

// newConfigBundle builds a Talos config bundle.
func newConfigBundle(cfg *BundleConfig, genOptions []generate.Option, cpPatches,
	workerPatches []string) (*bundle.Bundle, error) {
	opts := []bundle.Option{
		bundle.WithVerbose(false),
		bundle.WithInputOptions(&bundle.InputOptions{
			ClusterName: cfg.ClusterName,
			Endpoint:    cfg.Endpoint,
			KubeVersion: strings.TrimPrefix(cfg.KubeVersion, "v"),
			GenOptions:  genOptions,
		}),
	}
	if len(cpPatches) > 0 {
		patches, err := configpatcher.LoadPatches(cpPatches)
		if err != nil {
			return nil, fmt.Errorf("error parsing control plane config patch: %w", err)
		}
		opts = append(opts, bundle.WithPatchControlPlane(patches))
	}
	if len(workerPatches) > 0 {
		patches, err := configpatcher.LoadPatches(workerPatches)
		if err != nil {
			return nil, fmt.Errorf("error parsing worker config patch: %w", err)
		}
		opts = append(opts, bundle.WithPatchWorker(patches))
	}
	return bundle.NewBundle(opts...)
}

func TalosConfig(b *bundle.Bundle) *clientconfig.Config {
	return b.TalosConfig()
}

func versionContract(version string) (*config.VersionContract, error) {
	contract, err := config.ParseContractFromVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version contract %q: %w", version, err)
	}
	if contract.Greater(maxVersionContract) {
		return maxVersionContract, nil
	}
	return contract, nil
}

// DefaultInstallerImage returns the installer image to use for the given Talos version when the
// user doesn't provide one. The ghcr.io installer is not published starting with Talos v1.14.
func DefaultInstallerImage(version string) string {
	contract, err := config.ParseContractFromVersion(version)
	if err == nil && contract.Greater(config.TalosVersion1_13) {
		return fmt.Sprintf("%s:%s", images.InstallerImageRepository("metal"), version)
	}
	return fmt.Sprintf("%s:%s", DefaultTalosImage, version)
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

func cidrPatches(podCIDR, serviceCIDR *[]string) ([]string, error) {
	var cidrPatches []string

	if podCIDR != nil && len(*podCIDR) > 0 {
		marshaled, err := utils.MarshalStringSlice(*podCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pod CIDR: %w", err)
		}
		podSubnets := fmt.Sprintf(podSubnets, marshaled)
		cidrPatches = append(cidrPatches, podSubnets)
	}
	if serviceCIDR != nil && len(*serviceCIDR) > 0 {
		marshaled, err := utils.MarshalStringSlice(*serviceCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal service CIDR: %w", err)
		}
		serviceSubnets := fmt.Sprintf(serviceSubnets, marshaled)
		cidrPatches = append(cidrPatches, serviceSubnets)
	}
	return cidrPatches, nil
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

// cniPatch returns a config patch setting the cluster CNI.
func cniPatch(cni *v1alpha1.CNIConfig) (string, error) {
	marshaled, err := yaml.Marshal(map[string]any{
		"cluster": map[string]any{
			"network": map[string]any{
				"cni": convertCNIConfig(cni),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal CNI config: %w", err)
	}
	return string(marshaled), nil
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
			FlanneldExtraArgs:                 cni.Flannel.ExtraArgs,
			FlannelKubeNetworkPoliciesEnabled: cni.Flannel.KubeNetworkPoliciesEnabled,
		}
	}
	return talosCNI
}
