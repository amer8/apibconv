package model

// SecurityScheme represents a security scheme object in the API model.
type SecurityScheme struct {
	Type             SecuritySchemeType
	Description      string
	Name             string // For apiKey
	In               string // For apiKey: query, header, cookie
	Scheme           string // For http: basic, bearer
	BearerFormat     string // For http bearer
	Flows            *OAuthFlows
	OpenIDConnectURL string
}

// SecuritySchemeType defines the type of a security scheme.
type SecuritySchemeType string

const (
	// SecurityTypeAPIKey represents an API Key security scheme.
	SecurityTypeAPIKey        SecuritySchemeType = "apiKey"
	// SecurityTypeHTTP represents an HTTP security scheme.
	SecurityTypeHTTP          SecuritySchemeType = "http"
	// SecurityTypeOAuth2 represents an OAuth2 security scheme.
	SecurityTypeOAuth2        SecuritySchemeType = "oauth2"
	// SecurityTypeOpenIDConnect represents an OpenID Connect security scheme.
	SecurityTypeOpenIDConnect SecuritySchemeType = "openIdConnect"
)

// OAuthFlows contains details about the OAuth2 flows supported.
type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
}

// OAuthFlow contains details about a specific OAuth2 flow.
type OAuthFlow struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	Scopes           map[string]string
}

// SecurityRequirement is a map of security scheme names to a list of scopes required for the operation.
type SecurityRequirement map[string][]string

// NewSecurityScheme creates a new security scheme
func NewSecurityScheme(typ SecuritySchemeType) *SecurityScheme {
	return &SecurityScheme{
		Type: typ,
	}
}

// Validate validates the security scheme
func (s *SecurityScheme) Validate() error {
	// TODO: Validation is primarily handled by the `validator` package.
	return nil
}

// AddRequirement adds a requirement to the security requirement map
func (s *SecurityRequirement) AddRequirement(name string, scopes ...string) {
	(*s)[name] = scopes
}
