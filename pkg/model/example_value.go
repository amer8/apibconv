package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// NullExample represents an explicitly supplied JSON null, distinct from no example.
type NullExample struct{}

// MarshalJSON emits JSON null.
func (NullExample) MarshalJSON() ([]byte, error) { return []byte("null"), nil }

// MarshalYAML emits YAML null.
func (NullExample) MarshalYAML() (interface{}, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
}

// IsZero prevents an explicit null from being omitted by the YAML encoder.
func (NullExample) IsZero() bool { return false }

// JSONNumber retains the original precision of an example's JSON number.
type JSONNumber string

// MarshalJSON emits the number without rounding.
func (n JSONNumber) MarshalJSON() ([]byte, error) {
	if !json.Valid([]byte(n)) {
		return nil, fmt.Errorf("invalid JSON number %q", n)
	}
	return []byte(n), nil
}

// MarshalYAML emits a numeric scalar without rounding.
func (n JSONNumber) MarshalYAML() (interface{}, error) {
	if !json.Valid([]byte(n)) {
		return nil, fmt.Errorf("invalid JSON number %q", n)
	}
	tag := "!!int"
	if strings.ContainsAny(string(n), ".eE") {
		tag = "!!float"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(n)}, nil
}
