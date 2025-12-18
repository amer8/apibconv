package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

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
		// OpenAPI v2.0 to others
		{
			name:         "OpenAPI v2.0 (json) to API Blueprint",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_apiblueprint.apib",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAPIBlueprint,
		},
		{
			name:         "OpenAPI v2.0 (json) to AsyncAPI v2.6 (kafka)",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v2.0 (json) to AsyncAPI v2.6 (amqp)",
			inputFile:    "openapi_v2.json",
			expectedFile: "expected/expected_openapi_v2_json_to_asyncapi_v2_amqp.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "amqp",
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
			name:         "OpenAPI v3.0 (json) to AsyncAPI v2.6 (kafka)",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (json) to AsyncAPI v2.6 (http)",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v2_http.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "http",
		},
		{
			name:         "OpenAPI v3.0 (json) to AsyncAPI v3.0 (kafka)",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (json) to AsyncAPI v3.0 (http)",
			inputFile:    "openapi_v3_0.json",
			expectedFile: "expected/expected_openapi_v3_0_json_to_asyncapi_v3_http.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "http",
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
			name:         "OpenAPI v3.0 (yaml) to AsyncAPI v3.0 (kafka)",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "OpenAPI v3.0 (yaml) to AsyncAPI v3.0 (mqtt)",
			inputFile:    "openapi_v3_0.yaml",
			expectedFile: "expected/expected_openapi_v3_0_yaml_to_asyncapi_v3_mqtt.yaml",
			fromFormat:   format.FormatOpenAPI,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "mqtt",
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
		// AsyncAPI v2.x
		{
			name:         "AsyncAPI v2.x (yaml) to API Blueprint",
			inputFile:    "asyncapi_v2.yaml",
			expectedFile: "expected/expected_asyncapi_v2_to_apiblueprint.apib",
			fromFormat:   format.FormatAsyncAPI,
			toFormat:     format.FormatAPIBlueprint,
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
		// API Blueprint
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
			name:         "API Blueprint (apib) to AsyncAPI v2.6 (kafka)",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v2.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "kafka",
		},
		{
			name:         "API Blueprint (apib) to AsyncAPI v2.6 (ws)",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v2_ws.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "2.6",
			toProtocol: "ws",
		},
		{
			name:         "API Blueprint (apib) to AsyncAPI v3.0 (kafka)",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v3.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "kafka",
		},
		{
			name:         "API Blueprint (apib) to AsyncAPI v3.0 (wss)",
			inputFile:    "apiblueprint.apib",
			expectedFile: "expected/expected_apiblueprint_to_asyncapi_v3_wss.yaml",
			fromFormat:   format.FormatAPIBlueprint,
			toFormat:     format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol: "wss",
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

			compareContent(t, tt.expectedFile, expectedBytes, output.Bytes())
		})
	}
}

func compareContent(t *testing.T, filename string, expected, actual []byte) {
	ext := filepath.Ext(filename)
	switch ext {
	case ".json":
		var exp, act interface{}
		if err := json.Unmarshal(expected, &exp); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON %s: %v", filename, err)
		}
		if err := json.Unmarshal(actual, &act); err != nil {
			t.Errorf("Failed to unmarshal actual JSON output for %s: %v", filename, err)
			compareText(t, filename, expected, actual)
			return
		}
		// Normalize
		exp = normalize(exp)
		act = normalize(act)
		
		if !reflect.DeepEqual(exp, act) {
			t.Errorf("JSON output does not match expected file %s", filename)
			compareText(t, filename, expected, actual)
		}
	case ".yaml", ".yml":
		compareYAML(t, filename, expected, actual)
	default:
		compareText(t, filename, expected, actual)
	}
}

func compareYAML(t *testing.T, filename string, expected, actual []byte) {
	var expNode, actNode yaml.Node
	if err := yaml.Unmarshal(expected, &expNode); err != nil {
		t.Fatalf("Failed to unmarshal expected YAML %s: %v", filename, err)
	}
	if err := yaml.Unmarshal(actual, &actNode); err != nil {
		t.Errorf("Failed to unmarshal actual YAML output for %s: %v", filename, err)
		compareText(t, filename, expected, actual)
		return
	}

	if diff := diffNodes(&expNode, &actNode); diff != "" {
		t.Errorf("YAML output does not match expected file %s: %s", filename, diff)
		compareText(t, filename, expected, actual)
	}
}

func diffNodes(exp, act *yaml.Node) string {
	if exp.Kind != act.Kind {
		return fmt.Sprintf("Kind mismatch: %v vs %v", exp.Kind, act.Kind)
	}

	if exp.Kind == yaml.ScalarNode {
		if exp.Value != act.Value {
			return fmt.Sprintf("Value mismatch: %q vs %q", exp.Value, act.Value)
		}
	}

	if exp.Kind == yaml.SequenceNode {
		if len(exp.Content) != len(act.Content) {
			return fmt.Sprintf("Sequence length mismatch: %d vs %d", len(exp.Content), len(act.Content))
		}
		for i := range exp.Content {
			if d := diffNodes(exp.Content[i], act.Content[i]); d != "" {
				return fmt.Sprintf("Index %d: %s", i, d)
			}
		}
	}

	if exp.Kind == yaml.MappingNode {
		// Content is [key, val, key, val...]
		// Sort keys to compare
		expMap := make(map[string]*yaml.Node)
		actMap := make(map[string]*yaml.Node)

		for i := 0; i < len(exp.Content); i += 2 {
			expMap[exp.Content[i].Value] = exp.Content[i+1]
		}
		for i := 0; i < len(act.Content); i += 2 {
			actMap[act.Content[i].Value] = act.Content[i+1]
		}

		if len(expMap) != len(actMap) {
			// Find missing keys
			var missing []string
			for k := range expMap {
				if _, ok := actMap[k]; !ok {
					missing = append(missing, "-"+k)
				}
			}
			for k := range actMap {
				if _, ok := expMap[k]; !ok {
					missing = append(missing, "+"+k)
				}
			}
			sort.Strings(missing)
			return fmt.Sprintf("Map size mismatch: %d vs %d. Diff keys: %v", len(expMap), len(actMap), missing)
		}

		for k, vExp := range expMap {
			vAct, ok := actMap[k]
			if !ok {
				return fmt.Sprintf("Missing key: %s", k)
			}
			if d := diffNodes(vExp, vAct); d != "" {
				return fmt.Sprintf("Key %s: %s", k, d)
			}
		}
	}
	
	if exp.Kind == yaml.DocumentNode {
		if len(exp.Content) > 0 && len(act.Content) > 0 {
			return diffNodes(exp.Content[0], act.Content[0])
		}
	}

	return ""
}

func normalize(i interface{}) interface{} {
	b, err := json.Marshal(i)
	if err != nil {
		return i 
	}
	var res interface{}
	if err := json.Unmarshal(b, &res); err != nil {
		return i
	}
	return res
}

func compareText(t *testing.T, filename string, expected, actual []byte) {
	// Normalize line endings
	expStr := strings.ReplaceAll(string(expected), "\r\n", "\n")
	actStr := strings.ReplaceAll(string(actual), "\r\n", "\n")

	// Trim spaces
	expStr = strings.TrimSpace(expStr)
	actStr = strings.TrimSpace(actStr)

	if expStr != actStr {
		// Find first difference
		minLen := len(expStr)
		if len(actStr) < minLen {
			minLen = len(actStr)
		}
		diffIdx := 0
		for i := 0; i < minLen; i++ {
			if expStr[i] != actStr[i] {
				diffIdx = i
				break
			}
		}
		
		// Context around diff
		start := diffIdx - 50
		if start < 0 { start = 0 }
		end := diffIdx + 50
		if end > len(expStr) { end = len(expStr) }
		expCtx := expStr[start:end]
		
		endAct := diffIdx + 50
		if endAct > len(actStr) { endAct = len(actStr) }
		actCtx := actStr[start:endAct]

		t.Logf("First difference at index %d:\nExpected: ...%q...\nGot:      ...%q...", diffIdx, expCtx, actCtx)
	}
}
