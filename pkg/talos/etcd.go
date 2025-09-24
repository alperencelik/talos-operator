package talos

import (
	"context"
	"crypto/sha256"
	"io"
	"os"

	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
)

func (tc *TalosClient) TakeSnapshot(ctx context.Context) ([]byte, error) {
	req := &machineapi.EtcdSnapshotRequest{}
	resp, err := tc.EtcdSnapshot(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	// target
	dest, err := os.OpenFile("./.part", os.O_CREATE|os.O_WRONLY, 0600)

	size, err := io.Copy(dest, resp)

	if err = dest.Sync(); err != nil {
		return nil, err
	}

	if (size % 512) != sha256.Size {
		return nil, io.ErrUnexpectedEOF
	}

	if err = dest.Close(); err != nil {
		return nil, err
	}
	if err = os.Rename("./.part", "./etcd-snapshot.db"); err != nil {
		return nil, err
	}

	return nil, err
}
