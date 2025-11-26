package converter

import (
	"bytes"
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
	spec, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}

	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", spec.Info.Version)
	}

	if len(spec.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(spec.Servers))
	}

	if len(spec.Paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(spec.Paths))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseReader(t *testing.T) {
	reader := strings.NewReader(testOpenAPIJSON)
	spec, err := ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}
}

func TestFormat(t *testing.T) {
	spec, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result, err := Format(spec)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Check for expected API Blueprint elements
	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}

	if !strings.Contains(result, "# Test API") {
		t.Error("Expected API title in output")
	}

	if !strings.Contains(result, "https://api.example.com") {
		t.Error("Expected server URL in output")
	}

	if !strings.Contains(result, "/users") {
		t.Error("Expected /users path in output")
	}
}

func TestFormatNilSpec(t *testing.T) {
	_, err := Format(nil)
	if err == nil {
		t.Error("Expected error for nil spec, got nil")
	}
}

func TestFormatTo(t *testing.T) {
	spec, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var buf bytes.Buffer
	err = FormatTo(spec, &buf)
	if err != nil {
		t.Fatalf("FormatTo failed: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestFormatToNilSpec(t *testing.T) {
	var buf bytes.Buffer
	err := FormatTo(nil, &buf)
	if err == nil {
		t.Error("Expected error for nil spec, got nil")
	}
}

func TestFromJSON(t *testing.T) {
	result, err := FromJSON([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}

	if !strings.Contains(result, "# Test API") {
		t.Error("Expected API title in output")
	}
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON([]byte(`{invalid`))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestFromJSONString(t *testing.T) {
	result, err := FromJSONString(testOpenAPIJSON)
	if err != nil {
		t.Fatalf("FromJSONString failed: %v", err)
	}

	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestToBytes(t *testing.T) {
	result, err := ToBytes([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestConvertString(t *testing.T) {
	result, err := ConvertString(testOpenAPIJSON)
	if err != nil {
		t.Fatalf("ConvertString failed: %v", err)
	}

	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestMustFormat(t *testing.T) {
	spec, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should not panic
	result := MustFormat(spec)

	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestMustFormatPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil spec")
		}
	}()

	MustFormat(nil)
}

func TestMustFromJSON(t *testing.T) {
	// Should not panic
	result := MustFromJSON([]byte(testOpenAPIJSON))

	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestMustFromJSONPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid JSON")
		}
	}()

	MustFromJSON([]byte(`{invalid`))
}

// Test that Parse and Format are composable
func TestParseFormatComposition(t *testing.T) {
	// Parse
	spec, err := Parse([]byte(testOpenAPIJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Modify the spec
	spec.Info.Title = "Modified API"

	// Format
	result, err := Format(spec)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(result, "# Modified API") {
		t.Error("Expected modified title in output")
	}
}

// Test backward compatibility with existing Convert function
func TestConvertCompatibility(t *testing.T) {
	reader := strings.NewReader(testOpenAPIJSON)
	var buf bytes.Buffer

	err := Convert(reader, &buf)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

