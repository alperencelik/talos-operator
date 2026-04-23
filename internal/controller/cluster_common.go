package controller

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

// Represents a Talos machine's PXE specifications
type Machine struct {
	Id                string
	MacAddress        string
	IpAddress         string
	TalosVersion      string
	CpuArchitecture   string
	KernelCmdlineArgs string
}

// Represents a Talos cluster's PXE specifications
type Cluster struct {
	Name         string
	PxeInterface string
	PxeIpAddress string
	MatchboxPort string
	Machines     []Machine
}

// Loading configuration templates
//
//go:embed templates/dnsmasq-config.conf
var dnsmasqConfigTmpl string

//go:embed templates/matchbox-group.json
var matchboxGroupTmpl string

//go:embed templates/matchbox-profile.json
var matchboxProfileTmpl string

func getClustersPxeSpecs(tcList talosv1alpha1.TalosClusterList) []Cluster {
	var clusters []Cluster
	for i, tc := range tcList.Items {
		if tc.Spec.PxeServerSpec != nil {
			// Creating a structure for this cluster
			clusters = append(clusters, Cluster{
				tc.Name,
				*tc.Spec.PxeServerSpec.Interface,
				*tc.Spec.PxeServerSpec.Address,
				os.Getenv("MATCHBOX_PORT"),
				make([]Machine, 0),
			})
			// Adding machines in the cluster's structure
			var machineIndex = 0
			// Control plane machines
			if tc.Spec.ControlPlane != nil && tc.Spec.ControlPlane.Mode == TalosModeMetal {
				for _, m := range tc.Spec.ControlPlane.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						var kernelCmdline string
						if m.PxeClientSpec.KernelCmdlineArgs != nil {
							kernelCmdline = *m.PxeClientSpec.KernelCmdlineArgs
						} else {
							kernelCmdline = ""
						}
						clusters[i].Machines = append(clusters[i].Machines, Machine{
							fmt.Sprintf("%s-%d", tc.Name, machineIndex),
							*m.PxeClientSpec.MacAddress,
							*m.Address,
							tc.Spec.ControlPlane.Version,
							*m.PxeClientSpec.CpuArchitecture,
							kernelCmdline,
						})
						machineIndex++
					}
				}
			}
			// Worker machines
			if tc.Spec.Worker != nil && tc.Spec.Worker.Mode == TalosModeMetal {
				for _, m := range tc.Spec.Worker.MetalSpec.Machines {
					if m.PxeClientSpec != nil && m.Address != nil {
						var kernelCmdline string
						if m.PxeClientSpec.KernelCmdlineArgs != nil {
							kernelCmdline = *m.PxeClientSpec.KernelCmdlineArgs
						} else {
							kernelCmdline = ""
						}
						clusters[i].Machines = append(clusters[i].Machines, Machine{
							fmt.Sprintf("%s-%d", tc.Name, machineIndex),
							*m.PxeClientSpec.MacAddress,
							*m.Address,
							tc.Spec.Worker.Version,
							*m.PxeClientSpec.CpuArchitecture,
							kernelCmdline,
						})
						machineIndex++
					}
				}
			}
		}
	}

	return clusters
}

func updatePxeBootStackConfig(clusters []Cluster) error {
	// Cleaning up previous Matchbox configuration
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

	// Generating configurations from templates and saving to files
	// dnsmasq
	if err := generateConfiguration(dnsmasqConfigTmpl, DnsmasqConfigPath, clusters); err != nil {
		return err
	}
	// Matchbox
	for _, c := range clusters {
		for _, m := range c.Machines {
			// Groups
			if err := generateConfiguration(matchboxGroupTmpl,
				path.Join(MatchboxConfigPath, MatchboxGroupsDir, fmt.Sprintf("%s.json", m.Id)), m,
			); err != nil {
				return err
			}
			// Profiles
			if err := generateConfiguration(matchboxProfileTmpl,
				path.Join(MatchboxConfigPath, MatchboxProfilesDir, fmt.Sprintf("%s.json", m.Id)), m,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateConfiguration(tmplString string, destPath string, data any) error {
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	tmpl, err := template.New("").Parse(tmplString)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(file, data); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func downloadBootImages(clusters []Cluster) error {
	// Determining every combination of Talos version + CPU architecture to take into account when downloading boot images
	var downloadList = make(map[string]string) // Maps a download URL to a destination path
	for _, c := range clusters {
		for _, m := range c.Machines {
			// Kernel
			downloadList[fmt.Sprintf("%s/%s/vmlinuz-%s",
				os.Getenv("TALOS_IMAGES_BASE_URL"), m.TalosVersion, m.CpuArchitecture,
			)] = fmt.Sprintf("%s/%s/vmlinuz-%s-%s",
				MatchboxConfigPath, MatchboxAssetsDir, m.TalosVersion, m.CpuArchitecture,
			)
			// initramfs
			downloadList[fmt.Sprintf("%s/%s/initramfs-%s.xz",
				os.Getenv("TALOS_IMAGES_BASE_URL"), m.TalosVersion, m.CpuArchitecture,
			)] = fmt.Sprintf("%s/%s/initramfs-%s-%s.xz",
				MatchboxConfigPath, MatchboxAssetsDir, m.TalosVersion, m.CpuArchitecture,
			)
			// iPXE
			var ipxeArch = ""
			var ipxeFile = ""
			switch m.CpuArchitecture {
			case "amd64":
				ipxeArch = IpxeEfiX8664Arch
				ipxeFile = IpxeEfiX8664File
			case "arm64":
				ipxeArch = IpxeEfiArm64Arch
				ipxeFile = IpxeEfiArm64File
			}
			downloadList[fmt.Sprintf("%s/%s/%s",
				os.Getenv("IPXE_BASE_URL"), ipxeArch, IpxeDownloadFile,
			)] = fmt.Sprintf("%s/%s", TftpDir, ipxeFile)
		}
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
