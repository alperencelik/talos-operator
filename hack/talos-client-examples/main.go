package main

import (
	"context"
	"fmt"

	"github.com/siderolabs/talos/pkg/machinery/client"
)

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

	// 	c.Upgrade(ctx, "ghcr.io/siderolabs/installer:v1.10.5", false, false)

}
