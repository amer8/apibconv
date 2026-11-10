package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/model"
)

func TestPublicSecurityOverridesDoNotBlockBlueprint(t *testing.T) {
	input := `openapi: 3.0.0
info: {title: Test, version: '1'}
security: [{key: []}]
paths:
  /items:
    get:
      security: []
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
`
	var output bytes.Buffer
	if err := convertLowLevel(newTestConverter(t), context.Background(), strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatAPIBlueprint); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("public API produced no output")
	}
}

func TestOpenAPISecurityCannotBeSilentlyDiscarded(t *testing.T) {
	for _, version := range []string{"2.0", "3.0.0", "3.1.0"} {
		versionField := "openapi: " + version
		if version == "2.0" {
			versionField = "swagger: '2.0'"
		}
		for _, tc := range []struct {
			name       string
			global     string
			operation  string
			components string
		}{
			{name: "global", global: "security: [{key: []}]\n"},
			{name: "operation", operation: "      security: [{oauth: [read, write]}, {key: []}]\n"},
		} {
			input := fmt.Sprintf(`%s
info: {title: Test, version: '1'}
%s`+`paths:
  /items:
    get:
%s      responses: {'200': {description: OK}}
%s`, versionField, tc.global, tc.operation, tc.components)
			for _, target := range []struct {
				name    string
				format  format.Format
				version string
			}{
				{"blueprint", format.FormatAPIBlueprint, ""},
			} {
				t.Run(version+"/"+tc.name+"/"+target.name, func(t *testing.T) {
					ctx := context.Background()
					var output bytes.Buffer
					err := convertLowLevel(newTestConverter(t), ctx, strings.NewReader(input), &output, format.FormatOpenAPI, target.format)
					if !errors.Is(err, apierrors.ErrUnsupportedSecurity) {
						t.Fatalf("Convert() error = %v, want ErrUnsupportedSecurity", err)
					}
					if output.Len() != 0 {
						t.Fatalf("failed conversion produced output: %s", output.String())
					}
				})
			}
		}
	}
}

func TestWritersRejectSecurityInAllModelLocations(t *testing.T) {
	security := []model.SecurityRequirement{{"key": {}}}
	callback := model.Callback{"{$request.body#/url}": {Trace: &model.Operation{Security: security}}}
	cases := []struct {
		name string
		api  *model.API
	}{
		{"global", &model.API{Security: security, Paths: map[string]model.PathItem{"/x": {Get: &model.Operation{}}}}},
		{"get", &model.API{Paths: map[string]model.PathItem{"/x": {Get: &model.Operation{Security: security}}}}},
		{"put", &model.API{Paths: map[string]model.PathItem{"/x": {Put: &model.Operation{Security: security}}}}},
		{"post", &model.API{Paths: map[string]model.PathItem{"/x": {Post: &model.Operation{Security: security}}}}},
		{"delete", &model.API{Paths: map[string]model.PathItem{"/x": {Delete: &model.Operation{Security: security}}}}},
		{"options", &model.API{Paths: map[string]model.PathItem{"/x": {Options: &model.Operation{Security: security}}}}},
		{"head", &model.API{Paths: map[string]model.PathItem{"/x": {Head: &model.Operation{Security: security}}}}},
		{"patch", &model.API{Paths: map[string]model.PathItem{"/x": {Patch: &model.Operation{Security: security}}}}},
		{"trace", &model.API{Paths: map[string]model.PathItem{"/x": {Trace: &model.Operation{Security: security}}}}},
		{"webhook", &model.API{Webhooks: map[string]model.PathItem{"event": {Get: &model.Operation{Security: security}}}}},
		{"callback", &model.API{Paths: map[string]model.PathItem{"/x": {Post: &model.Operation{Callbacks: map[string]model.Callback{"notify": callback}}}}}},
		{"component callback", &model.API{Components: model.Components{Callbacks: map[string]model.Callback{"notify": callback}}}},
	}
	for _, tc := range cases {
		for _, target := range []struct {
			name   string
			writer format.Writer
		}{
			{"blueprint", apiblueprint.NewWriter()},
		} {
			t.Run(tc.name+"/"+target.name, func(t *testing.T) {
				var output bytes.Buffer
				err := target.writer.Write(context.Background(), tc.api, &output)
				if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
					t.Fatalf("Write() = %v, output %q; want security error and no output", err, output.String())
				}
			})
		}
	}
}

func TestSecurityCheckHandlesCallbackCycles(t *testing.T) {
	op := &model.Operation{}
	op.Callbacks = map[string]model.Callback{"self": {"/x": {Post: op}}}
	api := model.NewAPI()
	api.Info = model.Info{Title: "Test", Version: "1"}
	api.Paths["/x"] = model.PathItem{Post: op}
	var output bytes.Buffer
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &output); err == nil || output.Len() != 0 {
		t.Fatalf("Write() must reject the unsupported callback cycle: %v", err)
	}
}

// These tests exercise the low-level parser/writer security checks separately
// from the public conversion policy, which now rejects cross-format migrations.
func convertLowLevel(c *converter.Converter, ctx context.Context, input io.Reader, output io.Writer, from, to format.Format) error {
	api, err := c.ParseToModel(ctx, input, from)
	if err != nil {
		return err
	}
	return c.WriteFromModel(ctx, api, output, to)
}
