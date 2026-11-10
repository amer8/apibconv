package apiblueprint

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestParameterScopesRoundTrip(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Users"
	api.Paths["/users/{id}{?filter}"] = model.PathItem{
		Parameters: []model.Parameter{{Name: "id", In: model.ParameterInPath, Required: true, Description: "User ID", Schema: &model.Schema{Type: model.TypeInteger, Format: "int64"}}},
		Get:        &model.Operation{Parameters: []model.Parameter{{Name: "filter", In: model.ParameterInQuery, Required: false, Description: "Search filter", Schema: &model.Schema{Type: model.TypeString}}}, Responses: model.Responses{"200": {Description: "OK"}}},
	}
	var buf bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &buf); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	item := parsed.Paths["/users/{id}{?filter}"]
	if !reflect.DeepEqual(item.Parameters, api.Paths["/users/{id}{?filter}"].Parameters) {
		t.Fatalf("resource parameters got %#v", item.Parameters)
	}
	if item.Get == nil || !reflect.DeepEqual(item.Get.Parameters, api.Paths["/users/{id}{?filter}"].Get.Parameters) {
		t.Fatalf("action parameters got %#v", item.Get)
	}
}

func TestParameterSectionsIgnoreNestedValuesAndStopAtNextSection(t *testing.T) {
	input := `FORMAT: 1A
# Users
## User [/users/{id}{?status}]
+ Parameters
    + id: 42 (integer, required) - user ID
### Read [GET]
+ Parameters
    + status: active (string, optional) - current state
        + Members
            + active
            + inactive
+ Response 200 (application/json)
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/users/{id}{?status}"]
	if len(item.Parameters) != 1 || item.Parameters[0].Name != "id" || !item.Parameters[0].Required || item.Parameters[0].In != model.ParameterInPath {
		t.Fatalf("resource params: %#v", item.Parameters)
	}
	if item.Get == nil || len(item.Get.Parameters) != 1 || item.Get.Parameters[0].Name != "status" || item.Get.Parameters[0].In != model.ParameterInQuery || item.Get.Parameters[0].Required {
		t.Fatalf("action: %#v", item.Get)
	}
	if _, ok := item.Get.Responses["200"]; !ok {
		t.Fatal("response after parameter section not parsed")
	}
}

func TestNestedMSONRoundTrip(t *testing.T) {
	nested := &model.Schema{Type: model.TypeObject, Properties: map[string]*model.Schema{
		"child": {Type: model.TypeObject, Properties: map[string]*model.Schema{
			"id":       {Type: model.TypeInteger},
			"settings": {Type: model.TypeObject, Properties: map[string]*model.Schema{"enabled": {Type: model.TypeBoolean}}, Required: []string{"enabled"}},
		}, Required: []string{"id"}},
		"name":   {Type: model.TypeString},
		"users":  {Type: model.TypeArray, Items: &model.Schema{Type: model.TypeObject, Properties: map[string]*model.Schema{"id": {Type: model.TypeInteger}}, Required: []string{"id"}}},
		"labels": {Type: model.TypeArray, Items: &model.Schema{Type: model.TypeString}},
	}, Required: []string{"child"}}
	api := model.NewAPI()
	api.Info.Title = "Nested"
	api.Components.Schemas["Nested"] = nested
	api.Paths["/nested"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: nested}}}, Responses: model.Responses{"200": {Description: "OK"}}}}
	var buf bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &buf); err != nil {
		t.Fatal(err)
	}
	doc := buf.String()
	parsed, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	gotSchemas := map[string]*model.Schema{"component": parsed.Components.Schemas["Nested"]}
	post := parsed.Paths["/nested"].Post
	if post == nil || post.RequestBody == nil {
		t.Fatalf("request missing; document:\n%s", doc)
	}
	gotSchemas["request"] = post.RequestBody.Content["application/json"].Schema
	for name, got := range gotSchemas {
		if !reflect.DeepEqual(got, nested) {
			t.Errorf("%s schema mismatch:\n got: %#v\nwant: %#v\ndocument:\n%s", name, got, nested, doc)
		}
	}
}

func TestUnsupportedParameterLocationsFailBeforeOutput(t *testing.T) {
	for _, location := range []model.ParameterLocation{model.ParameterInHeader, model.ParameterInCookie} {
		for _, onResource := range []bool{true, false} {
			t.Run(string(location)+fmt.Sprint(onResource), func(t *testing.T) {
				api := model.NewAPI()
				item := model.PathItem{Get: &model.Operation{}}
				param := model.Parameter{Name: "session", In: location}
				if onResource {
					item.Parameters = []model.Parameter{param}
				} else {
					item.Get.Parameters = []model.Parameter{param}
				}
				api.Paths["/users"] = item
				var out bytes.Buffer
				err := NewWriter().Write(context.Background(), api, &out)
				if err == nil || out.Len() != 0 {
					t.Fatalf("expected error without output, got %v, %q", err, out.String())
				}
			})
		}
	}
}

func TestRootArrayMSONRoundTrip(t *testing.T) {
	array := &model.Schema{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeObject, Properties: map[string]*model.Schema{"id": {Type: model.TypeInteger}}, Required: []string{"id"}}}
	api := model.NewAPI()
	api.Components.Schemas["Users"] = array
	api.Paths["/users"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: array}}}}}
	var out bytes.Buffer
	if err := NewWriter().Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), strings.NewReader(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*model.Schema{"component": parsed.Components.Schemas["Users"], "request": parsed.Paths["/users"].Post.RequestBody.Content["application/json"].Schema}
	for name, schema := range got {
		if !reflect.DeepEqual(schema, array) {
			t.Errorf("%s = %#v; want %#v\n%s", name, schema, array, out.String())
		}
	}
}

func TestNestedMSONResponseArray(t *testing.T) {
	input := `FORMAT: 1A
# Users
## Users [/users]
### List [GET]
+ Response 200 (application/json)
    + Attributes (array[object])
            + profile (object, required)
                + id (integer, required)
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	schema := api.Paths["/users"].Get.Responses["200"].Content["application/json"].Schema
	if schema == nil || schema.Type != model.TypeArray || schema.Items == nil || schema.Items.Type != model.TypeObject {
		t.Fatalf("array items missing: %#v", schema)
	}
	profile := schema.Items.Properties["profile"]
	if profile == nil || profile.Type != model.TypeObject || profile.Properties["id"] == nil || profile.Properties["id"].Type != model.TypeInteger || !reflect.DeepEqual(profile.Required, []string{"id"}) || !reflect.DeepEqual(schema.Items.Required, []string{"profile"}) {
		t.Fatalf("nested structure lost: %#v", schema.Items)
	}
}
