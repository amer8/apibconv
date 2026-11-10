package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func checkSpec(t *testing.T, kind format.Format, data []byte) {
	t.Helper()
	if issues := specschema.Validate(kind, data); len(issues) != 0 {
		t.Fatalf("invalid spec: %v\n%s", issues, data)
	}
}

func TestP1CallbacksRoundTripAndUnsupportedTargets(t *testing.T) {
	input := `openapi: 3.0.3
info: {title: T, version: '1'}
paths:
  /subscribe:
    post:
      responses: {'200': {description: OK}}
      callbacks:
        onEvent: {$ref: '#/components/callbacks/Event'}
components:
  securitySchemes: {key: {type: apiKey, in: header, name: X-Key}}
  callbacks:
    Event:
      '{$request.query.url}':
        post:
          security: [{key: []}]
          requestBody: {required: true, content: {application/json: {schema: {type: string}}}}
          responses: {'200': {description: OK}}
`
	checkSpec(t, format.FormatOpenAPI, []byte(input))
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"3.0", "3.1"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				var out bytes.Buffer
				ctx := converter.WithEncoding(context.Background(), encoding)
				if err := openapi.NewWriter(openapi.WithWriterVersion(version)).Write(ctx, api, &out); err != nil {
					t.Fatal(err)
				}
				checkSpec(t, format.FormatOpenAPI, out.Bytes())
				round, err := openapi.NewParser().Parse(ctx, &out)
				if err != nil {
					t.Fatal(err)
				}
				cb := round.Paths["/subscribe"].Post.Callbacks["onEvent"]["{$request.query.url}"].Post
				if cb == nil || cb.RequestBody == nil || cb.RequestBody.Content["application/json"].Schema.Type != model.TypeString || len(cb.Security) != 1 {
					t.Fatalf("callback lost: %#v", cb)
				}
			})
		}
	}

	for _, writer := range []format.Writer{openapi.NewWriter(openapi.WithWriterVersion("2.0")), apiblueprint.NewWriter()} {
		var out bytes.Buffer
		if err := writer.Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
			t.Fatalf("%s must reject callbacks: %v %s", writer.Format(), err, out.String())
		}
	}
	broken := strings.Replace(input, "#/components/callbacks/Event", "#/components/callbacks/Missing", 1)
	if _, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(broken)); err == nil {
		t.Fatal("unresolved callback accepted")
	}
}

func compileSchema(t *testing.T, raw any) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", raw); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestP1ConstAndUnsupportedSchemaKeywords(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: T, version: '1'}
paths:
  /x:
    post:
      requestBody: {content: {application/json: {schema: {type: object, required: [kind], properties: {kind: {type: string, const: payment}}}}}}
      responses: {'200': {description: OK}}
`
	checkSpec(t, format.FormatOpenAPI, []byte(input))
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"3.1", "3.0", "2.0"} {
		t.Run(version, func(t *testing.T) {
			var out bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion(version), openapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, out.Bytes())
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			op := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["post"].(map[string]interface{})
			var raw any
			if version == "2.0" {
				raw = op["parameters"].([]interface{})[0].(map[string]interface{})["schema"]
			} else {
				raw = op["requestBody"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})["schema"]
			}
			s := compileSchema(t, raw)
			if err := s.Validate(map[string]interface{}{"kind": "payment"}); err != nil {
				t.Fatal(err)
			}
			if err := s.Validate(map[string]interface{}{"kind": "other"}); err == nil {
				t.Fatalf("const restriction lost: %s", out.String())
			}
		})
	}

	for _, target := range []struct {
		name   string
		writer format.Writer
		parser format.Parser
	}{
		{"blueprint", apiblueprint.NewWriter(), apiblueprint.NewParser()},
	} {
		t.Run(target.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := target.writer.Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			round, err := target.parser.Parse(context.Background(), &out)
			if err != nil {
				t.Fatal(err)
			}
			body := round.Paths["/x"].Post.RequestBody
			if body == nil {
				t.Fatal("request schema lost")
			}
			data, err := json.Marshal(body.Content["application/json"].Schema)
			if err != nil {
				t.Fatal(err)
			}
			var raw any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			schema := compileSchema(t, raw)
			if err := schema.Validate(map[string]interface{}{"kind": "payment"}); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(map[string]interface{}{"kind": "other"}); err == nil {
				t.Fatal("const lost on cross-format roundtrip")
			}
		})
	}
	for _, replacement := range []string{"if: {type: string}", "additionalProperties: {const: allowed, if: {type: string}}"} {
		bad := strings.Replace(input, "type: string, const: payment", replacement, 1)
		if _, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(bad)); err == nil || !strings.Contains(err.Error(), "unsupported schema keyword") {
			t.Fatalf("unsupported schema accepted: %v", err)
		}
	}
}

func TestP1ConstNullAndReferenceSiblings(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: T, version: '1'}
paths: {}
components:
  schemas:
    Base: {type: string}
    Narrow: {$ref: '#/components/schemas/Base', const: payment}
    NullOnly: {const: null, enum: [null, 3]}
`
	checkSpec(t, format.FormatOpenAPI, []byte(input))
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := openapi.NewWriter(openapi.WithWriterVersion("3.1"), openapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	checkSpec(t, format.FormatOpenAPI, out.Bytes())
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	nullSchema := doc["components"].(map[string]interface{})["schemas"].(map[string]interface{})["NullOnly"]
	compiled := compileSchema(t, nullSchema)
	if err := compiled.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(3.0); err == nil {
		t.Fatal("const null was lost when combined with enum")
	}
	for _, version := range []string{"2.0", "3.0"} {
		out.Reset()
		if err := openapi.NewWriter(openapi.WithWriterVersion(version)).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
			t.Fatalf("%s accepted incompatible constraints: %v", version, err)
		}
	}
}

func TestP1SwaggerRejectsUnsupportedSchemasAndContent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema *model.Schema
	}{
		{"oneOf", &model.Schema{OneOf: []*model.Schema{{Type: model.TypeString}, {Type: model.TypeInteger}}}},
		{"anyOf", &model.Schema{AnyOf: []*model.Schema{{Type: model.TypeString}, {Type: model.TypeInteger}}}},
		{"not", &model.Schema{Not: &model.Schema{Type: model.TypeString}}},
		{"nullable", &model.Schema{Type: model.TypeString, Nullable: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Info = model.Info{Title: "T", Version: "1"}
			api.Paths["/x"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: tc.schema}}}, Responses: model.Responses{"200": {Description: "OK"}}}}
			var out bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion("2.0")).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
				t.Fatalf("expected rejection: %v %s", err, out.String())
			}
		})
	}
	for _, request := range []bool{true, false} {
		t.Run(fmt.Sprintf("request=%v", request), func(t *testing.T) {
			api := model.NewAPI()
			api.Info = model.Info{Title: "T", Version: "1"}
			content := map[string]model.MediaType{"application/json": {Schema: &model.Schema{Type: model.TypeObject}}, "text/plain": {Schema: &model.Schema{Type: model.TypeString}}}
			op := &model.Operation{Responses: model.Responses{"200": {Description: "OK"}}}
			if request {
				op.RequestBody = &model.RequestBody{Content: content}
			} else {
				op.Responses["200"] = model.Response{Description: "OK", Content: content}
			}
			api.Paths["/x"] = model.PathItem{Post: op}
			var out bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion("2.0")).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
				t.Fatalf("expected rejection: %v %s", err, out.String())
			}
			content["text/plain"] = content["application/json"]
			if err := openapi.NewWriter(openapi.WithWriterVersion("2.0"), openapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, out.Bytes())
		})
	}
}

func TestP1BlueprintResponseAssetResolvesReferences(t *testing.T) {
	n := 4
	api := model.NewAPI()
	api.Info = model.Info{Title: "T", Version: "1"}
	api.Components.Schemas["Thing"] = &model.Schema{Type: model.TypeObject, Required: []string{"id"}, Properties: map[string]*model.Schema{"id": {Type: model.TypeString, MinLength: &n}, "child": {Ref: "#/components/schemas/Thing"}}}
	api.Paths["/x"] = model.PathItem{Get: &model.Operation{Responses: model.Responses{"200": {Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Ref: "#/components/schemas/Thing"}}}}}}}
	api.Paths["/send"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Ref: "#/components/schemas/Thing"}}}}, Responses: model.Responses{"200": {Description: "OK"}}}}
	var out bytes.Buffer
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(out.String(), "    + Schema\n\n        ", 2)
	if len(parts) != 2 {
		t.Fatal(out.String())
	}
	line := strings.SplitN(parts[1], "\n", 2)[0]
	var raw any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatal(err)
	}
	schema := compileSchema(t, raw)
	if err := schema.Validate(map[string]interface{}{"id": "abcd", "child": map[string]interface{}{"id": "valid"}}); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(map[string]interface{}{"id": "abc"}); err == nil {
		t.Fatal("referenced schema constraints lost")
	}
	round, err := apiblueprint.NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if round.Paths["/send"].Post.RequestBody.Content["application/json"].Schema.Ref != "#/components/schemas/Thing" {
		t.Fatal("request schema reference lost")
	}
	component := round.Components.Schemas["Thing"]
	if component == nil || component.Properties["id"].MinLength == nil || *component.Properties["id"].MinLength != 4 {
		t.Fatalf("component constraints lost on roundtrip: %#v", component)
	}
	api.Components.Schemas = map[string]*model.Schema{}
	out.Reset()
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
		t.Fatalf("missing reference accepted: %v %s", err, out.String())
	}
}
