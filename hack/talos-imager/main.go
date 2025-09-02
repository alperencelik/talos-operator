package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/siderolabs/talos/pkg/imager"
	"github.com/siderolabs/talos/pkg/imager/profile"
	"github.com/siderolabs/talos/pkg/reporter"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

var version = "v1.10.5"

func run() error {
	// Create a temporary directory for boot assets.
	tmpDir, err := os.MkdirTemp("", "talos-iso-creator")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download boot assets.
	// kernelPath, err := downloadAsset(tmpDir, "vmlinuz-amd64", fmt.Sprintf("https://github.com/siderolabs/talos/releases/download/%s/initramfs-amd64.xz", version))
	// if err != nil {
	// return fmt.Errorf("failed to download kernel: %w", err)
	// }

	// initramfsPath, err := downloadAsset(tmpDir, "initramfs-amd64.xz", fmt.Sprintf("https://github.com/siderolabs/talos/releases/download/%s/initramfs-amd64.xz", version))
	// if err != nil {
	// return fmt.Errorf("failed to download initramfs: %w", err)
	// }

	// SDStubPath, err := downloadAsset(tmpDir, "sd-stub.efi", fmt.Sprintf("https://github.com/siderolabs/talos/releases/download/%s/metal-amd64-uki.efi", version))
	// if err != nil {
	// return fmt.Errorf("failed to download sd-stub: %w", err)
	// }

	// SDBootPath, err := downloadAsset(tmpDir, "sd-boot.efi", fmt.Sprintf("https://github.com/siderolabs/talos/releases/download/%s/metal-amd64-uki.efi", version))
	// if err != nil {
	// return fmt.Errorf("failed to download sd-boot: %w", err)
	// }

	// Create a profile for ISO generation.
	prof := profile.Profile{
		Arch:       runtime.GOARCH,
		Platform:   "metal",
		Version:    version,
		SecureBoot: boolPtr(false),
		Output: profile.Output{
			Kind:      profile.OutKindISO,
			OutFormat: profile.OutFormatRaw,
		},
		Input: profile.Input{
			Kernel: profile.FileAsset{
				Path: "./vmlinuz-amd64",
			},
			Initramfs: profile.FileAsset{
				Path: "./initramfs-amd64.xz",
			},
			SDBoot: profile.FileAsset{
				Path: "./sd-boot-amd64.efi",
			},
			SDStub: profile.FileAsset{
				Path: "./sd-stub-amd64.efi",
			},
			BaseInstaller: profile.ContainerAsset{
				ImageRef: fmt.Sprintf("ghcr.io/siderolabs/talos:installer-base:%s", version),
			},
		},
	}

	// Create a new imager.
	img, err := imager.New(prof)
	if err != nil {
		return fmt.Errorf("failed to create imager: %w", err)
	}

	// Create an output directory.
	outputDir := "./output"
	err = os.Mkdir(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Execute the imager.
	if _, err := img.Execute(context.Background(), outputDir, reporter.New()); err != nil {
		return fmt.Errorf("failed to execute imager: %w", err)
	}

	fmt.Printf("ISO created successfully in %s\n", outputDir)

	return nil
}

func downloadAsset(dir, name, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return path, nil
}

func boolPtr(b bool) *bool {
	return &b
}
