package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"gopkg.in/yaml.v2"
)

func SecretBundleDecoder(bs string) (*secrets.Bundle, error) {
	decoder := yaml.NewDecoder(strings.NewReader(bs))
	var secretBundle secrets.Bundle
	if err := decoder.Decode(&secretBundle); err != nil {
		return nil, err
	}
	return &secretBundle, nil
}

func GenSans(name string, r *int) []string {
	var sans []string
	sans = append(sans, name)
	if r == nil {
		return sans
	}
	for i := 0; i < *r; i++ {
		sans = append(sans, fmt.Sprintf("%s-%d", name, i))
	}
	return sans
}

func MarshalStringSlice(slice []string) string {
	bytes, err := json.Marshal(slice)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal string slice: %v", err))
	}
	return string(bytes)
}

func PtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
