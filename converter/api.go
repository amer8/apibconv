package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Spec is a unified interface for any API specification (OpenAPI, AsyncAPI, API Blueprint).
type Spec interface {
	// Title returns the title of the specification.
	Title() string
	// Version returns the version of the specification.
	Version() string

	// AsOpenAPI attempts to return the underlying specification as *OpenAPI.
	// The boolean indicates if the conversion was successful.
	AsOpenAPI() (*OpenAPI, bool)
	// AsAsyncAPI attempts to return the underlying specification as *AsyncAPI.
	// The version parameter specifies the desired AsyncAPI major version (e.g., 2 or 3).
	// If version is 0, it returns the specification as-is if it is AsyncAPI.
	// The boolean indicates if the conversion was successful.
	AsAsyncAPI(version int) (*AsyncAPI, bool)
	// AsAPIBlueprint attempts to return the underlying specification as *APIBlueprint.
	// The boolean indicates if the conversion was successful.
	AsAPIBlueprint() (*APIBlueprint, bool)
}

// SpecFormat represents the specification format.
type SpecFormat string

const (
	// FormatAuto automatically detects the specification format.
	FormatAuto SpecFormat = "auto"
	// FormatBlueprint specifies the API Blueprint format.
	FormatBlueprint SpecFormat = "apib"
	// FormatOpenAPI specifies the OpenAPI format.
	FormatOpenAPI SpecFormat = "openapi"
	// FormatAsyncAPI specifies the AsyncAPI format.
	FormatAsyncAPI SpecFormat = "asyncapi"
)

// Parse parses API specification data into a unified Spec interface.
//
// It automatically detects the format if not specified.
//
// Parameters:
//   - data: specification content
//   - format: optional format hint (FormatAuto, FormatBlueprint, FormatOpenAPI, FormatAsyncAPI)
//
// Returns:
//   - Spec: The parsed specification (OpenAPI, AsyncAPI, or AsyncAPIV3 struct)
//   - error: Error if parsing fails
func Parse(data []byte, formats ...SpecFormat) (Spec, error) {
	format := FormatAuto
	if len(formats) > 0 {
		format = formats[0]
	}

	if format == FormatAuto {
		format = detectFormat(data)
	}

	switch format {
	case FormatBlueprint:
		// ParseBlueprint now returns *APIBlueprint
		return ParseBlueprint(data)
	case FormatOpenAPI:
		return parseOpenAPI(data)
	case FormatAsyncAPI:
		// Detect AsyncAPI version
		s, _, err := parseAsyncAPIAny(data)
		if err != nil {
			return nil, err
		}
		// s is *AsyncAPI, which implements Spec
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func detectFormat(data []byte) SpecFormat {
	// Simple detection logic
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return FormatAuto
	}

	// Check for API Blueprint
	if bytes.HasPrefix(trimmed, []byte("FORMAT:")) || (bytes.HasPrefix(trimmed, []byte("#")) && !bytes.HasPrefix(trimmed, []byte("{"))) {
		return FormatBlueprint
	}

	// Check for JSON/YAML signatures
	// This is a basic check.
	// We can check for "openapi" or "asyncapi" keys.
	// Since we are parsing anyway, we could just try parsing.
	// But let's try to be smart.

	if isJSON(data) || isYAML(data) {
		if bytes.Contains(data, []byte("asyncapi")) || bytes.Contains(data, []byte("asyncapi:")) {
			return FormatAsyncAPI
		}
		if bytes.Contains(data, []byte("openapi")) || bytes.Contains(data, []byte("openapi:")) || bytes.Contains(data, []byte("swagger")) {
			return FormatOpenAPI
		}
	}

	// Default to OpenAPI if JSON/YAML but not sure, or try Blueprint if text.
	// If it starts with {, it's likely JSON.
	if isJSON(data) {
		return FormatOpenAPI
	}

	return FormatBlueprint
}

func isYAML(data []byte) bool {
	// Heuristic: YAML usually doesn't start with { or [
	// but might start with comments or keys.
	// API Blueprint also text.
	// This is ambiguous.
	return !isJSON(data)
}

// parseOpenAPI is the old Parse logic for OpenAPI
func parseOpenAPI(data []byte) (*OpenAPI, error) {
	var spec OpenAPI

	// Try JSON first if it looks like JSON
	if isJSON(data) {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse OpenAPI JSON: %w", err)
		}
		return &spec, nil
	}

	// Try YAML
	if err := UnmarshalYAML(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI YAML: %w", err)
	}

	return &spec, nil
}

// isJSON checks if the data looks like JSON
func isJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}

// ParseWithConversion parses OpenAPI JSON/YAML and optionally converts to a target version.
//
// This function parses the OpenAPI spec and can automatically convert it to a different
// version if requested via the options. This is useful when you want to normalize
// all input to a specific version.
func ParseWithConversion(data []byte, opts *ConversionOptions) (*OpenAPI, error) {
	// Explicitly parse as OpenAPI for this specific function
	spec, err := parseOpenAPI(data)
	if err != nil {
		return nil, err
	}

	if opts == nil || opts.OutputVersion == "" {
		return spec, nil
	}

	// Convert if needed
	return spec.ConvertTo(opts.OutputVersion, opts)
}
