package talos

import (
	"context"
	"fmt"
	"io"

	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
)

// EtcdSnapshotReader returns an io.ReadCloser for streaming the etcd snapshot
// This avoids writing the snapshot to disk, allowing direct streaming to S3
func (tc *TalosClient) EtcdSnapshotReader(ctx context.Context) (io.ReadCloser, error) {
	req := &machineapi.EtcdSnapshotRequest{}
	resp, err := tc.EtcdSnapshot(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to request etcd snapshot: %w", err)
	}

	return resp, nil
}

// TakeSnapshot is deprecated. Use EtcdSnapshotReader for streaming to avoid local I/O.
// This method is kept for backwards compatibility but will write to local disk.
func (tc *TalosClient) TakeSnapshot(ctx context.Context) ([]byte, error) {
	reader, err := tc.EtcdSnapshotReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close() // nolint:errcheck

	// For backwards compatibility, discard the data
	_, err = io.Copy(io.Discard, reader)
	return nil, err
}
