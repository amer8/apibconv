package openapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/internal/specschema"
	"github.com/amer8/apibconv/internal/yamlguard"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

func TestWideMappingRejectedAtEachEntryPoint(t *testing.T) {
	var doc strings.Builder
	doc.WriteString("openapi: 3.1.0\npaths: {}\nx-map:\n")
	for i := 0; i < 2049; i++ {
		fmt.Fprintf(&doc, "  k%d: 1\n", i)
	}
	input := []byte(doc.String())
	_, _, detectErr := detect.NewDetector().DetectFromBytes(input)
	_, parseErr := NewParser().Parse(context.Background(), bytes.NewReader(input))
	for _, err := range []error{detectErr, parseErr} {
		if err == nil || !strings.Contains(err.Error(), "mapping exceeds") {
			t.Fatalf("wide mapping: %v", err)
		}
	}
	issues := specschema.Validate(format.FormatOpenAPI, input)
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "mapping exceeds") {
		t.Fatalf("validation: %v", issues)
	}
}

func TestCallbackBudgetSharedAcrossReferences(t *testing.T) {
	doc := OpenAPI{Components: Components{Callbacks: map[string]Callback{"Done": {Paths: map[string]PathItem{"event": {}}}}}}
	ref := Callback{Ref: "#/components/callbacks/Done"}
	// One valid reference consumes a callback visit, a reference hop and a path.
	r := callbackResolver{doc: &doc, remaining: 3, modelBytes: maxCallbackModelBytes}
	first, err := r.resolveCallback("Done", ref, 0)
	if err != nil || len(first.Paths) != 1 {
		t.Fatalf("first callback: %#v, %v", first, err)
	}
	if _, err := r.resolveCallback("Done", ref, 0); err == nil || !strings.Contains(err.Error(), "expansion limit") {
		t.Fatalf("reused budget: %v", err)
	}
}

func TestCallbackModelBudgetIncludesResolvedContents(t *testing.T) {
	r := callbackResolver{doc: &OpenAPI{}, remaining: maxCallbackVisits, modelBytes: 1000}
	callback := Callback{Extensions: map[string]any{"x-description": strings.Repeat("a", 1000)}}
	if _, err := r.resolveCallback("Done", callback, 0); err == nil || !strings.Contains(err.Error(), "model expansion") {
		t.Fatalf("expected payload budget error, got %v", err)
	}
}

func TestCallbackBudgetIncludesRetainedFields(t *testing.T) {
	ext := map[string]any{"x-note": "retained"}
	for _, tc := range []struct {
		name             string
		empty, populated any
	}{
		{"parameter ref", Parameter{}, Parameter{sourceRef: "#/components/parameters/ID"}},
		{"body ref", RequestBody{}, RequestBody{sourceRef: "#/components/requestBodies/Item"}},
		{"response ref", Response{}, Response{sourceRef: "#/components/responses/OK"}},
		{"parameter overlay", Parameter{}, Parameter{sourceRefExtensions: ext}},
		{"body overlay", RequestBody{}, RequestBody{sourceRefExtensions: ext}},
		{"response overlay", Response{}, Response{sourceRefExtensions: ext}},
		{"typed additional properties", Schema{}, Schema{additionalSchema: &Schema{Type: "string"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := callbackResolver{modelBytes: maxCallbackModelBytes}
			if !r.fitsModel(reflect.ValueOf(tc.empty)) {
				t.Fatal("empty value exceeded the budget")
			}
			r.modelBytes = maxCallbackModelBytes - r.modelBytes
			if r.fitsModel(reflect.ValueOf(tc.populated)) {
				t.Fatal("retained data did not consume additional model budget")
			}
		})
	}
}

func TestNestedCallbackErrorKeepsBoundedContext(t *testing.T) {
	name := strings.Repeat("é", 160)
	missing := Callback{Ref: "#/components/callbacks/Missing"}
	nested := PathItem{Post: &Operation{Callbacks: map[string]Callback{"inner": missing}}}
	outer := Callback{Paths: map[string]PathItem{name: nested}}
	doc := &OpenAPI{Paths: map[string]PathItem{"/jobs": {Post: &Operation{Callbacks: map[string]Callback{name: outer}}}}}
	_, err := NewParser().convertToModel(doc)
	if err == nil || !strings.Contains(err.Error(), "Missing") || !strings.Contains(err.Error(), "/jobs") ||
		strings.Contains(err.Error(), name) || strings.Contains(err.Error(), "inner") {
		t.Fatalf("unexpected callback diagnostic: %v", err)
	}
}

func TestSwaggerOperationSchemePreservation(t *testing.T) {
	const source = "swagger: '2.0'\ninfo: {title: Example, version: '1'}\nhost: api.example.test\nbasePath: /v1/@tenant\nschemes: [http, https]\npaths:\n  /items:\n    get: {schemes: [https], responses: {'200': {description: ok}}}\n"
	ctx := context.Background()
	for _, explicitHost := range []bool{true, false} {
		input := source
		if !explicitHost {
			input = strings.ReplaceAll(input, "host: api.example.test\n", "")
		}
		api, err := NewParser().Parse(ctx, strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		for _, version := range []string{"2.0", "3.1"} {
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion(version)).Write(ctx, api, &out)
			if version == "3.1" && !explicitHost {
				if err == nil || out.Len() != 0 {
					t.Fatalf("implicit host upgrade: output %q, err %v", out.String(), err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Paths    map[string]PathItem
				BasePath string `yaml:"basePath"`
				Schemes  []string
				Host     string
			}
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			op := doc.Paths["/items"].Get
			if op == nil {
				t.Fatal("missing operation")
			}
			if version == "2.0" {
				if doc.BasePath != "/v1/@tenant" || strings.Join(doc.Schemes, ",") != "http,https" || (doc.Host != "") != explicitHost {
					t.Fatalf("root transport changed: %+v", doc)
				}
				if len(op.Schemes) != 1 || op.Schemes[0] != "https" {
					t.Fatalf("Swagger schemes: %v", op.Schemes)
				}
			} else if len(op.Servers) != 1 || op.Servers[0].URL != "https://api.example.test/v1/@tenant" {
				t.Fatalf("OpenAPI servers: %v", op.Servers)
			}
		}
	}
}

func TestSchemaDecoderAcceptsSharedAlias(t *testing.T) {
	const input = "openapi: 3.1.0\ninfo: {title: Schema, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Text: &Text {type: string}\n    Labels: {type: object, additionalProperties: *Text}\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	child, ok := api.Components.Schemas["Labels"].AdditionalProperties.(*model.Schema)
	if !ok || child.Type != model.TypeString {
		t.Fatalf("missing typed map value: %#v", child)
	}
}

func TestImplicitHostBasePathStaysAPath(t *testing.T) {
	ctx := context.Background()
	for _, path := range []string{"/api", "//archive/items"} {
		source := "swagger: '2.0'\ninfo: {title: Test, version: '1'}\npaths: {}\nbasePath: " + path + "\n"
		api, err := NewParser().Parse(ctx, strings.NewReader(source))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := NewWriter(WithWriterVersion("2.0")).Write(ctx, api, &out); err != nil {
			t.Fatal(err)
		}
		var swagger Swagger20
		if err := yaml.Unmarshal(out.Bytes(), &swagger); err != nil || swagger.BasePath != path || swagger.Host != "" {
			t.Fatalf("host=%q basePath=%q err=%v", swagger.Host, swagger.BasePath, err)
		}
		out.Reset()
		err = NewWriter(WithWriterVersion("3.1")).Write(ctx, api, &out)
		if path == "/api" {
			if err != nil {
				t.Fatal(err)
			}
			var doc OpenAPI
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil || len(doc.Servers) != 1 || doc.Servers[0].URL != path {
				t.Fatalf("%q: %+v %v", path, doc.Servers, err)
			}
		} else if err == nil || out.Len() != 0 {
			t.Fatalf("ambiguous path produced %q", out.String())
		}
	}
}

func TestCallbackReferenceChargesExtensionCopies(t *testing.T) {
	resources := map[string]Response{"Shared": {Extensions: map[string]any{"x-label": "test"}}}
	r := callbackResolver{remaining: 10, modelBytes: 100}
	_, err := resolveResponse(Response{Ref: "#/components/responses/Shared"}, resources, "#/components/responses/", &r)
	if err == nil || !strings.Contains(err.Error(), "model expansion") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestCustomSchemaDecoderChecksExtensionGraph(t *testing.T) {
	// Exercise the hook with an opaque extension; a rejected graph must never
	// reach generic extension decoding or any recursively typed schema field.
	extension := &yaml.Node{Kind: yaml.SequenceNode}
	extension.Content = []*yaml.Node{{Kind: yaml.AliasNode, Alias: extension}}
	node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "x-example"}, extension}}
	var schema Schema
	if err := schema.UnmarshalYAML(node); !errors.Is(err, yamlguard.ErrUnsafe) {
		t.Fatalf("expected graph rejection, got %v", err)
	}
}

func TestSwaggerServerBudgetChargesEachOccurrence(t *testing.T) {
	const input = `swagger: '2.0'
info: {title: Items, version: '1'}
host: api.example.test
basePath: /v1
schemes: [http]
paths:
  /items:
    get: {schemes: [https], responses: {'200': {description: ok}}}
  /more: {$ref: '#/paths/~1items'}
`
	for _, tc := range []struct {
		name, source, version string
		schemeBytes           int
	}{
		{"explicit host", input, "3.1", len("http://api.example.test/v1") + 2*len("https://api.example.test/v1")},
		{"implicit host", strings.Replace(input, "host: api.example.test\n", "", 1), "2.0", len("http") + 2*len("https")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := yamlguard.Parse([]byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			exact := swaggerServerBudget(3*swaggerServerEntryBytes + tc.schemeBytes)
			parser := NewParser()
			ctx := context.Background()
			api, err := parser.parseV2(ctx, root, exact)
			if err != nil {
				t.Fatalf("parse at the limit: %v", err)
			}
			if partial, err := parser.parseV2(ctx, root, exact-1); err == nil || !strings.Contains(err.Error(), "server URL expansion limit") || partial != nil {
				t.Fatalf("parse over the limit returned a model=%t, err=%v", partial != nil, err)
			}
			// Both occurrences will be written even if a transformation reuses the operation.
			api.Paths["/more"] = api.Paths["/items"]
			if err := validateSchemaCompatibility(api, tc.version, exact); err != nil {
				t.Fatalf("write preflight at the limit: %v", err)
			}
			if err := validateSchemaCompatibility(api, tc.version, exact-1); err == nil || !strings.Contains(err.Error(), "server URL expansion limit") {
				t.Fatalf("write preflight over the limit: %v", err)
			}
		})
	}
}

func TestSwaggerRootServerSchemes(t *testing.T) {
	const address = "://api.example.test/v1"
	for _, tc := range []struct {
		name    string
		schemes []string
		wantErr string
	}{
		{"supported and repeated", []string{"http", "wss", "https", "ws", "http", "https"}, ""},
		{"unsupported first", []string{"ftp", "https"}, "unsupported Swagger scheme in root server 1"},
		{"unsupported later", []string{"https", "ftp"}, "unsupported Swagger scheme in root server 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Version = "3.1.0"
			for _, scheme := range tc.schemes {
				api.Servers = append(api.Servers, model.Server{URL: scheme + address})
			}
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr || out.Len() != 0 {
					t.Fatalf("wrote %q, err=%v", out.String(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var doc Swagger20
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Host != "api.example.test" || doc.BasePath != "/v1" || strings.Join(doc.Schemes, ",") != "http,wss,https,ws" {
				t.Fatalf("host=%q basePath=%q schemes=%v", doc.Host, doc.BasePath, doc.Schemes)
			}
		})
	}
}

func TestSwaggerRootTransportInheritance(t *testing.T) {
	const inherited = "//api.example.test/v1"
	const explicit = "https://api.example.test/v1"
	for _, tc := range []struct {
		name    string
		urls    []string
		wantErr string
	}{
		{"inherited alone", []string{inherited}, ""},
		{"both inherited", []string{inherited, inherited}, ""},
		{"different endpoint", []string{inherited, "https://elsewhere.example.test/v1"}, ""},
		{"inherited then explicit", []string{inherited, explicit}, "inherited and explicit"},
		{"explicit then inherited", []string{explicit, inherited}, "inherited and explicit"},
		{"mixed DNS case", []string{inherited, "https://API.example.test/v1"}, "inherited and explicit"},
		{"bare IPv6 with port", []string{"[::1]:8443", "https://[::1]:8443/"}, "inherited and explicit"},
		{"primary with query", []string{explicit + "?view=public"}, "queries"},
		{"document-relative primary", []string{"teams/items"}, "primary server URL"},
		{"secondary with query", []string{explicit, "http://api.example.test/v1?view=public"}, "queries"},
		{"empty query", []string{explicit, explicit + "?"}, "queries"},
		{"user information", []string{explicit, "https://reader@api.example.test/v1"}, "user information"},
		{"fragment", []string{explicit, explicit + "#section"}, "fragments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			for _, rawURL := range tc.urls {
				api.Servers = append(api.Servers, model.Server{URL: rawURL})
			}
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || out.Len() != 0 {
					t.Fatalf("expected %q before output, wrote %d bytes, err=%v", tc.wantErr, out.Len(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var doc Swagger20
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Host != "api.example.test" || doc.BasePath != "/v1" || doc.Schemes != nil {
				t.Fatalf("expected inherited transport, got host=%q basePath=%q schemes=%v", doc.Host, doc.BasePath, doc.Schemes)
			}
		})
	}
}

func TestSwaggerServerSchemesDistinguishTemplatesFromLiteralPaths(t *testing.T) {
	const literalPath = "/%7Bregion%7D"
	literal := model.Server{URL: "https://api.example.test" + literalPath}
	template := model.Server{
		URL:       "http://api.example.test/{region}",
		Variables: map[string]model.ServerVariable{"region": {Default: "west", Enum: []string{"west"}}},
	}
	for _, tc := range []struct {
		name        string
		servers     []model.Server
		overrides   []model.Server
		wantErr     string
		wantSchemes []string
	}{
		{name: "templated alternative", servers: []model.Server{literal, template}, wantErr: "root server 2"},
		{name: "templated primary", servers: []model.Server{template, literal}, wantErr: "root server 1"},
		{name: "operation with undeclared template", servers: []model.Server{literal}, overrides: []model.Server{{URL: template.URL}}, wantErr: "operation server override 1"},
		{name: "matching literal paths", servers: []model.Server{literal, {URL: "wss://api.example.test" + literalPath}}, wantSchemes: []string{"https", "wss"}},
		{name: "unrelated template", servers: []model.Server{literal, {URL: "http://elsewhere.example.test/{region}", Variables: template.Variables}}, wantSchemes: []string{"https"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Servers = tc.servers
			api.Paths["/items"] = model.PathItem{Get: &model.Operation{Servers: tc.overrides}}
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || out.Len() != 0 {
					t.Fatalf("expected %q before output, wrote %d bytes, err=%v", tc.wantErr, out.Len(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var doc Swagger20
			if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.BasePath != literalPath || !reflect.DeepEqual(doc.Schemes, tc.wantSchemes) {
				t.Fatalf("basePath=%q schemes=%v", doc.BasePath, doc.Schemes)
			}
		})
	}
}

func TestSwaggerInheritedRootWithExplicitOperationScheme(t *testing.T) {
	api := model.NewAPI()
	api.Servers = []model.Server{{URL: "//api.example.test"}}
	api.Paths["/items"] = model.PathItem{Get: &model.Operation{Servers: []model.Server{{URL: "https://API.example.test/"}}}}
	var out bytes.Buffer
	if err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc Swagger20
	if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	get := doc.Paths["/items"].Get
	if doc.Host != "api.example.test" || doc.Schemes != nil || get == nil || !reflect.DeepEqual(get.Schemes, []string{"https"}) {
		t.Fatalf("root host=%q schemes=%v, operation=%+v", doc.Host, doc.Schemes, get)
	}
}

func TestSwaggerOperationServerErrorOmitsURL(t *testing.T) {
	const root = "https://api.example.test/v1"
	for _, tc := range []struct{ name, url string }{
		{"credentials", "https://fixture-user:fixture-password@api.example.test/v1"},
		{"query token", root + "?token=fixture-token"},
		{"malformed URL", root + "/%zz?token=fixture-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := model.NewAPI()
			api.Servers = []model.Server{{URL: root}}
			api.Paths["/items"] = model.PathItem{Get: &model.Operation{
				Servers: []model.Server{{URL: root}, {URL: tc.url}},
			}}
			var out bytes.Buffer
			err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out)
			if err == nil || out.Len() != 0 {
				t.Fatalf("expected a rejection before output, wrote %d bytes", out.Len())
			}
			msg := err.Error()
			for _, value := range []string{tc.url, "fixture-user", "fixture-password", "fixture-token"} {
				if strings.Contains(msg, value) {
					t.Fatal("diagnostic exposed operation server URL data")
				}
			}
			if !strings.Contains(msg, "operation server override 2") {
				t.Fatalf("diagnostic did not identify the rejected server: %q", msg)
			}
		})
	}
}

func TestSwaggerWriterBudgetsOpenAPI3ServerOccurrences(t *testing.T) {
	const root = "https://api.example.test/v1"
	const override = "wss://api.example.test/v1"
	const equivalent = "WSS://api.example.test/v1"
	api := model.NewAPI()
	api.Version = "3.1.0"
	api.Servers = []model.Server{{URL: root}}
	op := &model.Operation{Servers: []model.Server{{URL: override}, {URL: equivalent}}}
	api.Paths["/items"] = model.PathItem{Get: op}
	api.Paths["/more"] = api.Paths["/items"]
	exact := swaggerServerBudget(5*swaggerServerEntryBytes + len(root) + 2*(len(override)+len(equivalent)))
	if err := validateSchemaCompatibility(api, "2.0", exact-1); err == nil || !strings.Contains(err.Error(), "server URL expansion limit") {
		t.Fatalf("one byte over: %v", err)
	}
	if err := validateSchemaCompatibility(api, "2.0", exact); err != nil {
		t.Fatalf("exact budget: %v", err)
	}
	var out bytes.Buffer
	if err := NewWriter(WithWriterVersion("2.0")).Write(context.Background(), api, &out); err != nil {
		t.Fatal(err)
	}
	var doc Swagger20
	if err := yaml.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for path := range api.Paths {
		get := doc.Paths[path].Get
		if get == nil || strings.Join(get.Schemes, ",") != "wss" {
			t.Fatalf("missing override on %s", path)
		}
	}
}

func TestImplicitHostBasePathRejectsControls(t *testing.T) {
	for _, path := range []string{"/api\tv1", "/api\nv1", "/api\rv1", "/api\x7fv1"} {
		input := fmt.Sprintf("swagger: '2.0'\ninfo: {title: Items, version: '1'}\nbasePath: %q\npaths: {}\n", path)
		api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err = NewWriter().Write(context.Background(), api, &out)
		if err == nil || !strings.Contains(err.Error(), "implicit host") || out.Len() != 0 {
			t.Errorf("path %q: wrote %d bytes, err=%v", path, out.Len(), err)
		}
	}
}

func TestAliasedSchemaKeywordPreservesAdditionalProperties(t *testing.T) {
	const input = `openapi: 3.1.0
info: {title: Labels, version: '1'}
paths: {}
x-key: &x-field additionalProperties
components:
  schemas:
    Labels:
      type: object
      ? *x-field
      : {const: active}
`
	for _, encoding := range []string{"json", "yaml"} {
		original, _, converted := securityRoundTrip(t, input, "3.1", encoding)
		for _, api := range []*model.API{original, converted} {
			nested, ok := api.Components.Schemas["Labels"].AdditionalProperties.(*model.Schema)
			if !ok || len(nested.AllOf) != 1 || len(nested.AllOf[0].Enum) != 1 || nested.AllOf[0].Enum[0] != "active" {
				t.Fatalf("%s: missing additional-property constraint: %#v", encoding, nested)
			}
		}
	}
}
