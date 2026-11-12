package openapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestReferencesConstraintsAndServersRoundTrip(t *testing.T) {
	input := `openapi: 3.0.3
info: {title: Test, version: '1'}
servers: [{url: 'https://root.example'}]
paths:
  /original:
    servers: [{url: 'https://path.example'}]
    get:
      servers: [{url: 'https://operation.example'}]
      parameters: [{$ref: '#/components/parameters/Limit'}]
      requestBody: {$ref: '#/components/requestBodies/Body'}
      responses: {'200': {$ref: '#/components/responses/Success'}}
  /alias: {$ref: '#/paths/~1original'}
components:
  parameters:
    Limit: {name: limit, in: query, schema: {type: integer, minimum: 1}}
  requestBodies:
    Body: {content: {application/json: {schema: {type: string, minLength: 2, maxLength: 8, pattern: '^[a-z]+$'}}}}
  responses:
    Success: {description: accepted, content: {application/json: {schema: {type: object, additionalProperties: false}}}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/original", "/alias"} {
		item := api.Paths[path]
		if item.Get == nil || len(item.Servers) != 1 || item.Servers[0].URL != "https://path.example" || len(item.Get.Servers) != 1 || item.Get.Servers[0].URL != "https://operation.example" {
			t.Fatalf("lost operation/server at %s: %#v", path, item)
		}
		op := item.Get
		if len(op.Parameters) != 1 || op.Parameters[0].Name != "limit" || op.Parameters[0].Schema.Minimum == nil {
			t.Fatalf("parameter = %#v", op.Parameters)
		}
		s := op.RequestBody.Content["application/json"].Schema
		if s.MinLength == nil || *s.MinLength != 2 || s.MaxLength == nil || *s.MaxLength != 8 || s.Pattern != "^[a-z]+$" {
			t.Fatalf("body schema = %#v", s)
		}
		if op.Responses["200"].Description != "accepted" || op.Responses["200"].Content["application/json"].Schema.AdditionalProperties != false {
			t.Fatalf("response = %#v", op.Responses["200"])
		}
	}
	var output bytes.Buffer
	if err := NewWriter(WithWriterVersion("3.0.3")).Write(context.Background(), api, &output); err != nil {
		t.Fatal(err)
	}
	if issues := specschema.Validate(format.FormatOpenAPI, output.Bytes()); len(issues) != 0 {
		t.Fatalf("invalid output: %v\n%s", issues, output.String())
	}
	reparsed, err := NewParser().Parse(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	op := reparsed.Paths["/alias"].Get
	if op == nil || op.Servers[0].URL != "https://operation.example" || op.RequestBody.Content["application/json"].Schema.Pattern != "^[a-z]+$" || op.Responses["200"].Content["application/json"].Schema.AdditionalProperties != false {
		t.Fatalf("roundtrip = %#v", op)
	}
}

func TestReferencesRejectUnresolvable(t *testing.T) {
	for _, ref := range []string{"#/paths/~1missing", "https://example.test/spec.yaml#/paths/~1x", "#/paths/~2bad"} {
		input := "openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths:\n  /x: {$ref: '" + ref + "'}\n"
		if _, err := NewParser().Parse(context.Background(), strings.NewReader(input)); err == nil {
			t.Fatalf("accepted path ref %q", ref)
		}
	}
}

func TestSwaggerFormParameters(t *testing.T) {
	input := `swagger: '2.0'
info: {title: Test, version: '1'}
consumes: [multipart/form-data]
parameters:
  File: {name: file, in: formData, type: file, required: true}
paths:
  /upload:
    parameters: [{$ref: '#/parameters/File'}]
    post:
      parameters: [{name: caption, in: formData, type: string}]
      responses: {'200': {description: OK}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	op := api.Paths["/upload"].Post
	if op == nil || len(op.Parameters) != 0 || op.RequestBody == nil {
		t.Fatalf("operation = %#v", op)
	}
	s := op.RequestBody.Content["multipart/form-data"].Schema
	if s == nil || s.Properties["file"].Type != model.TypeString || s.Properties["file"].Format != "binary" || s.Properties["caption"] == nil || len(s.Required) != 1 || s.Required[0] != "file" {
		t.Fatalf("schema = %#v", s)
	}
	var output bytes.Buffer
	if err := NewWriter(WithWriterVersion("3.0.3")).Write(context.Background(), api, &output); err != nil {
		t.Fatal(err)
	}
	if issues := specschema.Validate(format.FormatOpenAPI, output.Bytes()); len(issues) != 0 {
		t.Fatalf("invalid output: %v\n%s", issues, output.String())
	}
}

func TestSwaggerPathAndResponseReferences(t *testing.T) {
	input := `swagger: '2.0'
info: {title: Test, version: '1'}
parameters:
  Limit: {name: limit, in: query, type: integer}
responses:
  OK: {description: accepted, schema: {type: string, minLength: 2}}
paths:
  /original:
    get:
      parameters: [{$ref: '#/parameters/Limit'}]
      responses: {'200': {$ref: '#/responses/OK'}}
  /alias: {$ref: '#/paths/~1original'}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	op := api.Paths["/alias"].Get
	if op == nil || len(op.Parameters) != 1 || op.Parameters[0].Name != "limit" || op.Responses["200"].Description != "accepted" || op.Responses["200"].Content["application/json"].Schema.MinLength == nil {
		t.Fatalf("lost v2 references: %#v", op)
	}
}

func TestUnsupportedOutputVersionAndEncoding(t *testing.T) {
	for _, tc := range []struct{ version, encoding string }{{"5.0", "json"}, {"3.0", "potato"}} {
		var out bytes.Buffer
		err := NewWriter(WithWriterVersion(tc.version)).Write(converter.WithEncoding(context.Background(), tc.encoding), model.NewAPI(), &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("version %q encoding %q: err=%v output=%s", tc.version, tc.encoding, err, out.String())
		}
	}
}

func TestSchemaExclusiveBounds(t *testing.T) {
	for _, tc := range []struct{ version, fragment string }{{"3.0.3", "minimum: 5\n      exclusiveMinimum: true"}, {"3.1.0", "exclusiveMinimum: 5"}} {
		input := "openapi: " + tc.version + "\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    N:\n      type: number\n      " + tc.fragment + "\n"
		api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err = NewWriter(WithWriterVersion(tc.version)).Write(context.Background(), api, &out); err != nil {
			t.Fatal(err)
		}
		var doc map[string]interface{}
		if err = yaml.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		round, err := NewParser().Parse(context.Background(), &out)
		if err != nil {
			t.Fatal(err)
		}
		s := round.Components.Schemas["N"]
		if s.Minimum == nil || *s.Minimum != 5 || !s.ExclusiveMinimum {
			t.Fatalf("%s bound = %#v", tc.version, s)
		}
	}
}

func TestReferenceExtensionsArePreserved(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: Example, version: '1'}
paths:
  /original:
    x-source: source
    get:
      parameters: [{$ref: '#/components/parameters/Limit', x-local: local, x-shared: local}]
      responses: {'200': {$ref: '#/components/responses/Success', x-local: response}}
  /alias: {$ref: '#/paths/~1original', x-alias: alias}
components:
  parameters:
    Limit: {name: limit, in: query, schema: {type: integer}, x-shared: source}
  responses:
    Success: {description: OK, x-source: response}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/alias"]
	if item.Extensions["x-alias"] != "alias" || item.Extensions["x-source"] != "source" {
		t.Fatalf("path extensions = %#v", item.Extensions)
	}
	param := item.Get.Parameters[0]
	if param.Extensions["x-local"] != "local" || param.Extensions["x-shared"] != "local" {
		t.Fatalf("parameter extensions = %#v", param.Extensions)
	}
	response := item.Get.Responses["200"]
	if response.Extensions["x-local"] != "response" || response.Extensions["x-source"] != "response" {
		t.Fatalf("response extensions = %#v", response.Extensions)
	}
	var output bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &output); err != nil {
		t.Fatal(err)
	}
	reparsed, err := NewParser().Parse(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	alias := reparsed.Paths["/alias"]
	if alias.Extensions["x-alias"] != "alias" || alias.Get.Parameters[0].Extensions["x-shared"] != "local" {
		t.Fatal("reference extensions were lost after writing")
	}
}
