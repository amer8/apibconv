package converter

import (
	"strings"
	"testing"
)

const testOpenAPIJSON = `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0",
    "description": "A test API"
  },
  "servers": [
    {
      "url": "https://api.example.com"
    }
  ],
  "paths": {
    "/users": {
      "get": {
        "summary": "List users",
        "responses": {
          "200": {
            "description": "Success"
          }
        }
      }
    }
  }
}`

func TestParse(t *testing.T) {
	s, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	spec, ok := s.AsOpenAPI()
	    if !ok {
	        t.Fatalf("Expected *OpenAPI, got %T", s)
	    }
	
	    if spec.Title() != "Test API" {		t.Errorf("Expected title 'Test API', got '%s'", spec.Title())
	}

	// GetVersion returns the OpenAPI version
	if spec.Version() != "3.0.0" {
		t.Errorf("Expected version '3.0.0', got '%s'", spec.Version())
	}

	// Check API Info version directly
	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected info version '1.0.0', got '%s'", spec.Info.Version)
	}

	if len(spec.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(spec.Servers))
	}

	if len(spec.Paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(spec.Paths))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	// Parse with explicit format to force JSON parsing
	_, err := Parse([]byte(`{invalid json`), FormatOpenAPI)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// Test that Parse and Format are composable
func TestParseFormatComposition(t *testing.T) {
	// Parse
	s, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Modify the spec
	spec, ok := s.AsOpenAPI()
	if !ok {
		t.Fatalf("Expected *OpenAPI")
	}
	spec.Info.Title = "Modified API"

	// Format
	blueprint, err := spec.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("ToAPIBlueprint failed: %v", err)
	}

	if !strings.Contains(blueprint.String(), "# Modified API") {
		t.Error("Expected modified title in output")
	}
}

func TestParseYAML(t *testing.T) {
	yamlData := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`)
	s, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("Parse YAML failed: %v", err)
	}
	spec, ok := s.AsOpenAPI()
	if !ok {
		t.Fatalf("Expected *OpenAPI")
	}

	if spec.Title() != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Title())
	}
}

func TestParseInvalidYAML(t *testing.T) {
	// Invalid YAML type mismatch
	// The simplified parser handles indentation loosely, so we test type mismatch
	// openapi field should be string, not array
	yamlData := []byte(`
openapi: ["3.0.0"]
info:
  title: Test API
  version: 1.0.0
`)
	_, err := Parse(yamlData, FormatOpenAPI)
	if err == nil {
		t.Error("Expected error for invalid YAML type mismatch, got nil")
	}
}

func TestParseWithConversion(t *testing.T) {
	// 1. Test without options (should pass through)
	spec, err := ParseWithConversion([]byte(testOpenAPIJSON), nil)
	if err != nil {
		t.Fatalf("ParseWithConversion (nil opts) failed: %v", err)
	}
	if spec.OpenAPI != "3.0.0" {
		t.Errorf("Expected version 3.0.0, got %s", spec.OpenAPI)
	}

	// 2. Test with conversion to 3.1
	opts := &ConversionOptions{
		OutputVersion: Version31,
	}
	spec31, err := ParseWithConversion([]byte(testOpenAPIJSON), opts)
	if err != nil {
		t.Fatalf("ParseWithConversion (to 3.1) failed: %v", err)
	}
	if spec31.OpenAPI != "3.1.0" {
		t.Errorf("Expected version 3.1.0, got %s", spec31.OpenAPI)
	}

	// 3. Test with parse error
	_, err = ParseWithConversion([]byte(`{invalid`), nil)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
