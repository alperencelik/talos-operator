package main

import (
	"context"
	"fmt"

	"github.com/siderolabs/go-kubernetes/kubernetes/upgrade"
	"github.com/siderolabs/talos/pkg/cluster"
	k8s "github.com/siderolabs/talos/pkg/cluster/kubernetes"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

type tc struct {
	*client.Client
}

func main() {
	ctx := context.Background()
	// Initialize the talos client
	c, err := client.New(ctx, client.WithConfigFromFile("talosconfig"))
	if err != nil {
		panic(err)
	}
	resp, err := c.Version(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Talos version: %s\n", resp.Messages[0].Version.Tag)

	// Test upgrade Kubernetes
	talosClient := &tc{Client: c}
	if err := talosClient.upgradeK8s(ctx); err != nil {
		panic(fmt.Errorf("error upgrading Kubernetes: %w", err))
	}
}

func (c *tc) upgradeK8s(ctx context.Context) error {

	kubernetesEndpoint := "https://10.0.153.95:6443" // os.Getenv("KUBERNETES_ENDPOINT")
	// 	c.Upgrade(ctx, "ghcr.io/siderolabs/installer:v1.10.5", false, false)
	var upgradeOpts k8s.UpgradeOptions

	upgradeOpts.DryRun = true

	clientProvider := &cluster.ConfigClientProvider{
		DefaultClient: c.Client,
	}
	kubernetesProvider := &cluster.KubernetesClient{
		ClientProvider: clientProvider,
		ForceEndpoint:  kubernetesEndpoint,
	}
	// Use the kubernetes provider to create a new cluster upgrader
	upgradeProvider := struct {
		cluster.ClientProvider
		cluster.K8sProvider
	}{
		ClientProvider: clientProvider,
		K8sProvider:    kubernetesProvider,
	}

	fromVersion, _ := k8s.DetectLowestVersion(ctx, &upgradeProvider, upgradeOpts)
	toVersion := constants.DefaultKubernetesVersion

	upgradeOpts.Path, _ = upgrade.NewPath(fromVersion, toVersion)

	upgradeOpts.UpgradeKubelet = true
	// upgradeOpts.PrePullImages = true
	upgradeOpts.EncoderOpt = encoder.WithComments(0)

	upgradeOpts.KubeletImage = constants.KubeletImage
	upgradeOpts.APIServerImage = constants.KubernetesAPIServerImage
	upgradeOpts.ControllerManagerImage = constants.KubernetesControllerManagerImage
	upgradeOpts.SchedulerImage = constants.KubernetesSchedulerImage
	upgradeOpts.ProxyImage = constants.KubeProxyImage

	err := k8s.Upgrade(ctx, upgradeProvider, upgradeOpts)
	if err != nil {
		return fmt.Errorf("error upgrading Kubernetes: %w", err)
	}
	return nil
}
