package controller

// Shared constants for controller tests. Promoted from inline literals to satisfy
// goconst, and kept in one place so test fixtures stay consistent across files.
const (
	testTalosVersion        = "v1.12.1"
	testKubeVersion         = "v1.35.0"
	testModeCloud           = "cloud"
	testDeletionPolicyReset = "reset"

	testMachineIP = "192.168.1.10"

	testPodKind    = "Pod"
	testPodName    = "test-pod"
	testPodFieldIP = "status.podIP"
)
