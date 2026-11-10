package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSwaggerParametersRetainConstAndReferencedConstraints(t *testing.T) {
	for _, tc := range []struct {
		name, version, schema, components string
		want                              map[string]interface{}
	}{
		{"const", "3.1.0", "{type: string, const: admin}", "", map[string]interface{}{"type": "string", "enum": []interface{}{"admin"}}},
		{"ref", "3.0.3", "{$ref: '#/components/schemas/ID'}", "components: {schemas: {ID: {type: integer, minimum: 1, maximum: 9}}}", map[string]interface{}{"type": "integer", "minimum": float64(1), "maximum": float64(9)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := fmt.Sprintf("openapi: %s\ninfo: {title: T, version: '1'}\npaths:\n  /x:\n    get:\n      parameters: [{name: value, in: query, required: true, schema: %s}]\n      responses: {'200': {description: OK}}\n%s\n", tc.version, tc.schema, tc.components)
			checkSpec(t, format.FormatOpenAPI, []byte(input))
			api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion("2.0"), openapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, out.Bytes())
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			params := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["get"].(map[string]interface{})["parameters"].([]interface{})
			got := params[0].(map[string]interface{})
			for key, want := range tc.want {
				a, _ := json.Marshal(got[key])
				b, _ := json.Marshal(want)
				if !bytes.Equal(a, b) {
					t.Fatalf("%s=%s want %s", key, a, b)
				}
			}
		})
	}
	unsupported := `openapi: 3.0.3
info: {title: T, version: '1'}
paths: {/x: {get: {parameters: [{name: x, in: query, schema: {type: string, allOf: [{minLength: 3}]}}], responses: {'200': {description: OK}}}}}
`
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(unsupported))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := openapi.NewWriter(openapi.WithWriterVersion("2.0")).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
		t.Fatalf("lossy composition accepted: %v %s", err, out.String())
	}
}

// Repository JSON Schemas are pinned for 2.6 and 3.0 only.

func TestBlueprintAssetReferencesKeepTheirScope(t *testing.T) {
	input := `FORMAT: 1A
# T
## X [/x]
### X [POST]
+ Request (application/json)
    + Schema

        {"type":"object","properties":{"a":{"type":"integer"},"b":{"$ref":"#/properties/a"},"child":{"$ref":"#"}}}

+ Response 200 (application/json)
    + Schema

        {"type":"object","properties":{"a":{"type":"integer"},"b":{"$ref":"#/properties/a"},"child":{"$ref":"#"}}}
`
	api, err := apiblueprint.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"2.0", "3.0", "3.1"} {
		t.Run(version, func(t *testing.T) {
			var out bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion(version), openapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, out.Bytes())
			var doc any
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource("doc.json", doc); err != nil {
				t.Fatal(err)
			}
			pointer := "doc.json#/paths/~1x/post/requestBody/content/application~1json/schema"
			if version == "2.0" {
				pointer = "doc.json#/paths/~1x/post/parameters/0/schema"
			}
			schema, err := compiler.Compile(pointer)
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(map[string]interface{}{"b": float64(4), "child": map[string]interface{}{"b": float64(8)}}); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(map[string]interface{}{"b": "bad"}); err == nil {
				t.Fatal("local reference constraint lost")
			}
		})
	}
	broken := strings.Replace(input, "#/properties/a", "#/properties/missing", 1)
	if _, err := apiblueprint.NewParser().Parse(context.Background(), strings.NewReader(broken)); err == nil {
		t.Fatal("unresolved asset pointer accepted")
	}

	// A second Blueprint export/import must preserve the synthesized references.
	var blueprint bytes.Buffer
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &blueprint); err != nil {
		t.Fatal(err)
	}
	round, err := apiblueprint.NewParser().Parse(context.Background(), &blueprint)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := openapi.NewWriter(openapi.WithWriterVersion("3.1"), openapi.WithJSONOutput(true)).Write(context.Background(), round, &output); err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("roundtrip.json", document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("roundtrip.json#/paths/~1x/post/requestBody/content/application~1json/schema")
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(map[string]interface{}{"b": "wrong"}); err == nil {
		t.Fatal("constraint lost on Blueprint roundtrip")
	}
}
