package apiblueprint

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestParseRequestExample(t *testing.T) {
	input := `FORMAT: 1A
# API
## Things [/things]
### Create [POST]
+ Request (application/json)

        {"name":"test"}

+ Response 201 (application/json)
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	op := api.Paths["/things"].Post
	if op == nil || op.RequestBody == nil || !strings.Contains(op.RequestBody.Content["application/json"].Example.(string), "test") {
		t.Fatalf("request lost: %#v", op)
	}
	if _, ok := op.Responses["201"]; !ok {
		t.Fatal("response lost after request")
	}
}

func TestBlueprintSchemasRoundTrip(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "T"
	thing := &model.Schema{
		Type:       model.TypeObject,
		Properties: map[string]*model.Schema{"id": {Type: model.TypeString}},
		Required:   []string{"id"},
	}
	api.Components.AddSchema("Thing", thing)
	responseThing := &model.Schema{
		Type:       model.TypeObject,
		Properties: map[string]*model.Schema{"id": {Type: model.TypeString}},
		Required:   []string{"id"},
	}
	api.Paths["/things"] = model.PathItem{
		Post: &model.Operation{
			RequestBody: &model.RequestBody{Content: map[string]model.MediaType{
				"application/json": {Schema: &model.Schema{Ref: "#/components/schemas/Thing"}},
			}},
			Responses: model.Responses{
				"200": {Content: map[string]model.MediaType{"application/json": {Schema: responseThing}}},
				"201": {Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Ref: "#/components/schemas/Thing"}}}},
			},
		},
	}
	var out bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	schema := parsed.Components.Schemas["Thing"]
	if schema == nil || schema.Properties["id"] == nil || len(schema.Required) != 1 {
		t.Fatalf("schema lost: %#v\n%s", schema, out.String())
	}
	op := parsed.Paths["/things"].Post
	if op == nil || op.RequestBody == nil || op.RequestBody.Content["application/json"].Schema == nil || op.RequestBody.Content["application/json"].Schema.Ref != "#/components/schemas/Thing" {
		t.Fatalf("request lost: %#v\n%s", op, out.String())
	}
	resp := op.Responses["200"].Content["application/json"].Schema
	if resp == nil || resp.Properties["id"] == nil || len(resp.Required) != 1 {
		t.Fatalf("response schema lost: %#v\n%s", resp, out.String())
	}
	ref := op.Responses["201"].Content["application/json"].Schema
	if ref == nil || ref.Ref != "#/components/schemas/Thing" {
		t.Fatalf("response ref lost: %#v\n%s", ref, out.String())
	}
}

func TestResponseJSONSchemaAndExample(t *testing.T) {
	minLength := 3
	api := model.NewAPI()
	api.Info.Title = "T"
	original := &model.Schema{
		Type: model.TypeObject,
		Properties: map[string]*model.Schema{
			"name":   {Type: model.TypeString, MinLength: &minLength},
			"nested": {Type: model.TypeObject, Properties: map[string]*model.Schema{"count": {Type: model.TypeInteger}}},
		},
	}
	api.Paths["/things"] = model.PathItem{
		Get: &model.Operation{Responses: model.Responses{
			"200": {Content: map[string]model.MediaType{
				"application/json": {Schema: original, Example: `{"name":"abc"}`},
			}},
		}},
	}
	var out bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	mt := parsed.Paths["/things"].Get.Responses["200"].Content["application/json"]
	if mt.Schema == nil || mt.Schema.Properties["name"].MinLength == nil || *mt.Schema.Properties["name"].MinLength != 3 || mt.Schema.Properties["nested"].Properties["count"].Type != model.TypeInteger {
		t.Fatalf("schema lost: %#v\n%s", mt.Schema, out.String())
	}
	if example, ok := mt.Example.(string); !ok || !strings.Contains(example, "abc") {
		t.Fatalf("example lost: %#v\n%s", mt.Example, out.String())
	}
}
