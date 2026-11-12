package model

import (
	"fmt"
	"strings"
)

// SecurityScheme represents a security scheme object in the API model.
type SecurityScheme struct {
	Ref              string // OpenAPI reference to a security scheme
	Type             SecuritySchemeType
	Description      string
	Name             string // For apiKey
	In               string // For apiKey: query, header, cookie
	Scheme           string // For http: basic, bearer
	BearerFormat     string // For http bearer
	Flows            *OAuthFlows
	OpenIDConnectURL string
	Extensions       map[string]interface{}
}

// SecuritySchemeType defines the type of a security scheme.
type SecuritySchemeType string

const (
	// SecurityTypeAPIKey represents an API Key security scheme.
	SecurityTypeAPIKey SecuritySchemeType = "apiKey"
	// SecurityTypeHTTP represents an HTTP security scheme.
	SecurityTypeHTTP SecuritySchemeType = "http"
	// SecurityTypeOAuth2 represents an OAuth2 security scheme.
	SecurityTypeOAuth2 SecuritySchemeType = "oauth2"
	// SecurityTypeOpenIDConnect represents an OpenID Connect security scheme.
	SecurityTypeOpenIDConnect SecuritySchemeType = "openIdConnect"
)

// OAuthFlows contains details about the OAuth2 flows supported.
type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
	Extensions        map[string]interface{}
}

// OAuthFlow contains details about a specific OAuth2 flow.
type OAuthFlow struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	Scopes           map[string]string
	Extensions       map[string]interface{}
}

// SecurityRequirement is a map of security scheme names to a list of scopes required for the operation.
type SecurityRequirement map[string][]string

// NewSecurityScheme creates a security scheme of the given type.
func NewSecurityScheme(typ SecuritySchemeType) *SecurityScheme {
	return &SecurityScheme{
		Type: typ,
	}
}

// Validate checks the required fields for the security scheme and its OAuth flows.
func (s *SecurityScheme) Validate() error {
	if s == nil {
		return fmt.Errorf("security scheme is nil")
	}
	if s.Ref != "" {
		return nil
	}
	switch s.Type {
	case SecurityTypeAPIKey:
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("apiKey requires a name")
		}
		if s.In != "query" && s.In != "header" && s.In != "cookie" {
			return fmt.Errorf("apiKey location must be query, header, or cookie")
		}
	case SecurityTypeHTTP:
		if strings.TrimSpace(s.Scheme) == "" {
			return fmt.Errorf("HTTP security requires a scheme")
		}
	case SecurityTypeOpenIDConnect:
		if strings.TrimSpace(s.OpenIDConnectURL) == "" {
			return fmt.Errorf("OpenID Connect requires an openIdConnectUrl")
		}
	case SecurityTypeOAuth2:
		if s.Flows == nil {
			return fmt.Errorf("OAuth2 requires at least one flow")
		}
		flows := []struct {
			name                 string
			flow                 *OAuthFlow
			authorization, token bool
		}{
			{"implicit", s.Flows.Implicit, true, false},
			{"password", s.Flows.Password, false, true},
			{"clientCredentials", s.Flows.ClientCredentials, false, true},
			{"authorizationCode", s.Flows.AuthorizationCode, true, true},
		}
		found := false
		for _, entry := range flows {
			if entry.flow == nil {
				continue
			}
			found = true
			if entry.authorization && strings.TrimSpace(entry.flow.AuthorizationURL) == "" {
				return fmt.Errorf("OAuth2 %s flow requires an authorization URL", entry.name)
			}
			if entry.token && strings.TrimSpace(entry.flow.TokenURL) == "" {
				return fmt.Errorf("OAuth2 %s flow requires a token URL", entry.name)
			}
			if entry.flow.Scopes == nil {
				return fmt.Errorf("OAuth2 %s flow requires scopes", entry.name)
			}
		}
		if !found {
			return fmt.Errorf("OAuth2 requires at least one flow")
		}
	case "mutualTLS":
		// OpenAPI 3.1 mutual TLS has no additional required fields.
	default:
		return fmt.Errorf("unsupported security scheme type %q", s.Type)
	}
	return nil
}

// AddRequirement sets the scopes for a security scheme.
func (s *SecurityRequirement) AddRequirement(name string, scopes ...string) {
	(*s)[name] = scopes
}
