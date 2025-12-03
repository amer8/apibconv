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

	spec, err := ParseAPIBlueprint(data)
	if err != nil {
		t.Fatalf("ParseAPIBlueprint failed: %v", err)
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
