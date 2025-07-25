package main

import (
	"context"

	"github.com/siderolabs/talos/pkg/machinery/client"
)

func main() {
	ctx := context.Background()
	// Initialize the talos client
	c, err := client.New(ctx, client.WithConfigFromFile("talosconfig"))
	if err != nil {
		panic(err)
	}
	c.Upgrade(ctx, "ghcr.io/siderolabs/installer:v1.10.5", false, false)

}
