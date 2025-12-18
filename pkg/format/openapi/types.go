package openapi

// OpenAPI represents the root of an OpenAPI v3 document.
type OpenAPI struct {
	OpenAPI      string                `yaml:"openapi" json:"openapi"`
	Info         Info                  `yaml:"info" json:"info"`
	Servers      []Server              `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths        map[string]PathItem   `yaml:"paths" json:"paths"`
	Webhooks     map[string]PathItem   `yaml:"webhooks,omitempty" json:"webhooks,omitempty"` // OpenAPI 3.1
	Components   Components            `yaml:"components,omitempty" json:"components,omitempty"`
	Security     []map[string][]string `yaml:"security,omitempty" json:"security,omitempty"`
	Tags         []Tag                 `yaml:"tags,omitempty" json:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
}

// Info object provides metadata about the API.
type Info struct {
	Title          string   `yaml:"title" json:"title"`
	Description    string   `yaml:"description,omitempty" json:"description,omitempty"`
	TermsOfService string   `yaml:"termsOfService,omitempty" json:"termsOfService,omitempty"`
	Contact        *Contact `yaml:"contact,omitempty" json:"contact,omitempty"`
	License        *License `yaml:"license,omitempty" json:"license,omitempty"`
	Version        string   `yaml:"version" json:"version"`
}

// Contact information for the exposed API.
type Contact struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}

// License information for the exposed API.
type License struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Server object represents a server.
type Server struct {
	URL         string                    `yaml:"url" json:"url"`
	Description string                    `yaml:"description,omitempty" json:"description,omitempty"`
	Variables   map[string]ServerVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// ServerVariable object represents a variable for server URL substitution.
type ServerVariable struct {
	Enum        []string `yaml:"enum,omitempty" json:"enum,omitempty"`
	Default     string   `yaml:"default" json:"default"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	Ref         string      `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Summary     string      `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Get         *Operation  `yaml:"get,omitempty" json:"get,omitempty"`
	Put         *Operation  `yaml:"put,omitempty" json:"put,omitempty"`
	Post        *Operation  `yaml:"post,omitempty" json:"post,omitempty"`
	Delete      *Operation  `yaml:"delete,omitempty" json:"delete,omitempty"`
	Options     *Operation  `yaml:"options,omitempty" json:"options,omitempty"`
	Head        *Operation  `yaml:"head,omitempty" json:"head,omitempty"`
	Patch       *Operation  `yaml:"patch,omitempty" json:"patch,omitempty"`
	Trace       *Operation  `yaml:"trace,omitempty" json:"trace,omitempty"`
	Servers     []Server    `yaml:"servers,omitempty" json:"servers,omitempty"`
	Parameters  []Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Tags         []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
	Summary      string                `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description  string                `yaml:"description,omitempty" json:"description,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	OperationID  string                `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters   []Parameter           `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody  *RequestBody          `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses    map[string]Response   `yaml:"responses" json:"responses"`
	Callbacks    map[string]PathItem   `yaml:"callbacks,omitempty" json:"callbacks,omitempty"` // simplified
	Deprecated   bool                  `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Security     []map[string][]string `yaml:"security,omitempty" json:"security,omitempty"`
	Servers      []Server              `yaml:"servers,omitempty" json:"servers,omitempty"`

	// V2 specific fields
	Consumes []string `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Produces []string `yaml:"produces,omitempty" json:"produces,omitempty"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name            string               `yaml:"name,omitempty" json:"name,omitempty"`
	In              string               `yaml:"in,omitempty" json:"in,omitempty"`
	Description     string               `yaml:"description,omitempty" json:"description,omitempty"`
	Required        bool                 `yaml:"required,omitempty" json:"required,omitempty"`
	Deprecated      bool                 `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	AllowEmptyValue bool                 `yaml:"allowEmptyValue,omitempty" json:"allowEmptyValue,omitempty"`
	Style           string               `yaml:"style,omitempty" json:"style,omitempty"`
	Explode         bool                 `yaml:"explode,omitempty" json:"explode,omitempty"`
	Schema          *Schema              `yaml:"schema,omitempty" json:"schema,omitempty"` // V3
	Example         interface{}          `yaml:"example,omitempty" json:"example,omitempty"`
	Content         map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Ref             string               `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	// V2 specific fields (valid for non-body parameters)
	Type   string  `yaml:"type,omitempty" json:"type,omitempty"`
	Format string  `yaml:"format,omitempty" json:"format,omitempty"`
	Items  *Schema `yaml:"items,omitempty" json:"items,omitempty"` // Reusing Schema for items definition
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Required    bool                 `yaml:"required,omitempty" json:"required,omitempty"`
	Ref         string               `yaml:"$ref,omitempty" json:"$ref,omitempty"`
}

// Response describes a single response from an API Operation.
type Response struct {
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Headers     map[string]Header    `yaml:"headers,omitempty" json:"headers,omitempty"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Links       map[string]Link      `yaml:"links,omitempty" json:"links,omitempty"`
	Ref         string               `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	// V2 specific fields
	Schema *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// MediaType provides schema and examples for the media type identified by its key.
type MediaType struct {
	Schema   *Schema                `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example  interface{}            `yaml:"example,omitempty" json:"example,omitempty"`
	Examples map[string]interface{} `yaml:"examples,omitempty" json:"examples,omitempty"`
}

// Schema allows the definition of input and output data types.
type Schema struct {
	Ref         string             `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Type        interface{}        `yaml:"type,omitempty" json:"type,omitempty"` // string or []string (3.1)
	Format      string             `yaml:"format,omitempty" json:"format,omitempty"`
	Title       string             `yaml:"title,omitempty" json:"title,omitempty"`
	Description string             `yaml:"description,omitempty" json:"description,omitempty"`
	Default     interface{}        `yaml:"default,omitempty" json:"default,omitempty"`
	Enum        []interface{}      `yaml:"enum,omitempty" json:"enum,omitempty"`
	Properties  map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	Items       *Schema            `yaml:"items,omitempty" json:"items,omitempty"`
	Required    []string           `yaml:"required,omitempty" json:"required,omitempty"`

	// Composition
	AllOf []*Schema `yaml:"allOf,omitempty" json:"allOf,omitempty"`
	AnyOf []*Schema `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	OneOf []*Schema `yaml:"oneOf,omitempty" json:"oneOf,omitempty"`
	Not   *Schema   `yaml:"not,omitempty" json:"not,omitempty"`
}

// Components holds a set of reusable objects.
type Components struct {
	Schemas         map[string]*Schema        `yaml:"schemas,omitempty" json:"schemas,omitempty"`
	Responses       map[string]Response       `yaml:"responses,omitempty" json:"responses,omitempty"`
	Parameters      map[string]Parameter      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Examples        map[string]interface{}    `yaml:"examples,omitempty" json:"examples,omitempty"`
	RequestBodies   map[string]RequestBody    `yaml:"requestBodies,omitempty" json:"requestBodies,omitempty"`
	Headers         map[string]Header         `yaml:"headers,omitempty" json:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme `yaml:"securitySchemes,omitempty" json:"securitySchemes,omitempty"`
}

// Header follows the structure of the Parameter Object.
type Header struct {
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool    `yaml:"required,omitempty" json:"required,omitempty"`
	Deprecated  bool    `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// SecurityScheme defines a security scheme that can be used by the operations.
type SecurityScheme struct {
	Type             string `yaml:"type" json:"type"`
	Description      string `yaml:"description,omitempty" json:"description,omitempty"`
	Name             string `yaml:"name,omitempty" json:"name,omitempty"`
	In               string `yaml:"in,omitempty" json:"in,omitempty"`
	Scheme           string `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	BearerFormat     string `yaml:"bearerFormat,omitempty" json:"bearerFormat,omitempty"`
	OpenIDConnectURL string `yaml:"openIdConnectUrl,omitempty" json:"openIdConnectUrl,omitempty"`
}

// Tag allows adding metadata to a single tag.
type Tag struct {
	Name         string        `yaml:"name" json:"name"`
	Description  string        `yaml:"description,omitempty" json:"description,omitempty"`
	ExternalDocs *ExternalDocs `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
}

// ExternalDocs allows referencing an external resource for extended documentation.
type ExternalDocs struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	URL         string `yaml:"url" json:"url"`
}

// Link represents a possible design-time link for a response.
type Link struct {
	OperationRef string `yaml:"operationRef,omitempty" json:"operationRef,omitempty"`
	OperationID  string `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Description  string `yaml:"description,omitempty" json:"description,omitempty"`
}
