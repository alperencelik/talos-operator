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
	// ReconcileMode is the mode of the reconciliation, it could be Reconcile, Disable or DryRun
	ReconcileModeNormal  = "reconcile"
	ReconcileModeDisable = "disable"
	// ReconcileModeDryRun runs the reconciliation without performing any mutating operations.
	// Kubernetes writes are validated via server-side dry-run; Talos API and file-system
	// operations are skipped and reported as "Would do X" events.
	ReconcileModeDryRun = "dryrun"

	// ReconcileModeImport is the mode for importing existing Talos resources
	ReconcileModeImport = "import"

	// For tests
	DefaultNamespace = "default"

	// Deleting is the reason used in conditions when a resource is being deleted
	ConditionReasonDeleting = "Deleting"

	// AppLabelKey is the standard pod label used to select pods backing a Talos
	// control plane StatefulSet/Service.
	AppLabelKey = "app"

	// Field index keys for owner-ref lookups
	IndexControlPlaneRefName = "spec.controlPlaneRef.name"
	IndexWorkerRefName       = "spec.workerRef.name"

	// DeletionPolicyReset is the deletion policy that triggers a Talos reset
	DeletionPolicyReset = "reset"

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
