package integration

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestTopLevelExtensionsDuringConversion(t *testing.T) {
	for _, tc := range []struct {
		name, input string
	}{
		{"openapi", "openapi: 3.0.3\ninfo: {title: Example, version: '1'}\npaths: {}\nx-team: {name: platform, enabled: true}\n"},
		{"swagger", "swagger: '2.0'\ninfo: {title: Example, version: '1'}\npaths: {}\nx-team: {name: platform, enabled: true}\n"},
	} {
		for _, encoding := range []string{"json", "yaml"} {
			for _, preserve := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/preserve=%t", tc.name, encoding, preserve), func(t *testing.T) {
					conv, err := converter.New(converter.WithExtensions(preserve))
					if err != nil {
						t.Fatal(err)
					}
					conv.RegisterParser(openapi.NewParser())
					conv.RegisterWriter(openapi.NewWriter())
					ctx := converter.WithEncoding(context.Background(), encoding)
					var output bytes.Buffer
					if err := conv.Convert(ctx, strings.NewReader(tc.input), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
						t.Fatal(err)
					}
					var doc map[string]interface{}
					if err := yaml.Unmarshal(output.Bytes(), &doc); err != nil {
						t.Fatal(err)
					}
					value, ok := doc["x-team"]
					if ok != preserve {
						t.Fatalf("x-team presence = %v, want %v: %s", ok, preserve, output.String())
					}
					if preserve {
						team, ok := value.(map[string]interface{})
						if !ok || team["name"] != "platform" || team["enabled"] != true {
							t.Fatalf("x-team = %#v", value)
						}
					}
				})
			}
		}
	}
}

func TestWithExtensionsDoesNotMutateModel(t *testing.T) {
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader("openapi: 3.1.0\ninfo: {title: Example, version: '1'}\npaths: {}\nx-owner: team\n"))
	if err != nil {
		t.Fatal(err)
	}
	conv, err := converter.New(converter.WithExtensions(false))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterWriter(openapi.NewWriter())
	var output bytes.Buffer
	if err := conv.WriteFromModel(context.Background(), api, &output, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
	if api.Extensions["x-owner"] != "team" || strings.Contains(output.String(), "x-owner") {
		t.Fatalf("model or output changed unexpectedly: %#v %s", api.Extensions, output.String())
	}
}

func TestWarningHandlerAndStrictModeWithRealOpenAPI(t *testing.T) {
	input := "openapi: 3.1.0\ninfo: {title: Example, version: ''}\npaths: {}\n"
	for _, strict := range []bool{false, true} {
		var warnings []string
		conv, err := converter.New(converter.WithStrict(strict), converter.WithWarningHandler(func(message string) {
			warnings = append(warnings, message)
		}))
		if err != nil {
			t.Fatal(err)
		}
		conv.RegisterParser(openapi.NewParser())
		conv.RegisterWriter(openapi.NewWriter())
		var output bytes.Buffer
		err = conv.Convert(context.Background(), strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatOpenAPI)
		if strict {
			if err == nil || output.Len() != 0 || len(warnings) != 1 {
				t.Fatalf("strict: output=%q warnings=%v error=%v", output.String(), warnings, err)
			}
		} else if err != nil || len(warnings) != 2 || output.Len() == 0 {
			t.Fatalf("non-strict: output=%q warnings=%v error=%v", output.String(), warnings, err)
		}
	}
}

func TestSwaggerExtensionsSameVersion(t *testing.T) {
	input := "swagger: '2.0'\ninfo: {title: Example, version: '1'}\npaths: {}\nx-owner: team\n"
	for _, encoding := range []string{"json", "yaml"} {
		conv, err := converter.New()
		if err != nil {
			t.Fatal(err)
		}
		conv.RegisterParser(openapi.NewParser())
		conv.RegisterWriter(openapi.NewWriter(openapi.WithWriterVersion("2.0")))
		var output bytes.Buffer
		ctx := converter.WithEncoding(context.Background(), encoding)
		if err := conv.Convert(ctx, strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
			t.Fatal(err)
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(output.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc["x-owner"] != "team" || doc["swagger"] != "2.0" {
			t.Fatalf("%s: %s", encoding, output.String())
		}
	}
}

func TestStrictAllowsValidOpenAPINullSchema(t *testing.T) {
	input := "openapi: 3.1.0\ninfo: {title: Example, version: '1'}\npaths: {}\ncomponents: {schemas: {Value: {type: 'null'}}}\n"
	conv, err := converter.New(converter.WithStrict(true))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter())
	var output bytes.Buffer
	if err := conv.Convert(context.Background(), strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatalf("strict conversion rejected a valid null schema: %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("strict conversion produced no output")
	}
}

func TestNestedExtensionsInOpenAPI(t *testing.T) {
	input := `openapi: 3.0.3
info:
  title: Example
  version: '1'
  x-info: info
externalDocs: {url: 'https://example.test/docs', x-doc: root-doc}
tags:
  - name: items
    x-tag: tag
    externalDocs: {url: 'https://example.test/tag', x-doc: tag-doc}
servers:
  - url: https://example.test/{stage}
    x-server: server
    variables:
      stage: {default: production, x-variable: variable}
paths:
  x-paths: paths
  /items:
    x-path: path
    post:
      x-operation: operation
      externalDocs: {url: 'https://example.test/operation', x-doc: operation-doc}
      callbacks:
        onEvent:
          x-callback: callback
          '{$request.body#/url}':
            x-callback-path: callback-path
            post:
              x-callback-operation: callback-operation
              responses: {'200': {description: OK}}
      parameters:
        - name: filter
          in: query
          x-parameter: parameter
          schema: {type: string, x-schema: parameter-schema}
          examples:
            byName: {value: sample, x-example: parameter-example}
      requestBody:
        x-body: body
        content:
          multipart/form-data:
            x-media: media
            schema:
              type: object
              x-schema: object
              discriminator: {propertyName: id, x-discriminator: discriminator}
              externalDocs: {url: 'https://example.test/schema', x-doc: schema-doc}
              properties:
                id: {type: string, x-property: property}
            examples:
              sample: {value: {x-data: example-value}, x-example: media-example}
            encoding:
              id:
                contentType: text/plain
                x-encoding: encoding
                headers:
                  X-Meta: {schema: {type: string}, x-encoding-header: encoding-header}
      responses:
        x-responses: responses
        '200':
          description: OK
          x-response: response
          headers:
            X-Rate: {x-header: header, schema: {type: integer}, examples: {normal: {value: 42, x-example: header-example}}}
          links:
            Follow:
              operationId: getItem
              x-link: response-link
              server: {url: 'https://example.test', x-server: link-server}
components:
  x-components: components
  schemas:
    Item: {type: object, x-schema: component}
  examples:
    Sample: {value: {x-data: component-value}, x-example: component-example}
  links:
    Follow: {operationId: getItem, x-link: component-link}
  parameters:
    Filter: {name: filter, in: query, schema: {type: string}, x-component-parameter: parameter}
  requestBodies:
    Payload: {content: {application/json: {schema: {type: string}}}, x-component-body: body}
  responses:
    Success: {description: OK, x-component-response: response}
  headers:
    Rate: {schema: {type: integer}, x-component-header: header}
  securitySchemes:
    Key: {type: apiKey, name: X-Key, in: header, x-security: scheme}
    OAuth:
      type: oauth2
      x-security: oauth
      flows:
        x-flows: flows
        clientCredentials:
          tokenUrl: https://example.test/token
          scopes: {x-read: Read access}
          x-flow: flow
`
	for _, encoding := range []string{"json", "yaml"} {
		for _, preserve := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/preserve=%t", encoding, preserve), func(t *testing.T) {
				conv, err := converter.New(converter.WithExtensions(preserve))
				if err != nil {
					t.Fatal(err)
				}
				conv.RegisterParser(openapi.NewParser())
				conv.RegisterWriter(openapi.NewWriter())
				var out bytes.Buffer
				ctx := converter.WithEncoding(context.Background(), encoding)
				if err := conv.Convert(ctx, strings.NewReader(input), &out, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
					t.Fatal(err)
				}
				var doc map[string]interface{}
				if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
					t.Fatal(err)
				}
				obj := func(v interface{}) map[string]interface{} {
					t.Helper()
					m, ok := v.(map[string]interface{})
					if !ok {
						t.Fatalf("expected object, got %#v", v)
					}
					return m
				}
				item := obj(obj(doc["paths"])["/items"])
				op := obj(item["post"])
				parameters := op["parameters"].([]interface{})
				parameter := obj(parameters[0])
				callback := obj(obj(op["callbacks"])["onEvent"])
				callbackPath := obj(callback["{$request.body#/url}"])
				body := obj(op["requestBody"])
				media := obj(obj(body["content"])["multipart/form-data"])
				schema := obj(media["schema"])
				response := obj(obj(op["responses"])["200"])
				components := obj(doc["components"])
				server := obj(doc["servers"].([]interface{})[0])
				checks := []struct {
					object     map[string]interface{}
					key, value string
				}{
					{obj(doc["info"]), "x-info", "info"}, {server, "x-server", "server"},
					{obj(doc["externalDocs"]), "x-doc", "root-doc"},
					{obj(doc["tags"].([]interface{})[0]), "x-tag", "tag"},
					{obj(obj(doc["tags"].([]interface{})[0])["externalDocs"]), "x-doc", "tag-doc"},
					{obj(op["externalDocs"]), "x-doc", "operation-doc"}, {obj(schema["externalDocs"]), "x-doc", "schema-doc"},
					{obj(obj(server["variables"])["stage"]), "x-variable", "variable"},
					{obj(doc["paths"]), "x-paths", "paths"}, {obj(op["responses"]), "x-responses", "responses"},
					{item, "x-path", "path"}, {op, "x-operation", "operation"},
					{callback, "x-callback", "callback"}, {callbackPath, "x-callback-path", "callback-path"},
					{obj(callbackPath["post"]), "x-callback-operation", "callback-operation"},
					{parameter, "x-parameter", "parameter"}, {obj(parameter["schema"]), "x-schema", "parameter-schema"},
					{obj(obj(parameter["examples"])["byName"]), "x-example", "parameter-example"},
					{obj(obj(media["examples"])["sample"]), "x-example", "media-example"},
					{obj(obj(components["examples"])["Sample"]), "x-example", "component-example"},
					{body, "x-body", "body"}, {media, "x-media", "media"}, {schema, "x-schema", "object"},
					{obj(schema["discriminator"]), "x-discriminator", "discriminator"},
					{obj(obj(media["encoding"])["id"]), "x-encoding", "encoding"},
					{obj(obj(obj(obj(media["encoding"])["id"])["headers"])["X-Meta"]), "x-encoding-header", "encoding-header"},
					{obj(obj(schema["properties"])["id"]), "x-property", "property"},
					{response, "x-response", "response"}, {obj(obj(response["headers"])["X-Rate"]), "x-header", "header"},
					{obj(obj(obj(obj(response["headers"])["X-Rate"])["examples"])["normal"]), "x-example", "header-example"},
					{obj(obj(response["links"])["Follow"]), "x-link", "response-link"},
					{obj(obj(obj(response["links"])["Follow"])["server"]), "x-server", "link-server"},
					{obj(obj(components["links"])["Follow"]), "x-link", "component-link"},
					{components, "x-components", "components"}, {obj(obj(components["schemas"])["Item"]), "x-schema", "component"},
					{obj(obj(components["parameters"])["Filter"]), "x-component-parameter", "parameter"},
					{obj(obj(components["requestBodies"])["Payload"]), "x-component-body", "body"},
					{obj(obj(components["responses"])["Success"]), "x-component-response", "response"},
					{obj(obj(components["headers"])["Rate"]), "x-component-header", "header"},
					{obj(obj(components["securitySchemes"])["Key"]), "x-security", "scheme"},
					{obj(obj(components["securitySchemes"])["OAuth"]), "x-security", "oauth"},
					{obj(obj(obj(components["securitySchemes"])["OAuth"])["flows"]), "x-flows", "flows"},
					{obj(obj(obj(obj(components["securitySchemes"])["OAuth"])["flows"])["clientCredentials"]), "x-flow", "flow"},
				}
				for _, check := range checks {
					value, ok := check.object[check.key]
					if ok != preserve || (preserve && value != check.value) {
						t.Errorf("%s = %#v, present=%v, preserve=%v", check.key, value, ok, preserve)
					}
				}
				if obj(obj(obj(media["examples"])["sample"])["value"])["x-data"] != "example-value" ||
					obj(obj(obj(components["examples"])["Sample"])["value"])["x-data"] != "component-value" {
					t.Error("named example data was modified")
				}
				flow := obj(obj(obj(obj(components["securitySchemes"])["OAuth"])["flows"])["clientCredentials"])
				if obj(flow["scopes"])["x-read"] != "Read access" {
					t.Error("OAuth scope data was modified")
				}
			})
		}
	}
}

func TestNestedSwaggerExtensions(t *testing.T) {
	input := `swagger: '2.0'
info: {title: Example, version: '1', x-info: info}
paths:
  x-paths: paths
  /items:
    x-path: path
    post:
      x-operation: operation
      parameters:
        - {name: q, in: query, type: string, x-parameter: parameter}
        - {name: body, in: body, x-body: body, schema: {type: string, x-schema: schema}}
      responses:
        x-responses: responses
        '200': {description: OK, x-response: response, schema: {type: string, x-schema: response-schema}}
definitions:
  Item: {type: string, x-schema: component}
securityDefinitions:
  Key: {type: apiKey, name: X-Key, in: header, x-security: scheme}
`
	for _, version := range []string{"2.0", "3.1"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				conv, err := converter.New()
				if err != nil {
					t.Fatal(err)
				}
				conv.RegisterParser(openapi.NewParser())
				conv.RegisterWriter(openapi.NewWriter(openapi.WithWriterVersion(version)))
				var out bytes.Buffer
				ctx := converter.WithEncoding(context.Background(), encoding)
				if err := conv.Convert(ctx, strings.NewReader(input), &out, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
					t.Fatal(err)
				}
				var doc map[string]interface{}
				if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
					t.Fatal(err)
				}
				obj := func(v interface{}) map[string]interface{} {
					t.Helper()
					m, ok := v.(map[string]interface{})
					if !ok {
						t.Fatalf("expected object, got %#v", v)
					}
					return m
				}
				item := obj(obj(doc["paths"])["/items"])
				op := obj(item["post"])
				parameter := obj(op["parameters"].([]interface{})[0])
				response := obj(obj(op["responses"])["200"])
				checks := []struct {
					object     map[string]interface{}
					key, value string
				}{
					{obj(doc["info"]), "x-info", "info"}, {item, "x-path", "path"},
					{op, "x-operation", "operation"}, {parameter, "x-parameter", "parameter"},
					{obj(doc["paths"]), "x-paths", "paths"}, {obj(op["responses"]), "x-responses", "responses"},
					{response, "x-response", "response"},
				}
				if version == "2.0" {
					bodyParameter := obj(op["parameters"].([]interface{})[1])
					if obj(bodyParameter["schema"])["x-schema"] != "schema" || obj(response["schema"])["x-schema"] != "response-schema" {
						t.Error("Swagger body or response schema extension was lost")
					}
					checks = append(checks,
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(doc["definitions"])["Item"].(map[string]interface{}), "x-schema", "component"},
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(obj(doc["securityDefinitions"])["Key"]), "x-security", "scheme"},
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(op["parameters"].([]interface{})[1]), "x-body", "body"},
					)
				} else {
					bodyMedia := obj(obj(obj(op["requestBody"])["content"])["application/json"])
					responseMedia := obj(obj(response["content"])["application/json"])
					if obj(bodyMedia["schema"])["x-schema"] != "schema" || obj(responseMedia["schema"])["x-schema"] != "response-schema" {
						t.Error("upgraded body or response schema extension was lost")
					}
					components := obj(doc["components"])
					checks = append(checks,
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(obj(components["schemas"])["Item"]), "x-schema", "component"},
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(obj(components["securitySchemes"])["Key"]), "x-security", "scheme"},
						struct {
							object     map[string]interface{}
							key, value string
						}{obj(op["requestBody"]), "x-body", "body"},
					)
				}
				for _, check := range checks {
					if check.object[check.key] != check.value {
						t.Errorf("%s = %#v, want %q", check.key, check.object[check.key], check.value)
					}
				}
			})
		}
	}
}

func TestBlueprintSchemaAssetExtensionsRoundTrip(t *testing.T) {
	input := `FORMAT: 1A
# Example
## Items [/items]
### Create [POST]
+ Request (application/json)
    + Schema

        {"type":"object","x-schema":{"$ref":"https://example.test/data"},"properties":{"id":{"type":"string","x-field":"field"}},"discriminator":{"propertyName":"id","x-disc":"disc"},"additionalProperties":{"type":"string","x-extra":"extra"}}
+ Response 200 (application/json)
`
	parse := func(input string) *model.API {
		t.Helper()
		api, err := apiblueprint.NewParser().Parse(context.Background(), strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		return api
	}
	api := parse(input)
	var blueprint bytes.Buffer
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &blueprint); err != nil {
		t.Fatal(err)
	}
	roundTrip := parse(blueprint.String())
	schema := roundTrip.Paths["/items"].Post.RequestBody.Content["application/json"].Schema
	extension, ok := schema.Extensions["x-schema"].(map[string]interface{})
	if !ok || extension["$ref"] != "https://example.test/data" || schema.Properties["id"].Extensions["x-field"] != "field" || schema.Discriminator == nil || schema.Discriminator.Extensions["x-disc"] != "disc" {
		t.Fatalf("schema extensions lost: %#v", schema)
	}
	additional, ok := schema.AdditionalProperties.(*model.Schema)
	if !ok || additional.Extensions["x-extra"] != "extra" {
		t.Fatalf("additionalProperties = %#v", schema.AdditionalProperties)
	}
	var output bytes.Buffer
	if err := openapi.NewWriter().Write(context.Background(), roundTrip, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"x-field"`) || !strings.Contains(output.String(), `"x-extra"`) {
		t.Fatalf("OpenAPI output lost extensions: %s", output.String())
	}
}

func TestSwaggerFormParameterExtensionMovesToProperty(t *testing.T) {
	input := `swagger: '2.0'
info: {title: Example, version: '1'}
consumes: [multipart/form-data]
paths:
  /upload:
    post:
      parameters:
        - {name: file, in: formData, type: file, x-field: retained}
      responses: {'200': {description: OK}}
`
	conv, err := converter.New()
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter())
	var out bytes.Buffer
	if err := conv.Convert(context.Background(), strings.NewReader(input), &out, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]interface{})
	op := paths["/upload"].(map[string]interface{})["post"].(map[string]interface{})
	body := op["requestBody"].(map[string]interface{})
	content := body["content"].(map[string]interface{})["multipart/form-data"].(map[string]interface{})
	schema := content["schema"].(map[string]interface{})
	field := schema["properties"].(map[string]interface{})["file"].(map[string]interface{})
	if field["x-field"] != "retained" {
		t.Fatalf("form field extension lost: %s", out.String())
	}
}
