package talos

import (
	"fmt"
	"strings"
	"testing"

	v1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/config/configloader"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/config/validation"
)

const (
	testTalosVersion      = "v1.14.5"
	testKubernetesVersion = "v1.35.0"
	testClusterName       = "test-cluster"
)

func TestParseBundleConfig(t *testing.T) {
	// 1. Valid JSON input
	validJSON := `{
		"clusterName": "my-cluster",
		"endpoint": "https://1.2.3.4:6443",
		"version": "v1.5.0",
		"kubeVersion": "v1.28.0"
	}`

	cfg, err := ParseBundleConfig(validJSON)
	if err != nil {
		t.Fatalf("ParseBundleConfig(valid) failed: %v", err)
	}
	if cfg.ClusterName != "my-cluster" {
		t.Errorf("expected ClusterName 'my-cluster', got %q", cfg.ClusterName)
	}
	if cfg.Endpoint != "https://1.2.3.4:6443" {
		t.Errorf("expected Endpoint 'https://1.2.3.4:6443', got %q", cfg.Endpoint)
	}

	// 2. Invalid JSON
	invalidJSON := `{ "clusterName": "broken" `
	_, err = ParseBundleConfig(invalidJSON)
	if err == nil {
		t.Error("ParseBundleConfig(invalid JSON) expected error, got nil")
	}

	// 3. Missing required fields
	missingFields := `{
		"clusterName": "my-cluster"
	}`
	// ParseBundleConfig checks for Endpoint and KubeVersion
	_, err = ParseBundleConfig(missingFields)
	if err == nil {
		t.Error("ParseBundleConfig(missing fields) expected error, got nil")
	} else if err.Error() != "invalid bundle config: missing required fields" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewClock(t *testing.T) {
	c := NewClock()
	if c == nil {
		t.Error("NewClock() returned nil")
	}
	// Verify it returns 'now' roughly
	now := c.Now()
	if now.IsZero() {
		t.Error("NewClock().Now() returned zero time")
	}
}

func TestGenerateConfigs(t *testing.T) {
	// Setup SecretsBundle
	// NewSecretBundle returns (SecretBundle, error) where SecretBundle is *secrets.Bundle
	sb, err := NewSecretBundle()
	if err != nil {
		t.Fatalf("NewSecretBundle() failed: %v", err)
	}

	// Create a BundleConfig
	cfg := &BundleConfig{
		ClusterName:   testClusterName,
		Endpoint:      "https://1.2.3.4:6443",
		Version:       testTalosVersion,
		KubeVersion:   testKubernetesVersion,
		SecretsBundle: (*secrets.Bundle)(sb),
	}

	t.Run("GenerateControlPlaneConfig", func(t *testing.T) {
		configBytes, err := GenerateControlPlaneConfig(cfg, nil)
		if err != nil {
			t.Fatalf("GenerateControlPlaneConfig failed: %v", err)
		}
		if configBytes == nil || len(*configBytes) == 0 {
			t.Error("GenerateControlPlaneConfig returned empty config")
		}
		// We could try to parse it back to verify it's valid YAML, but checking non-empty is a good start
		// for integration/unit boundary.
	})

	t.Run("GenerateWorkerConfig", func(t *testing.T) {
		configBytes, err := GenerateWorkerConfig(cfg, nil)
		if err != nil {
			t.Fatalf("GenerateWorkerConfig failed: %v", err)
		}
		if configBytes == nil || len(*configBytes) == 0 {
			t.Error("GenerateWorkerConfig returned empty config")
		}
	})
}

func TestVersionContract(t *testing.T) {
	validVersion := "v1.10.4"
	contract, err := versionContract(validVersion)
	if err != nil {
		t.Fatalf("versionContract(valid) failed: %v", err)
	}
	if contract.Major != 1 || contract.Minor != 10 {
		t.Errorf("unexpected contract parsed: %+v", contract)
	}

	invalidVersion := "invalid-version"
	_, err = versionContract(invalidVersion)
	if err == nil {
		t.Error("versionContract(invalid) expected error, got nil")
	}

	// Versions newer than maxVersionContract must be capped
	for _, version := range []string{"v1.14.5", "v1.15.0-alpha.2", "v2.0.0"} {
		contract, err := versionContract(version)
		if err != nil {
			t.Fatalf("versionContract(%s) failed: %v", version, err)
		}
		if contract != maxVersionContract {
			t.Errorf("versionContract(%s) = %+v, want %+v", version, contract, maxVersionContract)
		}
	}
}

func TestDefaultInstallerImage(t *testing.T) {
	factoryInstaller := "factory.talos.dev/metal-installer/" +
		"376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
	tests := []struct {
		version string
		want    string
	}{
		{"v1.12.12", DefaultTalosImage + ":v1.12.12"},
		{"v1.13.12", DefaultTalosImage + ":v1.13.12"},
		{"v1.14.0", factoryInstaller + ":v1.14.0"},
		{"v1.14.5", factoryInstaller + ":v1.14.5"},
		{"v1.15.0-alpha.2", factoryInstaller + ":v1.15.0-alpha.2"},
	}
	for _, tt := range tests {
		if got := DefaultInstallerImage(tt.version); got != tt.want {
			t.Errorf("DefaultInstallerImage(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

// validationMode is a metal mode which requires an install section, like the machines managed by the operator.
type validationMode struct{}

func (validationMode) String() string        { return "metal" }
func (validationMode) RequiresInstall() bool { return true }
func (validationMode) InContainer() bool     { return false }

// TestGeneratedConfigsAreValid generates configs with every patch the operator can
// apply and makes sure the machinery of the pinned Talos version accepts them.
func TestGeneratedConfigsAreValid(t *testing.T) {
	sb, err := NewSecretBundle()
	if err != nil {
		t.Fatalf("NewSecretBundle() failed: %v", err)
	}
	podCIDR, serviceCIDR := []string{"10.100.0.0/16"}, []string{"10.200.0.0/16"}
	patches := []string{
		fmt.Sprintf(InstallDisk, "/dev/sda"),
		fmt.Sprintf(InstallImage, DefaultInstallerImage(testTalosVersion)),
		fmt.Sprintf(WipeDisk, true),
		AirGapp,
		AllowSchedulingOnControlPlanes,
		ImageCache,
	}
	versions := []string{"v1.8.4", "v1.9.6", "v1.10.8", "v1.11.6", "v1.12.12", "v1.13.12", "v1.14.5", "v1.15.0-alpha.2"}
	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			cfg := &BundleConfig{
				ClusterName:   testClusterName,
				Endpoint:      "https://1.2.3.4:6443",
				Version:       version,
				KubeVersion:   testKubernetesVersion,
				SecretsBundle: (*secrets.Bundle)(sb),
				PodCIDR:       &podCIDR,
				ServiceCIDR:   &serviceCIDR,
				CNI:           &v1alpha1.CNIConfig{Name: "none"},
			}
			cpConfig, err := GenerateControlPlaneConfig(cfg, &patches)
			if err != nil {
				t.Fatalf("GenerateControlPlaneConfig failed: %v", err)
			}
			workerConfig, err := GenerateWorkerConfig(cfg, &patches)
			if err != nil {
				t.Fatalf("GenerateWorkerConfig failed: %v", err)
			}
			for _, data := range [][]byte{*cpConfig, *workerConfig} {
				loaded, err := configloader.NewFromBytes(data)
				if err != nil {
					t.Fatalf("failed to load generated config: %v", err)
				}
				if _, err := loaded.ValidateAsClient(validationMode{}, validation.WithLocal()); err != nil {
					t.Errorf("generated config is invalid: %v", err)
				}
			}
		})
	}
}

func TestCNIPatch(t *testing.T) {
	sb, err := NewSecretBundle()
	if err != nil {
		t.Fatalf("NewSecretBundle() failed: %v", err)
	}
	policies := true
	tests := []struct {
		name string
		cni  *v1alpha1.CNIConfig
		want []string
	}{
		{"none", &v1alpha1.CNIConfig{Name: "none"}, []string{"name: none"}},
		{
			"custom",
			&v1alpha1.CNIConfig{Name: "custom", URLs: []string{"https://example.com/cni.yaml"}},
			[]string{"name: custom", "- https://example.com/cni.yaml"},
		},
		{
			"flannel",
			&v1alpha1.CNIConfig{Name: "flannel", Flannel: &v1alpha1.FlannelCNIConfig{
				ExtraArgs:                  []string{"--iface=eth1"},
				KubeNetworkPoliciesEnabled: &policies,
			}},
			[]string{"name: flannel", "- --iface=eth1", "kubeNetworkPoliciesEnabled: true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BundleConfig{
				ClusterName:   testClusterName,
				Endpoint:      "https://1.2.3.4:6443",
				Version:       testTalosVersion,
				KubeVersion:   testKubernetesVersion,
				SecretsBundle: (*secrets.Bundle)(sb),
				CNI:           tt.cni,
			}
			for kind, generate := range map[string]func(*BundleConfig, *[]string) (*[]byte, error){
				"controlplane": GenerateControlPlaneConfig,
				"worker":       GenerateWorkerConfig,
			} {
				data, err := generate(cfg, nil)
				if err != nil {
					t.Fatalf("generating %s config failed: %v", kind, err)
				}
				for _, want := range tt.want {
					if !strings.Contains(string(*data), want) {
						t.Errorf("%s config does not contain %q", kind, want)
					}
				}
			}
		})
	}
}

func TestCidrPatches(t *testing.T) {
	podCIDR := []string{"10.100.0.0/16", "10.150.0.0/16"}
	serviceCIDR := []string{"10.200.0.0/12"}

	patches, err := cidrPatches(&podCIDR, &serviceCIDR)
	if err != nil {
		t.Fatalf("cidrPatches() unexpected error: %v", err)
	}
	if len(patches) != 2 {
		t.Errorf("expected 2 CIDR patches, got %d", len(patches))
	}

	// Verify content
	// We expect the patches to be formatted strings using the package-level vars
	// Since those vars are unexported, we can't easily verify exact string equality without duplicating logic,
	// checking for containment of the CIDRs is a good proxy.

	foundPod := false
	foundService := false

	for _, p := range patches {
		if containsStr(p, "10.100.0.0/16") && containsStr(p, "10.150.0.0/16") {
			foundPod = true
		}
		if containsStr(p, "10.200.0.0/12") {
			foundService = true
		}
	}

	if !foundPod {
		t.Error("Pod CIDR patch not found or incorrect")
	}
	if !foundService {
		t.Error("Service CIDR patch not found or incorrect")
	}

	// Test nil input
	patchesNil, err := cidrPatches(nil, nil)
	if err != nil {
		t.Fatalf("cidrPatches(nil, nil) unexpected error: %v", err)
	}
	if len(patchesNil) != 0 {
		t.Errorf("expected 0 patches for nil input, got %d", len(patchesNil))
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
