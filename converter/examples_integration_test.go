package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamples_Parse(t *testing.T) {
	// Find the examples directory relative to this test file
	// We assume this test runs from the converter directory
	examplesDir := "../examples"

	// Walk through the examples directory
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read example file: %v", err)
			}

			// Try to parse as AsyncAPI first if filename contains "asyncapi"
			if strings.Contains(strings.ToLower(filepath.Base(path)), "asyncapi") {
				_, _, err := ParseAsyncAPIAny(data)
				if err != nil {
					t.Errorf("Failed to parse AsyncAPI example %s: %v", filepath.Base(path), err)
				}
			} else {
				// Assume OpenAPI/Swagger
				_, err := Parse(data)
				if err != nil {
					t.Errorf("Failed to parse OpenAPI example %s: %v", filepath.Base(path), err)
				}
			}
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk examples directory: %v", err)
	}
}
