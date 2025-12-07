package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Parse parses OpenAPI JSON or YAML data into an OpenAPI structure.
//
// This function is useful when you want to inspect or manipulate the parsed structure
// before converting to API Blueprint format. It accepts raw JSON or YAML bytes and returns
// a fully populated OpenAPI struct.
//
// The function automatically detects the format (JSON/YAML) and OpenAPI version (3.0 or 3.1).
// No conversion is performed - the spec is returned as-is.
//
// Parameters:
//   - data: OpenAPI specification as JSON or YAML bytes
//
// Returns:
//   - *OpenAPI: Parsed OpenAPI structure
//   - error: Error if parsing fails
//
// Example:
//
//	jsonData := []byte(`{
//	    "openapi": "3.0.0",
//	    "info": {"title": "My API", "version": "1.0.0"},
//	    "paths": {}
//	}`)
//
//	spec, err := converter.Parse(jsonData)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Modify the spec programmatically
//	spec.Info.Description = "An updated description"
//
//	// Convert to API Blueprint
//	result, err := converter.Format(spec)
func Parse(data []byte) (*OpenAPI, error) {
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
//
// Parameters:
//   - data: OpenAPI specification as JSON or YAML bytes
//   - opts: Conversion options (nil to keep original version)
//
// Returns:
//   - *OpenAPI: Parsed (and potentially converted) OpenAPI structure
//   - error: Error if parsing or conversion fails
//
// Example:
//
//	// Parse 3.0 spec and convert to 3.1
//	opts := &converter.ConversionOptions{
//	    OutputVersion: converter.Version31,
//	}
//	spec, err := converter.ParseWithConversion(jsonData, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Version: %s\n", spec.OpenAPI) // "3.1.0"
func ParseWithConversion(data []byte, opts *ConversionOptions) (*OpenAPI, error) {
	spec, err := Parse(data)
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
// This is the streaming version of Parse, useful for reading from files, network
// connections, or other io.Reader sources without loading the entire content into
// memory first.
//
// Parameters:
//   - r: An io.Reader containing OpenAPI 3.0 JSON or YAML data
//
// Returns:
//   - *OpenAPI: Parsed OpenAPI structure
//   - error: Error if reading fails or parsing is malformed
//
// Example:
//
//	file, err := os.Open("openapi.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	spec, err := converter.ParseReader(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("API Title: %s\n", spec.Info.Title)
func ParseReader(r io.Reader) (*OpenAPI, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Format converts an OpenAPI structure to API Blueprint format and returns it as a string.
//
// This function takes a parsed OpenAPI structure and formats it into API Blueprint
// markdown format. It uses zero-allocation buffer pooling internally for efficient
// memory usage.
//
// Parameters:
//   - spec: A parsed OpenAPI structure (must not be nil)
//
// Returns:
//   - string: The formatted API Blueprint content
//   - error: Error if spec is nil
//
// Example:
//
//	spec := &converter.OpenAPI{
//	    OpenAPI: "3.0.0",
//	    Info: converter.Info{
//	        Title:   "Pet Store API",
//	        Version: "1.0.0",
//	    },
//	    Paths: map[string]converter.PathItem{
//	        "/pets": {
//	            Get: &converter.Operation{
//	                Summary: "List all pets",
//	                Responses: map[string]converter.Response{
//	                    "200": {Description: "Success"},
//	                },
//	            },
//	        },
//	    },
//	}
//
//	blueprint, err := converter.Format(spec)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(blueprint)
func Format(spec *OpenAPI) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("spec cannot be nil")
	}

	buf := getBuffer()
	defer putBuffer(buf)

	writeAPIBlueprint(buf, spec)
	return buf.String(), nil
}

// FormatTo converts an OpenAPI structure to API Blueprint format and writes it to w.
//
// This is the streaming version of Format, useful when you want to write directly
// to a file, network connection, or other io.Writer without allocating an intermediate
// string. It's more memory-efficient for large specifications.
//
// Parameters:
//   - spec: A parsed OpenAPI structure (must not be nil)
//   - w: An io.Writer where the API Blueprint output will be written
//
// Returns an error if:
//   - spec is nil
//   - writing to w fails
//
// Example:
//
//	spec, err := converter.Parse(jsonData)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	output, err := os.Create("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	if err := converter.FormatTo(spec, output); err != nil {
//	    log.Fatal(err)
//	}
func FormatTo(spec *OpenAPI, w io.Writer) error {
	if spec == nil {
		return fmt.Errorf("spec cannot be nil")
	}

	buf := getBuffer()
	defer putBuffer(buf)

	writeAPIBlueprint(buf, spec)

	_, err := w.Write(buf.Bytes())
	return err
}

// FromJSON converts OpenAPI 3.0 JSON bytes directly to API Blueprint format string.
//
// This is the simplest one-step conversion function for programmatic usage.
// It combines parsing and formatting into a single call.
//
// Parameters:
//   - data: OpenAPI 3.0 specification as JSON bytes
//
// Returns:
//   - string: The formatted API Blueprint content
//   - error: Error if parsing or formatting fails
//
// Example:
//
//	openapiJSON := []byte(`{
//	    "openapi": "3.0.0",
//	    "info": {"title": "My API", "version": "1.0.0"},
//	    "paths": {
//	        "/users": {
//	            "get": {
//	                "summary": "List users",
//	                "responses": {"200": {"description": "Success"}}
//	            }
//	        }
//	    }
//	}`)
//
//	apiBlueprint, err := converter.FromJSON(openapiJSON)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(apiBlueprint)
func FromJSON(data []byte) (string, error) {
	spec, err := Parse(data)
	if err != nil {
		return "", err
	}
	return Format(spec)
}

// FromJSONString converts an OpenAPI 3.0 JSON string to API Blueprint format string.
//
// This is a convenience wrapper around FromJSON for string inputs. Useful when
// you're working with JSON strings instead of byte slices.
//
// Parameters:
//   - jsonStr: OpenAPI 3.0 specification as a JSON string
//
// Returns:
//   - string: The formatted API Blueprint content
//   - error: Error if parsing or formatting fails
//
// Example:
//
//	openapiJSON := `{"openapi": "3.0.0", "info": {"title": "API", "version": "1.0"}, "paths": {}}`
//	apiBlueprint, err := converter.FromJSONString(openapiJSON)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(apiBlueprint)
func FromJSONString(jsonStr string) (string, error) {
	return FromJSON([]byte(jsonStr))
}

// ToBytes converts OpenAPI 3.0 JSON bytes to API Blueprint format bytes.
//
// Returns raw bytes for maximum flexibility in handling the output.
// Useful when you need to further process the output or write it to a binary stream.
//
// Parameters:
//   - data: OpenAPI 3.0 specification as JSON bytes
//
// Returns:
//   - []byte: The formatted API Blueprint content as bytes
//   - error: Error if parsing or formatting fails
//
// Example:
//
//	openapiJSON := []byte(`{"openapi": "3.0.0", "info": {"title": "API", "version": "1.0"}, "paths": {}}`)
//	apiBlueprint, err := converter.ToBytes(openapiJSON)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Write to file
//	os.WriteFile("api.apib", apiBlueprint, 0644)
func ToBytes(data []byte) ([]byte, error) {
	spec, err := Parse(data)
	if err != nil {
		return nil, err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	writeAPIBlueprint(buf, spec)

	// Make a copy since we're returning the buffer to the pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// ConvertString provides string-to-string conversion of OpenAPI to API Blueprint.
//
// This is an alias for FromJSONString, provided for better discoverability and
// to match naming conventions when searching for conversion functions.
//
// Parameters:
//   - openapiJSON: OpenAPI 3.0 specification as a JSON string
//
// Returns:
//   - string: The formatted API Blueprint content
//   - error: Error if parsing or formatting fails
//
// Example:
//
//	result, err := converter.ConvertString(openapiJSON)
func ConvertString(openapiJSON string) (string, error) {
	return FromJSONString(openapiJSON)
}

// MustFormat is like Format but panics if an error occurs.
//
// Useful for testing and situations where you're certain the input is valid.
// Should not be used in production code where errors need to be handled gracefully.
//
// Parameters:
//   - spec: A parsed OpenAPI structure (must not be nil)
//
// Returns the formatted API Blueprint content or panics on error.
//
// Example:
//
//	spec := &converter.OpenAPI{
//	    OpenAPI: "3.0.0",
//	    Info: converter.Info{Title: "API", Version: "1.0"},
//	    Paths: map[string]converter.PathItem{},
//	}
//	blueprint := converter.MustFormat(spec)  // Panics if spec is nil
func MustFormat(spec *OpenAPI) string {
	result, err := Format(spec)
	if err != nil {
		panic(err)
	}
	return result
}

// MustFromJSON is like FromJSON but panics if an error occurs.
//
// Useful for testing and situations where you're certain the input is valid JSON.
// Should not be used in production code where errors need to be handled gracefully.
//
// Parameters:
//   - data: OpenAPI 3.0 specification as JSON bytes
//
// Returns the formatted API Blueprint content or panics on error.
//
// Example:
//
//	// In tests
//	apiBlueprint := converter.MustFromJSON([]byte(validOpenAPIJSON))
func MustFromJSON(data []byte) string {
	result, err := FromJSON(data)
	if err != nil {
		panic(err)
	}
	return result
}
