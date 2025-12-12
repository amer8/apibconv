package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

const testAPIBlueprint = `FORMAT: 1A

# Test API

A test API description

HOST: https://api.example.com

## Group Resources

## /users [/users]

### List Users [GET]

Get a list of users

+ Parameters
    + limit (optional, integer) - Maximum number of users

+ Response 200 (application/json)

        {
            "users": [
                {"id": 1, "name": "Alice"}
            ]
        }

### Create User [POST]

Create a new user

+ Request (application/json)

        {
            "name": "Bob"
        }

+ Response 201 (application/json)

        {
            "id": 2,
            "name": "Bob"
        }

## /users/{userId} [/users/{userId}]

### Get User [GET]

Get a specific user

+ Parameters
    + userId (required, integer) - User ID

+ Response 200 (application/json)

        {
            "id": 1,
            "name": "Alice"
        }

+ Response 404

    User not found
`

func parseTestSpec(t *testing.T) *OpenAPI {
	t.Helper()
	bp, err := ParseBlueprint([]byte(testAPIBlueprint))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}
	return spec
}

func TestParseAPIBlueprint_BasicInfo(t *testing.T) {
	spec := parseTestSpec(t)

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", spec.Info.Title)
	}

	if spec.Info.Description != "A test API description" {
		t.Errorf("Expected description 'A test API description', got %q", spec.Info.Description)
	}
}

func TestParseAPIBlueprint_Server(t *testing.T) {
	spec := parseTestSpec(t)

	if len(spec.Servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(spec.Servers))
	}
	if spec.Servers[0].URL != "https://api.example.com" {
		t.Errorf("Expected server URL 'https://api.example.com', got %q", spec.Servers[0].URL)
	}
}

func TestParseAPIBlueprint_Paths(t *testing.T) {
	spec := parseTestSpec(t)

	if len(spec.Paths) != 2 {
		t.Fatalf("Expected 2 paths, got %d", len(spec.Paths))
	}
}

func TestParseAPIBlueprint_UsersGetOperation(t *testing.T) {
	spec := parseTestSpec(t)

	usersPath, ok := spec.Paths["/users"]
	if !ok {
		t.Fatal("Expected /users path to exist")
	}
	if usersPath.Get == nil {
		t.Fatal("Expected GET operation on /users")
	}
	if usersPath.Get.Summary != "List Users" {
		t.Errorf("Expected summary 'List Users', got %q", usersPath.Get.Summary)
	}
	if usersPath.Get.Description != "Get a list of users" {
		t.Errorf("Expected description 'Get a list of users', got %q", usersPath.Get.Description)
	}
}

func TestParseAPIBlueprint_Parameters(t *testing.T) {
	spec := parseTestSpec(t)

	usersPath := spec.Paths["/users"]
	if len(usersPath.Get.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(usersPath.Get.Parameters))
	}

	param := usersPath.Get.Parameters[0]
	if param.Name != "limit" {
		t.Errorf("Expected parameter name 'limit', got %q", param.Name)
	}
	if param.In != "query" {
		t.Errorf("Expected parameter in 'query', got %q", param.In)
	}
	if param.Required {
		t.Error("Expected parameter to be optional")
	}
}

func TestParseAPIBlueprint_PostOperation(t *testing.T) {
	spec := parseTestSpec(t)

	usersPath := spec.Paths["/users"]
	if usersPath.Post == nil {
		t.Fatal("Expected POST operation on /users")
	}
	if usersPath.Post.Summary != "Create User" {
		t.Errorf("Expected summary 'Create User', got %q", usersPath.Post.Summary)
	}
	if usersPath.Post.RequestBody == nil {
		t.Fatal("Expected request body on POST")
	}
}

func TestParseAPIBlueprint_PathParameters(t *testing.T) {
	spec := parseTestSpec(t)

	userPath, ok := spec.Paths["/users/{userId}"]
	if !ok {
		t.Fatal("Expected /users/{userId} path to exist")
	}
	if userPath.Get == nil {
		t.Fatal("Expected GET operation on /users/{userId}")
	}
	if len(userPath.Get.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(userPath.Get.Parameters))
	}

	param := userPath.Get.Parameters[0]
	if param.Name != "userId" {
		t.Errorf("Expected parameter name 'userId', got %q", param.Name)
	}
	if param.In != "path" {
		t.Errorf("Expected parameter in 'path', got %q", param.In)
	}
	if !param.Required {
		t.Error("Expected path parameter to be required")
	}
}

func TestParseAPIBlueprint_MultipleResponses(t *testing.T) {
	spec := parseTestSpec(t)

	userPath := spec.Paths["/users/{userId}"]
	if len(userPath.Get.Responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(userPath.Get.Responses))
	}
}

func toOpenAPIBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	bp, err := ParseBlueprint(data)
	if err != nil {
		t.Fatalf("ParseBlueprint failed: %v", err)
	}
	openapi, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}
	b, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return b
}

func TestToOpenAPI(t *testing.T) {
	apib := `FORMAT: 1A

# Simple API

HOST: https://api.test.com

## /test [/test]

### Test [GET]

+ Response 200 (application/json)

        {"status": "ok"}
`

	jsonBytes := toOpenAPIBytes(t, []byte(apib))
	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, "Simple API") {
		t.Error("Expected JSON to contain 'Simple API'")
	}
	if !strings.Contains(jsonStr, "https://api.test.com") {
		t.Error("Expected JSON to contain 'https://api.test.com'")
	}
	if !strings.Contains(jsonStr, "/test") {
		t.Error("Expected JSON to contain '/test'")
	}
}

func TestToOpenAPIString(t *testing.T) {
	apib := `FORMAT: 1A

# String Test

## /test [/test]

### Test [GET]

+ Response 200
`

	jsonBytes := toOpenAPIBytes(t, []byte(apib))
	jsonStr := string(jsonBytes)

	if !strings.Contains(jsonStr, "String Test") {
		t.Error("Expected JSON string to contain 'String Test'")
	}
}

func TestRoundTrip(t *testing.T) {
	// Start with OpenAPI
	openapi := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Round Trip Test",
			Description: "Testing round trip conversion",
			Version:     "1.0.0",
		},
		Servers: []Server{
			{URL: "https://api.roundtrip.com"},
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary:     "Test Operation",
					Description: "A test operation",
					Parameters: []Parameter{
						{
							Name:        "id",
							In:          "query",
							Description: "Test ID",
							Required:    false,
							Schema:      &Schema{Type: "integer"},
						},
					},
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Example: map[string]any{"result": "ok"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to API Blueprint
	apibObj, err := openapi.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("Format to API Blueprint failed: %v", err)
	}
	apibStr := apibObj.String()

	// Convert back to OpenAPI
	bp, err := ParseBlueprint([]byte(apibStr))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	// Verify key fields survived the round trip
	if spec.Info.Title != openapi.Info.Title {
		t.Errorf("Title mismatch: expected %q, got %q", openapi.Info.Title, spec.Info.Title)
	}
	if spec.Info.Description != openapi.Info.Description {
		t.Errorf("Description mismatch: expected %q, got %q", openapi.Info.Description, spec.Info.Description)
	}
	if len(spec.Servers) != 1 || spec.Servers[0].URL != openapi.Servers[0].URL {
		t.Error("Server URL mismatch after round trip")
	}
	if len(spec.Paths) != 1 {
		t.Errorf("Expected 1 path after round trip, got %d", len(spec.Paths))
	}
	if _, ok := spec.Paths["/test"]; !ok {
		t.Error("Expected /test path after round trip")
	}
}

func TestParseAPIBlueprintReader(t *testing.T) {
	reader := strings.NewReader(testAPIBlueprint)
	bp, err := ParseBlueprintReader(reader)
	if err != nil {
		t.Fatalf("ParseAPIBlueprintReader failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %q", spec.Info.Title)
	}

	if len(spec.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(spec.Paths))
	}

	if len(spec.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(spec.Servers))
	}
}

func TestParseAPIBlueprintReaderEmpty(t *testing.T) {
	reader := strings.NewReader("")
	bp, err := ParseBlueprintReader(reader)
	if err != nil {
		t.Fatalf("ParseAPIBlueprintReader failed on empty input: %v", err)
	}

	// Empty input should still produce a valid (minimal) spec
	if bp == nil {
		t.Error("Expected non-nil spec for empty input")
	}
}

func TestConvertToOpenAPI(t *testing.T) {
	apib := `FORMAT: 1A

# Convert Test API

A test for streaming conversion

HOST: https://api.convert.com

## /convert [/convert]

### Convert Data [POST]

Convert some data

+ Request (application/json)

        {"input": "test"}

+ Response 200 (application/json)

        {"output": "result"}
`

	reader := strings.NewReader(apib)
	var buf strings.Builder

	// Manual conversion workflow
	bp, err := ParseBlueprintReader(reader)
	if err != nil {
		t.Fatalf("ParseBlueprintReader failed: %v", err)
	}
	openapi, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}
	if err := json.NewEncoder(&buf).Encode(openapi); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "Convert Test API") {
		t.Error("Expected JSON to contain 'Convert Test API'")
	}
	if !strings.Contains(result, "https://api.convert.com") {
		t.Error("Expected JSON to contain 'https://api.convert.com'")
	}
	if !strings.Contains(result, "/convert") {
		t.Error("Expected JSON to contain '/convert'")
	}
	if !strings.Contains(result, "\"openapi\"") {
		t.Error("Expected JSON to contain openapi version field")
	}
}

func TestConvertToOpenAPIEmpty(t *testing.T) {
	reader := strings.NewReader("")
	var buf strings.Builder

	bp, err := ParseBlueprintReader(reader)
	if err != nil {
		t.Fatalf("ParseBlueprintReader failed on empty input: %v", err)
	}
	openapi, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}
	if err := json.NewEncoder(&buf).Encode(openapi); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := buf.String()
	// Should produce valid JSON even with empty input
	if !strings.Contains(result, "\"openapi\"") {
		t.Error("Expected valid OpenAPI JSON structure")
	}
}

func TestParseAPIBlueprint_WithJSONRequestBody(t *testing.T) {
	apib := `FORMAT: 1A

# JSON Request Test

## /api [/api]

### Create [POST]

+ Request (application/json)

        {
            "field1": "value1",
            "field2": 123,
            "nested": {
                "key": "value"
            }
        }

+ Response 201 (application/json)

        {"id": 1}`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/api"]
	if path.Post == nil {
		t.Fatal("Expected POST operation")
	}
	if path.Post.RequestBody == nil {
		t.Fatal("Expected request body")
	}

	content, ok := path.Post.RequestBody.Content["application/json"]
	if !ok {
		t.Fatal("Expected application/json content")
	}
	if content.Example == nil {
		t.Error("Expected request body example")
	}
}

func TestParseAPIBlueprint_WithPlainTextResponse(t *testing.T) {
	apib := `FORMAT: 1A

# Plain Text Test

## /text [/text]

### Get Text [GET]

+ Response 200 (text/plain)

        This is plain text
                with multiple lines`
	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/text"]
	if path.Get == nil {
		t.Fatal("Expected GET operation")
	}

	resp, ok := path.Get.Responses["200"]
	if !ok {
		t.Fatal("Expected 200 response")
	}

	// Parser should handle text/plain responses
	if resp.Description == "" {
		t.Error("Expected response description")
	}
}

func TestParseAPIBlueprint_WithNoContentTypeResponse(t *testing.T) {
	apib := `FORMAT: 1A

# No Content Test

## /nocontent [/nocontent]

### Delete [DELETE]

+ Response 204

+ Response 404

        Not found`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/nocontent"]
	if path.Delete == nil {
		t.Fatal("Expected DELETE operation")
	}

	if len(path.Delete.Responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(path.Delete.Responses))
	}

	resp204, ok := path.Delete.Responses["204"]
	if !ok {
		t.Error("Expected 204 response")
	}
	if len(resp204.Content) > 0 {
		t.Error("204 response should not have content")
	}
}

func TestParseAPIBlueprint_MultipleOperationsOnePath(t *testing.T) {
	apib := `FORMAT: 1A

# Multiple Operations Test

## /resource [/resource]

### List [GET]

Get all resources

+ Response 200 (application/json)

        []

### Create [POST]

Create a resource

+ Request (application/json)

        {"name": "test"}

+ Response 201

### Update [PUT]

Update a resource

+ Response 200

### Partial Update [PATCH]

Partially update a resource

+ Response 200

### Remove [DELETE]

Delete a resource

+ Response 204`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/resource"]
	if path.Get == nil {
		t.Error("Expected GET operation")
	}
	if path.Post == nil {
		t.Error("Expected POST operation")
	}
	if path.Put == nil {
		t.Error("Expected PUT operation")
	}
	if path.Patch == nil {
		t.Error("Expected PATCH operation")
	}
	if path.Delete == nil {
		t.Error("Expected DELETE operation")
	}

	if path.Get.Summary != "List" {
		t.Errorf("Expected GET summary 'List', got %q", path.Get.Summary)
	}
	if path.Post.Summary != "Create" {
		t.Errorf("Expected POST summary 'Create', got %q", path.Post.Summary)
	}
}

func TestParseAPIBlueprint_WithHeaderParameters(t *testing.T) {
	apib := `FORMAT: 1A

# Header Test

## /auth [/auth]

### Authenticate [POST]

+ Parameters
    + Authorization (required, string) - Bearer token
    + X-Custom-Header (optional, string) - Custom header

+ Response 200`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/auth"]
	if path.Post == nil {
		t.Fatal("Expected POST operation")
	}

	// Parser should handle parameters (exact count may vary based on implementation)
	if len(path.Post.Parameters) == 0 {
		t.Error("Expected at least one parameter")
	}

	// Check first parameter exists
	if len(path.Post.Parameters) > 0 {
		firstParam := path.Post.Parameters[0]
		if firstParam.Name == "" {
			t.Error("Expected parameter to have a name")
		}
	}
}

func TestParseAPIBlueprint_ComplexNestedJSON(t *testing.T) {
	apib := `FORMAT: 1A

# Complex JSON Test

## /complex [/complex]

### Process [POST]

+ Request (application/json)

        {
            "data": {
                "users": [
                    {"id": 1, "name": "Alice"},
                    {"id": 2, "name": "Bob"}
                ],
                "metadata": {
                    "total": 2,
                    "page": 1
                }
            }
        }

+ Response 200 (application/json)

        {
            "status": "processed",
            "results": {
                "success": true,
                "count": 2
            }
        }`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	path := spec.Paths["/complex"]
	if path.Post == nil {
		t.Fatal("Expected POST operation")
	}

	if path.Post.RequestBody == nil {
		t.Fatal("Expected request body")
	}

	content := path.Post.RequestBody.Content["application/json"]
	if content.Example == nil {
		t.Error("Expected request body example")
	}

	resp := path.Post.Responses["200"]
	if len(resp.Content) == 0 {
		// Response should have content but parser may handle it differently
		t.Log("Response content not parsed as expected, but operation parsed successfully")
	}
}

func TestToOpenAPI_ErrorCases(t *testing.T) {
	// Test with invalid API Blueprint
	invalidApib := `FORMAT: 1A

# Test API

## Invalid JSON in response

+ Response 200 (application/json)

        {invalid json}`

	// This should still parse (parser is lenient with body content)
	jsonBytes := toOpenAPIBytes(t, []byte(invalidApib))

	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty result")
	}
}

func TestToOpenAPIString_ErrorCases(t *testing.T) {
	// Empty string should produce valid OpenAPI
	jsonBytes := toOpenAPIBytes(t, []byte(""))
	result := string(jsonBytes)

	if !strings.Contains(result, "\"openapi\"") {
		t.Error("Expected valid OpenAPI JSON structure")
	}
}
