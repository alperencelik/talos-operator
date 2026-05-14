package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"gopkg.in/yaml.v2"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func SecretBundleDecoder(bs string) (*secrets.Bundle, error) {
	decoder := yaml.NewDecoder(strings.NewReader(bs))
	var secretBundle secrets.Bundle
	if err := decoder.Decode(&secretBundle); err != nil {
		return nil, err
	}
	return &secretBundle, nil
}

func GenSans(name string, r *int) []string {
	capacity := 1
	if r != nil {
		capacity += *r
	}
	sans := make([]string, 0, capacity)
	sans = append(sans, name)
	if r == nil {
		return sans
	}
	for i := 0; i < *r; i++ {
		sans = append(sans, fmt.Sprintf("%s-%d", name, i))
	}
	return sans
}

func MarshalStringSlice(slice []string) (string, error) {
	bytes, err := json.Marshal(slice)
	if err != nil {
		return "", fmt.Errorf("failed to marshal string slice: %w", err)
	}
	return string(bytes), nil
}

func PtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// talosVersionPattern matches a Talos version: a leading "v", a dotted numeric core
// (e.g. "1.13.5"), and an optional semver-style pre-release suffix introduced by "-"
// (e.g. "-alpha.0", "-rc.1"). Pre-release identifiers are dot-separated alphanumerics.
const talosVersionPattern = `v\d+(\.\d+)*(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?`

func HasVersionSuffix(v string) bool {
	re := regexp.MustCompile(`:` + talosVersionPattern + `$`)
	return re.MatchString(v)
}

func IsValidTalosVersion(v string) bool {
	re := regexp.MustCompile(`^` + talosVersionPattern + `$`)
	return re.MatchString(v)
}

func StringToBytePtr(s string) *[]byte {
	b := []byte(s)
	return &b
}
