package talos

import (
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
)

func GenerateWorkerConfig(cfg *BundleConfig) (*[]byte, error) {
	// Create a new worker bundle using the provided configuration
	bundle, err := NewWorkerBundle(cfg)
	if err != nil {
		return nil, err
	}
	// // DEBUG
	// dir := fmt.Sprintf("./")
	// bundle.Write(dir, encoder.CommentsDisabled, machine.TypeWorker)
	// // DEBUG END

	// Generate the worker configuration
	bytes, err := bundle.Serialize(encoder.CommentsDisabled, machine.TypeWorker)
	if err != nil {
		return nil, err
	}
	return &bytes, nil
}
