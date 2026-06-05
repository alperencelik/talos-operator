/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
)

func newTalosMachineWithMode(mode string) *talosv1alpha1.TalosMachine {
	tm := &talosv1alpha1.TalosMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: DefaultNamespace,
		},
	}
	if mode != "" {
		tm.Annotations = map[string]string{ReconcileModeAnnotation: mode}
	}
	return tm
}

func TestGetReconciliationMode(t *testing.T) {
	r := &TalosMachineReconciler{}
	testCases := []struct {
		name       string
		annotation string
		expected   string
	}{
		{name: "absent annotation defaults to Normal", annotation: "", expected: ReconcileModeNormal},
		{name: "reconcile", annotation: "reconcile", expected: ReconcileModeNormal},
		{name: "disable", annotation: "disable", expected: ReconcileModeDisable},
		{name: "dryrun lowercase", annotation: "dryrun", expected: ReconcileModeDryRun},
		{name: "dryrun mixed case", annotation: "DryRun", expected: ReconcileModeDryRun},
		{name: "dryrun uppercase", annotation: "DRYRUN", expected: ReconcileModeDryRun},
		{name: "import", annotation: "import", expected: ReconcileModeImport},
		{name: "unknown defaults to Normal", annotation: "bogus", expected: ReconcileModeNormal},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tm := newTalosMachineWithMode(tc.annotation)
			if got := r.getReconciliationMode(context.Background(), tm); got != tc.expected {
				t.Errorf("getReconciliationMode() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestIsDryRun(t *testing.T) {
	r := &TalosMachineReconciler{}
	testCases := []struct {
		name       string
		annotation string
		expected   bool
	}{
		{name: "absent annotation", annotation: "", expected: false},
		{name: "dryrun lowercase", annotation: "dryrun", expected: true},
		{name: "dryrun mixed case", annotation: "DryRun", expected: true},
		{name: "disable", annotation: "disable", expected: false},
		{name: "reconcile", annotation: "reconcile", expected: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tm := newTalosMachineWithMode(tc.annotation)
			if got := r.isDryRun(tm); got != tc.expected {
				t.Errorf("isDryRun() = %v, want %v", got, tc.expected)
			}
		})
	}
}

// TestIsDryRunGeneric verifies the package-level helper works for any client.Object kind.
func TestIsDryRunGeneric(t *testing.T) {
	tc := &talosv1alpha1.TalosCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-cluster",
			Namespace:   DefaultNamespace,
			Annotations: map[string]string{ReconcileModeAnnotation: "DryRun"},
		},
	}
	if !isDryRun(tc) {
		t.Errorf("isDryRun() = false for a DryRun-annotated TalosCluster, want true")
	}
	tcp := &talosv1alpha1.TalosControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cp", Namespace: DefaultNamespace},
	}
	if isDryRun(tcp) {
		t.Errorf("isDryRun() = true for an unannotated TalosControlPlane, want false")
	}
}
