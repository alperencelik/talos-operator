package talos

import (
	"context"
	"fmt"

	"github.com/siderolabs/go-kubernetes/kubernetes/upgrade"
	"github.com/siderolabs/talos/pkg/cluster"
	tk8s "github.com/siderolabs/talos/pkg/cluster/kubernetes"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

func buildUpgradeOptions(currentVersion, targetVersion string) (tk8s.UpgradeOptions, error) {
	var upgradeOpts tk8s.UpgradeOptions
	var err error

	upgradeOpts.Path, err = upgrade.NewPath(currentVersion, targetVersion)
	if err != nil {
		return upgradeOpts, fmt.Errorf("error creating upgrade path: %w", err)
	}

	if !upgradeOpts.Path.IsSupported() {
		return upgradeOpts, fmt.Errorf("upgrade path %s to %s is not supported", currentVersion, targetVersion)
	}

	// Set the upgrade options
	upgradeOpts.UpgradeKubelet = true
	upgradeOpts.EncoderOpt = encoder.WithComments(0) // Disable comments in the generated config
	upgradeOpts.KubeletImage = constants.KubeletImage
	upgradeOpts.APIServerImage = constants.KubernetesAPIServerImage
	upgradeOpts.ControllerManagerImage = constants.KubernetesControllerManagerImage
	upgradeOpts.SchedulerImage = constants.KubernetesSchedulerImage
	upgradeOpts.ProxyImage = constants.KubeProxyImage

	return upgradeOpts, nil
}

func (tc *TalosClient) UpgradeKubeVersion(ctx context.Context, version string, endpoint string) error {

	clientProvider := &cluster.ConfigClientProvider{
		DefaultClient: tc.Client,
	}

	kubernetesProvider := &cluster.KubernetesClient{
		ClientProvider: clientProvider,
		ForceEndpoint:  endpoint,
	}
	upgradeProvider := struct {
		cluster.ClientProvider
		cluster.K8sProvider
	}{
		ClientProvider: clientProvider,
		K8sProvider:    kubernetesProvider,
	}
	// Check whether the version update path is valid
	currentVersion, err := tk8s.DetectLowestVersion(ctx, &upgradeProvider, tk8s.UpgradeOptions{})
	if err != nil {
		return fmt.Errorf("error detecting lowest version: %w", err)
	}

	upgradeOpts, err := buildUpgradeOptions(currentVersion, version)
	if err != nil {
		return err
	}

	if err := tk8s.Upgrade(ctx, upgradeProvider, upgradeOpts); err != nil {
		return fmt.Errorf("error upgrading Kubernetes: %w", err)
	}
	return nil
}
