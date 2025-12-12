package converter

import (
	"encoding/json"
	"testing"
)

func TestConvertToVersion_30to31(t *testing.T) {
	// Create a 3.0 spec with nullable schema
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test operation",
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type:     "string",
										Nullable: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	converted, err := spec.ConvertTo(Version31, nil)
	if err != nil {
		t.Fatalf("ConvertToVersion failed: %v", err)
	}

	if converted.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want %q", converted.OpenAPI, "3.1.0")
	}

	// Check that nullable was converted to type array
	schema := converted.Paths["/test"].Get.Responses["200"].Content["application/json"].Schema
	if schema.Nullable {
		t.Error("Schema should not have Nullable set in 3.1")
	}

	typeArr, ok := schema.Type.([]any)
	if !ok {
		t.Fatalf("Schema.Type should be []any, got %T", schema.Type)
	}

	if len(typeArr) != 2 {
		t.Errorf("Type array length = %d, want 2", len(typeArr))
	}

	hasString := false
	hasNull := false
	for _, v := range typeArr {
		if str, ok := v.(string); ok {
			if str == "string" {
				hasString = true
			}
			if str == "null" {
				hasNull = true
			}
		}
	}

	if !hasString || !hasNull {
		t.Errorf("Type array should contain 'string' and 'null', got %v", typeArr)
	}
}

func TestConvertToVersion_31to30(t *testing.T) {
	// Create a 3.1 spec with type array
	spec := &OpenAPI{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
			License: &License{
				Name:       "MIT",
				Identifier: "MIT",
			},
		},
		Webhooks: map[string]PathItem{
			"newUser": {
				Post: &Operation{
					Summary: "New user webhook",
					Responses: map[string]Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
		JSONSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test operation",
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: []any{"string", "null"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	converted, err := spec.ConvertTo(Version30, nil)
	if err != nil {
		t.Fatalf("ConvertToVersion failed: %v", err)
	}

	if converted.OpenAPI != "3.0.0" {
		t.Errorf("OpenAPI version = %q, want %q", converted.OpenAPI, "3.0.0")
	}

	// Check that 3.1-only features were removed
	if len(converted.Webhooks) != 0 {
		t.Error("Webhooks should be removed in 3.0")
	}

	if converted.JSONSchemaDialect != "" {
		t.Error("JSONSchemaDialect should be removed in 3.0")
	}

	if converted.Info.License != nil && converted.Info.License.Identifier != "" {
		t.Error("License.Identifier should be removed in 3.0")
	}

	// Check that type array was converted to nullable
	schema := converted.Paths["/test"].Get.Responses["200"].Content["application/json"].Schema
	if !schema.Nullable {
		t.Error("Schema should have Nullable set in 3.0")
	}

	typeStr, ok := schema.Type.(string)
	if !ok {
		t.Fatalf("Schema.Type should be string, got %T", schema.Type)
	}

	if typeStr != "string" {
		t.Errorf("Schema.Type = %q, want %q", typeStr, "string")
	}
}

func TestConvertToVersion_31to30_StrictMode(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Webhooks: map[string]PathItem{
			"newUser": {
				Post: &Operation{
					Summary: "Webhook",
					Responses: map[string]Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
		Paths: map[string]PathItem{},
	}

	opts := &ConversionOptions{
		OutputVersion: Version30,
		StrictMode:    true,
	}

	_, err := spec.ConvertTo(Version30, opts)
	if err == nil {
		t.Error("ConvertToVersion with StrictMode should fail when webhooks are present")
	}
}

func TestConvertToVersion_NoConversionNeeded(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	converted, err := spec.ConvertTo(Version30, nil)
	if err != nil {
		t.Fatalf("ConvertToVersion failed: %v", err)
	}

	if converted.OpenAPI != "3.0.0" {
		t.Errorf("OpenAPI version = %q, want %q", converted.OpenAPI, "3.0.0")
	}
}

func TestSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		expected string
	}{
		{
			name:     "Nil schema",
			schema:   nil,
			expected: "",
		},
		{
			name: "String type (3.0)",
			schema: &Schema{
				Type: "string",
			},
			expected: "string",
		},
		{
			name: "Type array with string and null (3.1)",
			schema: &Schema{
				Type: []any{"string", "null"},
			},
			expected: "string",
		},
		{
			name: "Type array with only null",
			schema: &Schema{
				Type: []any{"null"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schema.TypeName()
			if result != tt.expected {
				t.Errorf("SchemaType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsNullable(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		expected bool
	}{
		{
			name:     "Nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name: "3.0 nullable true",
			schema: &Schema{
				Type:     "string",
				Nullable: true,
			},
			expected: true,
		},
		{
			name: "3.0 nullable false",
			schema: &Schema{
				Type:     "string",
				Nullable: false,
			},
			expected: false,
		},
		{
			name: "3.1 type array with null",
			schema: &Schema{
				Type: []any{"string", "null"},
			},
			expected: true,
		},
		{
			name: "3.1 type array without null",
			schema: &Schema{
				Type: []any{"string"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schema.IsNullable()
			if result != tt.expected {
				t.Errorf("IsNullable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertNestedSchemas(t *testing.T) {
	// Test that nested schemas are properly converted
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Parameters: []Parameter{
						{
							Name:     "id",
							In:       "query",
							Required: true,
							Schema: &Schema{
								Type:     "string",
								Nullable: true,
							},
						},
					},
					RequestBody: &RequestBody{
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"name": {
											Type:     "string",
											Nullable: true,
										},
										"age": {
											Type: "integer",
										},
									},
								},
							},
						},
					},
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "array",
										Items: &Schema{
											Type:     "string",
											Nullable: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	converted, err := spec.ConvertTo(Version31, nil)
	if err != nil {
		t.Fatalf("ConvertToVersion failed: %v", err)
	}

	// Check parameter schema
	paramSchema := converted.Paths["/test"].Get.Parameters[0].Schema
	if paramSchema.IsNullable() != true {
		t.Error("Parameter schema should be nullable")
	}

	// Check request body nested property
	nameSchema := converted.Paths["/test"].Get.RequestBody.Content["application/json"].Schema.Properties["name"]
	if nameSchema.IsNullable() != true {
		t.Error("Request body 'name' property should be nullable")
	}

	// Check array items schema
	itemsSchema := converted.Paths["/test"].Get.Responses["200"].Content["application/json"].Schema.Items
	if itemsSchema.IsNullable() != true {
		t.Error("Array items schema should be nullable")
	}
}

func TestRoundTripConversion(t *testing.T) {
	// Test that converting 3.0 -> 3.1 -> 3.0 produces equivalent result
	original := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Test API",
			Version:     "1.0.0",
			Description: "A test API",
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary: "List users",
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type:     "array",
										Nullable: true,
										Items: &Schema{
											Type: "object",
											Properties: map[string]*Schema{
												"id": {
													Type: "string",
												},
												"name": {
													Type:     "string",
													Nullable: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to 3.1
	converted31, err := original.ConvertTo(Version31, nil)
	if err != nil {
		t.Fatalf("Conversion to 3.1 failed: %v", err)
	}

	// Convert back to 3.0
	converted30, err := converted31.ConvertTo(Version30, nil)
	if err != nil {
		t.Fatalf("Conversion back to 3.0 failed: %v", err)
	}

	// Compare key properties
	if converted30.OpenAPI != original.OpenAPI {
		t.Errorf("OpenAPI version mismatch: got %q, want %q", converted30.OpenAPI, original.OpenAPI)
	}

	if converted30.Info.Title != original.Info.Title {
		t.Errorf("Title mismatch: got %q, want %q", converted30.Info.Title, original.Info.Title)
	}

	// Check nullable was preserved
	schema := converted30.Paths["/users"].Get.Responses["200"].Content["application/json"].Schema
	if !schema.Nullable {
		t.Error("Top-level schema nullable property was not preserved")
	}

	nameSchema := schema.Items.Properties["name"]
	if !nameSchema.Nullable {
		t.Error("Nested schema nullable property was not preserved")
	}
}

func TestJSONMarshaling(t *testing.T) {
	// Test that converted specs can be marshaled to JSON without errors
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test",
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type:     "string",
										Nullable: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to 3.1
	converted, err := spec.ConvertTo(Version31, nil)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(converted, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshaling failed: %v", err)
	}

	// Unmarshal back
	var unmarshaled OpenAPI
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshaling failed: %v", err)
	}

	if unmarshaled.OpenAPI != "3.1.0" {
		t.Errorf("Unmarshaled version = %q, want %q", unmarshaled.OpenAPI, "3.1.0")
	}
}

//nolint:gocyclo // Test function complexity is acceptable for comprehensive table-driven tests
func TestNormalizeSchemaType(t *testing.T) {
	tests := []struct {
		name           string
		schema         *Schema
		version        Version
		expectedType   any
		expectedNull   bool
		expectedErrMsg string
	}{
		{
			name:    "Nil schema",
			schema:  nil,
			version: Version30,
		},
		{
			name: "3.0 - string type stays as string",
			schema: &Schema{
				Type: "string",
			},
			version:      Version30,
			expectedType: "string",
			expectedNull: false,
		},
		{
			name: "3.0 - type array converted to string",
			schema: &Schema{
				Type: []any{"string", "null"},
			},
			version:      Version30,
			expectedType: "string",
			expectedNull: false,
		},
		{
			name: "3.1 - string type stays as string",
			schema: &Schema{
				Type: "string",
			},
			version:      Version31,
			expectedType: "string",
			expectedNull: false,
		},
		{
			name: "3.1 - nullable converted to type array",
			schema: &Schema{
				Type:     "string",
				Nullable: true,
			},
			version:      Version31,
			expectedType: []any{"string", "null"},
			expectedNull: false,
		},
		{
			name: "3.1 - type array stays as type array",
			schema: &Schema{
				Type: []any{"integer", "null"},
			},
			version:      Version31,
			expectedType: []any{"integer", "null"},
			expectedNull: false,
		},
		{
			name: "3.0 - nested properties normalized",
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": {
						Type:     "string",
						Nullable: true,
					},
				},
			},
			version:      Version30,
			expectedType: "object",
			expectedNull: false,
		},
		{
			name: "3.1 - nested properties normalized",
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"age": {
						Type:     "integer",
						Nullable: true,
					},
				},
			},
			version:      Version31,
			expectedType: "object",
			expectedNull: false,
		},
		{
			name: "3.0 - array items normalized",
			schema: &Schema{
				Type: "array",
				Items: &Schema{
					Type:     "string",
					Nullable: true,
				},
			},
			version:      Version30,
			expectedType: "array",
			expectedNull: false,
		},
		{
			name: "3.1 - array items normalized",
			schema: &Schema{
				Type: "array",
				Items: &Schema{
					Type:     "boolean",
					Nullable: true,
				},
			},
			version:      Version31,
			expectedType: "array",
			expectedNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.schema == nil {
				tt.schema.NormalizeType(tt.version)
				return
			}

			tt.schema.NormalizeType(tt.version)

			// Check type
			if tt.expectedType != nil {
				switch expected := tt.expectedType.(type) {
				case string:
					if got, ok := tt.schema.Type.(string); !ok || got != expected {
						t.Errorf("Type = %v (%T), want %v", tt.schema.Type, tt.schema.Type, expected)
					}
				case []any:
					gotArr, ok := tt.schema.Type.([]any)
					if !ok {
						t.Errorf("Type should be []any, got %T", tt.schema.Type)
					} else if len(gotArr) != len(expected) {
						t.Errorf("Type array length = %d, want %d", len(gotArr), len(expected))
					}
				}
			}

			// Check nullable
			if tt.schema.Nullable != tt.expectedNull {
				t.Errorf("Nullable = %v, want %v", tt.schema.Nullable, tt.expectedNull)
			}

			// For nested properties test
			if len(tt.schema.Properties) > 0 {
				for _, prop := range tt.schema.Properties {
					if tt.version == Version31 {
						// Should have converted nullable to type array
						if typeArr, ok := prop.Type.([]any); !ok || len(typeArr) != 2 {
							t.Errorf("Nested property should have type array with 2 elements")
						}
						if prop.Nullable {
							t.Error("Nested property should not have nullable set in 3.1")
						}
					} else {
						// 3.0 should keep as string
						if typeStr, ok := prop.Type.(string); !ok || typeStr == "" {
							t.Errorf("Nested property should have string type in 3.0")
						}
					}
				}
			}

			// For array items test
			if tt.schema.Items != nil {
				if tt.version == Version31 {
					// Should have converted nullable to type array
					if typeArr, ok := tt.schema.Items.Type.([]any); !ok || len(typeArr) != 2 {
						t.Errorf("Array items should have type array with 2 elements")
					}
					if tt.schema.Items.Nullable {
						t.Error("Array items should not have nullable set in 3.1")
					}
				} else {
					// 3.0 should keep as string
					if typeStr, ok := tt.schema.Items.Type.(string); !ok || typeStr == "" {
						t.Errorf("Array items should have string type in 3.0")
					}
				}
			}
		})
	}
}

func TestNormalizeSchemaType_ComplexNesting(t *testing.T) {
	// Test deep nesting with multiple levels
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"user": {
				Type: "object",
				Properties: map[string]*Schema{
					"profile": {
						Type: "object",
						Properties: map[string]*Schema{
							"avatar": {
								Type:     "string",
								Nullable: true,
							},
						},
					},
				},
			},
			"tags": {
				Type: "array",
				Items: &Schema{
					Type:     "string",
					Nullable: true,
				},
			},
		},
	}

	schema.NormalizeType(Version31)

	// Check deeply nested property
	avatarSchema := schema.Properties["user"].Properties["profile"].Properties["avatar"]
	if avatarSchema.Nullable {
		t.Error("Deeply nested property should not have nullable in 3.1")
	}
	if typeArr, ok := avatarSchema.Type.([]any); !ok || len(typeArr) != 2 {
		t.Errorf("Deeply nested property should have type array, got %v", avatarSchema.Type)
	}

	// Check array items
	tagsItems := schema.Properties["tags"].Items
	if tagsItems.Nullable {
		t.Error("Array items should not have nullable in 3.1")
	}
	if typeArr, ok := tagsItems.Type.([]any); !ok || len(typeArr) != 2 {
		t.Errorf("Array items should have type array, got %v", tagsItems.Type)
	}
}
