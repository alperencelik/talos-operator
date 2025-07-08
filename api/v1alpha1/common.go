package v1alpha1

const (
	ConditionDeleting    = "Deleting"
	ConditionReady       = "Ready"
	ConditionFailed      = "Failed"
	ConditionProgressing = "Progressing"
	ConditionAvailable   = "Available"
	// State of the Talos control plane
	StatePending      = "Pending"      // Control plane is being created
	StateAvailable    = "Available"    // Control plane is ready to bootstrap the cluster
	StateBootstrapped = "Bootstrapped" // Control plane is ready to accept workloads
	StateReady        = "Ready"        // Control plane is fully operational
	StateFailed       = "Failed"       // Control plane creation failed

	// Finalizers
	TalosClusterFinalizer      = "taloscluster.talos.alperen.cloud/finalizer"
	TalosControlPlaneFinalizer = "taloscontrolplane.talos.alperen.cloud/finalizer"
	TalosWorkerFinalizer       = "talosworker.talos.alperen.cloud/finalizer"
	TalosMachineFinalizer      = "talosmachine.talos.alperen.cloud/finalizer"

	// GVK for the API group
	GroupName             = "talos.alperen.cloud"
	GroupKindCluster      = "TalosCluster"
	GroupKindControlPlane = "TalosControlPlane"
	GroupKindWorker       = "TalosWorker"
	GroupKindMachine      = "TalosMachine"
)
