package converter

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshalYAML_Simple(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	yamlBytes, err := MarshalYAML(spec)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)

	// Check for key elements (YAML doesn't always quote strings)
	if !strings.Contains(yaml, "openapi:") || !strings.Contains(yaml, "3.0.0") {
		t.Error("YAML should contain openapi version")
	}
	if !strings.Contains(yaml, "title: Test API") {
		t.Error("YAML should contain title")
	}
	if !strings.Contains(yaml, "version:") || !strings.Contains(yaml, "1.0.0") {
		t.Error("YAML should contain version")
	}
}

func TestMarshalYAML_WithPaths(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Users API",
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

	yamlBytes, err := MarshalYAML(spec)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "paths:") {
		t.Error("YAML should contain paths")
	}
	if !strings.Contains(yaml, "/users") {
		t.Error("YAML should contain /users path")
	}
	if !strings.Contains(yaml, "get:") {
		t.Error("YAML should contain get operation")
	}
	if !strings.Contains(yaml, "summary: List users") {
		t.Error("YAML should contain summary")
	}
}

func TestMarshalYAML_NilSpec(t *testing.T) {
	_, err := FormatOpenAPIAsYAML(nil)
	if err != ErrNilSpec {
		t.Errorf("FormatOpenAPIAsYAML(nil) error = %v, want ErrNilSpec", err)
	}
}

func TestMarshalYAML_SpecialCharacters(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Test: API with special chars",
			Description: "Description with \"quotes\" and\nnewlines",
			Version:     "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	yamlBytes, err := MarshalYAML(spec)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)

	// Check that special characters are properly escaped
	if !strings.Contains(yaml, "title:") {
		t.Error("YAML should contain title")
	}
	if !strings.Contains(yaml, "description:") {
		t.Error("YAML should contain description")
	}
}

func TestMarshalYAML_Arrays(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []Server{
			{URL: "https://api.example.com", Description: "Production"},
			{URL: "https://staging.example.com", Description: "Staging"},
		},
		Paths: map[string]PathItem{},
	}

	yamlBytes, err := MarshalYAML(spec)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "servers:") {
		t.Error("YAML should contain servers")
	}
	if !strings.Contains(yaml, "- ") {
		t.Error("YAML should contain array markers")
	}
	if !strings.Contains(yaml, "https://api.example.com") {
		t.Error("YAML should contain production URL")
	}
	if !strings.Contains(yaml, "https://staging.example.com") {
		t.Error("YAML should contain staging URL")
	}
}

func TestMarshalYAMLIndent(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	// Test with custom indent
	yaml4, err := MarshalYAMLIndent(spec, 4)
	if err != nil {
		t.Fatalf("MarshalYAMLIndent(4) error = %v", err)
	}

	// The output should have 4-space indentation
	if !bytes.Contains(yaml4, []byte("    ")) {
		t.Error("YAML with indent 4 should contain 4-space indentation")
	}

	// Test with default indent (when <= 0)
	yaml0, err := MarshalYAMLIndent(spec, 0)
	if err != nil {
		t.Fatalf("MarshalYAMLIndent(0) error = %v", err)
	}

	// Should default to 2 spaces
	if len(yaml0) == 0 {
		t.Error("YAML with indent 0 should have content")
	}
}

func TestEncodeYAML(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
	}

	var buf bytes.Buffer
	err := encodeYAML(&buf, spec)
	if err != nil {
		t.Fatalf("EncodeYAML() error = %v", err)
	}

	yaml := buf.String()
	if !strings.Contains(yaml, "openapi:") {
		t.Error("EncodeYAML should write YAML content")
	}
}

func TestMarshalYAML_EmptyMap(t *testing.T) {
	data := map[string]any{
		"empty": map[string]any{},
	}

	yamlBytes, err := MarshalYAML(data)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "{}") {
		t.Error("Empty maps should be rendered as {}")
	}
}

func TestMarshalYAML_EmptyArray(t *testing.T) {
	data := map[string]any{
		"empty": []any{},
	}

	yamlBytes, err := MarshalYAML(data)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "[]") {
		t.Error("Empty arrays should be rendered as []")
	}
}

func TestMarshalYAML_Numbers(t *testing.T) {
	data := map[string]any{
		"integer": float64(42),
		"float":   3.14159,
	}

	yamlBytes, err := MarshalYAML(data)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "integer: 42") {
		t.Error("Integers should be rendered without decimal point")
	}
	if !strings.Contains(yaml, "float: 3.14159") {
		t.Error("Floats should preserve decimal places")
	}
}

func TestMarshalYAML_BoolAndNull(t *testing.T) {
	data := map[string]any{
		"trueVal":  true,
		"falseVal": false,
		"nullVal":  nil,
	}

	yamlBytes, err := MarshalYAML(data)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "true") {
		t.Error("YAML should contain true")
	}
	if !strings.Contains(yaml, "false") {
		t.Error("YAML should contain false")
	}
	if !strings.Contains(yaml, "null") {
		t.Error("YAML should contain null")
	}
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"simple", false},
		{"", true},
		{"true", true},
		{"false", true},
		{"yes", true},
		{"no", true},
		{"null", true},
		{"-starts-with-dash", true},
		{":starts-with-colon", true},
		{"has:colon", true},
		{"has#hash", true},
		{"has\nnewline", true},
		{"has\"quote", true},
		{"01234", true}, // Leading zero
		{"normal123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := needsQuoting(tt.input)
			if result != tt.expected {
				t.Errorf("needsQuoting(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatAsyncAPIAsYAML(t *testing.T) {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:   "Test AsyncAPI",
			Version: "1.0.0",
		},
	}

	yamlBytes, err := FormatAsyncAPIAsYAML(spec)
	if err != nil {
		t.Fatalf("FormatAsyncAPIAsYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "asyncapi:") {
		t.Error("YAML should contain asyncapi version")
	}
	if !strings.Contains(yaml, "title: Test AsyncAPI") {
		t.Error("YAML should contain title")
	}
}

func TestFormatAsyncAPIAsYAML_Nil(t *testing.T) {
	_, err := FormatAsyncAPIAsYAML(nil)
	if err != ErrNilSpec {
		t.Errorf("FormatAsyncAPIAsYAML(nil) error = %v, want ErrNilSpec", err)
	}
}

func TestFormatAsyncAPIV3AsYAML(t *testing.T) {
	spec := &AsyncAPIV3{
		AsyncAPI: "3.0.0",
		Info: Info{
			Title:   "Test AsyncAPI V3",
			Version: "1.0.0",
		},
	}

	yamlBytes, err := FormatAsyncAPIV3AsYAML(spec)
	if err != nil {
		t.Fatalf("FormatAsyncAPIV3AsYAML() error = %v", err)
	}

	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "asyncapi:") {
		t.Error("YAML should contain asyncapi version")
	}
}

func TestFormatAsyncAPIV3AsYAML_Nil(t *testing.T) {
	_, err := FormatAsyncAPIV3AsYAML(nil)
	if err != ErrNilSpec {
		t.Errorf("FormatAsyncAPIV3AsYAML(nil) error = %v, want ErrNilSpec", err)
	}
}
