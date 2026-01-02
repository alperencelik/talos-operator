package talos

import (
	"testing"
)

func TestGenerateControlPlaneConfig(t *testing.T) {

	cfg := &BundleConfig{
		ClusterName: "test-cluster",
		Endpoint:    "https://10.0.0.1:6443",
		Version:     "v1.10.3",
		KubeVersion: "v1.31.0",
		// SecretsBundle: nil, // For testing purposes, we can set this to nil
		Sans:        []string{"example.com", "api.example.com"},
		PodCIDR:     &[]string{""},
		ServiceCIDR: &[]string{""},
	}
	// Add scheduling in controlplane patch
	patches := &[]string{AllowSchedulingOnControlPlanes}

	_, err := GenerateControlPlaneConfig(cfg, patches)
	if err != nil {
		t.Fatalf("GenerateControlPlaneConfig failed: %v", err)
	}
}
