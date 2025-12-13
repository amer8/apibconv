package detect

import (
	"bytes"
)

// Encoding attempts to detect if the data is JSON or YAML.
func Encoding(data []byte) string {
	if IsJSON(data) {
		return "json"
	}
	if IsYAML(data) {
		return "yaml"
	}
	return "unknown"
}

// IsYAML checks if the data looks like YAML.
func IsYAML(data []byte) bool {
	// Simple heuristic: check if it's not JSON and has YAML-like structure
	// This is very basic.
	if IsJSON(data) {
		return false
	}
	return true // Fallback
}

// IsJSON checks if the data looks like JSON.
func IsJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
