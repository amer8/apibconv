package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/converter"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/model"
)

func TestOpenAPISecurityRoundTripThroughAsyncAPI(t *testing.T) {
	input := `openapi: 3.0.0
info: {title: Test, version: '1'}
security: [{key: []}, {oauth: [read]}]
paths:
  /items:
    get:
      operationId: listItems
      responses: {'200': {description: OK}}
    post:
      operationId: createItem
      security: []
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
    oauth:
      type: oauth2
      flows:
        clientCredentials: {tokenUrl: 'https://example.com/token', scopes: {read: Read}}
`
	for _, version := range []string{"2.6", "3.0"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				conv := newTestConverter(t)
				ctx := converter.WithEncoding(converter.WithAsyncAPIVersion(context.Background(), version), encoding)
				var intermediate bytes.Buffer
				if err := conv.Convert(ctx, strings.NewReader(input), &intermediate, format.FormatOpenAPI, format.FormatAsyncAPI); err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				if err := conv.Convert(ctx, bytes.NewReader(intermediate.Bytes()), &output, format.FormatAsyncAPI, format.FormatOpenAPI); err != nil {
					t.Fatalf("convert back: %v\n%s", err, intermediate.String())
				}
				api, err := conv.ParseToModel(ctx, bytes.NewReader(output.Bytes()), format.FormatOpenAPI)
				if err != nil {
					t.Fatal(err)
				}
				item := api.Paths["/items"]
				want := []model.SecurityRequirement{{"key": {}}, {"oauth": {"read"}}}
				if item.Get == nil || !reflect.DeepEqual(item.Get.Security, want) {
					t.Fatalf("GET security = %#v, want %#v\n%s", item.Get, want, output.String())
				}
				if item.Post == nil || len(item.Post.Security) != 0 {
					t.Fatalf("public POST security = %#v\n%s", item.Post, output.String())
				}
				if api.Components.SecuritySchemes["key"].Name != "X-Key" || api.Components.SecuritySchemes["oauth"].Flows.ClientCredentials.Scopes["read"] != "Read" {
					t.Fatalf("security schemes = %#v", api.Components.SecuritySchemes)
				}
			})
		}
	}
}

func TestAsyncAPIServerSecurityCannotBeDiscarded(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
servers:
  production: {url: 'https://example.com', protocol: http, security: [{key: []}]}
channels:
  /items: {publish: {}}
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`
	for _, target := range []format.Format{format.FormatOpenAPI, format.FormatAPIBlueprint} {
		t.Run(string(target), func(t *testing.T) {
			var output bytes.Buffer
			err := newTestConverter(t).Convert(context.Background(), strings.NewReader(input), &output, format.FormatAsyncAPI, target)
			if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
				t.Fatalf("Convert() = %v, output %q; want unsupported server security and no output", err, output.String())
			}
		})
	}
}

func TestAsyncAPINativeSecurityCannotBeDiscardedByOpenAPI(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels: {/items: {publish: {security: [{auth: []}]}}}
components:
  securitySchemes:
    auth: {type: scramSha256}
`
	var output bytes.Buffer
	err := newTestConverter(t).Convert(context.Background(), strings.NewReader(input), &output, format.FormatAsyncAPI, format.FormatOpenAPI)
	if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
		t.Fatalf("Convert() = %v, output %q; want unsupported scheme and no output", err, output.String())
	}
}

func TestOpenAPIToAsyncAPISecurity(t *testing.T) {
	input := `openapi: 3.0.0
info: {title: Test, version: '1'}
security: [{key: []}]
paths:
  /items:
    get:
      operationId: listItems
      responses: {'200': {description: OK}}
    post:
      operationId: createItem
      security: []
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
`
	for _, version := range []string{"2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			ctx := converter.WithEncoding(converter.WithAsyncAPIVersion(context.Background(), version), "json")
			var output bytes.Buffer
			if err := newTestConverter(t).Convert(ctx, strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatAsyncAPI); err != nil {
				t.Fatal(err)
			}
			if issues := specschema.Validate(format.FormatAsyncAPI, output.Bytes()); len(issues) > 0 {
				t.Fatalf("invalid AsyncAPI output: %v\n%s", issues, output.String())
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(output.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			if version == "2.6" {
				channel := doc["channels"].(map[string]interface{})["/items"].(map[string]interface{})
				if _, ok := channel["publish"].(map[string]interface{})["security"]; !ok {
					t.Fatal("inherited GET security missing")
				}
				if _, ok := channel["subscribe"].(map[string]interface{})["security"]; ok {
					t.Fatal("public POST acquired security")
				}
			} else {
				ops := doc["operations"].(map[string]interface{})
				if _, ok := ops["listItems"].(map[string]interface{})["security"]; !ok {
					t.Fatal("inherited GET security missing")
				}
				if _, ok := ops["createItem"].(map[string]interface{})["security"]; ok {
					t.Fatal("public POST acquired security")
				}
			}
		})
	}
}

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
	if err := newTestConverter(t).Convert(context.Background(), strings.NewReader(input), &output, format.FormatOpenAPI, format.FormatAPIBlueprint); err != nil {
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
				{"asyncapi2", format.FormatAsyncAPI, "2.6"},
				{"asyncapi3", format.FormatAsyncAPI, "3.0"},
			} {
				t.Run(version+"/"+tc.name+"/"+target.name, func(t *testing.T) {
					ctx := converter.WithAsyncAPIVersion(context.Background(), target.version)
					var output bytes.Buffer
					err := newTestConverter(t).Convert(ctx, strings.NewReader(input), &output, format.FormatOpenAPI, target.format)
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
			{"asyncapi2", asyncapi.NewWriter(asyncapi.WithAsyncWriterVersion("2.6"))},
			{"asyncapi3", asyncapi.NewWriter(asyncapi.WithAsyncWriterVersion("3.0"))},
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
	if err := apiblueprint.NewWriter().Write(context.Background(), api, &output); err != nil {
		t.Fatalf("Write() with unsecured callback cycle failed: %v", err)
	}
}
