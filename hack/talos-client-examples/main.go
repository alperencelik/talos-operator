package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/siderolabs/go-kubernetes/kubernetes/upgrade"
	"github.com/siderolabs/talos/pkg/cluster"
	k8s "github.com/siderolabs/talos/pkg/cluster/kubernetes"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"google.golang.org/protobuf/types/known/durationpb"
)

type tc struct {
	*client.Client
}

func main() {
	ctx := context.Background()
	// Initialize the talos client
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For testing purposes, skip TLS verification
	}
	_ = tlsConfig
	c, err := client.New(ctx, client.WithConfigFromFile("talosconfig"),
		// client.WithTLSConfig(tlsConfig),
		client.WithDefaultConfig(),
		client.WithEndpoints("10.0.153.137"),
	)
	if err != nil {
		panic(err)
	}

	resp, err := c.Version(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Talos version: %s\n", resp.Messages[0].Version.Tag)

	// Setup a event listener

	stream, err := c.MachineClient.Events(ctx, &machineapi.EventsRequest{})
	if err != nil {
		panic(err)
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			// Print the error but continue listening
			fmt.Printf("Error receiving event: %v\n", err)
			continue
		}
		fmt.Printf("Event is received: %v\n", event)
	}

	// Test upgrade Kubernetes
	//	talosClient := &tc{Client: c}
	//if err := talosClient.upgradeK8s(ctx); err != nil {
	//panic(fmt.Errorf("error upgrading Kubernetes: %w", err))
	//}
	//// Test cert renewal
	//crt, key, err := talosClient.RenewTalosClientCert(ctx)
	//if err != nil {
	//panic(fmt.Errorf("error renewing Talos client certificate: %w", err))
	//}
	//fmt.Printf("New certificate: %s\n", *crt)
	//fmt.Printf("New key: %s\n", *key)
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

func (c *tc) RenewTalosClientCert(ctx context.Context) (*string, *string, error) {
	var roles []string
	roles = append(roles, "os:admin")
	// Set the duration for the certificate
	duration := 8760 * time.Hour // Default to 1 year

	resp, err := c.GenerateClientConfiguration(ctx, &machineapi.GenerateClientConfigurationRequest{
		Roles:  roles,
		CrtTtl: durationpb.New(duration),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error generating client configuration: %w", err)
	}
	// Retrieve the certificate and key from the response
	crt := string(resp.Messages[0].Crt)
	key := string(resp.Messages[0].Key)

	return &crt, &key, nil
}

func (c *tc) SetMetaKey(ctx context.Context, key uint8, value []byte) error {
	// Set the meta key
	return c.MetaWrite(ctx, key, value)
}
