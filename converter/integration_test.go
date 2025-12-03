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
	examplesRoot := "../examples"
	if _, err := os.Stat(examplesRoot); os.IsNotExist(err) {
		t.Logf("Examples directory not found at %s, skipping integration tests", examplesRoot)
		return
	}

	err := filepath.WalkDir(examplesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Skip hidden files or non-spec files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".json" && ext != ".apib" {
			return nil
		}

		// Run test for this file
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			switch ext {
			case ".json":
				// Determine type based on content
				var typeCheck struct {
					OpenAPI  string `json:"openapi"`
					AsyncAPI string `json:"asyncapi"`
				}
				if err := json.Unmarshal(content, &typeCheck); err != nil {
					t.Logf("Skipping %s: invalid JSON or not a spec file: %v", path, err)
					return
				}

				switch {
				case typeCheck.OpenAPI != "":
					testOpenAPIConversion(t, content)
				case typeCheck.AsyncAPI != "":
					testAsyncAPIConversion(t, content, typeCheck.AsyncAPI)
				default:
					t.Logf("Skipping %s: unknown JSON format (not OpenAPI or AsyncAPI)", path)
				}
			case ".apib":
				testAPIBConversion(t, content)
			default:
				t.Logf("Skipping %s: unknown file type %s", path, ext)
			}
		})
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
}

func testAPIBConversion(t *testing.T, content []byte) {
	// 1. Parse API Blueprint to OpenAPI
	// This implicitly validates the APIB parsing logic
	spec, err := ToOpenAPI(content)
	if err != nil {
		t.Fatalf("Failed to convert API Blueprint to OpenAPI: %v", err)
	}

	// 2. Validate the generated OpenAPI JSON
	// It should at least be valid JSON
	var js map[string]interface{}
	if err := json.Unmarshal(spec, &js); err != nil {
		t.Fatalf("Generated OpenAPI is not valid JSON: %v", err)
	}

	// Check for basics
	if _, ok := js["openapi"]; !ok {
		t.Error("Generated OpenAPI missing 'openapi' version field")
	}
	if _, ok := js["info"]; !ok {
		t.Error("Generated OpenAPI missing 'info' field")
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
