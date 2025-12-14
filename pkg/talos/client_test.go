package talos

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

func TestMetaKeyTemplateRendering(t *testing.T) {
	// Access the unexported metaKeyTemplate constant
	tplText := metaKeyTemplate

	tmpl, err := template.New("test").Parse(tplText)
	if err != nil {
		t.Fatalf("Failed to parse metaKeyTemplate: %v", err)
	}

	data := struct {
		*talosv1alpha1.META
		Endpoint string
	}{
		META: &talosv1alpha1.META{
			Interface:  "eth0",
			Subnet:     24,
			Gateway:    "192.168.1.1",
			Hostname:   "test-node",
			DNSServers: []string{"8.8.8.8", "1.1.1.1"},
		},
		Endpoint: "192.168.1.10",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	output := buf.String()

	// Verify key elements exist in the output
	expectedSubstrings := []string{
		"address: 192.168.1.10/24",
		"linkName: eth0",
		"gateway: 192.168.1.1",
		"hostname: test-node",
		"- 8.8.8.8",
		"- 1.1.1.1",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(output, s) {
			t.Errorf("Output missing expected substring %q. Got:\n%s", s, output)
		}
	}
}
