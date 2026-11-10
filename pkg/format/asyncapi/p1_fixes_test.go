package asyncapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestP1OneOfRejected(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels:
  topic:
    subscribe:
      message:
        oneOf:
          - {payload: {type: string}}
          - {payload: {type: integer}}
`
	_, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "oneOf") {
		t.Fatalf("expected oneOf error, got %v", err)
	}
}

func TestP1MessageContentTypeRoundTrip(t *testing.T) {
	for _, version := range []string{"2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			for _, ref := range []bool{false, true} {
				api := model.NewAPI()
				api.Info.Title = "T"
				api.Info.Version = "1"
				schema := &model.Schema{Type: model.TypeString}
				if ref {
					api.Components.AddSchema("S", schema)
					schema = &model.Schema{Ref: "#/components/schemas/S"}
				}
				api.Paths["events"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/xml": {Schema: schema}}}}}
				var out bytes.Buffer
				if err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out); err != nil {
					t.Fatal(err)
				}
				if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) != 0 {
					t.Fatalf("invalid output: %v\n%s", issues, out.String())
				}
				parsed, err := NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
				if err != nil {
					t.Fatalf("%v\n%s", err, out.String())
				}
				if _, ok := parsed.Paths["events"].Post.RequestBody.Content["application/xml"]; !ok {
					t.Fatalf("lost content type: %s", out.String())
				}
			}
		})
	}
}

func TestP1ChannelParameters(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: T, version: '1'}
channels:
  users/{id}:
    parameters:
      id: {description: User ID, schema: {type: string}}
    subscribe: {message: {payload: {type: string}}}
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = NewWriter(WithAsyncWriterVersion("2.6")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	params := parsed.Paths["users/{id}"].Parameters
	if len(params) != 1 || params[0].Schema == nil || params[0].Schema.Type != model.TypeString {
		t.Fatalf("lost parameter: %#v\n%s", params, out.String())
	}
	out.Reset()
	if err = NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out); err == nil || out.Len() != 0 {
		t.Fatalf("expected lossless rejection, err=%v out=%s", err, out.String())
	}
	item := api.Paths["users/{id}"]
	item.Parameters[0].Schema = nil
	api.Paths["users/{id}"] = item
	if err = NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) != 0 {
		t.Fatalf("invalid output: %v\n%s", issues, out.String())
	}
	parsed, err = NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if p := parsed.Paths["users/{id}"].Parameters; len(p) != 1 || p[0].Description != "User ID" {
		t.Fatalf("parameter lost: %#v", p)
	}
}

func TestP1MultiplePayloadsRejected(t *testing.T) {
	for _, version := range []string{"2.6", "3.0"} {
		api := model.NewAPI()
		api.Paths["events"] = model.PathItem{Post: &model.Operation{RequestBody: &model.RequestBody{Content: map[string]model.MediaType{"application/json": {Schema: &model.Schema{Type: model.TypeString}}, "application/xml": {Schema: &model.Schema{Type: model.TypeString}}}}}}
		var out bytes.Buffer
		err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("%s: err=%v, output=%s", version, err, out.String())
		}
	}
}
