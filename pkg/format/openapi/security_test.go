package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestSecurityMarshalingValueAndPointer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{}
	}{
		{"operation value", Operation{Security: []map[string][]string{}}},
		{"operation pointer", &Operation{Security: []map[string][]string{}}},
		{"openapi value", OpenAPI{Security: []map[string][]string{}}},
		{"openapi pointer", &OpenAPI{Security: []map[string][]string{}}},
		{"swagger value", Swagger20{Security: []map[string][]string{}}},
		{"swagger pointer", &Swagger20{Security: []map[string][]string{}}},
	} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(tc.name+"/"+encoding, func(t *testing.T) {
				var data []byte
				var err error
				if encoding == "json" {
					data, err = json.Marshal(tc.value)
				} else {
					data, err = yaml.Marshal(tc.value)
				}
				if err != nil {
					t.Fatal(err)
				}
				var doc map[string]interface{}
				if err := yaml.Unmarshal(data, &doc); err != nil {
					t.Fatal(err)
				}
				security, ok := doc["security"].([]interface{})
				if !ok || len(security) != 0 {
					t.Fatalf("security = %#v, want empty array; output: %s", doc["security"], data)
				}
			})
		}
	}
}

func TestOpenAPI3SecurityRoundTrip(t *testing.T) {
	for _, version := range []string{"3.0.0", "3.1.0"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				input := fmt.Sprintf(`
openapi: %s
info: {title: Security, version: '1'}
security:
  - Key: []
    OAuth: [read, write]
  - Bearer: []
paths:
  /items:
    get:
      responses: {'200': {description: OK}}
    post:
      security: []
      responses: {'200': {description: OK}}
    delete:
      security:
        - OAuth: [write]
        - {}
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    Key: {type: apiKey, in: header, name: X-Key, description: API key}
    Bearer: {type: http, scheme: bearer, bearerFormat: JWT}
    OpenID: {type: openIdConnect, openIdConnectUrl: 'https://example.com/openid'}
    Alias: {$ref: '#/components/securitySchemes/Key'}
    OAuth:
      type: oauth2
      flows:
        implicit: {authorizationUrl: 'https://example.com/authorize', scopes: {read: Read}}
        password: {tokenUrl: 'https://example.com/token', scopes: {write: Write}}
        clientCredentials: {tokenUrl: 'https://example.com/token', scopes: {}}
        authorizationCode: {authorizationUrl: 'https://example.com/authorize', tokenUrl: 'https://example.com/token', refreshUrl: 'https://example.com/refresh', scopes: {read: Read, write: Write}}
`, version)
				if version == "3.1.0" {
					input += `
webhooks:
  newItem:
    post:
      security: [{Key: []}]
      responses: {'200': {description: OK}}
`
				}
				api, output, roundTrip := securityRoundTrip(t, input, version, encoding)
				wantGlobal := []model.SecurityRequirement{{"Key": {}, "OAuth": {"read", "write"}}, {"Bearer": {}}}
				if !reflect.DeepEqual(api.Security, wantGlobal) {
					t.Fatalf("parsed global security = %#v, want %#v", api.Security, wantGlobal)
				}
				wantSchemes := map[string]model.SecurityScheme{
					"Key":    {Type: model.SecurityTypeAPIKey, In: "header", Name: "X-Key", Description: "API key"},
					"Bearer": {Type: model.SecurityTypeHTTP, Scheme: "bearer", BearerFormat: "JWT"},
					"OpenID": {Type: model.SecurityTypeOpenIDConnect, OpenIDConnectURL: "https://example.com/openid"},
					"Alias":  {Ref: "#/components/securitySchemes/Key"},
					"OAuth": {Type: model.SecurityTypeOAuth2, Flows: &model.OAuthFlows{
						Implicit:          &model.OAuthFlow{AuthorizationURL: "https://example.com/authorize", Scopes: map[string]string{"read": "Read"}},
						Password:          &model.OAuthFlow{TokenURL: "https://example.com/token", Scopes: map[string]string{"write": "Write"}},
						ClientCredentials: &model.OAuthFlow{TokenURL: "https://example.com/token", Scopes: map[string]string{}},
						AuthorizationCode: &model.OAuthFlow{AuthorizationURL: "https://example.com/authorize", TokenURL: "https://example.com/token", RefreshURL: "https://example.com/refresh", Scopes: map[string]string{"read": "Read", "write": "Write"}},
					}},
				}
				if !reflect.DeepEqual(api.Components.SecuritySchemes, wantSchemes) {
					t.Fatalf("parsed security schemes = %#v, want %#v", api.Components.SecuritySchemes, wantSchemes)
				}
				wantDelete := []model.SecurityRequirement{{"OAuth": {"write"}}, {}}
				if !reflect.DeepEqual(api.Paths["/items"].Delete.Security, wantDelete) {
					t.Fatalf("parsed operation security = %#v, want %#v", api.Paths["/items"].Delete.Security, wantDelete)
				}
				assertSecurityEqual(t, api, roundTrip)
				if version == "3.1.0" {
					want := []model.SecurityRequirement{{"Key": {}}}
					if got := roundTrip.Webhooks["newItem"].Post.Security; !reflect.DeepEqual(got, want) {
						t.Fatalf("webhook security = %#v, want %#v", got, want)
					}
				}
				if api.Paths["/items"].Get.Security != nil {
					t.Fatal("omitted operation security must inherit global security")
				}
				if got := api.Paths["/items"].Post.Security; got == nil || len(got) != 0 {
					t.Fatalf("explicitly empty security must disable global security: %#v", got)
				}
				var doc map[string]interface{}
				if err := yaml.Unmarshal(output, &doc); err != nil {
					t.Fatal(err)
				}
				paths := doc["paths"].(map[string]interface{})
				item := paths["/items"].(map[string]interface{})
				if _, ok := item["get"].(map[string]interface{})["security"]; ok {
					t.Fatal("inherited operation security was explicitly emitted")
				}
				if security, ok := item["post"].(map[string]interface{})["security"].([]interface{}); !ok || len(security) != 0 {
					t.Fatalf("explicit empty security was not emitted: %s", output)
				}
				if errs := specschema.Validate(format.FormatOpenAPI, output); len(errs) > 0 {
					t.Fatalf("invalid output: %v\n%s", errs, output)
				}
			})
		}
	}
}

func TestSecurityRequirementNilScopesAreWrittenAsEmptyArray(t *testing.T) {
	for _, encoding := range []string{"json", "yaml"} {
		t.Run(encoding, func(t *testing.T) {
			api := model.NewAPI()
			api.Info = model.Info{Title: "Test", Version: "1"}
			api.Security = []model.SecurityRequirement{{"key": nil}}
			api.Components.SecuritySchemes["key"] = model.SecurityScheme{Type: model.SecurityTypeAPIKey, Name: "X-Key", In: "header"}
			var out bytes.Buffer
			if err := NewWriter(WithYAML(encoding == "yaml")).Write(context.Background(), api, &out); err != nil {
				t.Fatal(err)
			}
			if errs := specschema.Validate(format.FormatOpenAPI, out.Bytes()); len(errs) > 0 {
				t.Fatalf("invalid output: %v\n%s", errs, out.String())
			}
			parsed, err := NewParser().Parse(context.Background(), bytes.NewReader(out.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if scopes := parsed.Security[0]["key"]; scopes == nil || len(scopes) != 0 {
				t.Fatalf("scope list = %#v, want empty array", scopes)
			}
		})
	}
}

func TestEmptyGlobalSecurityIsPreserved(t *testing.T) {
	for _, version := range []string{"2.0", "3.0.0", "3.1.0"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				api := model.NewAPI()
				api.Info = model.Info{Title: "Test", Version: "1"}
				api.Security = []model.SecurityRequirement{}
				var out bytes.Buffer
				if err := NewWriter(WithWriterVersion(version), WithYAML(encoding == "yaml")).Write(context.Background(), api, &out); err != nil {
					t.Fatal(err)
				}
				var doc map[string]interface{}
				if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
					t.Fatal(err)
				}
				if security, ok := doc["security"].([]interface{}); !ok || len(security) != 0 {
					t.Fatalf("global security = %#v, want empty array", doc["security"])
				}
				if errs := specschema.Validate(format.FormatOpenAPI, out.Bytes()); len(errs) > 0 {
					t.Fatalf("invalid output: %v\n%s", errs, out.String())
				}
			})
		}
	}
}

func TestSwaggerSecurityRoundTripAndUpgrade(t *testing.T) {
	input := `
swagger: '2.0'
info: {title: Security, version: '1'}
security:
  - Key: []
    Access: [read]
  - Basic: []
paths:
  /items:
    get:
      responses: {'200': {description: OK}}
    post:
      security: []
      responses: {'200': {description: OK}}
    delete:
      security: [{Access: [write]}]
      responses: {'200': {description: OK}}
securityDefinitions:
  Key: {type: apiKey, in: header, name: X-Key}
  Basic: {type: basic, description: Basic auth}
  Implicit: {type: oauth2, flow: implicit, authorizationUrl: 'https://example.com/auth', scopes: {read: Read}}
  Password: {type: oauth2, flow: password, tokenUrl: 'https://example.com/token', scopes: {}}
  Application: {type: oauth2, flow: application, tokenUrl: 'https://example.com/token', scopes: {read: Read}}
  Access: {type: oauth2, flow: accessCode, authorizationUrl: 'https://example.com/auth', tokenUrl: 'https://example.com/token', scopes: {read: Read, write: Write}}
`
	for _, version := range []string{"2.0", "3.0.0", "3.1.0"} {
		for _, encoding := range []string{"json", "yaml"} {
			t.Run(version+"/"+encoding, func(t *testing.T) {
				original, output, converted := securityRoundTrip(t, input, version, encoding)
				wantGlobal := []model.SecurityRequirement{{"Key": {}, "Access": {"read"}}, {"Basic": {}}}
				if !reflect.DeepEqual(original.Security, wantGlobal) {
					t.Fatalf("parsed global security = %#v, want %#v", original.Security, wantGlobal)
				}
				if security := original.Paths["/items"].Post.Security; security == nil || len(security) != 0 {
					t.Fatalf("parsed explicit empty security = %#v", security)
				}
				if security := original.Paths["/items"].Get.Security; security != nil {
					t.Fatalf("parsed inherited security = %#v", security)
				}
				if got := original.Components.SecuritySchemes["Basic"]; got.Type != model.SecurityTypeHTTP || got.Scheme != "basic" {
					t.Fatalf("basic scheme = %#v", got)
				}
				access := original.Components.SecuritySchemes["Access"].Flows
				if access == nil || access.AuthorizationCode == nil || access.AuthorizationCode.Scopes["write"] != "Write" {
					t.Fatalf("OAuth accessCode flow was not parsed: %#v", access)
				}
				assertSecurityEqual(t, original, converted)
				if errs := specschema.Validate(format.FormatOpenAPI, output); len(errs) > 0 {
					t.Fatalf("invalid output: %v\n%s", errs, output)
				}
				if version != "2.0" {
					_, downgrade, downgraded := securityRoundTrip(t, string(output), "2.0", encoding)
					assertSecurityEqual(t, original, downgraded)
					if errs := specschema.Validate(format.FormatOpenAPI, downgrade); len(errs) > 0 {
						t.Fatalf("invalid downgraded output: %v\n%s", errs, downgrade)
					}
				}
			})
		}
	}
}

func TestOpenAPISecurityCannotBeDowngraded(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scheme model.SecurityScheme
	}{
		{"bearer", model.SecurityScheme{Type: model.SecurityTypeHTTP, Scheme: "bearer"}},
		{"openid", model.SecurityScheme{Type: model.SecurityTypeOpenIDConnect, OpenIDConnectURL: "https://example.com"}},
		{"cookie", model.SecurityScheme{Type: model.SecurityTypeAPIKey, In: "cookie", Name: "key"}},
		{"reference", model.SecurityScheme{Ref: "#/components/securitySchemes/Other"}},
		{"multiple flows", model.SecurityScheme{Type: model.SecurityTypeOAuth2, Flows: &model.OAuthFlows{Implicit: &model.OAuthFlow{}, Password: &model.OAuthFlow{}}}},
		{"refresh url", model.SecurityScheme{Type: model.SecurityTypeOAuth2, Flows: &model.OAuthFlows{Password: &model.OAuthFlow{RefreshURL: "https://example.com"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Components.SecuritySchemes["auth"] = tc.scheme
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out)
			if err == nil || !strings.Contains(err.Error(), "security scheme \"auth\"") {
				t.Fatalf("expected a security scheme conversion error, got %v", err)
			}
			if out.Len() != 0 {
				t.Fatalf("failed conversion wrote output: %s", out.String())
			}
		})
	}
}

func TestMutualTLSCannotBeDowngradedToOpenAPI30(t *testing.T) {
	api := model.NewAPI()
	api.Components.SecuritySchemes["mtls"] = model.SecurityScheme{Type: "mutualTLS"}
	var out bytes.Buffer
	err := NewWriter(WithWriterVersion("3.0")).Write(context.Background(), api, &out)
	if err == nil || !strings.Contains(err.Error(), "mutualTLS cannot be represented in OpenAPI 3.0") {
		t.Fatalf("expected mutualTLS conversion error, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("failed conversion wrote output: %s", out.String())
	}
}

func securityRoundTrip(t *testing.T, input, version, encoding string) (original *model.API, output []byte, converted *model.API) {
	t.Helper()
	parser := NewParser()
	api, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse input: %v", err)
	}
	var out bytes.Buffer
	writer := NewWriter(WithWriterVersion(version), WithYAML(encoding == "yaml"))
	if err := writer.Write(context.Background(), api, &out); err != nil {
		t.Fatalf("write: %v", err)
	}
	converted, err = parser.Parse(context.Background(), bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("parse output: %v\n%s", err, out.String())
	}
	return api, out.Bytes(), converted
}

func assertSecurityEqual(t *testing.T, original, converted *model.API) {
	t.Helper()
	if !reflect.DeepEqual(original.Security, converted.Security) {
		t.Errorf("global security: got %#v, want %#v", converted.Security, original.Security)
	}
	if !reflect.DeepEqual(original.Components.SecuritySchemes, converted.Components.SecuritySchemes) {
		t.Errorf("security schemes: got %#v, want %#v", converted.Components.SecuritySchemes, original.Components.SecuritySchemes)
	}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		before := original.Paths["/items"]
		after := converted.Paths["/items"]
		if !reflect.DeepEqual(before.GetOperation(method).Security, after.GetOperation(method).Security) {
			t.Errorf("%s security: got %#v, want %#v", method, after.GetOperation(method).Security, before.GetOperation(method).Security)
		}
	}
}
