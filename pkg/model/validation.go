package model

import (
	"fmt"
	"math"
	"strings"
)

type modelValidator struct {
	api        *API
	schemas    map[*Schema]bool
	operations map[*Operation]bool
}

func validateAPI(api *API) error {
	if api == nil {
		return fmt.Errorf("API is nil")
	}
	if strings.TrimSpace(api.Info.Title) == "" {
		return fmt.Errorf("API title is required")
	}
	v := &modelValidator{
		api:        api,
		schemas:    make(map[*Schema]bool),
		operations: make(map[*Operation]bool),
	}
	for name := range api.Components.SecuritySchemes {
		scheme := api.Components.SecuritySchemes[name]
		if err := scheme.Validate(); err != nil {
			return fmt.Errorf("security scheme %q: %w", name, err)
		}
		if _, err := v.resolveSecurityScheme(name); err != nil {
			return err
		}
	}
	if err := v.security(api.Security); err != nil {
		return fmt.Errorf("API security: %w", err)
	}
	if err := v.servers(api.Servers); err != nil {
		return err
	}
	for name, schema := range api.Components.Schemas {
		if err := v.schema(schema); err != nil {
			return fmt.Errorf("schema %q: %w", name, err)
		}
	}
	for name := range api.Components.Parameters {
		parameter := api.Components.Parameters[name]
		if err := v.parameter(&parameter); err != nil {
			return fmt.Errorf("component parameter %q: %w", name, err)
		}
	}
	for name, body := range api.Components.RequestBodies {
		if err := v.content(body.Content); err != nil {
			return fmt.Errorf("component request body %q: %w", name, err)
		}
	}
	for name, response := range api.Components.Responses {
		if err := v.response(response); err != nil {
			return fmt.Errorf("component response %q: %w", name, err)
		}
	}
	for name, header := range api.Components.Headers {
		if err := v.header(header); err != nil {
			return fmt.Errorf("component header %q: %w", name, err)
		}
	}
	if err := v.examples(api.Components.Examples); err != nil {
		return fmt.Errorf("component examples: %w", err)
	}
	for name, link := range api.Components.Links {
		if err := v.link(&link); err != nil {
			return fmt.Errorf("component link %q: %w", name, err)
		}
	}
	for path := range api.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path %q must start with /", path)
		}
		item := api.Paths[path]
		if err := v.path(&item); err != nil {
			return fmt.Errorf("path %q: %w", path, err)
		}
	}
	for name := range api.Webhooks {
		item := api.Webhooks[name]
		if err := v.path(&item); err != nil {
			return fmt.Errorf("webhook %q: %w", name, err)
		}
	}
	for name, callback := range api.Components.Callbacks {
		if err := v.callback(callback); err != nil {
			return fmt.Errorf("component callback %q: %w", name, err)
		}
	}
	return nil
}

func (v *modelValidator) schema(schema *Schema) error {
	if schema == nil {
		return fmt.Errorf("schema is nil")
	}
	if v.schemas[schema] {
		return nil
	}
	v.schemas[schema] = true
	switch schema.Type {
	case "", "null", TypeString, TypeNumber, TypeInteger, TypeBoolean, TypeArray, TypeObject:
	default:
		return fmt.Errorf("invalid schema type %q", schema.Type)
	}
	if err := v.schemaReference(schema.Ref); err != nil {
		return err
	}
	if schema.Discriminator != nil && strings.TrimSpace(schema.Discriminator.PropertyName) == "" {
		return fmt.Errorf("discriminator propertyName is required")
	}
	for _, pair := range []struct {
		name     string
		min, max *int
	}{
		{"length", schema.MinLength, schema.MaxLength},
		{"items", schema.MinItems, schema.MaxItems},
		{"properties", schema.MinProperties, schema.MaxProperties},
	} {
		if pair.min != nil && *pair.min < 0 || pair.max != nil && *pair.max < 0 {
			return fmt.Errorf("%s bounds must not be negative", pair.name)
		}
		if pair.min != nil && pair.max != nil && *pair.min > *pair.max {
			return fmt.Errorf("minimum %s exceeds maximum", pair.name)
		}
	}
	for _, value := range []*float64{schema.Minimum, schema.Maximum, schema.MultipleOf} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return fmt.Errorf("numeric bounds must be finite")
		}
	}
	if schema.Minimum != nil && schema.Maximum != nil && *schema.Minimum > *schema.Maximum {
		return fmt.Errorf("minimum exceeds maximum")
	}
	if schema.MultipleOf != nil && *schema.MultipleOf <= 0 {
		return fmt.Errorf("multipleOf must be positive")
	}
	for name, child := range schema.Properties {
		if err := v.schema(child); err != nil {
			return fmt.Errorf("property %q: %w", name, err)
		}
	}
	children := append([]*Schema(nil), schema.AllOf...)
	children = append(children, schema.AnyOf...)
	children = append(children, schema.OneOf...)
	if schema.Items != nil {
		children = append(children, schema.Items)
	}
	if schema.Not != nil {
		children = append(children, schema.Not)
	}
	switch additional := schema.AdditionalProperties.(type) {
	case nil, bool:
	case *Schema:
		children = append(children, additional)
	default:
		return fmt.Errorf("additionalProperties must be a boolean or schema")
	}
	for _, child := range children {
		if err := v.schema(child); err != nil {
			return err
		}
	}
	return nil
}

func (v *modelValidator) schemaReference(ref string) error {
	if !strings.HasPrefix(ref, "#/components/schemas/") {
		return nil
	}
	_, err := (&Schema{Ref: ref}).ResolveRef(&v.api.Components)
	return err
}

func (v *modelValidator) path(item *PathItem) error {
	if err := v.servers(item.Servers); err != nil {
		return err
	}
	for i := range item.Parameters {
		if err := v.parameter(&item.Parameters[i]); err != nil {
			return err
		}
	}
	for _, op := range []*Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
		if op == nil || v.operations[op] {
			continue
		}
		v.operations[op] = true
		if err := v.operation(op); err != nil {
			return err
		}
	}
	return nil
}

func (v *modelValidator) operation(op *Operation) error {
	if err := v.security(op.Security); err != nil {
		return fmt.Errorf("operation security: %w", err)
	}
	if err := v.servers(op.Servers); err != nil {
		return err
	}
	for i := range op.Parameters {
		if err := v.parameter(&op.Parameters[i]); err != nil {
			return err
		}
	}
	if op.RequestBody != nil {
		if err := v.content(op.RequestBody.Content); err != nil {
			return fmt.Errorf("request body: %w", err)
		}
	}
	for code, response := range op.Responses {
		if err := v.response(response); err != nil {
			return fmt.Errorf("response %q: %w", code, err)
		}
	}
	for name, callback := range op.Callbacks {
		if err := v.callback(callback); err != nil {
			return fmt.Errorf("callback %q: %w", name, err)
		}
	}
	return nil
}

func (v *modelValidator) callback(callback Callback) error {
	for expression := range callback {
		item := callback[expression]
		if err := v.path(&item); err != nil {
			return fmt.Errorf("expression %q: %w", expression, err)
		}
	}
	return nil
}

func (v *modelValidator) parameter(parameter *Parameter) error {
	if strings.TrimSpace(parameter.Name) == "" {
		return fmt.Errorf("parameter name is required")
	}
	switch parameter.In {
	case ParameterInPath, ParameterInQuery, ParameterInHeader, ParameterInCookie:
	default:
		return fmt.Errorf("parameter %q has invalid location %q", parameter.Name, parameter.In)
	}
	if parameter.In == ParameterInPath && !parameter.Required {
		return fmt.Errorf("path parameter %q must be required", parameter.Name)
	}
	if parameter.Schema != nil {
		if err := v.schema(parameter.Schema); err != nil {
			return fmt.Errorf("parameter %q: %w", parameter.Name, err)
		}
	}
	if err := v.examples(parameter.Examples); err != nil {
		return fmt.Errorf("parameter %q: %w", parameter.Name, err)
	}
	return v.content(parameter.Content)
}

func (v *modelValidator) content(content map[string]MediaType) error {
	for mediaType, media := range content {
		if media.Schema != nil {
			if err := v.schema(media.Schema); err != nil {
				return fmt.Errorf("content %q: %w", mediaType, err)
			}
		}
		if err := v.examples(media.Examples); err != nil {
			return fmt.Errorf("content %q: %w", mediaType, err)
		}
		for name, encoding := range media.Encoding {
			for headerName, header := range encoding.Headers {
				if err := v.header(header); err != nil {
					return fmt.Errorf("encoding %q header %q: %w", name, headerName, err)
				}
			}
		}
	}
	return nil
}

func (v *modelValidator) examples(examples map[string]Example) error {
	for name, example := range examples {
		seen := make(map[string]bool)
		for {
			token, local := strings.CutPrefix(example.Ref, "#/components/examples/")
			if !local {
				break
			}
			if strings.Contains(token, "/") {
				return fmt.Errorf("invalid example reference %q", example.Ref)
			}
			target, err := decodePointerToken(token)
			if err != nil {
				return fmt.Errorf("invalid example reference %q: %w", example.Ref, err)
			}
			if seen[target] {
				return fmt.Errorf("cyclic example reference %q", target)
			}
			seen[target] = true
			var ok bool
			example, ok = v.api.Components.Examples[target]
			if !ok {
				return fmt.Errorf("example %q references missing example %q", name, target)
			}
		}
	}
	return nil
}

func (v *modelValidator) response(response Response) error {
	if err := v.content(response.Content); err != nil {
		return err
	}
	for name, header := range response.Headers {
		if err := v.header(header); err != nil {
			return fmt.Errorf("header %q: %w", name, err)
		}
	}
	for name, link := range response.Links {
		if err := v.link(&link); err != nil {
			return fmt.Errorf("link %q: %w", name, err)
		}
	}
	return nil
}

func (v *modelValidator) link(link *Link) error {
	current := *link
	seen := make(map[string]bool)
	for {
		token, local := strings.CutPrefix(current.Ref, "#/components/links/")
		if !local {
			break
		}
		if strings.Contains(token, "/") {
			return fmt.Errorf("invalid link reference %q", current.Ref)
		}
		name, err := decodePointerToken(token)
		if err != nil {
			return fmt.Errorf("invalid link reference %q: %w", current.Ref, err)
		}
		if seen[name] {
			return fmt.Errorf("cyclic link reference %q", name)
		}
		seen[name] = true
		var ok bool
		current, ok = v.api.Components.Links[name]
		if !ok {
			return fmt.Errorf("link %q not found", name)
		}
	}
	if current.OperationRef != "" && current.OperationID != "" {
		return fmt.Errorf("operationRef and operationId cannot both be set")
	}
	if current.Server != nil {
		return v.servers([]Server{*current.Server})
	}
	return nil
}

func (v *modelValidator) header(header Header) error {
	if header.Schema != nil {
		if err := v.schema(header.Schema); err != nil {
			return err
		}
	}
	return v.examples(header.Examples)
}

func (v *modelValidator) servers(servers []Server) error {
	for _, server := range servers {
		if err := v.security(server.Security); err != nil {
			return fmt.Errorf("server %q: %w", server.URL, err)
		}
	}
	return nil
}

func (v *modelValidator) security(requirements []SecurityRequirement) error {
	for _, requirement := range requirements {
		for name, scopes := range requirement {
			scheme, err := v.resolveSecurityScheme(name)
			if err != nil {
				return err
			}
			if len(scopes) > 0 && scheme.Ref == "" && scheme.Type != SecurityTypeOAuth2 && scheme.Type != SecurityTypeOpenIDConnect {
				return fmt.Errorf("security scheme %q does not support scopes", name)
			}
			if scheme.Ref != "" || scheme.Type != SecurityTypeOAuth2 || scheme.Flows == nil {
				continue
			}
			for _, scope := range scopes {
				if !hasOAuthScope(scheme.Flows, scope) {
					return fmt.Errorf("scope %q not found in security scheme %q", scope, name)
				}
			}
		}
	}
	return nil
}

func hasOAuthScope(flows *OAuthFlows, scope string) bool {
	for _, flow := range []*OAuthFlow{flows.Implicit, flows.Password, flows.ClientCredentials, flows.AuthorizationCode} {
		if flow == nil {
			continue
		}
		if _, ok := flow.Scopes[scope]; ok {
			return true
		}
	}
	return false
}

func (v *modelValidator) resolveSecurityScheme(name string) (SecurityScheme, error) {
	seen := make(map[string]bool)
	for {
		if seen[name] {
			return SecurityScheme{}, fmt.Errorf("cyclic security scheme reference %q", name)
		}
		seen[name] = true
		scheme, ok := v.api.Components.SecuritySchemes[name]
		if !ok {
			return SecurityScheme{}, fmt.Errorf("security scheme %q not found", name)
		}
		const prefix = "#/components/securitySchemes/"
		token, local := strings.CutPrefix(scheme.Ref, prefix)
		if !local {
			return scheme, nil
		}
		if strings.Contains(token, "/") {
			return SecurityScheme{}, fmt.Errorf("invalid security scheme reference %q", scheme.Ref)
		}
		decoded, err := decodePointerToken(token)
		if err != nil {
			return SecurityScheme{}, fmt.Errorf("invalid security scheme reference %q: %w", scheme.Ref, err)
		}
		name = decoded
	}
}
