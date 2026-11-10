package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
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

func TestAsyncSchemaDialectAndRoundTrip(t *testing.T) {
	input := `openapi: 3.0.3
info: {title: T, version: '1'}
paths:
  /event:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name: {type: string, nullable: true}
                amount: {type: integer, minimum: 0, exclusiveMinimum: true}
      responses: {'200': {description: OK}}
`
	api, err := openapi.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"2.0", "2.1", "2.2", "2.3", "2.4", "2.5", "2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			var out bytes.Buffer
			w := asyncapi.NewWriter(asyncapi.WithAsyncWriterVersion(version), asyncapi.WithJSONOutput(true))
			ctx := converter.WithProtocol(context.Background(), "kafka")
			if err := w.Write(ctx, api, &out); err != nil {
				t.Fatal(err)
			}
			// Repository JSON Schemas are pinned for 2.6 and 3.0 only.
			if version == "2.6" || version == "3.0" {
				checkSpec(t, format.FormatAsyncAPI, out.Bytes())
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			ch := doc["channels"].(map[string]interface{})["/event"].(map[string]interface{})
			var message map[string]interface{}
			if strings.HasPrefix(version, "2.") {
				message = ch["subscribe"].(map[string]interface{})["message"].(map[string]interface{})
			} else {
				message = ch["messages"].(map[string]interface{})["message_application_json"].(map[string]interface{})
			}
			raw := message["payload"]
			compiler := jsonschema.NewCompiler()
			compiler.DefaultDraft(jsonschema.Draft7)
			if err := compiler.AddResource("payload.json", raw); err != nil {
				t.Fatal(err)
			}
			schema, err := compiler.Compile("payload.json")
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(map[string]interface{}{"name": nil, "amount": float64(1)}); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(map[string]interface{}{"name": nil, "amount": float64(0)}); err == nil {
				t.Fatal("exclusive minimum lost")
			}
			round, err := asyncapi.NewParser().Parse(ctx, bytes.NewReader(out.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			got := round.Paths["/event"].Post.RequestBody.Content["application/json"].Schema
			if !got.Properties["name"].Nullable || !got.Properties["amount"].ExclusiveMinimum || *got.Properties["amount"].Minimum != 0 {
				t.Fatalf("schema roundtrip lost constraints: %#v", got)
			}
			var oas bytes.Buffer
			if err := openapi.NewWriter(openapi.WithWriterVersion("3.0"), openapi.WithJSONOutput(true)).Write(ctx, round, &oas); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, oas.Bytes())
			oas.Reset()
			if err := openapi.NewWriter(openapi.WithWriterVersion("2.0")).Write(ctx, round, &oas); err == nil || oas.Len() != 0 {
				t.Fatal("Swagger 2.0 must reject nullable schema")
			}
			oas.Reset()
			if err := openapi.NewWriter(openapi.WithWriterVersion("3.1"), openapi.WithJSONOutput(true)).Write(ctx, round, &oas); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatOpenAPI, oas.Bytes())
		})
	}
}

func TestAsyncNativeNumericBoundsAndTypeUnion(t *testing.T) {
	for _, version := range []string{"2.6.0", "3.0.0"} {
		t.Run(version, func(t *testing.T) {
			input := fmt.Sprintf(`asyncapi: %s
info: {title: T, version: '1'}
components:
  schemas:
    Event:
      type: object
      properties:
        name: {type: [string, 'null']}
        amount: {type: number, minimum: 10, exclusiveMinimum: 5, maximum: 20, exclusiveMaximum: 20}
`, version)
			if strings.HasPrefix(version, "2") {
				input += "channels: {}\n"
			}
			checkSpec(t, format.FormatAsyncAPI, []byte(input))
			api, err := asyncapi.NewParser().Parse(context.Background(), strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			schema := api.Components.Schemas["Event"]
			if !schema.Properties["name"].Nullable {
				t.Fatal("nullable union lost")
			}
			amount := schema.Properties["amount"]
			if *amount.Minimum != 10 || amount.ExclusiveMinimum || *amount.Maximum != 20 || !amount.ExclusiveMaximum {
				t.Fatalf("bounds lost: %+v", amount)
			}
			var out bytes.Buffer
			if err := asyncapi.NewWriter(asyncapi.WithAsyncWriterVersion(version), asyncapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			checkSpec(t, format.FormatAsyncAPI, out.Bytes())
		})
	}
}

func TestAsyncRejectsUnrepresentableMessageMetadata(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"v2 headers", `asyncapi: 2.6.0
info: {title: T, version: '1'}
channels: {events: {subscribe: {message: {headers: {type: object, required: [signature], properties: {signature: {type: string}}}, payload: {type: string}}}}}
`},
		{"v3 headers", `asyncapi: 3.0.0
info: {title: T, version: '1'}
channels: {events: {address: events, messages: {m: {headers: {type: object, properties: {signature: {type: string}}}, payload: {type: string}}}}}
operations: {send: {action: send, channel: {$ref: '#/channels/events'}, messages: [{$ref: '#/channels/events/messages/m'}]}}
`},
		{"v2 avro", `asyncapi: 2.6.0
info: {title: T, version: '1'}
channels: {events: {subscribe: {message: {schemaFormat: 'application/vnd.apache.avro+json;version=1.9.0', payload: {type: string}}}}}
`},
		{"v3 avro", `asyncapi: 3.0.0
info: {title: T, version: '1'}
channels: {events: {address: events, messages: {m: {payload: {schemaFormat: 'application/vnd.apache.avro+json;version=1.9.0', schema: {type: string}}}}}}
operations: {send: {action: send, channel: {$ref: '#/channels/events'}, messages: [{$ref: '#/channels/events/messages/m'}]}}
`},
		{"v2 trait headers", `asyncapi: 2.6.0
info: {title: T, version: '1'}
channels: {events: {subscribe: {message: {traits: [{$ref: '#/components/messageTraits/Auth'}], payload: {type: string}}}}}
components: {messageTraits: {Auth: {headers: {type: object, properties: {signature: {type: string}}}}}}
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkSpec(t, format.FormatAsyncAPI, []byte(tc.input))
			if _, err := asyncapi.NewParser().Parse(context.Background(), strings.NewReader(tc.input)); err == nil {
				t.Fatal("message metadata silently discarded")
			}
		})
	}
}

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
	for _, version := range []string{"2.6", "3.0"} {
		var out bytes.Buffer
		if err := asyncapi.NewWriter(asyncapi.WithAsyncWriterVersion(version), asyncapi.WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
			t.Fatal(err)
		}
		checkSpec(t, format.FormatAsyncAPI, out.Bytes())
		var doc map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft7)
		if err := compiler.AddResource("async.json", doc); err != nil {
			t.Fatal(err)
		}
		pointer := "async.json#/channels/~1x/subscribe/message/payload"
		if version == "3.0" {
			pointer = "async.json#/channels/~1x/messages/message_application_json/payload"
		}
		schema, err := compiler.Compile(pointer)
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(map[string]interface{}{"b": "wrong"}); err == nil {
			t.Fatal("local reference constraint lost in AsyncAPI")
		}
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
