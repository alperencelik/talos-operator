package talos

import (
	"testing"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

func TestGenerateWorkerConfig(t *testing.T) {

	sb, err := NewSecretBundle()
	if err != nil {
		t.Fatalf("NewSecretBundle failed: %v", err)
	}
	// Create a new worker bundle using the provided configuration
	cfg := &BundleConfig{
		ClusterName:   "test-cluster",
		Endpoint:      "https://10.0.0.1:6443",
		Version:       "v1.10.3",
		KubeVersion:   "v1.31.0",
		SecretsBundle: (*secrets.Bundle)(sb),
		Sans:          []string{"example.com", "api.example.com"},
		PodCIDR:       &[]string{""},
		ServiceCIDR:   &[]string{""},
	}

	_, err = GenerateWorkerConfig(cfg, nil)
	if err != nil {
		t.Fatalf("GenerateWorkerConfig failed: %v", err)
	}

}
