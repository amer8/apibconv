package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

// MarshalJSON preserves the distinction between inherited and explicitly empty security.
//
//nolint:gocritic // A value receiver preserves JSON marshaling for both values and pointers.
func (op Operation) MarshalJSON() ([]byte, error) {
	type operation Operation
	var security *[]map[string][]string
	if op.Security != nil {
		security = &op.Security
	}
	data, err := marshalExtensionsJSON(struct {
		operation
		Security *[]map[string][]string `json:"security,omitempty"`
	}{operation: operation(op), Security: security}, op.Extensions)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsJSON(data, "responses", op.ResponsesExtensions)
}

// MarshalYAML preserves the distinction between inherited and explicitly empty security.
//
//nolint:gocritic // A value receiver preserves YAML marshaling for both values and pointers.
func (op Operation) MarshalYAML() (interface{}, error) {
	type operation Operation
	node, err := marshalSecurityYAML(operation(op), op.Security)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsYAML(node, "responses", op.ResponsesExtensions)
}

// MarshalJSON preserves an explicitly empty global security array.
//
//nolint:gocritic // A value receiver preserves JSON marshaling for both values and pointers.
func (doc OpenAPI) MarshalJSON() ([]byte, error) {
	type openAPI OpenAPI
	var security *[]map[string][]string
	if doc.Security != nil {
		security = &doc.Security
	}
	data, err := marshalExtensionsJSON(struct {
		openAPI
		Security *[]map[string][]string `json:"security,omitempty"`
	}{openAPI: openAPI(doc), Security: security}, doc.Extensions)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsJSON(data, "paths", doc.PathsExtensions)
}

// MarshalYAML preserves an explicitly empty global security array.
//
//nolint:gocritic // A value receiver preserves YAML marshaling for both values and pointers.
func (doc OpenAPI) MarshalYAML() (interface{}, error) {
	type openAPI OpenAPI
	node, err := marshalSecurityYAML(openAPI(doc), doc.Security)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsYAML(node, "paths", doc.PathsExtensions)
}

// MarshalJSON preserves an explicitly empty global security array.
//
//nolint:gocritic // A value receiver preserves JSON marshaling for both values and pointers.
func (doc Swagger20) MarshalJSON() ([]byte, error) {
	type swagger Swagger20
	var security *[]map[string][]string
	if doc.Security != nil {
		security = &doc.Security
	}
	data, err := marshalExtensionsJSON(struct {
		swagger
		Security *[]map[string][]string `json:"security,omitempty"`
	}{swagger: swagger(doc), Security: security}, doc.Extensions)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsJSON(data, "paths", doc.PathsExtensions)
}

// MarshalYAML preserves an explicitly empty global security array.
//
//nolint:gocritic // A value receiver preserves YAML marshaling for both values and pointers.
func (doc Swagger20) MarshalYAML() (interface{}, error) {
	type swagger Swagger20
	node, err := marshalSecurityYAML(swagger(doc), doc.Security)
	if err != nil {
		return nil, err
	}
	return addMapExtensionsYAML(node, "paths", doc.PathsExtensions)
}

func marshalExtensionsJSON(value interface{}, extensions map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil || len(extensions) == 0 {
		return data, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	for key, value := range extensions {
		if !strings.HasPrefix(key, "x-") {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("extension %q: %w", key, err)
		}
		fields[key] = encoded
	}
	return json.Marshal(fields)
}

func marshalSecurityYAML(value interface{}, security []map[string][]string) (interface{}, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	if security != nil && len(security) == 0 {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "security"},
			&yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
		)
	}
	return &node, nil
}

func securityRequirementsToModel(requirements []map[string][]string) []model.SecurityRequirement {
	if requirements == nil {
		return nil
	}
	result := make([]model.SecurityRequirement, len(requirements))
	for i, requirement := range requirements {
		result[i] = model.SecurityRequirement(requirement)
	}
	return result
}

func securityRequirementsFromModel(requirements []model.SecurityRequirement) []map[string][]string {
	if requirements == nil {
		return nil
	}
	result := make([]map[string][]string, len(requirements))
	for i, requirement := range requirements {
		result[i] = make(map[string][]string, len(requirement))
		for name, scopes := range requirement {
			if scopes == nil {
				scopes = []string{}
			}
			result[i][name] = scopes
		}
	}
	return result
}

func securitySchemeToModel(s *SecurityScheme) model.SecurityScheme {
	result := model.SecurityScheme{
		Ref: s.Ref, Type: model.SecuritySchemeType(s.Type), Description: s.Description,
		Name: s.Name, In: s.In, Scheme: s.Scheme, BearerFormat: s.BearerFormat,
		OpenIDConnectURL: s.OpenIDConnectURL, Extensions: clonedExtensions(s.Extensions),
	}
	if s.Flows != nil {
		result.Flows = &model.OAuthFlows{
			Implicit: flowToModel(s.Flows.Implicit), Password: flowToModel(s.Flows.Password),
			ClientCredentials: flowToModel(s.Flows.ClientCredentials), AuthorizationCode: flowToModel(s.Flows.AuthorizationCode),
			Extensions: clonedExtensions(s.Flows.Extensions),
		}
	}
	return result
}

func flowToModel(f *OAuthFlow) *model.OAuthFlow {
	if f == nil {
		return nil
	}
	return &model.OAuthFlow{AuthorizationURL: f.AuthorizationURL, TokenURL: f.TokenURL, RefreshURL: f.RefreshURL, Scopes: f.Scopes, Extensions: clonedExtensions(f.Extensions)}
}

func securitySchemeFromModel(s *model.SecurityScheme) SecurityScheme {
	result := SecurityScheme{
		Ref: s.Ref, Type: string(s.Type), Description: s.Description,
		Name: s.Name, In: s.In, Scheme: s.Scheme, BearerFormat: s.BearerFormat,
		OpenIDConnectURL: s.OpenIDConnectURL, Extensions: clonedExtensions(s.Extensions),
	}
	if s.Flows != nil {
		result.Flows = &OAuthFlows{
			Implicit: flowFromModel(s.Flows.Implicit), Password: flowFromModel(s.Flows.Password),
			ClientCredentials: flowFromModel(s.Flows.ClientCredentials), AuthorizationCode: flowFromModel(s.Flows.AuthorizationCode),
			Extensions: clonedExtensions(s.Flows.Extensions),
		}
	}
	return result
}

func flowFromModel(f *model.OAuthFlow) *OAuthFlow {
	if f == nil {
		return nil
	}
	return &OAuthFlow{AuthorizationURL: f.AuthorizationURL, TokenURL: f.TokenURL, RefreshURL: f.RefreshURL, Scopes: f.Scopes, Extensions: clonedExtensions(f.Extensions)}
}

func securitySchemeV2ToModel(s *SecurityScheme) (model.SecurityScheme, error) {
	result := securitySchemeToModel(s)
	switch s.Type {
	case "basic":
		result.Type = model.SecurityTypeHTTP
		result.Scheme = "basic"
	case "oauth2":
		flow := &model.OAuthFlow{AuthorizationURL: s.AuthorizationURL, TokenURL: s.TokenURL}
		if s.Scopes != nil {
			flow.Scopes = *s.Scopes
		}
		result.Flows = &model.OAuthFlows{}
		switch s.Flow {
		case "implicit":
			result.Flows.Implicit = flow
		case "password":
			result.Flows.Password = flow
		case "application":
			result.Flows.ClientCredentials = flow
		case "accessCode":
			result.Flows.AuthorizationCode = flow
		default:
			return result, fmt.Errorf("unsupported Swagger 2.0 OAuth flow %q", s.Flow)
		}
	}
	return result, nil
}

func securitySchemeToV2(s *model.SecurityScheme) (SecurityScheme, error) {
	result := SecurityScheme{Description: s.Description, Extensions: clonedExtensions(s.Extensions)}
	if s.Ref != "" {
		return result, fmt.Errorf("security scheme reference %q cannot be represented in Swagger 2.0", s.Ref)
	}
	switch s.Type {
	case model.SecurityTypeAPIKey:
		if s.In != "header" && s.In != "query" {
			return result, fmt.Errorf("apiKey location %q cannot be represented in Swagger 2.0", s.In)
		}
		result.Type, result.Name, result.In = "apiKey", s.Name, s.In
	case model.SecurityTypeHTTP:
		if !strings.EqualFold(s.Scheme, "basic") || s.BearerFormat != "" {
			return result, fmt.Errorf("HTTP security scheme %q cannot be represented in Swagger 2.0", s.Scheme)
		}
		result.Type = "basic"
	case model.SecurityTypeOAuth2:
		if s.Flows == nil {
			return result, fmt.Errorf("OAuth2 scheme has no flow")
		}
		var selected *model.OAuthFlow
		for _, candidate := range []struct {
			name string
			flow *model.OAuthFlow
		}{
			{"implicit", s.Flows.Implicit}, {"password", s.Flows.Password},
			{"application", s.Flows.ClientCredentials}, {"accessCode", s.Flows.AuthorizationCode},
		} {
			if candidate.flow == nil {
				continue
			}
			if selected != nil {
				return result, fmt.Errorf("multiple OAuth2 flows cannot be represented in a single Swagger 2.0 security definition")
			}
			result.Flow, selected = candidate.name, candidate.flow
		}
		if selected == nil {
			return result, fmt.Errorf("OAuth2 scheme has no flow")
		}
		if selected.RefreshURL != "" {
			return result, fmt.Errorf("OAuth2 refreshUrl cannot be represented in Swagger 2.0")
		}
		result.Type, result.AuthorizationURL, result.TokenURL = "oauth2", selected.AuthorizationURL, selected.TokenURL
		scopes := selected.Scopes
		if scopes == nil {
			scopes = map[string]string{}
		}
		result.Scopes = &scopes
	default:
		return result, fmt.Errorf("security scheme type %q cannot be represented in Swagger 2.0", s.Type)
	}
	return result, nil
}
