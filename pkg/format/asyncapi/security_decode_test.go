package asyncapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/specschema"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestParseV2SecurityServerAndOperation(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
servers:
  production:
    url: example.com
    protocol: http
    security: [{key: []}]
channels:
  /items:
    publish:
      security: [{oauth: [read]}]
    subscribe:
      security: []
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
    oauth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://example.com/token
          scopes: {read: Read}
`
	for _, encoding := range []string{"yaml", "json"} {
		t.Run(encoding, func(t *testing.T) {
			api := parseSecurityFixture(t, input, encoding)
			wantServer := []model.SecurityRequirement{{"key": {}}}
			if api.Security != nil || len(api.Servers) != 1 || !reflect.DeepEqual(api.Servers[0].Security, wantServer) {
				t.Fatalf("server security = %#v, global = %#v", api.Servers, api.Security)
			}
			item := api.Paths["/items"]
			wantPost := []model.SecurityRequirement{{"oauth": {"read"}}}
			if !reflect.DeepEqual(item.Post.Security, wantPost) {
				t.Fatalf("publish security = %#v", item.Post.Security)
			}
			if item.Get.Security != nil {
				t.Fatalf("unexpected operation security: %#v", item.Get.Security)
			}
			key := api.Components.SecuritySchemes["key"]
			if key.Type != model.SecurityTypeAPIKey || key.Name != "X-Key" || key.In != "header" {
				t.Fatalf("key = %#v", key)
			}
			oauth := api.Components.SecuritySchemes["oauth"]
			if oauth.Flows == nil || oauth.Flows.ClientCredentials.Scopes["read"] != "Read" {
				t.Fatalf("oauth = %#v", oauth)
			}
		})
	}
}

func TestAsyncAPIServerAndOperationSecurityRoundTrip(t *testing.T) {
	input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
servers:
  production: {url: 'https://example.com', protocol: http, security: [{key: []}]}
channels:
  /items:
    publish: {security: [{oauth: [read]}]}
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
    oauth:
      type: oauth2
      flows:
        clientCredentials: {tokenUrl: 'https://example.com/token', scopes: {read: Read}}
`
	api := parseSecurityFixture(t, input, "yaml")
	for _, version := range []string{"2.6", "3.0"} {
		t.Run(version, func(t *testing.T) {
			var output bytes.Buffer
			if err := NewWriter(WithAsyncWriterVersion(version)).Write(context.Background(), api, &output); err != nil {
				t.Fatal(err)
			}
			if issues := specschema.Validate(format.FormatAsyncAPI, output.Bytes()); len(issues) > 0 {
				t.Fatalf("invalid output: %v\n%s", issues, output.String())
			}
			reparsed, err := NewParser().Parse(context.Background(), bytes.NewReader(output.Bytes()))
			if err != nil {
				t.Fatalf("parse output: %v\n%s", err, output.String())
			}
			if len(reparsed.Servers) != 1 || !reflect.DeepEqual(reparsed.Servers[0].Security, api.Servers[0].Security) {
				t.Fatalf("server security = %#v", reparsed.Servers)
			}
			if got, want := reparsed.Paths["/items"].Post.Security, api.Paths["/items"].Post.Security; !reflect.DeepEqual(got, want) {
				t.Fatalf("operation security = %#v, want %#v", got, want)
			}
		})
	}
}

func TestAsyncAPINativeSecuritySchemesRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeName string
		location string
	}{
		{"user password", "userPassword", ""},
		{"SASL SCRAM", "scramSha256", ""},
		{"certificate", "X509", ""},
		{"username API key", "apiKey", "user"},
		{"password API key", "apiKey", "password"},
	} {
		for _, target := range []string{"2.6", "3.0"} {
			t.Run(tc.name+"/"+target, func(t *testing.T) {
				input := `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels: {/items: {publish: {security: [{auth: []}]}}}
components:
  securitySchemes:
    auth: {type: ` + tc.typeName
				if tc.location != "" {
					input += ", in: " + tc.location
				}
				input += "}\n"
				api := parseSecurityFixture(t, input, "yaml")
				var output bytes.Buffer
				if err := NewWriter(WithAsyncWriterVersion(target)).Write(context.Background(), api, &output); err != nil {
					t.Fatal(err)
				}
				if issues := specschema.Validate(format.FormatAsyncAPI, output.Bytes()); len(issues) > 0 {
					t.Fatalf("invalid output: %v\n%s", issues, output.String())
				}
				reparsed, err := NewParser().Parse(context.Background(), bytes.NewReader(output.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				if got, want := reparsed.Components.SecuritySchemes["auth"], api.Components.SecuritySchemes["auth"]; !reflect.DeepEqual(got, want) {
					t.Fatalf("scheme = %#v, want %#v", got, want)
				}
			})
		}
	}
}

func TestParseV3SecurityInlineReferencesAndScopes(t *testing.T) {
	input := `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels:
  /items: {address: /items}
operations:
  readItems:
    action: receive
    channel: {$ref: '#/channels/~1items'}
    security:
      - $ref: '#/components/securitySchemes/key'
      - type: oauth2
        flows:
          clientCredentials: {tokenUrl: 'https://example.com/token', availableScopes: {read: Read}}
        scopes: [read]
  publicItems:
    action: send
    channel: {$ref: '#/channels/~1items'}
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
    oauth:
      type: oauth2
      flows:
        clientCredentials: {tokenUrl: 'https://example.com/token', availableScopes: {read: Read}}
`
	for _, encoding := range []string{"yaml", "json"} {
		t.Run(encoding, func(t *testing.T) {
			api := parseSecurityFixture(t, input, encoding)
			item, ok := api.Paths["/items"]
			if !ok || item.Get == nil || item.Post == nil {
				t.Fatalf("missing operations: %#v", api.Paths)
			}
			want := []model.SecurityRequirement{{"key": {}}, {"oauth": {"read"}}}
			if !reflect.DeepEqual(item.Get.Security, want) {
				t.Fatalf("security = %#v, want %#v", item.Get.Security, want)
			}
			if item.Post.Security != nil {
				t.Fatalf("public operation = %#v", item.Post.Security)
			}
		})
	}
}

func TestParseRejectsDifferentServerPoliciesEvenWithChannelBindings(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"v2", `asyncapi: 2.6.0
info: {title: Test, version: '1'}
servers:
  secure: {url: secure.example, protocol: http, security: [{key: []}]}
  public: {url: public.example, protocol: http}
channels:
  /secure: {servers: [secure], publish: {}}
  /public: {servers: [public], publish: {}}
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`},
		{"v3", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
servers:
  secure: {host: secure.example, protocol: http, security: [{$ref: '#/components/securitySchemes/key'}]}
  public: {host: public.example, protocol: http}
channels:
  secureChannel: {address: /secure, servers: [{$ref: '#/servers/secure'}]}
  publicChannel: {address: /public, servers: [{$ref: '#/servers/public'}]}
operations:
  secure: {action: send, channel: {$ref: '#/channels/secureChannel'}}
  public: {action: send, channel: {$ref: '#/channels/publicChannel'}}
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, err := NewParser().Parse(context.Background(), strings.NewReader(tc.input))
			if api != nil || !errors.Is(err, apierrors.ErrUnsupportedSecurity) {
				t.Fatalf("Parse() = %#v, %v; want unsupported server security", api, err)
			}
		})
	}
}

func TestParseRejectsUnrepresentableSecurity(t *testing.T) {
	for _, tc := range []struct{ name, input, contains string }{
		{"server alternatives differ", `asyncapi: 2.6.0
info: {title: Test, version: '1'}
servers:
  secure: {url: a, protocol: http, security: [{key: []}]}
  public: {url: b, protocol: http}
channels: {/x: {publish: {}}}
components: {securitySchemes: {key: {type: httpApiKey, name: X-Key, in: header}}}
`, "different server security"},
		{"unsupported type", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations: {get: {action: receive, channel: {$ref: '#/channels/x'}, security: [{type: custom}]}}
`, "custom"},
		{"external reference", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations: {get: {action: receive, channel: {$ref: '#/channels/x'}, security: [{$ref: 'https://example.com/security.yaml'}]}}
`, "external security"},
		{"security trait", `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels: {/x: {publish: {traits: [{$ref: '#/components/operationTraits/auth'}]}}}
`, "trait"},
		{"duplicate secured operations", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations:
  one: {action: send, channel: {$ref: '#/channels/x'}, security: [{type: http, scheme: bearer}]}
  two: {action: send, channel: {$ref: '#/channels/x'}}
`, "multiple secured"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, err := NewParser().Parse(context.Background(), strings.NewReader(tc.input))
			if api != nil || !errors.Is(err, apierrors.ErrUnsupportedSecurity) || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("Parse() = %#v, %v", api, err)
			}
		})
	}
}

func TestParseSecurityInOperationTraits(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        []model.SecurityRequirement
	}{
		{"v2 trait overrides operation", `asyncapi: 2.6.0
info: {title: Test, version: '1'}
channels:
  /x:
    publish:
      security: [{key: []}]
      traits: [{$ref: '#/components/operationTraits/auth'}]
components:
  operationTraits:
    auth: {security: [{oauth: [read]}]}
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
    oauth:
      type: oauth2
      flows:
        clientCredentials: {tokenUrl: 'https://example.com/token', scopes: {read: Read}}
`, []model.SecurityRequirement{{"oauth": {"read"}}}},
		{"v3 operation overrides trait", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations:
  send:
    action: send
    channel: {$ref: '#/channels/x'}
    security: [{$ref: '#/components/securitySchemes/key'}]
    traits: [{$ref: '#/components/operationTraits/auth'}]
components:
  operationTraits:
    auth: {security: [{type: http, scheme: bearer}]}
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`, []model.SecurityRequirement{{"key": {}}}},
		{"v3 trait supplies security", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations:
  send:
    action: send
    channel: {$ref: '#/channels/x'}
    traits: [{$ref: '#/components/operationTraits/auth'}]
components:
  operationTraits:
    auth: {security: [{$ref: '#/components/securitySchemes/key'}]}
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`, []model.SecurityRequirement{{"key": {}}}},
		{"v3 explicit security ignores unresolved trait", `asyncapi: 3.0.0
info: {title: Test, version: '1'}
channels: {x: {address: /x}}
operations:
  send:
    action: send
    channel: {$ref: '#/channels/x'}
    security: [{$ref: '#/components/securitySchemes/key'}]
    traits: [{$ref: 'https://example.com/trait.yaml'}]
components:
  securitySchemes:
    key: {type: httpApiKey, name: X-Key, in: header}
`, []model.SecurityRequirement{{"key": {}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := parseSecurityFixture(t, tc.input, "yaml")
			if got := api.Paths["/x"].Post.Security; !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("security = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func parseSecurityFixture(t *testing.T, input, encoding string) *model.API {
	t.Helper()
	data := []byte(input)
	if encoding == "json" {
		var document interface{}
		if err := yaml.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		var err error
		data, err = json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
	}
	api, err := NewParser().Parse(context.Background(), bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	return api
}
