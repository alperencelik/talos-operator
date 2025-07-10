package talos

import (
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
)

func GenerateWorkerConfig(cfg *BundleConfig, patches *[]string) (*[]byte, error) {
	// Create a new worker bundle using the provided configuration
	bundle, err := NewWorkerBundle(cfg, patches)
	if err != nil {
		return nil, err
	}
	// DEBUG
	// err = WriteWorkerConfig(bundle, "./")
	// if err != nil {
	// return nil, err
	// }
	// DEBUG END

	// Generate the worker configuration
	bytes, err := bundle.Serialize(encoder.CommentsDisabled, machine.TypeWorker)
	if err != nil {
		return nil, err
	}
	return &bytes, nil
}
