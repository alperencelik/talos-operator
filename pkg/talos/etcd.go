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
