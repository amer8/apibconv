package asyncapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestOperationParametersArePreservedOrRejected(t *testing.T) {
	for _, version := range []string{"2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			t.Run("query rejected", func(t *testing.T) {
				api := model.NewAPI()
				api.Paths["/orders"] = model.PathItem{Get: &model.Operation{Parameters: []model.Parameter{{Name: "tenant", In: model.ParameterInQuery, Required: true}}}}
				var out bytes.Buffer
				err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out)
				if err == nil || !strings.Contains(err.Error(), "tenant") || out.Len() != 0 {
					t.Fatalf("expected error for query parameter and no output, got %v, %q", err, out.String())
				}
			})
			t.Run("path promoted", func(t *testing.T) {
				api := model.NewAPI()
				param := model.Parameter{Name: "id", In: model.ParameterInPath, Required: true, Description: "Order id"}
				if version == "2.6" {
					param.Schema = &model.Schema{Type: model.TypeString}
				}
				api.Paths["/orders/{id}"] = model.PathItem{Get: &model.Operation{Parameters: []model.Parameter{param}}}
				var out bytes.Buffer
				if err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &out); err != nil {
					t.Fatal(err)
				}
				parsed, err := NewParser().Parse(context.Background(), &out)
				if err != nil {
					t.Fatal(err)
				}
				params := parsed.Paths["/orders/{id}"].Parameters
				if len(params) != 1 || params[0].Name != "id" || params[0].Description != "Order id" || !params[0].Required {
					t.Fatalf("lost channel parameter: %#v", params)
				}
				if version == "2.6" && (params[0].Schema == nil || params[0].Schema.Type != model.TypeString) {
					t.Fatalf("lost schema: %#v", params[0].Schema)
				}
				if len(api.Paths["/orders/{id}"].Parameters) != 0 {
					t.Fatal("writer mutated input model")
				}
			})
		})
	}
}

func TestOperationParameterConflictRejected(t *testing.T) {
	api := model.NewAPI()
	api.Paths["/orders/{id}"] = model.PathItem{
		Get:  &model.Operation{Parameters: []model.Parameter{{Name: "id", In: model.ParameterInPath, Required: true, Description: "first"}}},
		Post: &model.Operation{Parameters: []model.Parameter{{Name: "id", In: model.ParameterInPath, Required: true, Description: "second"}}},
	}
	var out bytes.Buffer
	err := NewWriter(WithAsyncWriterVersion("2.6")).Write(context.Background(), api, &out)
	if err == nil || !strings.Contains(err.Error(), "conflicting") || out.Len() != 0 {
		t.Fatalf("expected conflict error with no output; got %v, %q", err, out.String())
	}
}

func TestV3OperationPathParameterSchemaRejected(t *testing.T) {
	api := model.NewAPI()
	api.Paths["/orders/{id}"] = model.PathItem{Get: &model.Operation{Parameters: []model.Parameter{{Name: "id", In: model.ParameterInPath, Required: true, Schema: &model.Schema{Type: model.TypeString}}}}}
	var out bytes.Buffer
	err := NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &out)
	if err == nil || !strings.Contains(err.Error(), "schema") || out.Len() != 0 {
		t.Fatalf("expected schema error with no output; got %v, %q", err, out.String())
	}
}
