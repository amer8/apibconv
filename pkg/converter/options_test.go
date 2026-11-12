package converter

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestStrictAndWarningHandler(t *testing.T) {
	for _, strict := range []bool{false, true} {
		var warnings []string
		conv, err := New(WithStrict(strict), WithWarningHandler(func(warning string) {
			warnings = append(warnings, warning)
		}))
		if err != nil {
			t.Fatal(err)
		}
		conv.validator.AddRule(warningRule{})
		api := model.NewAPI()
		api.Info.Title = "Example"
		api.Info.Version = "1"
		conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
		conv.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
		var output bytes.Buffer
		err = conv.Convert(context.Background(), strings.NewReader(""), &output, format.FormatOpenAPI, format.FormatOpenAPI)
		if strict {
			if !errors.Is(err, apierrors.ErrValidationFailed) || output.Len() != 0 || len(warnings) != 1 {
				t.Fatalf("strict conversion: output=%q warnings=%v err=%v", output.String(), warnings, err)
			}
		} else if err != nil || output.String() != "mock output" || len(warnings) != 2 {
			t.Fatalf("conversion: output=%q warnings=%v err=%v", output.String(), warnings, err)
		}
		if !strings.Contains(warnings[0], "warning only") {
			t.Fatalf("warning = %q", warnings[0])
		}
	}
}

func TestStrictReportsMissingOpenAPIVersion(t *testing.T) {
	conv, err := New(WithStrict(true))
	if err != nil {
		t.Fatal(err)
	}
	api := model.NewAPI()
	api.Info.Title = "Example"
	conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	conv.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	err = conv.Convert(context.Background(), strings.NewReader(""), &output, format.FormatOpenAPI, format.FormatOpenAPI)
	if !errors.Is(err, apierrors.ErrValidationFailed) || !strings.Contains(err.Error(), "API version is empty") || output.Len() != 0 {
		t.Fatalf("strict conversion: output=%q err=%v", output.String(), err)
	}
}

func TestParseWithoutExtensionsDoesNotMutateParserModel(t *testing.T) {
	api := model.NewAPI()
	api.Version = "3.1.0"
	api.Extensions["x-root"] = true
	op := &model.Operation{Extensions: map[string]interface{}{"x-op": true}}
	api.Paths["/items"] = model.PathItem{Get: op}
	conv, err := New(WithExtensions(false))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	result, err := conv.ParseToModel(context.Background(), strings.NewReader(""), format.FormatOpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Extensions) != 0 || len(result.Paths["/items"].Get.Extensions) != 0 {
		t.Fatal("nested extensions were not removed")
	}
	if api.Extensions["x-root"] != true || op.Extensions["x-op"] != true {
		t.Fatal("parser's model was modified")
	}
}

func TestConversionValidationChecksNestedModel(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Example"
	api.Info.Version = "1"
	api.Paths["/items"] = model.PathItem{Post: &model.Operation{
		RequestBody: &model.RequestBody{Content: map[string]model.MediaType{
			"application/json": {Schema: &model.Schema{Type: model.TypeObject, Properties: map[string]*model.Schema{
				"id": {Type: "invalid"},
			}}},
		}},
	}}
	conv, err := New(WithValidation(true, false))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	conv.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	err = conv.Convert(context.Background(), strings.NewReader(""), &output, format.FormatOpenAPI, format.FormatOpenAPI)
	if !errors.Is(err, apierrors.ErrValidationFailed) || !strings.Contains(err.Error(), "invalid schema type") || output.Len() != 0 {
		t.Fatalf("conversion = %q, %v", output.String(), err)
	}
}

func TestConversionValidationHandlesRecursiveModel(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Example"
	api.Info.Version = "1"
	schema := &model.Schema{Type: model.TypeObject}
	schema.Properties = map[string]*model.Schema{"child": schema}
	api.Components.Schemas["Node"] = schema
	conv, err := New(WithValidation(true, false))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	conv.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	if err := conv.Convert(context.Background(), strings.NewReader(""), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
}

func TestConversionValidationResolvesNestedSchemaPointers(t *testing.T) {
	api := model.NewAPI()
	api.Info.Title = "Example"
	api.Info.Version = "1"
	api.Components.Schemas["a/b"] = &model.Schema{Type: model.TypeObject, Properties: map[string]*model.Schema{"id": {Type: model.TypeString}}}
	api.Components.Schemas["alias"] = &model.Schema{Ref: "#/components/schemas/a~1b/properties/id"}
	conv, err := New(WithValidation(true, false))
	if err != nil {
		t.Fatal(err)
	}
	conv.RegisterParser(&mockParser{fmt: format.FormatOpenAPI, api: api})
	conv.RegisterWriter(&mockWriter{fmt: format.FormatOpenAPI})
	var output bytes.Buffer
	if err := conv.Convert(context.Background(), strings.NewReader(""), &output, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		t.Fatal(err)
	}
}
