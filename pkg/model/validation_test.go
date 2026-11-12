package model

import (
	"strings"
	"testing"
)

func TestAPIValidate(t *testing.T) {
	valid := NewAPI()
	valid.Info.Title = "Example"
	valid.Paths["/items"] = PathItem{Get: &Operation{}}
	valid.Components.Schemas["Item"] = &Schema{Type: TypeObject, Properties: map[string]*Schema{"id": {Type: TypeString}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid API: %v", err)
	}
	for _, tc := range []struct {
		name string
		api  *API
		want string
	}{
		{"nil", nil, "nil"},
		{"missing title", NewAPI(), "title"},
		{"invalid path", &API{Info: Info{Title: "Example"}, Paths: map[string]PathItem{"items": {}}}, "must start"},
		{"nil schema", &API{Info: Info{Title: "Example"}, Components: Components{Schemas: map[string]*Schema{"Bad": nil}}}, "schema is nil"},
		{"invalid nested type", &API{Info: Info{Title: "Example"}, Components: Components{Schemas: map[string]*Schema{"Bad": {Properties: map[string]*Schema{"field": {Type: "nonsense"}}}}}}, "invalid schema type"},
		{"invalid security", &API{Info: Info{Title: "Example"}, Components: Components{SecuritySchemes: map[string]SecurityScheme{"Key": {Type: SecurityTypeAPIKey}}}}, "security scheme"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.api.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v; want error containing %q", err, tc.want)
			}
		})
	}
}

func TestAPIValidateHandlesRecursiveSchemas(t *testing.T) {
	api := NewAPI()
	api.Info.Title = "Example"
	schema := &Schema{Type: TypeObject}
	schema.Properties = map[string]*Schema{"child": schema}
	api.Components.Schemas["Node"] = schema
	if err := api.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSecuritySchemeValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scheme *SecurityScheme
		valid  bool
	}{
		{"nil", nil, false},
		{"reference", &SecurityScheme{Ref: "#/components/securitySchemes/Auth"}, true},
		{"api key", &SecurityScheme{Type: SecurityTypeAPIKey, Name: "X-Key", In: "header"}, true},
		{"api key missing name", &SecurityScheme{Type: SecurityTypeAPIKey, In: "header"}, false},
		{"api key bad location", &SecurityScheme{Type: SecurityTypeAPIKey, Name: "Key", In: "path"}, false},
		{"http", &SecurityScheme{Type: SecurityTypeHTTP, Scheme: "bearer"}, true},
		{"http missing scheme", &SecurityScheme{Type: SecurityTypeHTTP}, false},
		{"openid", &SecurityScheme{Type: SecurityTypeOpenIDConnect, OpenIDConnectURL: "https://example.test/.well-known/openid-configuration"}, true},
		{"openid missing url", &SecurityScheme{Type: SecurityTypeOpenIDConnect}, false},
		{"oauth authorization code", &SecurityScheme{Type: SecurityTypeOAuth2, Flows: &OAuthFlows{AuthorizationCode: &OAuthFlow{AuthorizationURL: "https://example.test/auth", TokenURL: "https://example.test/token", Scopes: map[string]string{}}}}, true},
		{"oauth missing flow", &SecurityScheme{Type: SecurityTypeOAuth2, Flows: &OAuthFlows{}}, false},
		{"oauth missing token", &SecurityScheme{Type: SecurityTypeOAuth2, Flows: &OAuthFlows{Password: &OAuthFlow{Scopes: map[string]string{}}}}, false},
		{"oauth missing scopes", &SecurityScheme{Type: SecurityTypeOAuth2, Flows: &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://example.test/auth"}}}, false},
		{"mutual tls", &SecurityScheme{Type: "mutualTLS"}, true},
		{"unknown", &SecurityScheme{Type: "other"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.scheme.Validate()
			if (err == nil) != tc.valid {
				t.Fatalf("Validate() = %v, valid = %v", err, tc.valid)
			}
		})
	}
}

func TestAPIValidateChecksNestedLocations(t *testing.T) {
	bad := func() *Schema { return &Schema{Type: "invalid"} }
	for _, tc := range []struct {
		name string
		set  func(*API)
		want string
	}{
		{"component parameter", func(api *API) {
			api.Components.Parameters["p"] = Parameter{Name: "q", In: ParameterInQuery, Schema: bad()}
		}, "component parameter"},
		{"component request body", func(api *API) {
			api.Components.RequestBodies["b"] = RequestBody{Content: map[string]MediaType{"application/json": {Schema: bad()}}}
		}, "component request body"},
		{"component response header", func(api *API) {
			api.Components.Responses["r"] = Response{Headers: map[string]Header{"X": {Schema: bad()}}}
		}, "header"},
		{"component header", func(api *API) { api.Components.Headers["h"] = Header{Schema: bad()} }, "component header"},
		{"webhook", func(api *API) {
			api.Webhooks["event"] = PathItem{Trace: &Operation{Responses: Responses{"200": {Content: map[string]MediaType{"application/json": {Schema: bad()}}}}}}
		}, "webhook"},
		{"callback", func(api *API) {
			api.Components.Callbacks["onEvent"] = Callback{"{$request.body#/url}": {Head: &Operation{Parameters: []Parameter{{Name: "id", In: ParameterInQuery, Schema: bad()}}}}}
		}, "callback"},
		{"encoding header", func(api *API) {
			api.Paths["/p"] = PathItem{Post: &Operation{RequestBody: &RequestBody{Content: map[string]MediaType{"application/json": {Encoding: map[string]Encoding{"id": {Headers: map[string]Header{"X": {Schema: bad()}}}}}}}}}
		}, "encoding"},
		{"missing example reference", func(api *API) {
			api.Components.Examples["sample"] = Example{Ref: "#/components/examples/missing"}
		}, "missing example"},
		{"invalid component link", func(api *API) {
			api.Components.Links["next"] = Link{OperationRef: "#/paths/~1items/get", OperationID: "getItems"}
		}, "operationRef"},
		{"missing link reference", func(api *API) {
			api.Components.Links["next"] = Link{Ref: "#/components/links/missing"}
		}, "not found"},
		{"missing reference", func(api *API) { api.Components.Schemas["A"] = &Schema{Ref: "#/components/schemas/missing"} }, "not found"},
		{"invalid bounds", func(api *API) {
			minimum, maximum := 5, 1
			api.Components.Schemas["A"] = &Schema{MinLength: &minimum, MaxLength: &maximum}
		}, "minimum length"},
		{"missing OAuth scope", func(api *API) {
			api.Components.SecuritySchemes["oauth"] = SecurityScheme{Type: SecurityTypeOAuth2, Flows: &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://example.test/token", Scopes: map[string]string{"read": "Read"}}}}
			api.Security = []SecurityRequirement{{"oauth": {"write"}}}
		}, "scope"},
		{"missing security scheme", func(api *API) { api.Security = []SecurityRequirement{{"missing": nil}} }, "not found"},
		{"path parameter not required", func(api *API) {
			api.Paths["/p/{id}"] = PathItem{Parameters: []Parameter{{Name: "id", In: ParameterInPath}}}
		}, "must be required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := NewAPI()
			api.Info.Title = "Example"
			tc.set(api)
			if err := api.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestAPIValidateEscapedReferencesAndCallbackCycles(t *testing.T) {
	api := NewAPI()
	api.Info.Title = "Example"
	api.Components.Schemas["a/b"] = &Schema{Type: TypeString}
	api.Components.Schemas["ref"] = &Schema{Ref: "#/components/schemas/a~1b"}
	op := &Operation{}
	op.Callbacks = map[string]Callback{"loop": {"event": {Get: op}}}
	api.Paths["/items"] = PathItem{Get: op}
	if err := api.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveRefNestedAndEscapedSchemas(t *testing.T) {
	components := NewComponents()
	leaf := &Schema{Type: TypeString}
	components.Schemas["a/b"] = &Schema{Properties: map[string]*Schema{"x~y": {AllOf: []*Schema{leaf}}}}
	for _, ref := range []string{
		"#/components/schemas/a~1b/properties/x~0y/allOf/0",
		"#/components/schemas/a%2Fb/properties/x~0y/allOf/0",
	} {
		resolved, err := (&Schema{Ref: ref}).ResolveRef(&components)
		if err != nil || resolved != leaf {
			t.Fatalf("ResolveRef(%q) = %#v, %v", ref, resolved, err)
		}
	}
	for _, ref := range []string{
		"#/components/schemas/a~2b", "#/components/schemas/a~1b/properties/missing",
		"#/components/schemas/a~1b/properties/x~0y/allOf/9", "#/components/schemas/a~1b/type",
	} {
		if _, err := (&Schema{Ref: ref}).ResolveRef(&components); err == nil {
			t.Fatalf("accepted invalid reference %q", ref)
		}
	}
}

func TestAPIValidateSecurityReferences(t *testing.T) {
	api := NewAPI()
	api.Info.Title = "Example"
	api.Components.SecuritySchemes["oauth/main"] = SecurityScheme{
		Type:  SecurityTypeOAuth2,
		Flows: &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://example.test/token", Scopes: map[string]string{"read": "Read"}}},
	}
	api.Components.SecuritySchemes["alias"] = SecurityScheme{Ref: "#/components/securitySchemes/oauth~1main"}
	api.Security = []SecurityRequirement{{"alias": {"read"}}}
	if err := api.Validate(); err != nil {
		t.Fatal(err)
	}
	api.Security = []SecurityRequirement{{"alias": {"write"}}}
	if err := api.Validate(); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("missing scope: %v", err)
	}
	api.Security = nil
	api.Components.SecuritySchemes["broken"] = SecurityScheme{Ref: "#/components/securitySchemes/missing"}
	if err := api.Validate(); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing reference: %v", err)
	}
	delete(api.Components.SecuritySchemes, "broken")
	api.Components.SecuritySchemes["loop"] = SecurityScheme{Ref: "#/components/securitySchemes/loop"}
	if err := api.Validate(); err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("cyclic reference: %v", err)
	}
}

func TestAPIValidateOAuthScopesAcrossFlows(t *testing.T) {
	api := NewAPI()
	api.Info.Title = "Example"
	api.Components.SecuritySchemes["oauth"] = SecurityScheme{
		Type: SecurityTypeOAuth2,
		Flows: &OAuthFlows{
			Implicit: &OAuthFlow{
				AuthorizationURL: "https://example.test/auth",
				Scopes:           map[string]string{"read": "Read"},
			},
			AuthorizationCode: &OAuthFlow{
				AuthorizationURL: "https://example.test/auth",
				TokenURL:         "https://example.test/token",
				Scopes:           map[string]string{"write": "Write"},
			},
		},
	}
	api.Security = []SecurityRequirement{{"oauth": {"read", "write"}}}
	if err := api.Validate(); err != nil {
		t.Fatalf("Validate() with scopes from different flows: %v", err)
	}
	api.Security = []SecurityRequirement{{"oauth": {"read", "missing"}}}
	if err := api.Validate(); err == nil || !strings.Contains(err.Error(), `scope "missing"`) {
		t.Fatalf("Validate() with missing scope = %v", err)
	}
}
