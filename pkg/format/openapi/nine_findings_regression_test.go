package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

func TestTransformedReferencedObjectsAreWrittenInline(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: API, version: '1'}
paths:
  /x:
    post:
      parameters: [{$ref: '#/components/parameters/Param'}]
      requestBody: {$ref: '#/components/requestBodies/Body'}
      responses: {'200': {$ref: '#/components/responses/OK'}}
components:
  parameters:
    Param: {name: p, in: query, schema: {type: string}}
  requestBodies:
    Body: {content: {application/json: {schema: {type: string}}}}
  responses:
    OK: {description: original}
`
	c, err := converter.New(converter.WithTransform(func(api *model.API) error {
		op := api.Paths["/x"].Post
		op.Parameters[0].Description = "changed parameter"
		op.RequestBody.Description = "changed body"
		response := op.Responses["200"]
		response.Description = "changed response"
		op.Responses["200"] = response
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(NewParser())
	c.RegisterWriter(NewWriter(WithJSONOutput(true)))
	var out bytes.Buffer
	if err := c.Convert(context.Background(), strings.NewReader(input), &out, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	op := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["post"].(map[string]interface{})
	p := op["parameters"].([]interface{})[0].(map[string]interface{})
	body := op["requestBody"].(map[string]interface{})
	resp := op["responses"].(map[string]interface{})["200"].(map[string]interface{})
	if p["description"] != "changed parameter" || body["description"] != "changed body" || resp["description"] != "changed response" {
		t.Fatalf("transform lost: %s", out.String())
	}
}

func TestBlueprintJSONNullAndLargeNumbers(t *testing.T) {
	input := `FORMAT: 1A
# API
## X [/x]
### Create [POST]
+ Request (application/json)

        null
+ Response 200 (application/json)

        {"id":9007199254740993,"nested":[null,9007199254740993]}
+ Response 201 (application/json)

        null
`
	api, err := apiblueprint.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, asYAML := range []bool{false, true} {
		var out bytes.Buffer
		if err := NewWriter(WithYAML(asYAML)).Write(context.Background(), api, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "9007199254740993") || strings.Contains(out.String(), "9007199254740992") {
			t.Fatalf("number changed: %s", out.String())
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		post := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["post"].(map[string]interface{})
		mt := post["requestBody"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})
		if value, ok := mt["example"]; !ok || value != nil {
			t.Fatalf("null lost: %s", out.String())
		}
		responses := post["responses"].(map[string]interface{})
		content := responses["201"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})
		if value, ok := content["example"]; !ok || value != nil {
			t.Fatalf("response null lost: %s", out.String())
		}
		number := responses["200"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})["example"].(map[string]interface{})["id"]
		if _, isString := number.(string); isString {
			t.Fatalf("numeric example became a string: %s", out.String())
		}
	}
}

func TestNullableReferenceAcceptsNull(t *testing.T) {
	input := `FORMAT: 1A
# API
## X [/x]
### Read [GET]
+ Response 200 (application/json)
    + Attributes (object)
        + address (Address, nullable)
# Data Structures
## Address (object)
+ street (string)
`
	api, err := apiblueprint.NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := NewWriter(WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("doc.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("doc.json#/paths/~1x/get/responses/200/content/application~1json/schema/properties/address")
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(nil); err != nil {
		t.Fatalf("null rejected: %v\n%s", err, out.String())
	}
	if err := schema.Validate("invalid"); err == nil {
		t.Fatal("invalid non-object accepted")
	}
}

func TestTransformedSwaggerReferencesAreWrittenInline(t *testing.T) {
	input := `swagger: '2.0'
info: {title: API, version: '1'}
parameters:
  Param: {name: p, in: query, type: string}
  Body: {name: item, in: body, schema: {type: string}}
responses:
  OK: {description: original}
paths:
  /x:
    post:
      parameters: [{$ref: '#/parameters/Param'}, {$ref: '#/parameters/Body'}]
      responses: {'200': {$ref: '#/responses/OK'}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	op := api.Paths["/x"].Post
	op.Parameters[0].Description = "new parameter"
	op.RequestBody.Description = "new body"
	response := op.Responses["200"]
	response.Description = "new response"
	op.Responses["200"] = response
	var out bytes.Buffer
	if err := NewWriter(WithWriterVersion("2.0"), WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	written := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["post"].(map[string]interface{})
	params := written["parameters"].([]interface{})
	if params[0].(map[string]interface{})["description"] != "new parameter" || params[1].(map[string]interface{})["description"] != "new body" || written["responses"].(map[string]interface{})["200"].(map[string]interface{})["description"] != "new response" {
		t.Fatalf("transforms lost: %s", out.String())
	}
}

func TestEditingComponentRetainsUnchangedReference(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: API, version: '1'}
paths:
  /x:
    get:
      responses: {'200': {$ref: '#/components/responses/OK'}}
components:
  responses:
    OK: {description: original}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	component := api.Components.Responses["OK"]
	component.Description = "updated component"
	api.Components.Responses["OK"] = component
	var out bytes.Buffer
	if err := NewWriter(WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	response := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})["200"].(map[string]interface{})
	if response["$ref"] != "#/components/responses/OK" {
		t.Fatalf("reference lost: %s", out.String())
	}
	componentOut := doc["components"].(map[string]interface{})["responses"].(map[string]interface{})["OK"].(map[string]interface{})
	if componentOut["description"] != "updated component" {
		t.Fatalf("component lost: %s", out.String())
	}
}

func TestTransformedNestedSchemaExtensionsAreNotLost(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: API, version: '1'}
paths:
  /x:
    post:
      parameters: [{$ref: '#/components/parameters/Param'}]
      requestBody: {$ref: '#/components/requestBodies/Body'}
      responses: {'200': {$ref: '#/components/responses/OK'}}
components:
  parameters:
    Param: {name: p, in: query, schema: {type: string, x-mark: original}}
  requestBodies:
    Body: {content: {application/json: {schema: {type: object, properties: {p: {type: string, x-mark: original}}}}}}
  responses:
    OK: {description: original, content: {application/json: {schema: {type: string, x-mark: original}}}}
`
	c, err := converter.New(converter.WithTransform(func(api *model.API) error {
		op := api.Paths["/x"].Post
		op.Parameters[0].Schema.Extensions["x-mark"] = "changed"
		op.RequestBody.Content["application/json"].Schema.Properties["p"].Extensions["x-mark"] = "changed"
		response := op.Responses["200"]
		response.Content["application/json"].Schema.Extensions["x-mark"] = "changed"
		op.Responses["200"] = response
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(NewParser())
	c.RegisterWriter(NewWriter(WithJSONOutput(true)))
	var out bytes.Buffer
	if err := c.Convert(context.Background(), strings.NewReader(input), &out, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	op := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})["post"].(map[string]interface{})
	param := op["parameters"].([]interface{})[0].(map[string]interface{})
	body := op["requestBody"].(map[string]interface{})
	response := op["responses"].(map[string]interface{})["200"].(map[string]interface{})
	paramSchema := param["schema"].(map[string]interface{})
	bodySchema := body["content"].(map[string]interface{})["application/json"].(map[string]interface{})["schema"].(map[string]interface{})
	nested := bodySchema["properties"].(map[string]interface{})["p"].(map[string]interface{})
	responseSchema := response["content"].(map[string]interface{})["application/json"].(map[string]interface{})["schema"].(map[string]interface{})
	if paramSchema["x-mark"] != "changed" || nested["x-mark"] != "changed" || responseSchema["x-mark"] != "changed" {
		t.Fatalf("extension edits lost: %s", out.String())
	}
}
