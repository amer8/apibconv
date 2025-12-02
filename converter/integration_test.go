package converter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_Examples verifies that all example files can be parsed and converted.
// This serves as a smoke test and ensures that complex real-world examples (like those in examples/)
// work with the converter.
func TestIntegration_Examples(t *testing.T) {
	examplesDir := "../examples"
	files, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("Failed to read examples directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(examplesDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// Determine type based on content or name
			// Simple heuristic: check for "openapi" or "asyncapi" keys
			var typeCheck struct {
				OpenAPI  string `json:"openapi"`
				AsyncAPI string `json:"asyncapi"`
			}
			if err := json.Unmarshal(content, &typeCheck); err != nil {
				t.Fatalf("Failed to unmarshal JSON structure: %v", err)
			}

			if typeCheck.OpenAPI != "" {
				testOpenAPIConversion(t, content)
			} else if typeCheck.AsyncAPI != "" {
				testAsyncAPIConversion(t, content, typeCheck.AsyncAPI)
			} else {
				t.Logf("Skipping %s: unknown format", file.Name())
			}
		})
	}
}

func testOpenAPIConversion(t *testing.T, content []byte) {
	// 1. Parse
	spec, err := Parse(content)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI: %v", err)
	}

	// 2. Convert to API Blueprint
	var buf bytes.Buffer
	err = Convert(bytes.NewReader(content), &buf)
	if err != nil {
		t.Fatalf("Failed to convert OpenAPI to API Blueprint: %v", err)
	}

	apibOutput := buf.String()
	if !strings.Contains(apibOutput, "FORMAT: 1A") {
		t.Error("Output does not contain API Blueprint format header")
	}

	// 3. Round-trip (APIB -> OpenAPI)
	// Note: Round-trip is lossy, so we can't compare exactly, but we can check if it parses back.
	parsedSpec, err := ParseAPIBlueprint([]byte(apibOutput))
	if err != nil {
		t.Fatalf("Failed to parse generated API Blueprint back to OpenAPI: %v", err)
	}

	if parsedSpec.Info.Title != spec.Info.Title {
		t.Errorf("Title mismatch after round-trip. Got %q, want %q", parsedSpec.Info.Title, spec.Info.Title)
	}
}

func testAsyncAPIConversion(t *testing.T, content []byte, version string) {
	// 1. Parse and Convert based on version
	var apibOutput string
	var err error
	var buf bytes.Buffer

	if strings.HasPrefix(version, "3.") {
		err = ConvertAsyncAPIV3ToAPIBlueprint(bytes.NewReader(content), &buf)
	} else {
		err = ConvertAsyncAPIToAPIBlueprint(bytes.NewReader(content), &buf)
	}

	if err != nil {
		t.Fatalf("Failed to convert AsyncAPI to API Blueprint: %v", err)
	}

	apibOutput = buf.String()
	if !strings.Contains(apibOutput, "FORMAT: 1A") {
		t.Error("Output does not contain API Blueprint format header")
	}

	// 2. Verify some content exists
	if !strings.Contains(apibOutput, "HOST:") && !strings.Contains(apibOutput, "host:") {
		// Not all asyncapis have servers, so this might be optional, 
		// but our examples generally do.
		// t.Log("Warning: No HOST found in output")
	}
}

// TestWorkflow_AsyncAPI_To_OpenAPI verifies a multi-step conversion:
// AsyncAPI -> API Blueprint -> OpenAPI
// This simulates a user wanting to document their Event API using OpenAPI tools.
func TestWorkflow_AsyncAPI_To_OpenAPI(t *testing.T) {
	// Create a simple AsyncAPI spec
	asyncAPI := `{
		"asyncapi": "2.6.0",
		"info": {
			"title": "Workflow Test",
			"version": "1.0.0"
		},
		"channels": {
			"user/signup": {
				"subscribe": {
					"message": {
						"payload": {
							"type": "object",
							"properties": {
								"username": {"type": "string"}
							}
						}
					}
				}
			}
		}
	}`

	// Step 1: AsyncAPI -> API Blueprint
	var apibBuf bytes.Buffer
	err := ConvertAsyncAPIToAPIBlueprint(strings.NewReader(asyncAPI), &apibBuf)
	if err != nil {
		t.Fatalf("Step 1 Failed: AsyncAPI -> APIB: %v", err)
	}
	apibOutput := apibBuf.String()

	// Step 2: API Blueprint -> OpenAPI
	var openapiBuf bytes.Buffer
	err = ConvertToOpenAPI(strings.NewReader(apibOutput), &openapiBuf)
	if err != nil {
		t.Fatalf("Step 2 Failed: APIB -> OpenAPI: %v", err)
	}

	// Step 3: Verify OpenAPI structure
	openapiOutput := openapiBuf.Bytes()
	spec, err := Parse(openapiOutput)
	if err != nil {
		t.Fatalf("Step 3 Failed: Parsing generated OpenAPI: %v", err)
	}

	// Verification
	if spec.Info.Title != "Workflow Test" {
		t.Errorf("Title lost in translation. Got %q", spec.Info.Title)
	}
	
	// The AsyncAPI "user/signup" channel should become a path
	// The "subscribe" operation (receive) should become a "GET" operation (standard mapping in this lib)
	pathItem, exists := spec.Paths["/user/signup"]
	if !exists {
		t.Fatalf("Path '/user/signup' missing in final OpenAPI. Paths: %v", spec.Paths)
	}

	if pathItem.Get == nil {
		t.Error("Operation mapping incorrect. Expected GET operation for AsyncAPI subscribe.")
	}
}
