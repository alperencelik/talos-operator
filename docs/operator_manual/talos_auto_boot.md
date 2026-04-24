# Booting Talos Automatically

You can boot machines into Talos automatically from the network using PXE by enabling the *PXE boot stack*.

!!!info
    This feature only supports x86_64 and arm64 EFI firmwares for now.

## Usage

To enable the *PXE boot stack*, set the value `featureFlags.enablePxeBootStack` to `true` in the Helm chart.

Then, create a `TalosCluster` resource and fill the property `spec.pxeServerSpec` with the IP address of the server you are running talos-operator on and the name of the interface (as given by the system) that connects it to the machines network.

You also need to fill the property `spec.[controlPlane/worker].metalSpec.machines.pxeClientSpec` for every machine that needs to be booted automatically with the MAC address used by its PXE firmware, its CPU architecture, and optional kernel command line arguments to inject on boot.

Here is an example of such a `TalosCluster` resource:
```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosCluster
metadata:
  name: auto-boot-cluster
spec:
  pxeServerSpec:
    address: "10.0.0.1"
    interface: enp0s1
  controlPlane:
    version: "v1.12.6"
    mode: metal
    metalSpec:
      machines:
        - address: "10.0.0.2"
          pxeClientSpec:
            macAddress: "aa:aa:aa:aa:aa:aa"
            cpuArchitecture: "amd64"
        - address: "10.0.0.3"
          pxeClientSpec:
            macAddress: "bb:bb:bb:bb:bb:bb"
            cpuArchitecture: "amd64"
            kernelCmdlineArgs: "net.ifnames=0 nomodeset"
```

Finally, boot your machines in PXE mode.

## How it works

The *PXE boot stack* consists of a container running [dnsmasq](https://dnsmasq.org/doc.html) that exposes a DHCP and a TFTP server as well as a container running [Matchbox](https://matchbox.psdn.io/) that delivers boot images depending on the machine sending the request.

Machines booted in PXE will automatically send a *DHCP discover* packet to which `dnsmasq` will answer with an IP address and the URL to an executable file that contains the [iPXE](https://ipxe.org/) firmware, which is downloaded using TFTP. iPXE will then boot and send a second DHCP request. This time, `dnsmasq` answers with the URL to an iPXE script. Matchbox generates and delivers this script depending on the node's MAC address. It instructs iPXE to download the Talos kernel and initrd images to boot Talos and specifies the kernel command line arguments. Machines will then start Talos in "Maintenance" mode and wait for the operator to start the installation.