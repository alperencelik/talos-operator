package controller

const (
	TalosPlatformKey = "PLATFORM"
	// TalosModeContainer is the mode for Talos running in a container
	TalosModeContainer = "container"
	// TalosModeMetal is the mode for Talos running on bare metal
	TalosModeMetal = "metal"

	// MachineType
	TalosMachineTypeControlPlane = "controlplane"
	TalosMachineTypeWorker       = "worker"

	// Reconcile Modes

	// ReconcileModeAnnotation is the annotation key for the reconcile mode
	ReconcileModeAnnotation = "talos.alperen.cloud/reconcile-mode"
	// ReconcileMode is the mode of the reconciliation it could be Normal, WatchOnly, EnsureExists, Disable, DryRun (to be implemented)
	ReconcileModeNormal  = "reconcile"
	ReconcileModeDisable = "disable"
	ReconcileModeDryRun  = "dryrun" // TODO: Implement DryRun mode

	// ReconcileModeImport is the mode for importing existing Talos resources
	ReconcileModeImport = "import"

	// For tests
	DefaultNamespace = "default"

	// PXE boot stack

	// PXE boot stack enabled value
	PxeBootStackEnabled = "true"

	// proc related paths
	ProcPath        = "/proc"
	ProcCmdlineFile = "cmdline"
	DnsmasqCmdline  = "/sbin/tini\u0000--\u0000/usr/bin/dnsmasq.sh\u0000"

	// dnsmasq configuration directory mount point in the talos-operator container
	DnsmasqConfigPath = "/etc/dnsmasq.d"
	// dnsmasq configuration file name
	DnsmasqConfigFile = "dnsmasq.conf"
	// Base dnsmasq configuration that disables DNS, enables TFTP, sets PXE boot file name and tags iPXE clients
	BaseDnsmasqConfig = `bind-interfaces

# Disable DNS
port=0

# TFTP server:
enable-tftp
tftp-root=/var/lib/tftp

# UEFI iPXE boot file:
dhcp-match=set:efi,option:client-arch,7
dhcp-boot=tag:efi,ipxe.efi

# Tagging iPXE clients:
dhcp-userclass=set:ipxe,iPXE`

	// Matchbox configuration directory mount point in the talos-operator container
	MatchboxConfigPath = "/var/lib/matchbox"
	// Matchbox configuration subdirectories
	MatchboxAssetsDir   = "assets"
	MatchboxGroupsDir   = "groups"
	MatchboxProfilesDir = "profiles"
)
