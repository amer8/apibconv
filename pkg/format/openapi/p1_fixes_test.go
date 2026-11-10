package openapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestP1NestedAdditionalPropertiesReference(t *testing.T) {
	for _, tc := range []struct{ input, version, expected string }{
		{"swagger: '2.0'\ninfo: {title: T, version: '1'}\npaths: {}\ndefinitions:\n  Pet: {type: string}\n  Map: {type: object, additionalProperties: {$ref: '#/definitions/Pet'}}\n", "3.0", "#/components/schemas/Pet"},
		{"openapi: 3.0.0\ninfo: {title: T, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string}\n    Map: {type: object, additionalProperties: {$ref: '#/components/schemas/Pet'}}\n", "2.0", "#/definitions/Pet"},
	} {
		api, err := NewParser().Parse(context.Background(), strings.NewReader(tc.input))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err = NewWriter(WithWriterVersion(tc.version)).Write(context.Background(), api, &out); err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		var schemas map[string]any
		if tc.version == "2.0" {
			schemas = doc["definitions"].(map[string]any)
		} else {
			schemas = doc["components"].(map[string]any)["schemas"].(map[string]any)
		}
		ref := schemas["Map"].(map[string]any)["additionalProperties"].(map[string]any)["$ref"]
		if ref != tc.expected {
			t.Fatalf("ref=%v, want %s\n%s", ref, tc.expected, out.String())
		}
	}
}

func TestP1IndependentNumericBounds(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: T, version: '1'}
paths: {}
components:
  schemas:
    A: {type: number, minimum: 10, exclusiveMinimum: 5, maximum: 20, exclusiveMaximum: 25}
    B: {type: number, minimum: 5, exclusiveMinimum: 10, maximum: 25, exclusiveMaximum: 20}
    C: {type: number, minimum: 10, exclusiveMinimum: 10}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := api.Components.Schemas["A"], api.Components.Schemas["B"], api.Components.Schemas["C"]
	if *a.Minimum != 10 || a.ExclusiveMinimum || *a.Maximum != 20 || a.ExclusiveMaximum {
		t.Fatalf("A: %+v", a)
	}
	if *b.Minimum != 10 || !b.ExclusiveMinimum || *b.Maximum != 20 || !b.ExclusiveMaximum {
		t.Fatalf("B: %+v", b)
	}
	if *c.Minimum != 10 || !c.ExclusiveMinimum {
		t.Fatalf("C: %+v", c)
	}
}
