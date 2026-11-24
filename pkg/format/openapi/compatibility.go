package openapi

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/amer8/apibconv/pkg/model"
)

// validateSchemaCompatibility rejects schema constructs the target dialect
// cannot express, before the writer emits any bytes.
//
//nolint:gocyclo // The traversal checks the complete model before output.
func validateSchemaCompatibility(api *model.API, version string, serverBudget swaggerServerBudget) error {
	swagger := strings.HasPrefix(version, "2")
	// Both reading from and writing to Swagger can expand server URLs. Charge
	// each emitted occurrence even when several paths share an operation.
	checkServers := func(servers []model.Server, implicitSchemes []string) error {
		if swagger || api.Version == "2.0" {
			for i := range servers {
				if err := serverBudget.reserve(int64(len(servers[i].URL))); err != nil {
					return err
				}
			}
		}
		return serverBudget.reserveSchemes(implicitSchemes, 0)
	}
	var rootSchemes []string
	if origin := api.Swagger2Origin; origin != nil {
		if len(api.Servers) > 0 {
			return fmt.Errorf("cannot combine explicit servers with an implicit Swagger host")
		}
		if !swagger && (len(origin.Schemes) > 0 || !validImplicitServerPath(origin.BasePath)) {
			return fmt.Errorf("cannot translate Swagger origin with an implicit host to an OpenAPI server")
		}
		rootSchemes = origin.Schemes
	}
	if err := checkServers(api.Servers, rootSchemes); err != nil {
		return err
	}
	var rootServer swaggerServerLocation
	if swagger && len(api.Servers) > 0 {
		rootServer = swaggerOverrideLocation(api.Servers[0].URL, true)
	}
	oas30 := strings.HasPrefix(version, "3.0")
	seen := map[*model.Schema]bool{}
	activeSchemas := map[*model.Schema]bool{}
	var schema func(*model.Schema) error
	schema = func(s *model.Schema) error {
		if s == nil {
			return nil
		}
		if activeSchemas[s] {
			return fmt.Errorf("cyclic schema cannot be written directly; use a reference")
		}
		if seen[s] {
			return nil
		}
		seen[s] = true
		activeSchemas[s] = true
		defer delete(activeSchemas, s)
		if (swagger || oas30) && s.Ref != "" {
			withoutRef := *s
			withoutRef.Ref = ""
			if !reflect.DeepEqual(withoutRef, model.Schema{}) {
				return fmt.Errorf("schema reference siblings cannot be represented in OpenAPI %s", version)
			}
		}
		if swagger && s.Discriminator != nil {
			return fmt.Errorf("schema discriminator cannot be represented in OpenAPI 2.0")
		}
		if swagger && (s.Nullable || len(s.AnyOf) != 0 || len(s.OneOf) != 0 || s.Not != nil || s.WriteOnly || s.Deprecated) {
			return fmt.Errorf("schema contains constraints not representable in OpenAPI 2.0")
		}
		if (swagger || oas30) && s.Type == "null" {
			return fmt.Errorf("null schema type is not representable in OpenAPI %s", version)
		}
		if swagger || (oas30 && !s.Nullable) {
			for _, v := range s.Enum {
				if v == nil {
					return fmt.Errorf("null enum value is not representable in OpenAPI %s", version)
				}
			}
		}
		children := []*model.Schema{s.Items, s.Not}
		for _, p := range s.Properties {
			children = append(children, p)
		}
		children = append(children, s.AllOf...)
		children = append(children, s.AnyOf...)
		children = append(children, s.OneOf...)
		if nested, ok := s.AdditionalProperties.(*model.Schema); ok {
			children = append(children, nested)
		}
		for _, child := range children {
			if err := schema(child); err != nil {
				return err
			}
		}
		return nil
	}
	checkContent := func(content map[string]model.MediaType) error {
		var first *model.Schema
		firstSet := false
		for _, media := range content {
			if err := schema(media.Schema); err != nil {
				return err
			}
			for _, encoding := range media.Encoding {
				for _, header := range encoding.Headers {
					if err := schema(header.Schema); err != nil {
						return err
					}
				}
			}
			if swagger && firstSet && !reflect.DeepEqual(first, media.Schema) {
				return fmt.Errorf("different content type schemas cannot be represented in OpenAPI 2.0")
			}
			first, firstSet = media.Schema, true
		}
		return nil
	}
	checkParameters := func(parameters []model.Parameter) error {
		for i := range parameters {
			parameter := &parameters[i]
			if swagger && parameter.In != "body" && (parameter.In != "formData" || parameter.Schema == nil || parameter.Schema.Type != "file") {
				if _, err := swaggerParameterSchema(parameter.Schema, api); err != nil {
					return err
				}
			}
			if err := schema(parameter.Schema); err != nil {
				return err
			}
			if err := checkContent(parameter.Content); err != nil {
				return err
			}
		}
		return nil
	}
	checkHeaders := func(headers map[string]model.Header) error {
		for _, header := range headers {
			if swagger {
				if _, err := swaggerHeaderSchema(header.Schema, api); err != nil {
					return err
				}
			}
			if err := schema(header.Schema); err != nil {
				return err
			}
		}
		return nil
	}
	checkResponse := func(response model.Response) error {
		if err := checkContent(response.Content); err != nil {
			return err
		}
		return checkHeaders(response.Headers)
	}
	var path func(model.PathItem) error
	seenOps := map[*model.Operation]bool{}
	activeOps := map[*model.Operation]bool{}
	path = func(item model.PathItem) error {
		if swagger && len(item.Servers) > 0 {
			return fmt.Errorf("OpenAPI 2.0 cannot represent path server override")
		}
		if err := checkParameters(item.Parameters); err != nil {
			return err
		}
		for _, op := range []*model.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op == nil {
				continue
			}
			if err := checkServers(op.Servers, op.Swagger2HostSchemes); err != nil {
				return err
			}
			if len(op.Swagger2HostSchemes) > 0 && (!swagger || len(op.Servers) > 0) {
				return fmt.Errorf("cannot translate Swagger operation schemes with an implicit host to explicit servers")
			}
			if swagger {
				if err := checkSwaggerOperationServers(rootServer, op.Servers); err != nil {
					return err
				}
			}
			if activeOps[op] {
				return fmt.Errorf("cyclic callback operation cannot be written")
			}
			if seenOps[op] {
				continue
			}
			seenOps[op] = true
			activeOps[op] = true
			if err := checkParameters(op.Parameters); err != nil {
				return err
			}
			if op.RequestBody != nil {
				if err := checkContent(op.RequestBody.Content); err != nil {
					return err
				}
			}
			for _, response := range op.Responses {
				if err := checkResponse(response); err != nil {
					return err
				}
			}
			for _, callback := range op.Callbacks {
				for expression := range callback {
					item := callback[expression]
					if err := path(item); err != nil {
						return err
					}
				}
			}
			delete(activeOps, op)
		}
		return nil
	}
	for _, s := range api.Components.Schemas {
		if err := schema(s); err != nil {
			return err
		}
	}
	for name := range api.Components.Parameters {
		parameter := api.Components.Parameters[name]
		if err := checkParameters([]model.Parameter{parameter}); err != nil {
			return fmt.Errorf("parameter component %q: %w", name, err)
		}
	}
	if swagger {
		for name := range api.Components.Swagger2Parameters {
			parameter := api.Components.Swagger2Parameters[name]
			if err := checkParameters([]model.Parameter{parameter}); err != nil {
				return fmt.Errorf("swagger parameter %q: %w", name, err)
			}
		}
	}
	for name, body := range api.Components.RequestBodies {
		if err := checkContent(body.Content); err != nil {
			return fmt.Errorf("request body component %q: %w", name, err)
		}
	}
	for name, response := range api.Components.Responses {
		if err := checkResponse(response); err != nil {
			return fmt.Errorf("response component %q: %w", name, err)
		}
	}
	if err := checkHeaders(api.Components.Headers); err != nil {
		return fmt.Errorf("header component: %w", err)
	}
	for _, paths := range []map[string]model.PathItem{api.Paths, api.Webhooks} {
		for name := range paths {
			item := paths[name]
			if err := path(item); err != nil {
				return err
			}
		}
	}
	for _, callback := range api.Components.Callbacks {
		for name := range callback {
			item := callback[name]
			if err := path(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func validImplicitServerPath(path string) bool {
	// Browsers discard some control characters before resolving URL authorities.
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") &&
		!strings.ContainsAny(path, `\?#`) && strings.IndexFunc(path, unicode.IsControl) < 0
}

// Swagger operation overrides can change only the scheme of the root server.
type swaggerServerLocation struct{ host, path string }

func swaggerOverrideLocation(rawURL string, allowInherited bool) swaggerServerLocation {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || hasUnsupportedSwaggerURLParts(rawURL, u) {
		return swaggerServerLocation{}
	}
	if validSwaggerScheme(u.Scheme) || (allowInherited && u.Scheme == "") {
		return newSwaggerServerLocation(u.Host, u.EscapedPath())
	}
	return swaggerServerLocation{}
}

func hasUnsupportedSwaggerURLParts(rawURL string, u *url.URL) bool {
	// EscapedPath renders template braces and literal percent-encoded braces alike.
	return strings.ContainsAny(rawURL, "{}") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != ""
}

func newSwaggerServerLocation(host, path string) swaggerServerLocation {
	if host != "" && path == "" {
		path = "/"
	}
	// ASCII DNS names are case-insensitive. Preserve non-ASCII hosts and IPv6 zones exactly.
	for i := range len(host) {
		if host[i] >= utf8.RuneSelf || host[i] == '%' {
			return swaggerServerLocation{host: host, path: path}
		}
	}
	return swaggerServerLocation{host: strings.ToLower(host), path: path}
}

func checkSwaggerOperationServers(base swaggerServerLocation, overrides []model.Server) error {
	for i, server := range overrides {
		if base.host == "" || !reflect.DeepEqual(server, model.Server{URL: server.URL}) || swaggerOverrideLocation(server.URL, false) != base {
			return fmt.Errorf("OpenAPI 2.0 cannot represent operation server override %d (only the root server scheme can change)", i+1)
		}
	}
	return nil
}

func validSwaggerScheme(scheme string) bool {
	switch scheme {
	case "http", "https", "ws", "wss":
		return true
	}
	return false
}
