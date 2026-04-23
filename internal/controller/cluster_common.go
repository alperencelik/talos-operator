package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

func updatePxeBootStackConfig(tcList talosv1alpha1.TalosClusterList, machines map[*talosv1alpha1.Machine]string) error {
	// Cleaning up previous configuration
	// Matchbox
	for _, dir := range []string{MatchboxGroupsDir, MatchboxProfilesDir} {
		dirPath := path.Join(MatchboxConfigPath, dir)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := os.Remove(path.Join(dirPath, f.Name())); err != nil {
				return err
			}
		}
	}

	// Generating dnsmasq and Matchbox configurations
	// Base dnsmasq configuration
	var dnsmasqConfigString = BaseDnsmasqConfig
	// These variables map a file name with its content
	var matchboxGroupsMap = make(map[string]string)
	var matchboxProfilesMap = make(map[string]string)
	for _, tc := range tcList.Items {
		if tc.Spec.PxeServerSpec != nil {
			// Global cluster config : tag hosts depending on the interface receiving the request, enable static IP assignment and set iPXE boot file name
			dnsmasqConfigString = fmt.Sprintf(`%s

# %s
tag-if=set:if_%s,tag:%s
dhcp-range=tag:if_%s,%s,static
dhcp-boot=tag:if_%s,tag:ipxe,http://%s:%s/boot.ipxe`,
				dnsmasqConfigString, tc.Name, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Address, *tc.Spec.PxeServerSpec.Interface, *tc.Spec.PxeServerSpec.Address, os.Getenv("MATCHBOX_PORT"),
			)

			// Machines specific config
			// Combining control plane and worker machines in a single array and mapping machines with the version of Talos they require
			if tc.Spec.ControlPlane != nil && tc.Spec.ControlPlane.Mode == TalosModeMetal {
				for _, m := range tc.Spec.ControlPlane.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						machines[&m] = tc.Spec.ControlPlane.Version
					}
				}
			}
			if tc.Spec.Worker != nil && tc.Spec.Worker.Mode == TalosModeMetal {
				for _, m := range tc.Spec.Worker.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						machines[&m] = tc.Spec.Worker.Version
					}
				}
			}
			var machineIndex = 0
			for m, version := range machines {
				// dnsmasq config : static IP addresses assignment
				dnsmasqConfigString = fmt.Sprintf("%s\ndhcp-host=tag:if_%s,%s,%s",
					dnsmasqConfigString, *tc.Spec.PxeServerSpec.Interface, *m.PxeClientSpec.MacAddress, *m.Address,
				)
				// Matchbox config : matching hosts with their MAC address
				var matchboxId = fmt.Sprintf("%s-%d", tc.Name, machineIndex)
				var matchboxFileName = fmt.Sprintf("%s.json", matchboxId)
				matchboxGroupsMap[matchboxFileName] = fmt.Sprintf(`{
	"id": "%s",
	"name": "%s",
	"profile": "%s",
	"selector": {
		"mac": "%s"
	}
}`,
					matchboxId, matchboxId, matchboxId, *m.PxeClientSpec.MacAddress,
				)
				var kernelCmdline string // Additional kernel command line arguments
				if m.PxeClientSpec.KernelCmdlineArgs != nil {
					kernelCmdline = *m.PxeClientSpec.KernelCmdlineArgs
				} else {
					kernelCmdline = ""
				}
				matchboxProfilesMap[matchboxFileName] = fmt.Sprintf(`{
	"id": "%s",
	"name": "%s",
	"boot": {
		"kernel": "/assets/vmlinuz-%s-%s",
		"initrd": ["/assets/initramfs-%s-%s.xz"],
		"args": [
			"initrd=initramfs.xz",
			"slab_nomerge",
			"pti=on",
			"console=tty0",
			"printk.devkmsg=on",
			"talos.platform=metal",
			"%s"
		]
	}
}`,
					matchboxId, matchboxId, version, *m.PxeClientSpec.CpuArchitecture, version, *m.PxeClientSpec.CpuArchitecture, kernelCmdline,
				)
				machineIndex++
			}
		}
	}

	// Saving configs to files
	// dnsmasq
	if err := os.WriteFile(path.Join(DnsmasqConfigPath, DnsmasqConfigFile), []byte(dnsmasqConfigString), os.ModePerm); err != nil {
		return err
	}
	// Matchbox groups
	for file, content := range matchboxGroupsMap {
		if err := os.WriteFile(path.Join(MatchboxConfigPath, MatchboxGroupsDir, file), []byte(content), os.ModePerm); err != nil {
			return err
		}
	}
	// Matchbox profiles
	for file, content := range matchboxProfilesMap {
		if err := os.WriteFile(path.Join(MatchboxConfigPath, MatchboxProfilesDir, file), []byte(content), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func downloadTalosBootImages(machines map[*talosv1alpha1.Machine]string) error {
	// Parsing the machines array to determine every combination of Talos version + CPU architecture to take into account when downloading boot images
	var downloadList = make(map[string]string) // Maps a download URL to a destination path
	for m, version := range machines {
		// Kernel
		downloadList[fmt.Sprintf("%s/%s/vmlinuz-%s", TalosBootImageBaseUrl, version, *m.PxeClientSpec.CpuArchitecture)] = fmt.Sprintf("%s/%s/vmlinuz-%s-%s", MatchboxConfigPath, MatchboxAssetsDir, version, *m.PxeClientSpec.CpuArchitecture)
		// initramfs
		downloadList[fmt.Sprintf("%s/%s/initramfs-%s.xz", TalosBootImageBaseUrl, version, *m.PxeClientSpec.CpuArchitecture)] = fmt.Sprintf("%s/%s/initramfs-%s-%s.xz", MatchboxConfigPath, MatchboxAssetsDir, version, *m.PxeClientSpec.CpuArchitecture)
	}

	// Downloading files if they do not exist yet
	for url, path := range downloadList {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := downloadFile(url, path); err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadFile(url string, path string) error {
	// Create destination file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// Write downloaded file
	if resp.StatusCode == 200 {
		if _, err := io.Copy(file, resp.Body); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unable to download '%s' (HTTP status code: %d)", url, resp.StatusCode)
	}

	return nil
}

func restartPxeBootStack() error {
	// Restart dnsmasq
	// Retrieving PID of process
	var pid = -1
	procs, err := os.ReadDir(ProcPath)
	if err != nil {
		return err
	}
	for _, proc := range procs {
		currentPid, err := strconv.Atoi(proc.Name())
		// If there is an error, then it is not a PID
		if err == nil {
			cmdlinePath := filepath.Join(ProcPath, proc.Name(), ProcCmdlineFile)
			content, err := os.ReadFile(cmdlinePath)
			if err != nil {
				return err
			} else {
				if string(content) == DnsmasqCmdline {
					pid = currentPid
				}
			}
		}
	}
	if pid != -1 {
		// Retrieving process object from PID
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		// Sending SIGTERM to restart dnsmasq
		if err := p.Signal(syscall.SIGTERM); err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("could not find dnsmasq process")
	}
}
