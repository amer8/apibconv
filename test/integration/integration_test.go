package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/format/openapi"
)

func TestConversions(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
		fromFormat   format.Format
		toFormat          format.Format
		toAsyncAPIVersion string
		toOpenAPIVersion  string
		toProtocol        string
	}{
		// OpenAPI v3 to others
		// OpenAPI v2.0
		{
			name:         "OpenAPI v2.0 (json) to API Blueprint",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v2.0 (json) to AsyncAPI v2.6",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v2.0 (json) to AsyncAPI v3.0",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v2.0 (json) to OpenAPI v3.0",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "OpenAPI v2.0 (json) to OpenAPI v3.1",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "OpenAPI v2.0 (yaml) to API Blueprint",
			inputFile:    "openapi_v2.yaml",
			expectedFile: "expected/expected_openapi_v2_yaml_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v2.0 (yaml) to AsyncAPI v2.6",
			inputFile:    "openapi_v2.yaml",
			expectedFile: "expected/expected_openapi_v2_yaml_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v2.0 (yaml) to AsyncAPI v3.0",
			inputFile:    "openapi_v2.yaml",
			expectedFile: "expected/expected_openapi_v2_yaml_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v2.0 (yaml) to OpenAPI v3.0",
			inputFile:    "openapi_v2.yaml",
			expectedFile: "expected/expected_openapi_v2_yaml_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "OpenAPI v2.0 (yaml) to OpenAPI v3.1",
			inputFile:    "openapi_v2.yaml",
			expectedFile: "expected/expected_openapi_v2_yaml_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		// OpenAPI v3.0
		{
			name:         "OpenAPI v3.0 (json) to API Blueprint",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v3.0 (json) to AsyncAPI v2.6",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (json) to AsyncAPI v3.0",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (json) to OpenAPI v2.0",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_openapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "OpenAPI v3.0 (json) to OpenAPI v3.1",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "OpenAPI v3.0 (yaml) to API Blueprint",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v3.0 (yaml) to AsyncAPI v2.6",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (yaml) to AsyncAPI v3.0",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (yaml) to OpenAPI v2.0",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_openapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "OpenAPI v3.0 (yaml) to OpenAPI v3.1",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		// OpenAPI v3.1
		{
			name:         "OpenAPI v3.1 (json) to API Blueprint",
			inputFile:    "openapi_v3_1.json",
			expectedFile: "expected/expected_openapi_v3_1_json_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v3.1 (json) to AsyncAPI v2.6",
			inputFile:    "openapi_v3_1.json",
			expectedFile: "expected/expected_openapi_v3_1_json_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.1 (json) to AsyncAPI v3.0",
			inputFile:    "openapi_v3_1.json",
			expectedFile: "expected/expected_openapi_v3_1_json_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.1 (json) to OpenAPI v2.0",
			inputFile:    "openapi_v3_1.json",
			expectedFile: "expected/expected_openapi_v3_1_json_to_openapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "OpenAPI v3.1 (json) to OpenAPI v3.0",
			inputFile:    "openapi_v3_1.json",
			expectedFile: "expected/expected_openapi_v3_1_json_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "OpenAPI v3.1 (yaml) to API Blueprint",
			inputFile:    "openapi_v3_1.yaml",
			expectedFile: "expected/expected_openapi_v3_1_yaml_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v3.1 (yaml) to AsyncAPI v2.6",
			inputFile:    "openapi_v3_1.yaml",
			expectedFile: "expected/expected_openapi_v3_1_yaml_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.1 (yaml) to AsyncAPI v3.0",
			inputFile:    "openapi_v3_1.yaml",
			expectedFile: "expected/expected_openapi_v3_1_yaml_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.1 (yaml) to OpenAPI v2.0",
			inputFile:    "openapi_v3_1.yaml",
			expectedFile: "expected/expected_openapi_v3_1_yaml_to_openapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "OpenAPI v3.1 (yaml) to OpenAPI v3.0",
			inputFile:    "openapi_v3_1.yaml",
			expectedFile: "expected/expected_openapi_v3_1_yaml_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		// AsyncAPI v2.x
		{
			name:         "AsyncAPI v2.x (yaml) to API Blueprint",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_apiblueprint.apib",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "AsyncAPI v2.x (yaml) to OpenAPI v2.0",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v2.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "AsyncAPI v2.x (yaml) to OpenAPI v3.0",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "AsyncAPI v2.x (yaml) to OpenAPI v3.1",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "AsyncAPI v2.x (yaml) to AsyncAPI v3.0",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "AsyncAPI v2.x (json) to API Blueprint",
			inputFile:    "asyncapi_v2.json",
			expectedFile: "expected/expected_asyncapi_v2_to_apiblueprint.apib",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "AsyncAPI v2.x (json) to OpenAPI v2.0",
			inputFile:    "asyncapi_v2.json",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v2.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "AsyncAPI v2.x (json) to OpenAPI v3.0",
			inputFile:    "asyncapi_v2.json",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "AsyncAPI v2.x (json) to OpenAPI v3.1",
			inputFile:    "asyncapi_v2.json",
			expectedFile: "expected/expected_asyncapi_v2_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "AsyncAPI v2.x (json) to AsyncAPI v3.0",
			inputFile:    "asyncapi_v2.json",
			expectedFile: "expected/expected_asyncapi_v2_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		// AsyncAPI v3.0
		{
			name:         "AsyncAPI v3.0 (yaml) to API Blueprint",
			inputFile:    "asyncapi_v3.yaml",
			expectedFile: "expected/expected_asyncapi_v3_to_apiblueprint.apib",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "AsyncAPI v3.0 (yaml) to OpenAPI v2.0",
			inputFile:    "asyncapi_v3.yaml",
			expectedFile: "expected/expected_asyncapi_v3_to_openapi_v2.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "AsyncAPI v3.0 (yaml) to OpenAPI v3.0",
			inputFile:    "asyncapi_v3.yaml",
			expectedFile: "expected/expected_asyncapi_v3_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "AsyncAPI v3.0 (yaml) to OpenAPI v3.1",
			inputFile:    "asyncapi_v3.yaml",
			expectedFile: "expected/expected_asyncapi_v3_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "AsyncAPI v3.0 (yaml) to AsyncAPI v2.6",
			inputFile:    "asyncapi_v3.yaml",
			expectedFile: "expected/expected_asyncapi_v3_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		// API Blueprint
		{
			name:         "API Blueprint (apib) to OpenAPI v2.0",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_openapi_v2.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		},
		{
			name:         "API Blueprint (apib) to OpenAPI v3.0",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_openapi_v3_0.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		},
		{
			name:         "API Blueprint (apib) to OpenAPI v3.1",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_openapi_v3_1.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		},
		{
			name:         "API Blueprint (apib) to AsyncAPI v2.6",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "API Blueprint (apib) to AsyncAPI v3.0",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup converter
			conv, err := converter.New()
			if err != nil {
				t.Fatalf("Failed to create converter: %v", err)
			}
			conv.RegisterParser(openapi.NewParser())
			conv.RegisterParser(apiblueprint.NewParser())
			conv.RegisterParser(asyncapi.NewParser())
			conv.RegisterWriter(openapi.NewWriter())
			conv.RegisterWriter(apiblueprint.NewWriter())
			conv.RegisterWriter(asyncapi.NewWriter())

			// Read input file
			inputPath := filepath.Join("testdata", tt.inputFile)
			f, err := os.Open(inputPath)
			if err != nil {
				t.Fatalf("Failed to open input file: %v", err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					t.Errorf("Failed to close input file: %v", err)
				}
			}()

			// Create context with default values matching CLI
			ctx := context.Background()
			ctx = converter.WithOpenAPIVersion(ctx, "3.0")
			if tt.toOpenAPIVersion != "" {
				ctx = converter.WithOpenAPIVersion(ctx, tt.toOpenAPIVersion)
			}
			if tt.toAsyncAPIVersion != "" {
				ctx = converter.WithAsyncAPIVersion(ctx, tt.toAsyncAPIVersion)
			}
			if tt.toProtocol != "" {
				ctx = converter.WithProtocol(ctx, tt.toProtocol)
			}
			
			// Infer encoding from expected file extension
			ext := filepath.Ext(tt.expectedFile)
			switch ext {
			case ".yaml", ".yml":
				ctx = converter.WithEncoding(ctx, "yaml")
			case ".json":
				ctx = converter.WithEncoding(ctx, "json")
			}

			// Perform conversion
			var output bytes.Buffer
			err = conv.Convert(ctx, f, &output, tt.fromFormat, tt.toFormat)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			if output.Len() == 0 {
				t.Error("Output is empty")
			}

			// Verify output against expected file
			expectedPath := filepath.Join("testdata", tt.expectedFile)
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected file: %v", err)
			}

			if !bytes.Equal(output.Bytes(), expectedBytes) {
				t.Errorf("Output does not match expected file %s", tt.expectedFile)
				// Print first 500 chars of diff
				expectedStr := string(expectedBytes)
				actualStr := output.String()
				if len(expectedStr) > 500 { expectedStr = expectedStr[:500] + "..." }
				if len(actualStr) > 500 { actualStr = actualStr[:500] + "..." }
				t.Logf("Expected (start):\n%s\nGot (start):\n%s", expectedStr, actualStr)
			}
		})
	}
}
