package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
)

type conversionCase struct {
	name              string
	inputFile         string
	fromFormat        format.Format
	toFormat          format.Format
	toAsyncAPIVersion string
	toOpenAPIVersion  string
	toProtocol        string
	encoding          string
}

func TestConversions(t *testing.T) {
	for _, tc := range localConversionCases() {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join("testdata", tc.inputFile)
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read input file %s: %v", tc.inputFile, err)
			}

			assertConverts(t, &tc, input)
		})
	}
}

func TestValidSpecifications(t *testing.T) {
	for _, tc := range validSpecificationCases() {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join("testdata", tc.inputFile)
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read valid fixture %s: %v", tc.inputFile, err)
			}

			assertConverts(t, &tc, input)
		})
	}
}

func TestInvalidSpecifications(t *testing.T) {
	for _, tc := range invalidSpecificationCases() {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := filepath.Join("testdata", "invalid", tc.inputFile)
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read invalid fixture %s: %v", tc.inputFile, err)
			}

			conv := newTestConverter(t)
			errs, err := conv.Validate(context.Background(), bytes.NewReader(input), tc.fromFormat)
			if err == nil && len(errs) == 0 {
				t.Fatalf("Validate() accepted invalid %s fixture %s", tc.fromFormat, tc.inputFile)
			}
		})
	}
}

func localConversionCases() []conversionCase {
	var cases []conversionCase
	add := func(tc conversionCase) {
		cases = append(cases, expandEncodingVariants(&tc)...)
	}

	openAPISources := []struct {
		label   string
		version string
		files   []string
	}{
		{label: "OpenAPI v2.0", version: "2.0", files: []string{"openapi_v2.json", "openapi_v2.yaml"}},
		{label: "OpenAPI v3.0", version: "3.0", files: []string{"openapi_v3_0.json", "openapi_v3_0.yaml"}},
		{label: "OpenAPI v3.1", version: "3.1", files: []string{"openapi_v3_1.json", "openapi_v3_1.yaml"}},
	}

	for _, src := range openAPISources {
		for _, file := range src.files {
			inputEncoding := filepath.Ext(file)[1:]

			add(conversionCase{
				name:       fmt.Sprintf("%s (%s) to API Blueprint", src.label, inputEncoding),
				inputFile:  file,
				fromFormat: format.FormatOpenAPI,
				toFormat:   format.FormatAPIBlueprint,
			})
			add(conversionCase{
				name:              fmt.Sprintf("%s (%s) to AsyncAPI v2.6 (kafka)", src.label, inputEncoding),
				inputFile:         file,
				fromFormat:        format.FormatOpenAPI,
				toFormat:          format.FormatAsyncAPI,
				toAsyncAPIVersion: "2.6",
				toProtocol:        "kafka",
			})
			add(conversionCase{
				name:              fmt.Sprintf("%s (%s) to AsyncAPI v3.0 (kafka)", src.label, inputEncoding),
				inputFile:         file,
				fromFormat:        format.FormatOpenAPI,
				toFormat:          format.FormatAsyncAPI,
				toAsyncAPIVersion: "3.0",
				toProtocol:        "kafka",
			})

			if src.version != "2.0" {
				add(conversionCase{
					name:             fmt.Sprintf("%s (%s) to OpenAPI v2.0", src.label, inputEncoding),
					inputFile:        file,
					fromFormat:       format.FormatOpenAPI,
					toFormat:         format.FormatOpenAPI,
					toOpenAPIVersion: "2.0",
				})
			}
			switch src.version {
			case "2.0":
				add(conversionCase{
					name:             fmt.Sprintf("%s (%s) to OpenAPI v3.0", src.label, inputEncoding),
					inputFile:        file,
					fromFormat:       format.FormatOpenAPI,
					toFormat:         format.FormatOpenAPI,
					toOpenAPIVersion: "3.0",
				})
				add(conversionCase{
					name:             fmt.Sprintf("%s (%s) to OpenAPI v3.1", src.label, inputEncoding),
					inputFile:        file,
					fromFormat:       format.FormatOpenAPI,
					toFormat:         format.FormatOpenAPI,
					toOpenAPIVersion: "3.1",
				})
			case "3.0":
				add(conversionCase{
					name:             fmt.Sprintf("%s (%s) to OpenAPI v3.1", src.label, inputEncoding),
					inputFile:        file,
					fromFormat:       format.FormatOpenAPI,
					toFormat:         format.FormatOpenAPI,
					toOpenAPIVersion: "3.1",
				})
			}
		}
	}

	add(conversionCase{
		name:              "OpenAPI v2.0 (json) to AsyncAPI v2.6 (amqp)",
		inputFile:         "openapi_v2.json",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "amqp",
	})
	add(conversionCase{
		name:              "OpenAPI v3.0 (json) to AsyncAPI v2.6 (http)",
		inputFile:         "openapi_v3_0.json",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "http",
	})
	add(conversionCase{
		name:              "OpenAPI v3.0 (json) to AsyncAPI v3.0 (http)",
		inputFile:         "openapi_v3_0.json",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "http",
	})
	add(conversionCase{
		name:              "OpenAPI v3.0 (yaml) to AsyncAPI v3.0 (mqtt)",
		inputFile:         "openapi_v3_0.yaml",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "mqtt",
	})

	for _, file := range []string{"asyncapi_v2.yaml", "asyncapi_v2.json"} {
		inputEncoding := filepath.Ext(file)[1:]

		add(conversionCase{
			name:       fmt.Sprintf("AsyncAPI v2.x (%s) to API Blueprint", inputEncoding),
			inputFile:  file,
			fromFormat: format.FormatAsyncAPI,
			toFormat:   format.FormatAPIBlueprint,
		})
		add(conversionCase{
			name:             fmt.Sprintf("AsyncAPI v2.x (%s) to OpenAPI v3.0", inputEncoding),
			inputFile:        file,
			fromFormat:       format.FormatAsyncAPI,
			toFormat:         format.FormatOpenAPI,
			toOpenAPIVersion: "3.0",
		})
		add(conversionCase{
			name:             fmt.Sprintf("AsyncAPI v2.x (%s) to OpenAPI v2.0", inputEncoding),
			inputFile:        file,
			fromFormat:       format.FormatAsyncAPI,
			toFormat:         format.FormatOpenAPI,
			toOpenAPIVersion: "2.0",
		})
		add(conversionCase{
			name:             fmt.Sprintf("AsyncAPI v2.x (%s) to OpenAPI v3.1", inputEncoding),
			inputFile:        file,
			fromFormat:       format.FormatAsyncAPI,
			toFormat:         format.FormatOpenAPI,
			toOpenAPIVersion: "3.1",
		})
		add(conversionCase{
			name:              fmt.Sprintf("AsyncAPI v2.x (%s) to AsyncAPI v3.0", inputEncoding),
			inputFile:         file,
			fromFormat:        format.FormatAsyncAPI,
			toFormat:          format.FormatAsyncAPI,
			toAsyncAPIVersion: "3.0",
			toProtocol:        "kafka",
		})
	}

	add(conversionCase{
		name:       "AsyncAPI v3.0 (yaml) to API Blueprint",
		inputFile:  "asyncapi_v3.yaml",
		fromFormat: format.FormatAsyncAPI,
		toFormat:   format.FormatAPIBlueprint,
	})
	add(conversionCase{
		name:             "AsyncAPI v3.0 (yaml) to OpenAPI v3.0",
		inputFile:        "asyncapi_v3.yaml",
		fromFormat:       format.FormatAsyncAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.0",
	})
	add(conversionCase{
		name:             "AsyncAPI v3.0 (yaml) to OpenAPI v2.0",
		inputFile:        "asyncapi_v3.yaml",
		fromFormat:       format.FormatAsyncAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "2.0",
	})
	add(conversionCase{
		name:             "AsyncAPI v3.0 (yaml) to OpenAPI v3.1",
		inputFile:        "asyncapi_v3.yaml",
		fromFormat:       format.FormatAsyncAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.1",
	})

	add(conversionCase{
		name:             "API Blueprint (apib) to OpenAPI v3.0",
		inputFile:        "apiblueprint.apib",
		fromFormat:       format.FormatAPIBlueprint,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.0",
	})
	add(conversionCase{
		name:             "API Blueprint (apib) to OpenAPI v2.0",
		inputFile:        "apiblueprint.apib",
		fromFormat:       format.FormatAPIBlueprint,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "2.0",
	})
	add(conversionCase{
		name:             "API Blueprint (apib) to OpenAPI v3.1",
		inputFile:        "apiblueprint.apib",
		fromFormat:       format.FormatAPIBlueprint,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.1",
	})
	add(conversionCase{
		name:              "API Blueprint (apib) to AsyncAPI v2.6 (kafka)",
		inputFile:         "apiblueprint.apib",
		fromFormat:        format.FormatAPIBlueprint,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "kafka",
	})
	add(conversionCase{
		name:              "API Blueprint (apib) to AsyncAPI v2.6 (ws)",
		inputFile:         "apiblueprint.apib",
		fromFormat:        format.FormatAPIBlueprint,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "ws",
	})
	add(conversionCase{
		name:              "API Blueprint (apib) to AsyncAPI v3.0 (kafka)",
		inputFile:         "apiblueprint.apib",
		fromFormat:        format.FormatAPIBlueprint,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "kafka",
	})
	add(conversionCase{
		name:              "API Blueprint (apib) to AsyncAPI v3.0 (wss)",
		inputFile:         "apiblueprint.apib",
		fromFormat:        format.FormatAPIBlueprint,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "wss",
	})

	return cases
}

type invalidSpecificationCase struct {
	name       string
	inputFile  string
	fromFormat format.Format
}

func invalidSpecificationCases() []invalidSpecificationCase {
	return []invalidSpecificationCase{
		{
			name:       "OpenAPI v2.0 missing info",
			inputFile:  "openapi_v2_missing_info.yaml",
			fromFormat: format.FormatOpenAPI,
		},
		{
			name:       "OpenAPI v3.0 missing info version",
			inputFile:  "openapi_v3_missing_info_version.yaml",
			fromFormat: format.FormatOpenAPI,
		},
		{
			name:       "OpenAPI v2.0 missing paths",
			inputFile:  "openapi_v2_missing_paths.yaml",
			fromFormat: format.FormatOpenAPI,
		},
		{
			name:       "OpenAPI v3.1 missing paths",
			inputFile:  "openapi_v3_1_missing_paths.yaml",
			fromFormat: format.FormatOpenAPI,
		},
		{
			name:       "AsyncAPI v2.6 missing info version",
			inputFile:  "asyncapi_v2_missing_info_version.yaml",
			fromFormat: format.FormatAsyncAPI,
		},
		{
			name:       "AsyncAPI v2.6 missing channels",
			inputFile:  "asyncapi_v2_missing_channels.yaml",
			fromFormat: format.FormatAsyncAPI,
		},
		{
			name:       "AsyncAPI v3.0 missing info version",
			inputFile:  "asyncapi_v3_missing_info_version.yaml",
			fromFormat: format.FormatAsyncAPI,
		},
	}
}

func validSpecificationCases() []conversionCase {
	var cases []conversionCase
	add := func(tc conversionCase) {
		cases = append(cases, expandEncodingVariants(&tc)...)
	}

	add(conversionCase{
		name:       "Valid API Blueprint fixture to OpenAPI v3.0",
		inputFile:  "valid/warehouse.apib",
		fromFormat: format.FormatAPIBlueprint,
		toFormat:   format.FormatOpenAPI,
	})
	add(conversionCase{
		name:              "Valid API Blueprint fixture to AsyncAPI v2.6",
		inputFile:         "valid/warehouse.apib",
		fromFormat:        format.FormatAPIBlueprint,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "kafka",
	})
	add(conversionCase{
		name:             "Valid API Blueprint support fixture to OpenAPI v3.1",
		inputFile:        "valid/support.apib",
		fromFormat:       format.FormatAPIBlueprint,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.1",
	})
	add(conversionCase{
		name:       "Valid OpenAPI fixture to API Blueprint",
		inputFile:  "valid/checkout-openapi.yaml",
		fromFormat: format.FormatOpenAPI,
		toFormat:   format.FormatAPIBlueprint,
	})
	add(conversionCase{
		name:             "Valid OpenAPI fixture to OpenAPI v2.0",
		inputFile:        "valid/checkout-openapi.yaml",
		fromFormat:       format.FormatOpenAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "2.0",
	})
	add(conversionCase{
		name:              "Valid OpenAPI fixture to AsyncAPI v3.0",
		inputFile:         "valid/checkout-openapi.yaml",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "kafka",
	})
	add(conversionCase{
		name:             "Valid OpenAPI v2 fixture to OpenAPI v3.0",
		inputFile:        "valid/catalog-openapi-v2.json",
		fromFormat:       format.FormatOpenAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.0",
	})
	add(conversionCase{
		name:              "Valid OpenAPI v2 fixture to AsyncAPI v2.6",
		inputFile:         "valid/catalog-openapi-v2.json",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "2.6",
		toProtocol:        "amqp",
	})
	add(conversionCase{
		name:       "Valid OpenAPI v3.1 fixture to API Blueprint",
		inputFile:  "valid/subscriptions-openapi-v3_1.yaml",
		fromFormat: format.FormatOpenAPI,
		toFormat:   format.FormatAPIBlueprint,
	})
	add(conversionCase{
		name:              "Valid OpenAPI v3.1 fixture to AsyncAPI v3.0",
		inputFile:         "valid/subscriptions-openapi-v3_1.yaml",
		fromFormat:        format.FormatOpenAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "http",
	})
	add(conversionCase{
		name:       "Valid AsyncAPI fixture to API Blueprint",
		inputFile:  "valid/fulfillment-asyncapi.yaml",
		fromFormat: format.FormatAsyncAPI,
		toFormat:   format.FormatAPIBlueprint,
	})
	add(conversionCase{
		name:             "Valid AsyncAPI fixture to OpenAPI v3.0",
		inputFile:        "valid/fulfillment-asyncapi.yaml",
		fromFormat:       format.FormatAsyncAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.0",
	})
	add(conversionCase{
		name:              "Valid AsyncAPI fixture to AsyncAPI v3.0",
		inputFile:         "valid/fulfillment-asyncapi.yaml",
		fromFormat:        format.FormatAsyncAPI,
		toFormat:          format.FormatAsyncAPI,
		toAsyncAPIVersion: "3.0",
		toProtocol:        "kafka",
	})
	add(conversionCase{
		name:       "Valid AsyncAPI v3 fixture to API Blueprint",
		inputFile:  "valid/notifications-asyncapi-v3.yaml",
		fromFormat: format.FormatAsyncAPI,
		toFormat:   format.FormatAPIBlueprint,
	})
	add(conversionCase{
		name:             "Valid AsyncAPI v3 fixture to OpenAPI v3.1",
		inputFile:        "valid/notifications-asyncapi-v3.yaml",
		fromFormat:       format.FormatAsyncAPI,
		toFormat:         format.FormatOpenAPI,
		toOpenAPIVersion: "3.1",
	})

	return cases
}

func assertConverts(t *testing.T, tc *conversionCase, input []byte) {
	t.Helper()

	conv := newTestConverter(t)
	ctx := testContext(tc)

	sourceModel, err := conv.ParseToModel(ctx, bytes.NewReader(input), tc.fromFormat)
	if err != nil {
		t.Fatalf("failed to parse source %s: %v", tc.fromFormat, err)
	}
	assertMeaningfulModel(t, "source", sourceModel)

	var output bytes.Buffer
	if err := conv.Convert(ctx, bytes.NewReader(input), &output, tc.fromFormat, tc.toFormat); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("conversion produced empty output")
	}

	assertFormatMarker(t, tc, output.Bytes())

	convertedModel, err := conv.ParseToModel(ctx, bytes.NewReader(output.Bytes()), tc.toFormat)
	if err != nil {
		t.Fatalf("failed to parse converted %s output: %v\n%s", tc.toFormat, err, output.String())
	}
	assertMeaningfulModel(t, "converted output", convertedModel)
	assertSemanticShape(t, sourceModel, convertedModel, tc)

	errs, err := conv.Validate(ctx, bytes.NewReader(output.Bytes()), tc.toFormat)
	if err != nil {
		t.Fatalf("failed to validate converted %s output: %v", tc.toFormat, err)
	}
	if len(errs) > 0 {
		t.Fatalf("converted %s output has validation errors: %v", tc.toFormat, errs)
	}
}

func expandEncodingVariants(tc *conversionCase) []conversionCase {
	if tc.encoding != "" || (tc.toFormat != format.FormatOpenAPI && tc.toFormat != format.FormatAsyncAPI) {
		return []conversionCase{*tc}
	}

	yamlCase := *tc
	yamlCase.name = tc.name + " (yaml output)"
	yamlCase.encoding = "yaml"

	jsonCase := *tc
	jsonCase.name = tc.name + " (json output)"
	jsonCase.encoding = "json"

	return []conversionCase{yamlCase, jsonCase}
}

func newTestConverter(t *testing.T) *converter.Converter {
	t.Helper()

	conv, err := converter.New()
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterParser(apiblueprint.NewParser())
	conv.RegisterParser(asyncapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter())
	conv.RegisterWriter(apiblueprint.NewWriter())
	conv.RegisterWriter(asyncapi.NewWriter())

	return conv
}

func testContext(tc *conversionCase) context.Context {
	ctx := context.Background()
	if tc.encoding != "" {
		ctx = converter.WithEncoding(ctx, tc.encoding)
	}
	if tc.toOpenAPIVersion != "" {
		ctx = converter.WithOpenAPIVersion(ctx, tc.toOpenAPIVersion)
	}
	if tc.toAsyncAPIVersion != "" {
		ctx = converter.WithAsyncAPIVersion(ctx, tc.toAsyncAPIVersion)
	}
	if tc.toProtocol != "" {
		ctx = converter.WithProtocol(ctx, tc.toProtocol)
	}
	return ctx
}

func assertFormatMarker(t *testing.T, tc *conversionCase, output []byte) {
	t.Helper()

	switch tc.toFormat {
	case format.FormatAPIBlueprint:
		if !strings.HasPrefix(strings.TrimSpace(string(output)), "FORMAT: 1A") {
			t.Fatalf("API Blueprint output does not start with FORMAT: 1A:\n%s", string(output))
		}
	case format.FormatOpenAPI:
		doc := decodeObject(t, output)
		version := normalizeOpenAPIVersion(tc.toOpenAPIVersion)
		if version == "2.0" {
			assertFieldEquals(t, doc, "swagger", "2.0")
			return
		}
		assertFieldEquals(t, doc, "openapi", version)
	case format.FormatAsyncAPI:
		doc := decodeObject(t, output)
		assertFieldEquals(t, doc, "asyncapi", normalizeAsyncAPIVersion(tc.toAsyncAPIVersion))
	default:
		t.Fatalf("unsupported target format in test: %s", tc.toFormat)
	}
}

func decodeObject(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to decode generated document: %v\n%s", err, string(data))
	}
	if len(doc) == 0 {
		t.Fatalf("generated document decoded to an empty object:\n%s", string(data))
	}
	return doc
}

func assertFieldEquals(t *testing.T, doc map[string]interface{}, field, want string) {
	t.Helper()

	got, ok := doc[field]
	if !ok {
		t.Fatalf("generated document is missing %q", field)
	}
	if fmt.Sprint(got) != want {
		t.Fatalf("generated document %q = %q, want %q", field, got, want)
	}
}

func normalizeOpenAPIVersion(version string) string {
	switch version {
	case "", "3.0":
		return "3.0.0"
	case "3.1":
		return "3.1.0"
	case "2.0", "2.0.0":
		return "2.0"
	default:
		return version
	}
}

func normalizeAsyncAPIVersion(version string) string {
	switch version {
	case "", "2.6":
		return "2.6.0"
	case "3.0":
		return "3.0.0"
	default:
		return version
	}
}

func assertMeaningfulModel(t *testing.T, label string, api *model.API) {
	t.Helper()

	if api == nil {
		t.Fatalf("%s parsed to nil model", label)
	}
	if len(api.Paths) == 0 && len(api.Webhooks) == 0 && len(api.Components.Schemas) == 0 {
		t.Fatalf("%s has no paths, webhooks, or schemas", label)
	}
	if countOperations(api) == 0 && len(api.Components.Schemas) == 0 {
		t.Fatalf("%s has no operations or schemas", label)
	}
}

func assertSemanticShape(t *testing.T, source, converted *model.API, tc *conversionCase) {
	t.Helper()

	if countOperations(source) > 0 && countOperations(converted) == 0 {
		t.Fatalf("converted output lost all operations from source")
	}

	if tc.fromFormat == tc.toFormat && (tc.toFormat == format.FormatOpenAPI || tc.toFormat == format.FormatAsyncAPI) {
		assertPathCountPreserved(t, source, converted)
	}

	if tc.fromFormat == format.FormatOpenAPI && tc.toFormat == format.FormatOpenAPI {
		assertSchemaNamesPreserved(t, source, converted)
		assertOperationIDsPreserved(t, source, converted)
	}
}

func assertPathCountPreserved(t *testing.T, source, converted *model.API) {
	t.Helper()

	if len(source.Paths) == 0 {
		return
	}
	if len(converted.Paths) != len(source.Paths) {
		t.Fatalf("converted output path count = %d, want %d", len(converted.Paths), len(source.Paths))
	}
}

func assertSchemaNamesPreserved(t *testing.T, source, converted *model.API) {
	t.Helper()

	for name := range source.Components.Schemas {
		if _, ok := converted.Components.Schemas[name]; !ok {
			t.Fatalf("converted output is missing schema %q", name)
		}
	}
}

func assertOperationIDsPreserved(t *testing.T, source, converted *model.API) {
	t.Helper()

	convertedIDs := operationIDs(converted)
	for id := range operationIDs(source) {
		if _, ok := convertedIDs[id]; !ok {
			t.Fatalf("converted output is missing operation ID %q", id)
		}
	}
}

func operationIDs(api *model.API) map[string]struct{} {
	ids := make(map[string]struct{})
	collectOperationIDs := func(item model.PathItem) {
		for _, op := range pathItemOperations(&item) {
			if op.OperationID != "" {
				ids[op.OperationID] = struct{}{}
			}
		}
	}
	for path := range api.Paths {
		collectOperationIDs(api.Paths[path])
	}
	return ids
}

func countOperations(api *model.API) int {
	total := 0
	for path := range api.Paths {
		item := api.Paths[path]
		total += len(pathItemOperations(&item))
	}
	for name := range api.Webhooks {
		item := api.Webhooks[name]
		total += len(pathItemOperations(&item))
	}
	return total
}

func pathItemOperations(item *model.PathItem) []*model.Operation {
	operations := make([]*model.Operation, 0, 8)
	if item.Get != nil {
		operations = append(operations, item.Get)
	}
	if item.Put != nil {
		operations = append(operations, item.Put)
	}
	if item.Post != nil {
		operations = append(operations, item.Post)
	}
	if item.Delete != nil {
		operations = append(operations, item.Delete)
	}
	if item.Options != nil {
		operations = append(operations, item.Options)
	}
	if item.Head != nil {
		operations = append(operations, item.Head)
	}
	if item.Patch != nil {
		operations = append(operations, item.Patch)
	}
	if item.Trace != nil {
		operations = append(operations, item.Trace)
	}
	return operations
}
