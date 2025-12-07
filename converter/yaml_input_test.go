package converter

import (
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

	var result map[string]interface{}
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
	var result map[string]interface{}
	if err := UnmarshalYAML(yaml, &result); err != nil {
		t.Fatalf("UnmarshalYAML error: %v", err)
	}

	info := result["info"].(map[string]interface{})
	if info["title"] != "Nested" {
		t.Errorf("Nested title mismatch")
	}

	servers := result["servers"].([]interface{})
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}
	
	server1 := servers[0].(map[string]interface{})
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

	spec, err := Parse(yaml)
	if err != nil {
		t.Fatalf("Parse(YAML) error: %v", err)
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
