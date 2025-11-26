package converter

import (
	"strings"
	"testing"
)

func TestValidateOpenAPI_Valid(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary: "List users",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	result := ValidateOpenAPI(spec)
	if !result.Valid {
		t.Errorf("ValidateOpenAPI() should be valid, got errors: %v", result.Errors)
	}
	if result.Format != "OpenAPI 3.0.0" {
		t.Errorf("ValidateOpenAPI().Format = %q, want %q", result.Format, "OpenAPI 3.0.0")
	}
}

func TestValidateOpenAPI_Nil(t *testing.T) {
	result := ValidateOpenAPI(nil)
	if result.Valid {
		t.Error("ValidateOpenAPI(nil) should be invalid")
	}
	if result.Errors.Count() == 0 {
		t.Error("ValidateOpenAPI(nil) should have errors")
	}
}

func TestValidateOpenAPI_MissingVersion(t *testing.T) {
	spec := &OpenAPI{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when openapi version is missing")
	}

	hasVersionError := false
	for _, err := range result.Errors.Errors {
		if err.Field == "openapi" {
			hasVersionError = true
			break
		}
	}
	if !hasVersionError {
		t.Error("ValidateOpenAPI() should have error for missing openapi version")
	}
}

func TestValidateOpenAPI_MissingTitle(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when title is missing")
	}

	hasTitleError := false
	for _, err := range result.Errors.Errors {
		if err.Field == "info.title" {
			hasTitleError = true
			break
		}
	}
	if !hasTitleError {
		t.Error("ValidateOpenAPI() should have error for missing title")
	}
}

func TestValidateOpenAPI_MissingInfoVersion(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title: "Test API",
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when info.version is missing")
	}
}

func TestValidateOpenAPI_InvalidPath(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"users": { // Missing leading /
				Get: &Operation{
					Responses: map[string]Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when path doesn't start with /")
	}
}

func TestValidateOpenAPI_MissingResponses(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary: "List users",
					// Missing responses
				},
			},
		},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when responses are missing")
	}
}

func TestValidateOpenAPI_InvalidParameterLocation(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Parameters: []Parameter{
						{
							Name: "id",
							In:   "invalid", // Invalid location
						},
					},
					Responses: map[string]Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when parameter location is invalid")
	}
}

func TestValidateOpenAPI_PathParameterNotRequired(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/users/{id}": {
				Get: &Operation{
					Parameters: []Parameter{
						{
							Name:     "id",
							In:       "path",
							Required: false, // Path params should be required
						},
					},
					Responses: map[string]Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
	}

	result := ValidateOpenAPI(spec)
	// Should produce warning, not error
	if len(result.Warnings) == 0 {
		t.Error("ValidateOpenAPI() should warn when path parameter is not required")
	}
}

func TestValidateOpenAPI_ServerURLRequired(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []Server{
			{URL: ""}, // Empty URL
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when server URL is empty")
	}
}

func TestValidateAsyncAPI_Valid(t *testing.T) {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:   "Test AsyncAPI",
			Version: "1.0.0",
		},
		Channels: map[string]Channel{
			"user/signedup": {},
		},
	}

	result := ValidateAsyncAPI(spec)
	if !result.Valid {
		t.Errorf("ValidateAsyncAPI() should be valid, got errors: %v", result.Errors)
	}
	if !strings.Contains(result.Format, "AsyncAPI 2.6.0") {
		t.Errorf("ValidateAsyncAPI().Format = %q, want to contain %q", result.Format, "AsyncAPI 2.6.0")
	}
}

func TestValidateAsyncAPI_Nil(t *testing.T) {
	result := ValidateAsyncAPI(nil)
	if result.Valid {
		t.Error("ValidateAsyncAPI(nil) should be invalid")
	}
}

func TestValidateAsyncAPI_MissingVersion(t *testing.T) {
	spec := &AsyncAPI{
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
	}

	result := ValidateAsyncAPI(spec)
	if result.Valid {
		t.Error("ValidateAsyncAPI() should be invalid when asyncapi version is missing")
	}
}

func TestValidateAsyncAPI_NoChannels(t *testing.T) {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
	}

	result := ValidateAsyncAPI(spec)
	// Should produce warning, not error
	if len(result.Warnings) == 0 {
		t.Error("ValidateAsyncAPI() should warn when no channels are defined")
	}
}

func TestValidateAsyncAPIV3_Valid(t *testing.T) {
	spec := &AsyncAPIV3{
		AsyncAPI: "3.0.0",
		Info: Info{
			Title:   "Test AsyncAPI V3",
			Version: "1.0.0",
		},
	}

	result := ValidateAsyncAPIV3(spec)
	if !result.Valid {
		t.Errorf("ValidateAsyncAPIV3() should be valid, got errors: %v", result.Errors)
	}
}

func TestValidateAsyncAPIV3_Nil(t *testing.T) {
	result := ValidateAsyncAPIV3(nil)
	if result.Valid {
		t.Error("ValidateAsyncAPIV3(nil) should be invalid")
	}
}

func TestValidateAsyncAPIV3_WrongVersion(t *testing.T) {
	spec := &AsyncAPIV3{
		AsyncAPI: "2.6.0", // Wrong version for V3 spec
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
	}

	result := ValidateAsyncAPIV3(spec)
	if len(result.Warnings) == 0 {
		t.Error("ValidateAsyncAPIV3() should warn when version is not 3.x")
	}
}

func TestValidateAPIBlueprint_Valid(t *testing.T) {
	content := `FORMAT: 1A

# Test API

A test API

## GET /users

+ Response 200 (application/json)
`

	result := ValidateAPIBlueprint(content)
	if !result.Valid {
		t.Errorf("ValidateAPIBlueprint() should be valid, got errors: %v", result.Errors)
	}
	if result.Format != "API Blueprint" {
		t.Errorf("ValidateAPIBlueprint().Format = %q, want %q", result.Format, "API Blueprint")
	}
}

func TestValidateAPIBlueprint_Empty(t *testing.T) {
	result := ValidateAPIBlueprint("")
	if result.Valid {
		t.Error("ValidateAPIBlueprint() should be invalid for empty content")
	}
}

func TestValidateAPIBlueprint_NoTitle(t *testing.T) {
	content := `FORMAT: 1A

## GET /users

+ Response 200
`

	result := ValidateAPIBlueprint(content)
	if result.Valid {
		t.Error("ValidateAPIBlueprint() should be invalid when title is missing")
	}
}

func TestValidateAPIBlueprint_NoFormat(t *testing.T) {
	content := `# Test API

## GET /users

+ Response 200
`

	result := ValidateAPIBlueprint(content)
	// Should produce warning but still be valid
	if len(result.Warnings) == 0 {
		t.Error("ValidateAPIBlueprint() should warn when FORMAT header is missing")
	}
}

func TestValidateJSON_Valid(t *testing.T) {
	data := []byte(`{"openapi": "3.0.0", "info": {"title": "Test"}}`)
	result := ValidateJSON(data)

	if !result.Valid {
		t.Error("ValidateJSON() should be valid for valid JSON")
	}
	if !strings.Contains(result.Format, "OpenAPI") {
		t.Errorf("ValidateJSON().Format = %q, should contain OpenAPI", result.Format)
	}
}

func TestValidateJSON_AsyncAPI(t *testing.T) {
	data := []byte(`{"asyncapi": "2.6.0", "info": {"title": "Test"}}`)
	result := ValidateJSON(data)

	if !result.Valid {
		t.Error("ValidateJSON() should be valid for valid JSON")
	}
	if !strings.Contains(result.Format, "AsyncAPI") {
		t.Errorf("ValidateJSON().Format = %q, should contain AsyncAPI", result.Format)
	}
}

func TestValidateJSON_Invalid(t *testing.T) {
	data := []byte(`{invalid json}`)
	result := ValidateJSON(data)

	if result.Valid {
		t.Error("ValidateJSON() should be invalid for invalid JSON")
	}
}

func TestValidateJSON_UnknownFormat(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	result := ValidateJSON(data)

	if len(result.Warnings) == 0 {
		t.Error("ValidateJSON() should warn for unknown format")
	}
}

func TestValidateBytes_Empty(t *testing.T) {
	result := ValidateBytes([]byte{})
	if result.Valid {
		t.Error("ValidateBytes() should be invalid for empty input")
	}
}

func TestValidateBytes_JSON(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)

	result := ValidateBytes(data)
	if !result.Valid {
		t.Errorf("ValidateBytes() should be valid for valid OpenAPI: %v", result.Errors)
	}
}

func TestValidateBytes_APIBlueprint(t *testing.T) {
	data := []byte(`FORMAT: 1A

# Test API

## GET /users

+ Response 200
`)

	result := ValidateBytes(data)
	if !result.Valid {
		t.Errorf("ValidateBytes() should be valid for valid API Blueprint: %v", result.Errors)
	}
}

func TestValidateReader(t *testing.T) {
	data := `{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`

	reader := strings.NewReader(data)
	result, err := ValidateReader(reader)

	if err != nil {
		t.Fatalf("ValidateReader() error = %v", err)
	}
	if !result.Valid {
		t.Errorf("ValidateReader() should be valid: %v", result.Errors)
	}
}

func TestValidationResult_Fields(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)

	if result.Version != "3.1.0" {
		t.Errorf("ValidationResult.Version = %q, want %q", result.Version, "3.1.0")
	}
	if !strings.Contains(result.Format, "3.1.0") {
		t.Errorf("ValidationResult.Format = %q, should contain version", result.Format)
	}
}

func TestValidateOpenAPI_UnsupportedVersion(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "4.0.0", // Unsupported version
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	result := ValidateOpenAPI(spec)
	// Should produce warning for unsupported version
	if len(result.Warnings) == 0 {
		t.Error("ValidateOpenAPI() should warn for unsupported OpenAPI version")
	}
}

func TestValidateOpenAPI_Components(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
		Components: &Components{
			Schemas: map[string]*Schema{
				"Valid": {Type: "object"},
				"Null":  nil, // Invalid null schema
			},
		},
	}

	result := ValidateOpenAPI(spec)
	if result.Valid {
		t.Error("ValidateOpenAPI() should be invalid when component schema is null")
	}
}
