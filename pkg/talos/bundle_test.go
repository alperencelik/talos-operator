package talos

import (
	"strings"
	"testing"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
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
		ClusterName:   "test-cluster",
		Endpoint:      "https://1.2.3.4:6443",
		Version:       "v1.10.3",
		KubeVersion:   "v1.31.0",
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
}

func TestCidrPatches(t *testing.T) {
	podCIDR := []string{"10.100.0.0/16", "10.150.0.0/16"}
	serviceCIDR := []string{"10.200.0.0/12"}

	patches := cidrPatches(&podCIDR, &serviceCIDR)
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
	patchesNil := cidrPatches(nil, nil)
	if len(patchesNil) != 0 {
		t.Errorf("expected 0 patches for nil input, got %d", len(patchesNil))
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
