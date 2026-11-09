package asyncapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/converter"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestSecurityConversion(t *testing.T) {
	api := model.NewAPI()
	api.Info = model.Info{Title: "Test", Version: "1"}
	api.Security = []model.SecurityRequirement{{"key": {}}, {"bearer": {}}}
	api.Components.SecuritySchemes["key"] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}
	api.Components.SecuritySchemes["bearer"] = model.SecurityScheme{Type: model.SecurityTypeHTTP, Scheme: "bearer", BearerFormat: "JWT"}
	api.Components.SecuritySchemes["oauth"] = model.SecurityScheme{Type: model.SecurityTypeOAuth2, Flows: &model.OAuthFlows{ClientCredentials: &model.OAuthFlow{TokenURL: "https://example.com/token", Scopes: map[string]string{"read": "Read"}}}}
	api.Components.SecuritySchemes["openid"] = model.SecurityScheme{Type: model.SecurityTypeOpenIDConnect, OpenIDConnectURL: "https://example.com/.well-known/openid-configuration"}
	api.Paths["/events"] = model.PathItem{Name: "events", Get: &model.Operation{OperationID: "readEvents"}, Post: &model.Operation{OperationID: "publicEvents", Security: []model.SecurityRequirement{}}}
	api.Paths["/scoped"] = model.PathItem{Name: "scoped", Post: &model.Operation{OperationID: "writeEvents", Security: []model.SecurityRequirement{{"oauth": {"read"}}}}}
	api.Paths["/identity"] = model.PathItem{Name: "identity", Get: &model.Operation{OperationID: "identity", Security: []model.SecurityRequirement{{"openid": {"profile"}}}}}
	for _, version := range []string{"2.6", "3.0"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				ctx := converter.WithEncoding(converter.WithProtocol(context.Background(), "http"), encoding)
				var out bytes.Buffer
				if err := NewWriter(WithAsyncWriterVersion(version)).Write(ctx, api, &out); err != nil {
					t.Fatal(err)
				}
				if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) > 0 {
					t.Fatalf("invalid AsyncAPI: %v\n%s", issues, out.String())
				}
				// JSON output makes the assertions independent of YAML map types.
				ctx = converter.WithEncoding(ctx, "json")
				out.Reset()
				if err := NewWriter(WithAsyncWriterVersion(version)).Write(ctx, api, &out); err != nil {
					t.Fatal(err)
				}
				var doc map[string]interface{}
				if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
					t.Fatal(err)
				}
				components := doc["components"].(map[string]interface{})
				schemes := components["securitySchemes"].(map[string]interface{})
				if schemes["key"].(map[string]interface{})["type"] != "httpApiKey" || schemes["bearer"].(map[string]interface{})["scheme"] != "bearer" {
					t.Fatalf("schemes = %#v", schemes)
				}
				if schemes["openid"].(map[string]interface{})["openIdConnectUrl"] != "https://example.com/.well-known/openid-configuration" {
					t.Fatalf("OpenID scheme = %#v", schemes["openid"])
				}
				oauth := schemes["oauth"].(map[string]interface{})
				flow := oauth["flows"].(map[string]interface{})["clientCredentials"].(map[string]interface{})
				scopeField := "scopes"
				if version == "3.0" {
					scopeField = "availableScopes"
				}
				if flow[scopeField].(map[string]interface{})["read"] != "Read" {
					t.Fatalf("OAuth flow = %#v", flow)
				}
				if version == "2.6" {
					channels := doc["channels"].(map[string]interface{})
					events := channels["/events"].(map[string]interface{})
					get := events["publish"].(map[string]interface{})
					if len(get["security"].([]interface{})) != 2 {
						t.Fatalf("inherited security = %#v", get["security"])
					}
					if _, ok := events["subscribe"].(map[string]interface{})["security"]; ok {
						t.Fatal("public operation is secured")
					}
					scoped := channels["/scoped"].(map[string]interface{})["subscribe"].(map[string]interface{})["security"].([]interface{})[0].(map[string]interface{})
					if scoped["oauth"].([]interface{})[0] != "read" {
						t.Fatalf("scoped security = %#v", scoped)
					}
					identity := channels["/identity"].(map[string]interface{})["publish"].(map[string]interface{})["security"].([]interface{})[0].(map[string]interface{})
					if identity["openid"].([]interface{})[0] != "profile" {
						t.Fatalf("OpenID security = %#v", identity)
					}
				} else {
					operations := doc["operations"].(map[string]interface{})
					if len(operations["readEvents"].(map[string]interface{})["security"].([]interface{})) != 2 {
						t.Fatal("inherited security missing")
					}
					if _, ok := operations["publicEvents"].(map[string]interface{})["security"]; ok {
						t.Fatal("public operation is secured")
					}
					scoped := operations["writeEvents"].(map[string]interface{})["security"].([]interface{})[0].(map[string]interface{})
					if scoped["type"] != "oauth2" || scoped["scopes"].([]interface{})[0] != "read" {
						t.Fatalf("scoped security = %#v", scoped)
					}
					identity := operations["identity"].(map[string]interface{})["security"].([]interface{})[0].(map[string]interface{})
					if identity["type"] != "openIdConnect" || identity["scopes"].([]interface{})[0] != "profile" {
						t.Fatalf("OpenID security = %#v", identity)
					}
				}
			})
		}
	}
}

func TestSecurityUnsupportedCases(t *testing.T) {
	for _, tc := range []struct {
		name, version, protocol string
		req                     []model.SecurityRequirement
		scheme                  model.SecurityScheme
	}{
		{"v3 AND", "3.0", "http", []model.SecurityRequirement{{"key": {}, "other": {}}}, model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}},
		{"v3 optional", "3.0", "http", []model.SecurityRequirement{{}, {"key": {}}}, model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}},
		{"http over kafka", "2.6", "kafka", []model.SecurityRequirement{{"key": {}}}, model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}},
		{"unresolved", "3.0", "http", []model.SecurityRequirement{{"key": {}}}, model.SecurityScheme{Ref: "https://example.com/auth"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Components.SecuritySchemes["key"] = tc.scheme
			api.Paths["/x"] = model.PathItem{Get: &model.Operation{Security: tc.req}}
			var out bytes.Buffer
			err := NewWriter(WithAsyncWriterVersion(tc.version)).Write(converter.WithProtocol(context.Background(), tc.protocol), api, &out)
			if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || out.Len() != 0 {
				t.Fatalf("Write() = %v, output %q", err, out.String())
			}
		})
	}
}

func TestLocalSecuritySchemeReference(t *testing.T) {
	for _, version := range []string{"2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			api := model.NewAPI()
			api.Info = model.Info{Title: "Test", Version: "1"}
			api.Components.SecuritySchemes["original"] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}
			api.Components.SecuritySchemes["alias"] = model.SecurityScheme{Ref: "#/components/securitySchemes/original"}
			api.Paths["/x"] = model.PathItem{Get: &model.Operation{Security: []model.SecurityRequirement{{"alias": {}}}}}
			var out bytes.Buffer
			if err := NewWriter(WithAsyncWriterVersion(version), WithJSONOutput(true)).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) > 0 {
				t.Fatalf("invalid output: %v\n%s", issues, out.String())
			}
			var doc map[string]interface{}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			schemes := doc["components"].(map[string]interface{})["securitySchemes"].(map[string]interface{})
			if schemes["alias"].(map[string]interface{})["name"] != "X-Key" {
				t.Fatalf("resolved schemes = %#v", schemes)
			}
		})
	}
}

func TestCyclicSecuritySchemeReference(t *testing.T) {
	api := model.NewAPI()
	api.Components.SecuritySchemes["a"] = model.SecurityScheme{Ref: "#/components/securitySchemes/b"}
	api.Components.SecuritySchemes["b"] = model.SecurityScheme{Ref: "#/components/securitySchemes/a"}
	api.Paths["/x"] = model.PathItem{Get: &model.Operation{Security: []model.SecurityRequirement{{"a": {}}}}}
	var output bytes.Buffer
	err := NewWriter(WithAsyncWriterVersion("3.0")).Write(context.Background(), api, &output)
	if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
		t.Fatalf("Write() = %v, output %q; want security error and no output", err, output.String())
	}
}

func TestSecuredOperationCollisionsFailBeforeWriting(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version string
		paths   map[string]model.PathItem
	}{
		{"v2 channel", "2.6", map[string]model.PathItem{
			"/x#one": {Get: &model.Operation{}}, "/x#two": {Get: &model.Operation{}},
		}},
		{"v3 channel", "3.0", map[string]model.PathItem{
			"/x": {Name: "same", Get: &model.Operation{}}, "/y": {Name: "same", Post: &model.Operation{}},
		}},
		{"v3 operation", "3.0", map[string]model.PathItem{
			"/x": {Get: &model.Operation{OperationID: "same"}}, "/y": {Post: &model.Operation{OperationID: "same"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Security = []model.SecurityRequirement{{"key": {}}}
			api.Components.SecuritySchemes["key"] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}
			api.Paths = tc.paths
			var output bytes.Buffer
			err := NewWriter(WithAsyncWriterVersion(tc.version)).Write(context.Background(), api, &output)
			if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
				t.Fatalf("Write() = %v, output %q; want security error and no output", err, output.String())
			}
		})
	}
}

func TestSecurityRejectsIncompatibleDetectedServerProtocol(t *testing.T) {
	api := model.NewAPI()
	api.Servers = []model.Server{{URL: "kafka://broker.example"}}
	api.Components.SecuritySchemes["key"] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}
	api.Paths["/x"] = model.PathItem{Get: &model.Operation{Security: []model.SecurityRequirement{{"key": {}}}}}
	var output bytes.Buffer
	err := NewWriter(WithAsyncWriterVersion("3.0")).Write(converter.WithProtocol(context.Background(), "auto"), api, &output)
	if !errors.Is(err, apierrors.ErrUnsupportedSecurity) || output.Len() != 0 {
		t.Fatalf("Write() = %v, output %q; want security error and no output", err, output.String())
	}
}

func TestAsyncAPITwoSupportsCombinedSecurity(t *testing.T) {
	api := model.NewAPI()
	api.Info = model.Info{Title: "Test", Version: "1"}
	for _, name := range []string{"first", "second"} {
		api.Components.SecuritySchemes[name] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: name, In: "header"}
	}
	api.Paths["/x"] = model.PathItem{Get: &model.Operation{Security: []model.SecurityRequirement{{"first": {}, "second": {}}}}}
	var out bytes.Buffer
	if err := NewWriter(WithAsyncWriterVersion("2.6"), WithJSONOutput(true)).Write(converter.WithProtocol(context.Background(), "http"), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	channels := doc["channels"].(map[string]interface{})
	op := channels["/x"].(map[string]interface{})["publish"].(map[string]interface{})
	requirements := op["security"].([]interface{})
	combined := requirements[0].(map[string]interface{})
	if len(requirements) != 1 || len(combined) != 2 || combined["first"] == nil || combined["second"] == nil {
		t.Fatalf("combined requirements = %#v", requirements)
	}
	if issues := specschema.Validate(format.FormatAsyncAPI, out.Bytes()); len(issues) > 0 {
		t.Fatalf("invalid output: %v", issues)
	}
}
