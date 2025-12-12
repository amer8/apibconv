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
		if !isValidExtension(ext) {
			return nil
		}

		// Run test for this file
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			processExampleFile(t, path, content, ext)
		})
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
}

func isValidExtension(ext string) bool {
	return ext == ".json" || ext == ".apib" || ext == ".yaml" || ext == ".yml"
}

func processExampleFile(t *testing.T, path string, content []byte, ext string) {
	switch ext {
	case ".json", ".yaml", ".yml":
		// Determine type based on content
		var typeCheck struct {
			OpenAPI  string `json:"openapi"`
			AsyncAPI string `json:"asyncapi"`
		}

		var unmarshalErr error
		if ext == ".json" {
			unmarshalErr = json.Unmarshal(content, &typeCheck)
		} else {
			unmarshalErr = UnmarshalYAML(content, &typeCheck)
		}

		if unmarshalErr != nil {
			t.Logf("Skipping %s: invalid format or not a spec file: %v", path, unmarshalErr)
			return
		}

		switch {
		case typeCheck.OpenAPI != "":
			testOpenAPIConversion(t, content)
		case typeCheck.AsyncAPI != "":
			testAsyncAPIConversion(t, content, typeCheck.AsyncAPI)
		default:
			t.Logf("Skipping %s: unknown spec format (not OpenAPI or AsyncAPI)", path)
		}
	case ".apib":
		testAPIBConversion(t, content)
	default:
		t.Logf("Skipping %s: unknown file type %s", path, ext)
	}
}
func testAPIBConversion(t *testing.T, content []byte) {
	// 1. Parse API Blueprint
	bp, err := ParseBlueprint(content)
	if err != nil {
		t.Fatalf("Failed to parse API Blueprint: %v", err)
	}

	// Convert to OpenAPI
	specObj, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("Failed to convert to OpenAPI: %v", err)
	}

	spec, err := json.Marshal(specObj)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	// 2. Validate the generated OpenAPI JSON
	// It should at least be valid JSON
	var js map[string]any
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
	s, err := Parse(content)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI: %v", err)
	}
	spec, ok := s.AsOpenAPI()
	if !ok {
		t.Fatalf("Expected *OpenAPI")
	}

	// 2. Convert to API Blueprint
	var buf bytes.Buffer
	specToConvert, err := Parse(content, FormatOpenAPI)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI for conversion: %v", err)
	}
	_, err = specToConvert.(*OpenAPI).WriteTo(&buf)
	if err != nil {
		t.Fatalf("Failed to convert OpenAPI to API Blueprint: %v", err)
	}

	apibOutput := buf.String()
	if !strings.Contains(apibOutput, "FORMAT: 1A") {
		t.Error("Output does not contain API Blueprint format header")
	}

	// 3. Round-trip (APIB -> OpenAPI)
	// Note: Round-trip is lossy, so we can't compare exactly, but we can check if it parses back.
	parsedSpec, err := ParseBlueprint([]byte(apibOutput))
	if err != nil {
		t.Fatalf("Failed to parse generated API Blueprint back to OpenAPI: %v", err)
	}

	if parsedSpec.Title() != spec.Title() {
		t.Errorf("Title mismatch: expected %q, got %q", "Test API", spec.Title())
	}
}

func testAsyncAPIConversion(t *testing.T, content []byte, version string) {
	var apibOutput string
	var err error
	var buf bytes.Buffer

	var spec Spec

	if strings.HasPrefix(version, "3.") {
		s, e := parseAsyncV3(content)
		spec = s
		err = e
	} else {
		s, e := parseAsync(content)
		spec = s
		err = e
	}

	if err == nil {
		var blueprintStr string
		if s, ok := spec.(*AsyncAPI); ok {
			bpObj, err := s.ToAPIBlueprint()
			if err != nil {
				t.Fatalf("ToAPIBlueprint failed: %v", err)
			}
			blueprintStr = bpObj.String()
		} else {
			// Should not happen for AsyncAPI parsing
			t.Fatalf("Parsed spec is not *AsyncAPI")
		}

		if err == nil {
			buf.WriteString(blueprintStr)
		}
	}

	if err != nil {
		t.Fatalf("Failed to convert AsyncAPI to API Blueprint: %v", err)
	}

	apibOutput = buf.String()
	if !strings.Contains(apibOutput, "FORMAT: 1A") {
		t.Error("Output does not contain API Blueprint format header")
	}
}


