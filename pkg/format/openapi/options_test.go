package openapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/model"
)

func TestParserVersionOption(t *testing.T) {
	for _, tc := range []struct {
		name, expected, input string
		valid                 bool
	}{
		{"matching minor", "3.0", "openapi: 3.0.3", true},
		{"matching patch", "3.0.3", "openapi: 3.0.3", true},
		{"different patch", "3.0.2", "openapi: 3.0.3", false},
		{"different minor", "3.1", "openapi: 3.0.3", false},
		{"swagger", "2.0", "swagger: '2.0'", true},
		{"invalid expected", "4.0", "openapi: 4.0.0", false},
		{"missing actual", "3.0", "info: {}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewParser(WithVersion(tc.expected)).Parse(context.Background(), strings.NewReader(tc.input))
			if (err == nil) != tc.valid {
				t.Fatalf("Parse() = %v, valid = %v", err, tc.valid)
			}
			if !tc.valid && !errors.Is(err, apierrors.ErrIncompatibleVersion) {
				t.Fatalf("version mismatch error = %v", err)
			}
		})
	}
}

func TestParserStrictValidatesSpecification(t *testing.T) {
	valid := "openapi: 3.0.3\ninfo: {title: Example, version: '1'}\npaths: {}\n"
	invalid := "openapi: 3.0.3\ninfo: {title: Example}\npaths: {}\n"
	for _, tc := range []struct {
		input         string
		strict, valid bool
	}{{valid, true, true}, {invalid, true, false}, {invalid, false, true}} {
		_, err := NewParser(WithStrict(tc.strict)).Parse(context.Background(), strings.NewReader(tc.input))
		if (err == nil) != tc.valid {
			t.Fatalf("strict=%v Parse(%q) = %v", tc.strict, tc.input, err)
		}
		if !tc.valid && !errors.Is(err, apierrors.ErrValidationFailed) {
			t.Fatalf("strict validation error = %v", err)
		}
	}
}

func TestWriterRejectsDirectSchemaCycles(t *testing.T) {
	api := model.NewAPI()
	schema := &model.Schema{Type: model.TypeObject}
	schema.Properties = map[string]*model.Schema{"child": schema}
	api.Components.Schemas["Node"] = schema
	var output bytes.Buffer
	err := NewWriter().Write(context.Background(), api, &output)
	if err == nil || !strings.Contains(err.Error(), "cyclic schema") || output.Len() != 0 {
		t.Fatalf("Write() = %v, output %q", err, output.String())
	}
}

func TestStrictParserAcceptsNestedExtensions(t *testing.T) {
	input := `openapi: 3.1.0
info: {title: Example, version: '1', x-info: info}
paths:
  x-paths: paths
  /items:
    x-path: path
    get:
      x-operation: operation
      responses:
        x-responses: responses
        '200': {description: OK, x-response: response}
components:
  schemas:
    Item: {type: string, x-schema: schema}
`
	api, err := NewParser(WithStrict(true)).Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if api.PathsExtensions["x-paths"] != "paths" || api.Paths["/items"].Get.ResponsesExtensions["x-responses"] != "responses" || api.Components.Schemas["Item"].Extensions["x-schema"] != "schema" {
		t.Fatal("nested extensions were lost")
	}
}

func TestDuplicateMapExtensionsAreRejected(t *testing.T) {
	input := "openapi: 3.1.0\ninfo: {title: Example, version: '1'}\npaths:\n  x-duplicate: first\n  x-duplicate: second\n"
	if _, err := NewParser().Parse(context.Background(), strings.NewReader(input)); err == nil || !strings.Contains(err.Error(), "duplicate extension") {
		t.Fatalf("Parse() error = %v", err)
	}
}
