package utils

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	key := "TEST_ENV_VAR"
	val := "test_value"
	os.Setenv(key, val)
	defer os.Unsetenv(key)

	if got := GetEnv(key, "fallback"); got != val {
		t.Errorf("GetEnv() = %v, want %v", got, val)
	}

	if got := GetEnv("NON_EXISTENT_VAR", "fallback"); got != "fallback" {
		t.Errorf("GetEnv() = %v, want %v", got, "fallback")
	}
}

func TestGenSans(t *testing.T) {
	name := "test-node"
	replicas := 3
	expected := []string{"test-node", "test-node-0", "test-node-1", "test-node-2"}

	got := GenSans(name, &replicas)
	if len(got) != len(expected) {
		t.Fatalf("GenSans() returned %d items, want %d", len(got), len(expected))
	}
	for i, s := range got {
		if s != expected[i] {
			t.Errorf("GenSans()[%d] = %v, want %v", i, s, expected[i])
		}
	}

	gotNil := GenSans(name, nil)
	if len(gotNil) != 1 || gotNil[0] != name {
		t.Errorf("GenSans() with nil replicas = %v, want [%v]", gotNil, name)
	}
}

func TestPtrToString(t *testing.T) {
	s := "test"
	if got := PtrToString(&s); got != s {
		t.Errorf("PtrToString() = %v, want %v", got, s)
	}
	if got := PtrToString(nil); got != "" {
		t.Errorf("PtrToString(nil) = %v, want \"\"", got)
	}
}

func TestHasVersionSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"image:v1.0.0", true},
		{"image:v1.2", true}, // Regex :v\d+(\.\d+)*$ matches v1.2
		{"image:v1", true},
		{"image:v1.2.3.4", true}, // Regex allows multiple components
		{"image:1.0.0", false},
		{"image", false},
		{"image:v1.2.3", true},
		{"image:1.0.0", false},
		{"image", false},
	}

	for _, tt := range tests {
		if got := HasVersionSuffix(tt.input); got != tt.want {
			t.Errorf("HasVersionSuffix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsValidTalosVersion(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"v1.0.0", true},
		{"v1.2", true},
		{"v1", true},
		{"1.0.0", false},
		{"v1.0.0-beta", false}, // Regex `^v\d+(\.\d+)*$` doesn't seem to support suffixes like -beta based on my read, but check source.
	}
	// Re-reading IsValidTalosVersion regex: `^v\d+(\.\d+)*$`
	// It strictly matches vDigits(.Digits)*

	for _, tt := range tests {
		if got := IsValidTalosVersion(tt.input); got != tt.want {
			t.Errorf("IsValidTalosVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStringToBytePtr(t *testing.T) {
	s := "test"
	ptr := StringToBytePtr(s)
	if ptr == nil {
		t.Fatal("StringToBytePtr() returned nil")
	}
	if string(*ptr) != s {
		t.Errorf("StringToBytePtr() content = %v, want %v", string(*ptr), s)
	}
}
