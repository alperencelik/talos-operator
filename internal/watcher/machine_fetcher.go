// Package watcher implements kube-external-watcher fetchers for the
// talos-operator resources, replacing fixed-interval RequeueAfter polling with
// drift-triggered reconciliation (see docs/proposals/0001-event-driven-reconciliation.md).
package watcher

import (
	"context"
	"fmt"
	"strings"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/alperencelik/talos-operator/pkg/talos"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	watcher "github.com/alperencelik/kube-external-watcher/watcher"
)

// MachineState represents the comparable desired/actual state for a TalosMachine.
// Checks for Version and MachineConfig diff.
type MachineState struct {
	Version    string
	ConfigDiff string
}

type MachineStateComparator struct{}

var _ watcher.StateComparator = MachineStateComparator{}

func (MachineStateComparator) HasDrifted(desired, actual any) (bool, error) {
	d, ok := desired.(MachineState)
	if !ok {
		return false, fmt.Errorf("unexpected desired state type: %T", desired)
	}
	a, ok := actual.(MachineState)
	if !ok {
		return false, fmt.Errorf("unexpected actual state type: %T", actual)
	}
	return d.Version != a.Version || a.ConfigDiff != "", nil
}

func (MachineStateComparator) Diff(desired, actual any) string {
	d, dok := desired.(MachineState)
	a, aok := actual.(MachineState)
	if !dok || !aok {
		return ""
	}
	var sb strings.Builder
	if d.Version != a.Version {
		fmt.Fprintf(&sb, "version: desired %s, node reports %s", d.Version, a.Version)
	}
	if a.ConfigDiff != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("config diff reported by node:\n")
		sb.WriteString(a.ConfigDiff)
	}
	return sb.String()
}

// ConfigResolver resolves the Talos client bundle and the desired machine
// configuration for a machine.
type ConfigResolver interface {
	GetBundleConfig(ctx context.Context, tm *talosv1alpha1.TalosMachine) (*talos.BundleConfig, error)
	BuildDesiredConfig(ctx context.Context, tm *talosv1alpha1.TalosMachine, bc *talos.BundleConfig) (*[]byte, error)
}

// TalosMachineFetcher implements watcher.ResourceStateFetcher for TalosMachine resources.
type TalosMachineFetcher struct {
	Reader   client.Reader
	Resolver ConfigResolver
	Log      logr.Logger
}

var _ watcher.ResourceStateFetcher = (*TalosMachineFetcher)(nil)

func (f *TalosMachineFetcher) GetDesiredState(ctx context.Context, key types.NamespacedName) (any, error) {
	tm, err := f.get(ctx, key)
	if err != nil {
		return nil, err
	}
	// The desired side is always in sync (empty diff); a node reporting
	// otherwise has drifted.
	return MachineState{Version: tm.Spec.Version}, nil
}

// FetchExternalResource connects to the machine's Talos endpoint and reads the
// node's reported version plus whether the desired config is already applied
func (f *TalosMachineFetcher) FetchExternalResource(ctx context.Context, objKey any) (any, error) {
	key, ok := objKey.(types.NamespacedName)
	if !ok {
		return nil, fmt.Errorf("unexpected resource key type: %T", objKey)
	}
	tm, err := f.get(ctx, key)
	if err != nil {
		return nil, err
	}
	bc, err := f.Resolver.GetBundleConfig(ctx, tm)
	if err != nil {
		return nil, fmt.Errorf("resolve bundle config for %s: %w", key, err)
	}
	if bc == nil {
		return nil, fmt.Errorf("bundle config not ready for %s", key)
	}
	desiredConfig, err := f.Resolver.BuildDesiredConfig(ctx, tm, bc)
	if err != nil {
		return nil, fmt.Errorf("build desired config for %s: %w", key, err)
	}
	// Target this specific machine, not the cluster endpoint.
	bc.ClientEndpoint = &[]string{tm.Spec.Endpoint}
	tc, err := talos.NewClient(ctx, bc, false)
	if err != nil {
		return nil, fmt.Errorf("create talos client for %s: %w", key, err)
	}
	defer tc.Close() //nolint:errcheck

	version, err := tc.GetTalosVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get talos version for %s: %w", key, err)
	}
	diff, err := tc.DryRunConfigDiff(ctx, *desiredConfig)
	if err != nil {
		return nil, fmt.Errorf("dry-run config diff for %s: %w", key, err)
	}
	if version != tm.Spec.Version || diff != "" {
		// Log the specific divergence so recurring drift is diagnosable.
		f.Log.Info("TalosMachine drift detected",
			"machine", key,
			"desiredVersion", tm.Spec.Version,
			"observedVersion", version,
			"configDiff", diff,
		)
	}
	return MachineState{Version: version, ConfigDiff: diff}, nil
}

// TransformExternalState is the identity transform: FetchExternalResource
// already returns the normalized MachineState shape.
func (f *TalosMachineFetcher) TransformExternalState(raw any) (any, error) {
	return raw, nil
}

// IsResourceReadyToWatch reports whether a machine should be watched for drift.
// Only "Available" machines are watched.
func (f *TalosMachineFetcher) IsResourceReadyToWatch(ctx context.Context, key types.NamespacedName) bool {
	tm, err := f.get(ctx, key)
	if err != nil {
		return false
	}
	if !tm.DeletionTimestamp.IsZero() {
		return false
	}
	return tm.Status.State == talosv1alpha1.StateAvailable && tm.Spec.Endpoint != ""
}

func (f *TalosMachineFetcher) get(ctx context.Context, key types.NamespacedName) (*talosv1alpha1.TalosMachine, error) {
	var tm talosv1alpha1.TalosMachine
	if err := f.Reader.Get(ctx, key, &tm); err != nil {
		return nil, err
	}
	return &tm, nil
}

// TalosMachineConfigExtractor extracts watcher.ResourceConfig from a TalosMachine object.
func TalosMachineConfigExtractor(obj client.Object) watcher.ResourceConfig {
	return watcher.ResourceConfig{
		ResourceKey: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()},
	}
}
