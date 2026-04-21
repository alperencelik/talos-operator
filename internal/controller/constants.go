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

	// PXE boot stack pods annotation
	PxeBootStackPodsAnnotation      = "talos.alperen.cloud/deployment-name"
	PxeBootStackPodsAnnotationValue = "pxe-boot"

	// Matchbox assets directory mount point in the talos-operator container
	MatchboxAssetsPath = "/matchbox-assets"

	// Talos boot images download URL
	TalosBootImageBaseUrl = "https://github.com/siderolabs/talos/releases/download"
)
