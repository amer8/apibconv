package converter

import (
	"bytes"
	"testing"
)

func TestUnmarshalYAML_SimpleMap(t *testing.T) {
	yaml := []byte(`
title: Test API
version: 1.0.0
description: >
  This is a
  multiline description
`)

	var result map[string]any
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("UnmarshalYAML error: %v", err)
	}

	if result["title"] != "Test API" {
		t.Errorf("Expected title 'Test API', got '%v'", result["title"])
	}
	if result["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", result["version"])
	}

	desc := result["description"].(string)
	if len(desc) < 10 {
		t.Errorf("Description parsing failed, got '%v'", desc)
	}
}

func TestUnmarshalYAML_Nested(t *testing.T) {
	yaml := []byte(`
info:
  title: Nested
  contact:
    name: API Support
servers:
  - url: https://api.example.com
    description: Production
  - url: https://test.example.com
`)

	// Parse into generic map first to check structure
	var result map[string]any
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("UnmarshalYAML error: %v", err)
	}

	info := result["info"].(map[string]any)
	if info["title"] != "Nested" {
		t.Errorf("Nested title mismatch")
	}

	servers := result["servers"].([]any)
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	server1 := servers[0].(map[string]any)
	if server1["url"] != "https://api.example.com" {
		t.Errorf("Server 1 URL mismatch")
	}
}

func TestParse_YAML_OpenAPI(t *testing.T) {
	yaml := []byte(`
openapi: 3.0.0
info:
  title: YAML API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get Users
      responses:
        "200":
          description: OK
`)

	s, err := Parse(yaml)
	if err != nil {
		t.Fatalf("Parse(YAML) error: %v", err)
	}
	spec, ok := s.(*OpenAPI)
	if !ok {
		t.Fatalf("Expected *OpenAPI")
	}

	if spec.OpenAPI != "3.0.0" {
		t.Errorf("Expected OpenAPI 3.0.0, got %s", spec.OpenAPI)
	}
	if spec.Info.Title != "YAML API" {
		t.Errorf("Expected title 'YAML API', got '%s'", spec.Info.Title)
	}

	pathItem, ok := spec.Paths["/users"]
	if !ok {
		t.Fatal("Path /users not found")
	}
	if pathItem.Get == nil {
		t.Fatal("GET operation not found")
	}
	if pathItem.Get.Summary != "Get Users" {
		t.Errorf("Summary mismatch")
	}
}

func TestParseAsyncAPI_YAML(t *testing.T) {
	yaml := []byte(`
asyncapi: 2.6.0
info:
  title: Async YAML
  version: 1.0.0
channels:
  user/signup:
    subscribe:
      message:
        name: UserSignup
`)

	spec, err := ParseAsyncAPI(yaml)
	if err != nil {
		t.Fatalf("ParseAsyncAPI(YAML) error: %v", err)
	}

	if spec.AsyncAPI != "2.6.0" {
		t.Errorf("Expected AsyncAPI 2.6.0, got %s", spec.AsyncAPI)
	}
	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}
}

func TestParseAsyncAPIAny_YAML_V3(t *testing.T) {
	yaml := []byte(`
asyncapi: 3.0.0
info:
  title: Async V3 YAML
  version: 1.0.0
channels:
  userSignup:
    address: user/signup
operations:
  onSignup:
    action: receive
    channel:
      $ref: "#/channels/userSignup"
`)

	spec, version, err := ParseAsyncAPIAny(yaml)
	if err != nil {
		t.Fatalf("ParseAsyncAPIAny(YAML) error: %v", err)
	}

	if version != 3 {
		t.Errorf("Expected version 3, got %d", version)
	}

	v3Spec, ok := spec.(*AsyncAPIV3)
	if !ok {
		t.Fatalf("Expected *AsyncAPIV3 type, got %T", spec)
	}

	if v3Spec.Info.Title != "Async V3 YAML" {
		t.Errorf("Title mismatch")
	}
}

func TestUnmarshalYAMLReader(t *testing.T) {
	yaml := []byte(`name: Test`)
	reader := bytes.NewReader(yaml)

	var result map[string]any
	if err := UnmarshalYAMLReader(reader, &result); err != nil {
		t.Fatalf("UnmarshalYAMLReader error: %v", err)
	}

	if result["name"] != "Test" {
		t.Errorf("Expected 'Test', got %v", result["name"])
	}
}

func TestParseYAML_ComplexStructures(t *testing.T) {
	yaml := []byte(`
array_root:
  - item1
  - item2: value2
    item3: value3
  - 
    nested: value
compact_map:
  - key: value
    key2: value2
scalars:
  quoted: "value with \"quotes\""
  single_quoted: 'value with ''quotes'''
  boolean: true
  number: 123
  float: 12.34
  null_val: null
  plain: just a string
blocks:
  literal: |
    Line 1
    Line 2
  folded: >
    Line 1
    Line 2
`)

	var result map[string]any
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("Complex YAML parse failed: %v", err)
	}

	// Verify array
	arr := result["array_root"].([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 items in array, got %d", len(arr))
	}
	if arr[0] != "item1" {
		t.Errorf("Item 1 mismatch")
	}
	item2 := arr[1].(map[string]any)
	if item2["item2"] != "value2" {
		t.Errorf("Item 2 mismatch")
	}

	// Verify compact map
	compact := result["compact_map"].([]any)
	cMap := compact[0].(map[string]any)
	if cMap["key"] != "value" || cMap["key2"] != "value2" {
		t.Errorf("Compact map mismatch: %v", cMap)
	}

	// Verify scalars
	sc := result["scalars"].(map[string]any)
	if sc["quoted"] != `value with "quotes"` {
		t.Errorf("Quoted string mismatch: %s", sc["quoted"])
	}
	if sc["boolean"] != true {
		t.Errorf("Boolean mismatch")
	}
	if sc["number"] != 123.0 { // JSON numbers are float64
		t.Errorf("Number mismatch")
	}
	if sc["null_val"] != nil {
		t.Errorf("Null mismatch")
	}

	// Verify blocks
	bl := result["blocks"].(map[string]any)
	if bl["literal"] != "Line 1\nLine 2" {
		t.Errorf("Literal block mismatch: %q", bl["literal"])
	}
	if bl["folded"] != "Line 1 Line 2" {
		t.Errorf("Folded block mismatch: %q", bl["folded"])
	}
}

func TestParseYAML_RootArray(t *testing.T) {
	yaml := []byte(`
- item1
- item2
`)
	var result []any
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("Root array parse failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

func TestParseYAML_InlineJSON(t *testing.T) {
	yaml := []byte(`
key: {"nested": "value", "arr": [1, 2]}
`)
	var result map[string]any
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("Inline JSON parse failed: %v", err)
	}

	val := result["key"].(map[string]any)
	if val["nested"] != "value" {
		t.Errorf("Nested value mismatch")
	}
	arr := val["arr"].([]any)
	if len(arr) != 2 {
		t.Errorf("Array length mismatch")
	}
}

func TestParseYAML_EdgeCases(t *testing.T) {
	// Empty
	var res map[string]any
	if err := UnmarshalYAML([]byte(""), &res); err != nil {
		t.Fatal(err)
	}

	// Comments only
	if err := UnmarshalYAML([]byte("# Comment only"), &res); err != nil {
		t.Fatal(err)
	}

	// Quoted keys
	yaml := []byte(`
"quoted key": value
'single quoted': value
`)
	if err := UnmarshalYAML(yaml, &res); err != nil {
		t.Fatal(err)
	}
	if res["quoted key"] != "value" {
		t.Error("Quoted key failed")
	}
}
