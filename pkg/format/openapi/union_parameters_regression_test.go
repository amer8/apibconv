package openapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/format"
	"go.yaml.in/yaml/v3"
)

func TestTypeUnionAndParameterExplode(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: Test, version: '1'}
paths:
  /items:
    get:
      parameters:
        - {name: ids, in: query, style: form, explode: false, schema: {type: array, items: {type: string}}}
        - {name: enabled, in: query, schema: {type: boolean}}
      responses: {'200': {description: ok}}
components:
  schemas:
    Id: {type: [string, integer, 'null'], anyOf: [{minLength: 1}, {minimum: 0}]}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = NewWriter(WithWriterVersion("3.1")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	if issues := specschema.Validate(format.FormatOpenAPI, out.Bytes()); len(issues) != 0 {
		t.Fatalf("output invalid: %v\n%s", issues, out.String())
	}
	var doc map[string]any
	if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	schema := doc["components"].(map[string]any)["schemas"].(map[string]any)["Id"].(map[string]any)
	if len(schema["anyOf"].([]any)) != 2 {
		t.Fatalf("original anyOf lost: %#v", schema)
	}
	all := schema["allOf"].([]any)[0].(map[string]any)["anyOf"].([]any)
	got := map[string]bool{}
	for _, v := range all {
		got[v.(map[string]any)["type"].(string)] = true
	}
	if !got["string"] || !got["integer"] || !got["null"] {
		t.Fatalf("union lost: %#v", schema)
	}
	params := doc["paths"].(map[string]any)["/items"].(map[string]any)["get"].(map[string]any)["parameters"].([]any)
	p := params[0].(map[string]any)
	if value, ok := p["explode"].(bool); !ok || value {
		t.Fatalf("explicit false lost: %#v", p)
	}
	if _, ok := params[1].(map[string]any)["explode"]; ok {
		t.Fatal("unspecified explode became explicit")
	}
}
