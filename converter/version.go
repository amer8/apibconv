package converter

import (
	"fmt"
	"strings"
)

// Version represents an OpenAPI specification version.
type Version string

const (
	// Version30 represents OpenAPI 3.0.x
	Version30 Version = "3.0"
	// Version31 represents OpenAPI 3.1.x
	Version31 Version = "3.1"
)

// String returns the string representation of the version.
func (v Version) String() string {
	return string(v)
}

// DetectVersion parses the openapi field to determine the major.minor version.
//
// It extracts the version from strings like "3.0.0", "3.0.3", "3.1.0", etc.
// and returns the appropriate Version constant.
//
// Parameters:
//   - openapiField: The value of the "openapi" field in the spec (e.g., "3.0.0", "3.1.0")
//
// Returns:
//   - Version: The detected version (Version30 or Version31)
//
// Example:
//
//	version := DetectVersion("3.0.0")  // Returns Version30
//	version := DetectVersion("3.1.0")  // Returns Version31
func DetectVersion(openapiField string) Version {
	if strings.HasPrefix(openapiField, "3.1") {
		return Version31
	}
	return Version30
}

// ConversionOptions configures how API Blueprint to OpenAPI conversion should behave.
//
// This allows fine-grained control over the output format, including which OpenAPI
// version to target and whether to fail on incompatible features.
type ConversionOptions struct {
	// OutputVersion specifies the desired OpenAPI version for the output.
	// Default: Version30 (OpenAPI 3.0.0)
	OutputVersion Version

	// StrictMode when enabled will cause conversion to fail if features are used
	// that are incompatible with the target version (e.g., webhooks in 3.0 output).
	// When disabled, incompatible features are silently dropped.
	// Default: false
	StrictMode bool
}

// DefaultConversionOptions returns the default conversion options.
//
// By default, output is OpenAPI 3.0.0 with StrictMode disabled.
func DefaultConversionOptions() *ConversionOptions {
	return &ConversionOptions{
		OutputVersion: Version30,
		StrictMode:    false,
	}
}

// Validate checks if the conversion options are valid.
func (opts *ConversionOptions) Validate() error {
	if opts.OutputVersion != Version30 && opts.OutputVersion != Version31 {
		return fmt.Errorf("invalid output version: %s (must be %s or %s)",
			opts.OutputVersion, Version30, Version31)
	}
	return nil
}

// ToFullVersion returns the full version string (e.g., "3.0.0" or "3.1.0").
func (v Version) ToFullVersion() string {
	switch v {
	case Version31:
		return "3.1.0"
	default:
		return "3.0.0"
	}
}
