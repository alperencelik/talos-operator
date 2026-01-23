package controller

import (
	"context"
	"testing"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetMachineIPAddress(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = talosv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name          string
		machine       *talosv1alpha1.Machine
		existingObjs  []client.Object
		expectedIP    string
		expectedError bool
	}{
		{
			name: "Direct Address",
			machine: &talosv1alpha1.Machine{
				Address: func() *string { s := "192.168.1.10"; return &s }(),
			},
			expectedIP:    "192.168.1.10",
			expectedError: false,
		},
		{
			name: "Valid MachineRef",
			machine: &talosv1alpha1.Machine{
				MachineRef: &corev1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
					FieldPath:  "status.podIP",
				},
			},
			existingObjs: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test-pod",
							"namespace": "default",
						},
						"status": map[string]interface{}{
							"podIP": "10.244.0.5",
						},
					},
				},
			},
			expectedIP:    "10.244.0.5",
			expectedError: false,
		},
		{
			name: "Invalid IP in MachineRef",
			machine: &talosv1alpha1.Machine{
				MachineRef: &corev1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "invalid-pod",
					Namespace:  "default",
					FieldPath:  "status.podIP",
				},
			},
			existingObjs: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "invalid-pod",
							"namespace": "default",
						},
						"status": map[string]interface{}{
							"podIP": "not-an-ip",
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Empty field path",
			machine: &talosv1alpha1.Machine{
				MachineRef: &corev1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "test-pod",
					Namespace:  "default",
					FieldPath:  "",
				},
			},
			expectedError: true,
		},
		{
			name: "Object not found",
			machine: &talosv1alpha1.Machine{
				MachineRef: &corev1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "missing-pod",
					Namespace:  "default",
					FieldPath:  "status.podIP",
				},
			},
			expectedError: true, // Client Get error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert existing objects to unstructured if needed for the fake client to work with generic jsonpath on known types?
			// Actually the fake client supports getting unstructured from typed objects if registered in scheme?
			// But wait, the code uses `c.Get(ctx, ..., obj)`. `obj` is `&unstructured.Unstructured{}`.
			// The fake client might not support getting a typed object into an unstructured object directly if not strictly set up that way?
			// Let's safe bet: register types, create typed objects. Fake client usually handles it.
			// However, for the code to work: `c.Get` is called with `&unstructured.Unstructured{}`.
			// The fake client needs to be able to serve that.

			// Use the objects directly as they are now defined as Unstructured where needed
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.existingObjs...)
			cli := builder.Build()

			gotIP, err := getMachineIPAddress(context.Background(), cli, tt.machine)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none, IP: %v", gotIP)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if gotIP == nil {
					t.Error("Expected IP but got nil")
				} else if *gotIP != tt.expectedIP {
					t.Errorf("Expected IP %s, got %s", tt.expectedIP, *gotIP)
				}
			}
		})
	}
}
