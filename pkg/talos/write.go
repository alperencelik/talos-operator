package talos

import (
	"os"

	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"gopkg.in/yaml.v2"
)

func WriteControlPlaneConfig(bundle *bundle.Bundle, dir string) error {
	// Write the control plane configuration to the specified directory
	return bundle.Write(dir, encoder.CommentsDisabled, machine.TypeControlPlane)
}

func WriteTalosConfig(bundle *bundle.Bundle, dir string) error {
	// Write the Talos configuration to the specified directory
	data, err := yaml.Marshal(bundle.TalosConfig())
	if err != nil {
		return err
	}
	return os.WriteFile(dir+"talosconfig.yaml", data, 0o600)
}

func WriteWorkerConfig(bundle *bundle.Bundle, dir string) error {
	// Write the worker configuration to the specified directory
	return bundle.Write(dir, encoder.CommentsDisabled, machine.TypeWorker)
}
