package converter

import (
	"os"
	"testing"
)

func TestMSONParsing(t *testing.T) {
	data, err := os.ReadFile("../examples/mson-example.apib")
	if err != nil {
		t.Skip("skipping test because examples/mson-example.apib is missing")
	}

	bp, err := ParseBlueprint(data)
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	// Verify Components
	if len(spec.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas, got %d", len(spec.Components.Schemas))
	}

	if _, ok := spec.Components.Schemas["User"]; !ok {
		t.Error("Schema User not found")
	}
	if _, ok := spec.Components.Schemas["Item"]; !ok {
		t.Error("Schema Item not found")
	}

	userSchema := spec.Components.Schemas["User"]
	if len(userSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields in User, got %d", len(userSchema.Required))
	}

	// Verify Paths
	path := spec.Paths["/users"]
	if path.Post == nil {
		t.Fatal("POST /users not found")
	}

	// Verify Request Body
	req := path.Post.RequestBody
	if req == nil {
		t.Fatal("POST /users request body is nil")
	}
	content := req.Content["application/json"]
	if content.Schema == nil {
		t.Fatal("POST /users request schema is nil")
	}
	if content.Schema.Ref != "#/components/schemas/User" {
		t.Errorf("Expected ref to User, got %s", content.Schema.Ref)
	}
}

func TestMSONParsing_Attributes(t *testing.T) {
	apib := `FORMAT: 1A
# Test API

## Data Structures

### ComplexObject (object)
+ id (number, required)
+ name: Alice (string, optional) - The user name
+ tags (array[string])
+ active: true (boolean)
+ parent (User, optional)
`

	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	schema := spec.Components.Schemas["ComplexObject"]
	if schema == nil {
		t.Fatal("ComplexObject schema not found")
	}

	t.Run("ID Property", func(t *testing.T) {
		validateIDProperty(t, schema)
	})

	t.Run("Name Property", func(t *testing.T) {
		validateNameProperty(t, schema)
	})

	t.Run("Tags Property", func(t *testing.T) {
		validateTagsProperty(t, schema)
	})

	t.Run("Active Property", func(t *testing.T) {
		validateActiveProperty(t, schema)
	})

	t.Run("Parent Property", func(t *testing.T) {
		validateParentProperty(t, schema)
	})
}

func validateIDProperty(t *testing.T, schema *Schema) {
	t.Helper()
	idProp := schema.Properties["id"]
	if idProp.Type != "number" {
		t.Errorf("Expected id type number, got %v", idProp.Type)
	}
	if !sliceContains(schema.Required, "id") {
		t.Error("Expected id to be required")
	}
}

func validateNameProperty(t *testing.T, schema *Schema) {
	t.Helper()
	nameProp := schema.Properties["name"]
	if nameProp.Type != "string" {
		t.Errorf("Expected name type string, got %v", nameProp.Type)
	}
	if nameProp.Example != "Alice" {
		t.Errorf("Expected name example Alice, got %v", nameProp.Example)
	}
	if nameProp.Description != "The user name" {
		t.Errorf("Expected name description 'The user name', got %q", nameProp.Description)
	}
	if sliceContains(schema.Required, "name") {
		t.Error("Expected name to be optional")
	}
}

func validateTagsProperty(t *testing.T, schema *Schema) {
	t.Helper()
	tagsProp := schema.Properties["tags"]
	if tagsProp.Type != "array" {
		t.Errorf("Expected tags type array, got %v", tagsProp.Type)
	}
	if tagsProp.Items == nil || tagsProp.Items.Type != "string" {
		t.Error("Expected tags items to be string")
	}
}

func validateActiveProperty(t *testing.T, schema *Schema) {
	t.Helper()
	activeProp := schema.Properties["active"]
	if activeProp.Type != "boolean" {
		t.Errorf("Expected active type boolean, got %v", activeProp.Type)
	}
	if activeProp.Example != true {
		t.Errorf("Expected active example true, got %v", activeProp.Example)
	}
}

func validateParentProperty(t *testing.T, schema *Schema) {
	t.Helper()
	parentProp := schema.Properties["parent"]
	if parentProp.Ref != "#/components/schemas/User" {
		t.Errorf("Expected parent ref to User, got %s", parentProp.Ref)
	}
}

func TestMSONParsing_InlineAttributes(t *testing.T) {
	apib := `FORMAT: 1A
# Test API

## /test [/test]

### POST [POST]

+ Request (application/json)
    + Attributes
        + count: 10 (number)
`
	bp, err := ParseBlueprint([]byte(apib))
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
	}
	spec, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("ToOpenAPI failed: %v", err)
	}

	op := spec.Paths["/test"].Post
	content := op.RequestBody.Content["application/json"]
	schema := content.Schema

	if schema == nil {
		t.Fatal("Expected schema in request body")
	}

	countProp := schema.Properties["count"]
	if countProp == nil {
		t.Fatal("Expected count property")
	}
	if countProp.Type != "number" {
		t.Errorf("Expected type number, got %v", countProp.Type)
	}
	if countProp.Example != 10.0 {
		t.Errorf("Expected example 10, got %v", countProp.Example)
	}
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
