package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestSwaggerParameterConstraintsSurviveUpgradeAndRoundTrip(t *testing.T) {
	const input = `swagger: "2.0"
info: {title: test, version: "1"}
paths:
  /things:
    get:
      parameters:
        - name: state
          in: query
          type: string
          enum: [active, inactive]
          default: active
          minLength: 2
          maxLength: 8
          pattern: '^[a-z]+$'
        - name: count
          in: query
          type: integer
          minimum: 2
          maximum: 20
          exclusiveMinimum: true
          multipleOf: 2
        - name: tags
          in: query
          type: array
          items: {type: string, enum: [red, blue]}
          minItems: 1
          maxItems: 3
          uniqueItems: true
      responses:
        '200': {description: ok}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"3.0", "2.0"} {
		t.Run(version, func(t *testing.T) {
			var out bytes.Buffer
			if err := NewWriter(WithWriterVersion(version)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			params := doc["paths"].(map[string]interface{})["/things"].(map[string]interface{})["get"].(map[string]interface{})["parameters"].([]interface{})
			schema := func(i int) map[string]interface{} {
				p := params[i].(map[string]interface{})
				if version == "3.0" {
					return p["schema"].(map[string]interface{})
				}
				return p
			}
			state := schema(0)
			if fmt.Sprint(state["enum"]) != "[active inactive]" || state["default"] != "active" || state["pattern"] != "^[a-z]+$" || state["minLength"] != float64(2) || state["maxLength"] != float64(8) {
				t.Fatalf("lost string constraints: %#v", state)
			}
			count := schema(1)
			if count["minimum"] != float64(2) || count["maximum"] != float64(20) || count["exclusiveMinimum"] != true || count["multipleOf"] != float64(2) {
				t.Fatalf("lost numeric constraints: %#v", count)
			}
			tags := schema(2)
			if tags["minItems"] != float64(1) || tags["maxItems"] != float64(3) || tags["uniqueItems"] != true || fmt.Sprint(tags["items"].(map[string]interface{})["enum"]) != "[red blue]" {
				t.Fatalf("lost array constraints: %#v", tags)
			}
		})
	}
}

func TestSwaggerArrayCollectionFormats(t *testing.T) {
	cases := []struct {
		name, location, format, style string
		explode                       bool
		v2                            string
	}{
		{"query default csv", "query", "", "form", false, ""},
		{"query csv", "query", "csv", "form", false, ""},
		{"query multi", "query", "multi", "form", true, "multi"},
		{"query ssv", "query", "ssv", "spaceDelimited", false, "ssv"},
		{"query pipes", "query", "pipes", "pipeDelimited", false, "pipes"},
		{"path csv", "path", "csv", "simple", false, ""},
		{"header csv", "header", "csv", "simple", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			collection := ""
			if tc.format != "" {
				collection = "\n    collectionFormat: " + tc.format
			}
			input := fmt.Sprintf(`swagger: "2.0"
info: {title: test, version: "1"}
parameters:
  Tags:
    name: tags
    in: %s
    required: %t
    type: array
    items: {type: string}%s
paths:
  /things/{tags}:
    parameters:
      - {$ref: '#/parameters/Tags'}
    get:
      parameters:
        - {$ref: '#/parameters/Tags'}
      responses:
        '200': {description: ok}
`, tc.location, tc.location == "path", collection)
			api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			item := api.Paths["/things/{tags}"]
			for _, p := range []model.Parameter{item.Parameters[0], item.Get.Parameters[0]} {
				if p.Style != tc.style || !p.ExplodeSet || p.Explode != tc.explode {
					t.Fatalf("serialization mismatch: %+v", p)
				}
			}
			var upgraded bytes.Buffer
			if err := NewWriter(WithWriterVersion("3.0")).Write(context.Background(), api, &upgraded); err != nil {
				t.Fatal(err)
			}
			var upgradedDoc map[string]interface{}
			if err := json.Unmarshal(upgraded.Bytes(), &upgradedDoc); err != nil {
				t.Fatal(err)
			}
			upgradedItem := upgradedDoc["paths"].(map[string]interface{})["/things/{tags}"].(map[string]interface{})
			for _, raw := range []interface{}{upgradedItem["parameters"].([]interface{})[0], upgradedItem["get"].(map[string]interface{})["parameters"].([]interface{})[0]} {
				p := raw.(map[string]interface{})
				if p["style"] != tc.style || p["explode"] != tc.explode {
					t.Fatalf("wrong upgraded serialization: %#v", p)
				}
			}
			var out bytes.Buffer
			if err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			itemOut := doc["paths"].(map[string]interface{})["/things/{tags}"].(map[string]interface{})
			for _, raw := range []interface{}{itemOut["parameters"].([]interface{})[0], itemOut["get"].(map[string]interface{})["parameters"].([]interface{})[0]} {
				p := raw.(map[string]interface{})
				got, _ := p["collectionFormat"].(string)
				if got != tc.v2 {
					t.Fatalf("collectionFormat = %q; want %q", got, tc.v2)
				}
			}
		})
	}
}

func TestSwaggerUnsupportedArrayCollectionFormatsFailExplicitly(t *testing.T) {
	for _, tc := range []struct{ location, format string }{{"query", "tsv"}, {"header", "pipes"}, {"formData", "multi"}} {
		t.Run(tc.location+"_"+tc.format, func(t *testing.T) {
			input := fmt.Sprintf(`swagger: "2.0"
info: {title: test, version: "1"}
paths:
  /things:
    get:
      parameters:
        - {name: things, in: %s, type: array, items: {type: string}, collectionFormat: %s}
      responses:
        '200': {description: ok}
`, tc.location, tc.format)
			_, err := NewParser().Parse(context.Background(), strings.NewReader(input))
			if err == nil || !strings.Contains(err.Error(), "cannot be represented without loss") {
				t.Fatalf("expected unsupported serialization error, got %v", err)
			}
		})
	}
}
