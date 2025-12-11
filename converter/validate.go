package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ValidationResult represents the result of a specification validation.
type ValidationResult struct {
	// Valid indicates whether the specification passed validation.
	Valid bool
	// Format is the detected specification format (e.g., "OpenAPI 3.0", "AsyncAPI 2.6").
	Format string
	// Version is the specification version string.
	Version string
	// Errors contains any validation errors found.
	Errors *ValidationErrors
	// Warnings contains non-fatal validation warnings.
	Warnings []string
}

// ValidateOpenAPI validates an OpenAPI specification.
// It checks for required fields and common issues.
//
// Example:
//
//	spec, _ := Parse(jsonData)
//	result := ValidateOpenAPI(spec)
//	if !result.Valid {
//	    for _, err := range result.Errors.Errors {
//	        fmt.Printf("Error at %s: %s\n", err.Field, err.Message)
//	    }
//	}
func ValidateOpenAPI(spec *OpenAPI) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Format: "OpenAPI",
		Errors: &ValidationErrors{},
	}

	if spec == nil {
		result.Valid = false
		result.Errors.Add("", "specification is nil")
		return result
	}

	// Validate OpenAPI version
	if spec.OpenAPI == "" {
		result.Valid = false
		result.Errors.Add("openapi", "openapi version is required")
	} else {
		result.Version = spec.OpenAPI
		if !strings.HasPrefix(spec.OpenAPI, "3.0") && !strings.HasPrefix(spec.OpenAPI, "3.1") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("openapi version '%s' may not be fully supported", spec.OpenAPI))
		}
		result.Format = fmt.Sprintf("OpenAPI %s", spec.OpenAPI)
	}

	// Validate info object
	validateOpenAPIInfo(spec, result)

	// Validate paths
	validateOpenAPIPaths(spec, result)

	// Validate components if present
	if spec.Components != nil {
		validateOpenAPIComponents(spec, result)
	}

	// Validate servers
	validateOpenAPIServers(spec, result)

	return result
}

func validateOpenAPIInfo(spec *OpenAPI, result *ValidationResult) {
	if spec.Info.Title == "" {
		result.Valid = false
		result.Errors.Add("info.title", "title is required")
	}
	if spec.Info.Version == "" {
		result.Valid = false
		result.Errors.Add("info.version", "version is required")
	}
}

func validateOpenAPIPaths(spec *OpenAPI, result *ValidationResult) {
	if spec.Paths == nil {
		result.Warnings = append(result.Warnings, "paths object is empty")
		return
	}

	for path, pathItem := range spec.Paths {
		// Validate path format
		if !strings.HasPrefix(path, "/") {
			result.Errors.Add(fmt.Sprintf("paths.%s", path), "path must start with /")
			result.Valid = false
		}

		// Validate operations
		validateOperation(path, "get", pathItem.Get, result)
		validateOperation(path, "post", pathItem.Post, result)
		validateOperation(path, "put", pathItem.Put, result)
		validateOperation(path, "delete", pathItem.Delete, result)
		validateOperation(path, "patch", pathItem.Patch, result)
	}
}

func validateOperation(path, method string, op *Operation, result *ValidationResult) {
	if op == nil {
		return
	}

	fieldPath := fmt.Sprintf("paths.%s.%s", path, method)

	// Validate responses
	if len(op.Responses) == 0 {
		result.Valid = false
		result.Errors.Add(fieldPath+".responses", "at least one response is required")
	} else {
		for code, resp := range op.Responses {
			if resp.Description == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("%s.responses.%s: description is recommended", fieldPath, code))
			}
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		paramPath := fmt.Sprintf("%s.parameters[%d]", fieldPath, i)
		if param.Name == "" {
			result.Valid = false
			result.Errors.Add(paramPath+".name", "parameter name is required")
		}
		if param.In == "" {
			result.Valid = false
			result.Errors.Add(paramPath+".in", "parameter location (in) is required")
		} else if param.In != "query" && param.In != "header" && param.In != "path" && param.In != "cookie" {
			result.Errors.Add(paramPath+".in",
				fmt.Sprintf("invalid parameter location '%s' (must be query, header, path, or cookie)", param.In))
			result.Valid = false
		}
		if param.In == "path" && !param.Required {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s: path parameters should be marked as required", paramPath))
		}
	}
}

func validateOpenAPIComponents(spec *OpenAPI, result *ValidationResult) {
	if spec.Components.Schemas != nil {
		for name, schema := range spec.Components.Schemas {
			if schema == nil {
				result.Errors.Add(fmt.Sprintf("components.schemas.%s", name), "schema cannot be null")
				result.Valid = false
			}
		}
	}
}

func validateOpenAPIServers(spec *OpenAPI, result *ValidationResult) {
	for i, server := range spec.Servers {
		if server.URL == "" {
			result.Errors.Add(fmt.Sprintf("servers[%d].url", i), "server URL is required")
			result.Valid = false
		}
	}
}

// ValidateAsyncAPI validates an AsyncAPI 2.x specification.
func ValidateAsyncAPI(spec *AsyncAPI) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Format: "AsyncAPI",
		Errors: &ValidationErrors{},
	}

	if spec == nil {
		result.Valid = false
		result.Errors.Add("", "specification is nil")
		return result
	}

	// Validate AsyncAPI version
	if spec.AsyncAPI == "" {
		result.Valid = false
		result.Errors.Add("asyncapi", "asyncapi version is required")
	} else {
		result.Version = spec.AsyncAPI
		result.Format = fmt.Sprintf("AsyncAPI %s", spec.AsyncAPI)
	}

	// Validate info
	if spec.Info.Title == "" {
		result.Valid = false
		result.Errors.Add("info.title", "title is required")
	}
	if spec.Info.Version == "" {
		result.Valid = false
		result.Errors.Add("info.version", "version is required")
	}

	// Validate channels
	if len(spec.Channels) == 0 {
		result.Warnings = append(result.Warnings, "no channels defined")
	}

	return result
}

// ValidateAsyncAPIV3 validates an AsyncAPI 3.x specification.
func ValidateAsyncAPIV3(spec *AsyncAPIV3) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Format: "AsyncAPI",
		Errors: &ValidationErrors{},
	}

	if spec == nil {
		result.Valid = false
		result.Errors.Add("", "specification is nil")
		return result
	}

	// Validate AsyncAPI version
	if spec.AsyncAPI == "" {
		result.Valid = false
		result.Errors.Add("asyncapi", "asyncapi version is required")
	} else {
		result.Version = spec.AsyncAPI
		result.Format = fmt.Sprintf("AsyncAPI %s", spec.AsyncAPI)
		if !strings.HasPrefix(spec.AsyncAPI, "3.") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("asyncapi version '%s' is not a 3.x version", spec.AsyncAPI))
		}
	}

	// Validate info
	if spec.Info.Title == "" {
		result.Valid = false
		result.Errors.Add("info.title", "title is required")
	}
	if spec.Info.Version == "" {
		result.Valid = false
		result.Errors.Add("info.version", "version is required")
	}

	return result
}

// ValidateAPIBlueprint validates an API Blueprint specification string.
func ValidateAPIBlueprint(content string) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Format: "API Blueprint",
		Errors: &ValidationErrors{},
	}

	if content == "" {
		result.Valid = false
		result.Errors.Add("", "content is empty")
		return result
	}

	lines := strings.Split(content, "\n")

	// Check for FORMAT header (recommended)
	hasFormat := false
	hasTitle := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "FORMAT:") {
			hasFormat = true
			result.Version = strings.TrimSpace(strings.TrimPrefix(trimmed, "FORMAT:"))
		}
		if strings.HasPrefix(trimmed, "# ") && !hasTitle {
			hasTitle = true
		}
	}

	if !hasFormat {
		result.Warnings = append(result.Warnings, "FORMAT: header is recommended")
	}

	if !hasTitle {
		result.Valid = false
		result.Errors.Add("title", "API title (# Title) is required")
	}

	return result
}

// ValidateJSON validates that the input is valid JSON and detects its format.
// It returns a ValidationResult with the detected format.
func ValidateJSON(data []byte) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: &ValidationErrors{},
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		result.Valid = false
		result.Errors.Add("", fmt.Sprintf("invalid JSON: %v", err))
		return result
	}

	// Detect format
	if _, ok := raw["openapi"]; ok {
		result.Format = "OpenAPI"
		if v, ok := raw["openapi"].(string); ok {
			result.Version = v
			result.Format = fmt.Sprintf("OpenAPI %s", v)
		}
	} else if _, ok := raw["asyncapi"]; ok {
		result.Format = "AsyncAPI"
		if v, ok := raw["asyncapi"].(string); ok {
			result.Version = v
			result.Format = fmt.Sprintf("AsyncAPI %s", v)
		}
	} else {
		result.Format = "Unknown JSON"
		result.Warnings = append(result.Warnings,
			"could not detect specification format (missing 'openapi' or 'asyncapi' field)")
	}

	return result
}

// ValidateReader validates a specification from an io.Reader.
// It auto-detects the format (OpenAPI, AsyncAPI, or API Blueprint).
func ValidateReader(r io.Reader) (*ValidationResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	return ValidateBytes(data), nil
}

// ValidateBytes validates a specification from bytes.
// It auto-detects the format and performs appropriate validation.
func ValidateBytes(data []byte) *ValidationResult {
	if len(data) == 0 {
		return &ValidationResult{
			Valid:  false,
			Errors: &ValidationErrors{Errors: []*ValidationError{{Message: "input is empty"}}},
		}
	}

	// Trim whitespace to check first character
	trimmed := strings.TrimSpace(string(data))

	// Check if it's JSON (starts with {)
	if strings.HasPrefix(trimmed, "{") {
		return validateJSONSpec(data)
	}

	// Assume API Blueprint
	return ValidateAPIBlueprint(string(data))
}

func validateJSONSpec(data []byte) *ValidationResult {
	// First, check basic JSON validity
	jsonResult := ValidateJSON(data)
	if !jsonResult.Valid {
		return jsonResult
	}

	// Detect and validate specific format
	var raw map[string]any
	_ = json.Unmarshal(data, &raw) // Already validated above

	if _, ok := raw["openapi"]; ok {
		spec, err := Parse(data)
		if err != nil {
			return &ValidationResult{
				Valid:  false,
				Format: "OpenAPI",
				Errors: &ValidationErrors{Errors: []*ValidationError{
					{Message: fmt.Sprintf("failed to parse OpenAPI spec: %v", err)},
				}},
			}
		}
		return ValidateOpenAPI(spec)
	}

	if v, ok := raw["asyncapi"].(string); ok {
		version := DetectAsyncAPIVersion(v)
		if version == 3 {
			spec, err := ParseAsyncAPIV3(data)
			if err != nil {
				return &ValidationResult{
					Valid:  false,
					Format: "AsyncAPI 3.x",
					Errors: &ValidationErrors{Errors: []*ValidationError{
						{Message: fmt.Sprintf("failed to parse AsyncAPI 3.x spec: %v", err)},
					}},
				}
			}
			return ValidateAsyncAPIV3(spec)
		}
		spec, err := ParseAsyncAPI(data)
		if err != nil {
			return &ValidationResult{
				Valid:  false,
				Format: "AsyncAPI 2.x",
				Errors: &ValidationErrors{Errors: []*ValidationError{
					{Message: fmt.Sprintf("failed to parse AsyncAPI 2.x spec: %v", err)},
				}},
			}
		}
		return ValidateAsyncAPI(spec)
	}

	return &ValidationResult{
		Valid:  false,
		Format: "Unknown",
		Errors: &ValidationErrors{Errors: []*ValidationError{{Message: "unknown specification format"}}},
	}
}
