package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Spec is a unified interface for any API specification (OpenAPI, AsyncAPI, API Blueprint).
type Spec interface {
	// ToBlueprint converts the specification to API Blueprint format.
	ToBlueprint() (string, error)

	// ToOpenAPI converts the specification to OpenAPI 3.0 format.
	ToOpenAPI() (*OpenAPI, error)

	// ToAsyncAPI converts the specification to AsyncAPI 2.6 format.
	// protocol is required for AsyncAPI server definitions.
	ToAsyncAPI(protocol Protocol) (*AsyncAPI, error)

	// ToAsyncAPIV3 converts the specification to AsyncAPI 3.0 format.
	// protocol is required for AsyncAPI server definitions.
	ToAsyncAPIV3(protocol Protocol) (*AsyncAPIV3, error)
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
		return ParseBlueprint(data)
	case FormatOpenAPI:
		return parseOpenAPI(data)
	case FormatAsyncAPI:
		// Detect AsyncAPI version
		spec, _, err := ParseAsyncAPIAny(data)
		if err != nil {
			return nil, err
		}
		if s, ok := spec.(Spec); ok {
			return s, nil
		}
		return nil, fmt.Errorf("parsed AsyncAPI spec does not implement Spec interface")
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
	return ConvertToVersion(spec, opts.OutputVersion, opts)
}

// ParseReader parses OpenAPI 3.0 JSON or YAML from an io.Reader into an OpenAPI structure.
//
// Deprecated: Use Parse with io.ReadAll instead.
func ParseReader(r io.Reader) (*OpenAPI, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseOpenAPI(data)
}

// Format converts a Spec to API Blueprint format and returns it as a string.
//
// Deprecated: Use Spec.ToBlueprint instead.
func Format(spec Spec) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("spec cannot be nil")
	}
	return spec.ToBlueprint()
}

// FormatTo converts a Spec to API Blueprint format and writes it to w.
//
// Deprecated: Use write logic manually or new interface methods.
func FormatTo(spec *OpenAPI, w io.Writer) error {
	if spec == nil {
		return fmt.Errorf("spec cannot be nil")
	}

	return spec.WriteBlueprint(w)
}

// FromJSON converts OpenAPI 3.0 JSON bytes directly to API Blueprint format string.
//
// Deprecated: Use Parse and Spec.ToBlueprint instead.
func FromJSON(data []byte) (string, error) {
	spec, err := Parse(data, FormatOpenAPI)
	if err != nil {
		return "", err
	}
	return spec.ToBlueprint()
}

// FromJSONString converts an OpenAPI 3.0 JSON string to API Blueprint format string.
//
// Deprecated: Use Parse and Spec.ToBlueprint instead.
func FromJSONString(jsonStr string) (string, error) {
	return FromJSON([]byte(jsonStr))
}

// ToBytes converts OpenAPI 3.0 JSON bytes to API Blueprint format bytes.
//
// Deprecated: Use Parse and Spec.ToBlueprint instead.
func ToBytes(data []byte) ([]byte, error) {
	spec, err := Parse(data, FormatOpenAPI)
	if err != nil {
		return nil, err
	}

	bp, err := spec.ToBlueprint()
	if err != nil {
		return nil, err
	}

	return []byte(bp), nil
}

// ConvertString provides string-to-string conversion of OpenAPI to API Blueprint.
//
// Deprecated: Use Parse and Spec.ToBlueprint instead.
func ConvertString(openapiJSON string) (string, error) {
	return FromJSONString(openapiJSON)
}

// MustFormat is like Format but panics if an error occurs.
//
// Deprecated: Use Format and handle errors instead.
func MustFormat(spec *OpenAPI) string {
	result, err := Format(spec)
	if err != nil {
		panic(err)
	}
	return result
}

// MustFromJSON is like FromJSON but panics if an error occurs.
//
// Deprecated: Use FromJSON and handle errors instead.
func MustFromJSON(data []byte) string {
	result, err := FromJSON(data)
	if err != nil {
		panic(err)
	}
	return result
}
