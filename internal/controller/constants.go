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

	// dnsmasq configuration path
	DnsmasqConfigPath = "/etc/dnsmasq.d/dnsmasq.conf"
	// Default dnsmasq configuration that disables DNS
	DefaultDnsmasqConfig = "port=0"
	// TFTP files
	TftpDir          = "/var/lib/tftp"
	IpxeEfiX8664File = "ipxe-efi-x86_64.efi"
	IpxeEfiArm64File = "ipxe-efi-arm64.efi"
	IpxeEfiX8664Arch = "x86_64-efi"
	IpxeEfiArm64Arch = "arm64-efi"
	IpxeDownloadFile = "ipxe.efi"

	// Matchbox configuration directory mount point in the talos-operator container
	MatchboxConfigPath = "/var/lib/matchbox"
	// Matchbox configuration subdirectories
	MatchboxAssetsDir   = "assets"
	MatchboxGroupsDir   = "groups"
	MatchboxProfilesDir = "profiles"
)
