package openapi

import (
	"fmt"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

// OpenAPI represents the root of an OpenAPI v3 document.
type OpenAPI struct {
	OpenAPI         string                 `yaml:"openapi" json:"openapi"`
	Info            Info                   `yaml:"info" json:"info"`
	Servers         []Server               `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths           map[string]PathItem    `yaml:"paths" json:"paths"`
	Webhooks        map[string]PathItem    `yaml:"webhooks,omitempty" json:"webhooks,omitempty"` // OpenAPI 3.1
	Components      Components             `yaml:"components,omitempty" json:"components,omitempty"`
	Security        []map[string][]string  `yaml:"security,omitempty" json:"security,omitempty"`
	Tags            []Tag                  `yaml:"tags,omitempty" json:"tags,omitempty"`
	ExternalDocs    *ExternalDocs          `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	Extensions      map[string]interface{} `yaml:",inline" json:"-"`
	PathsExtensions map[string]interface{} `yaml:"-" json:"-"`
}

// Info object provides metadata about the API.
type Info struct {
	Title          string                 `yaml:"title" json:"title"`
	Description    string                 `yaml:"description,omitempty" json:"description,omitempty"`
	TermsOfService string                 `yaml:"termsOfService,omitempty" json:"termsOfService,omitempty"`
	Contact        *Contact               `yaml:"contact,omitempty" json:"contact,omitempty"`
	License        *License               `yaml:"license,omitempty" json:"license,omitempty"`
	Version        string                 `yaml:"version" json:"version"`
	Extensions     map[string]interface{} `yaml:",inline" json:"-"`
}

// Contact information for the exposed API.
type Contact struct {
	Name       string                 `yaml:"name,omitempty" json:"name,omitempty"`
	URL        string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Email      string                 `yaml:"email,omitempty" json:"email,omitempty"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
}

// License information for the exposed API.
type License struct {
	Name       string                 `yaml:"name" json:"name"`
	URL        string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
}

// Server object represents a server.
type Server struct {
	URL         string                    `yaml:"url" json:"url"`
	Description string                    `yaml:"description,omitempty" json:"description,omitempty"`
	Variables   map[string]ServerVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
	Extensions  map[string]interface{}    `yaml:",inline" json:"-"`
}

// ServerVariable object represents a variable for server URL substitution.
type ServerVariable struct {
	Enum        []string               `yaml:"enum,omitempty" json:"enum,omitempty"`
	Default     string                 `yaml:"default" json:"default"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Extensions  map[string]interface{} `yaml:",inline" json:"-"`
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	Ref         string                 `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Summary     string                 `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Get         *Operation             `yaml:"get,omitempty" json:"get,omitempty"`
	Put         *Operation             `yaml:"put,omitempty" json:"put,omitempty"`
	Post        *Operation             `yaml:"post,omitempty" json:"post,omitempty"`
	Delete      *Operation             `yaml:"delete,omitempty" json:"delete,omitempty"`
	Options     *Operation             `yaml:"options,omitempty" json:"options,omitempty"`
	Head        *Operation             `yaml:"head,omitempty" json:"head,omitempty"`
	Patch       *Operation             `yaml:"patch,omitempty" json:"patch,omitempty"`
	Trace       *Operation             `yaml:"trace,omitempty" json:"trace,omitempty"`
	Servers     []Server               `yaml:"servers,omitempty" json:"servers,omitempty"`
	Parameters  []Parameter            `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Extensions  map[string]interface{} `yaml:",inline" json:"-"`
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
	Callbacks    map[string]Callback   `yaml:"callbacks,omitempty" json:"callbacks,omitempty"`
	Deprecated   bool                  `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Security     []map[string][]string `yaml:"security,omitempty" json:"security,omitempty"`
	Servers      []Server              `yaml:"servers,omitempty" json:"servers,omitempty"`

	// V2 specific fields
	Consumes            []string               `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Produces            []string               `yaml:"produces,omitempty" json:"produces,omitempty"`
	Extensions          map[string]interface{} `yaml:",inline" json:"-"`
	ResponsesExtensions map[string]interface{} `yaml:"-" json:"-"`
}

// Callback is either a reference or a map of runtime expressions to path items.
type Callback struct {
	Ref        string
	Paths      map[string]PathItem
	Extensions map[string]interface{}
}

// UnmarshalYAML reads an inline callback or a reference.
func (c *Callback) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("callback must be an object")
	}
	wrapper := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "callback"}, node}}
	clean, extensions, err := splitMapExtensions(wrapper, "callback")
	if err != nil {
		return err
	}
	node = clean.Content[1]
	c.Extensions = extensions
	if len(node.Content) >= 2 && node.Content[0].Value == "$ref" {
		if len(node.Content) != 2 {
			return fmt.Errorf("callback reference has unsupported siblings")
		}
		return node.Content[1].Decode(&c.Ref)
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "$ref" {
			return fmt.Errorf("callback reference has unsupported siblings")
		}
	}
	return node.Decode(&c.Paths)
}

// MarshalJSON emits a callback object.
func (c Callback) MarshalJSON() ([]byte, error) {
	if c.Ref != "" {
		return marshalExtensionsJSON(map[string]string{"$ref": c.Ref}, c.Extensions)
	}
	return marshalExtensionsJSON(c.Paths, c.Extensions)
}

// MarshalYAML emits a callback object.
func (c Callback) MarshalYAML() (interface{}, error) {
	result := make(map[string]interface{}, len(c.Paths)+len(c.Extensions)+1)
	if c.Ref != "" {
		result["$ref"] = c.Ref
	} else {
		for key := range c.Paths {
			result[key] = c.Paths[key]
		}
	}
	copyExtensions(result, c.Extensions)
	return result, nil
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
	Explode         *bool                `yaml:"explode,omitempty" json:"explode,omitempty"`
	Schema          *Schema              `yaml:"schema,omitempty" json:"schema,omitempty"` // V3
	Example         interface{}          `yaml:"example,omitempty" json:"example,omitempty"`
	Content         map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Ref             string               `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	// V2 specific fields (valid for non-body parameters)
	Type             string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Format           string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Items            *Schema                `yaml:"items,omitempty" json:"items,omitempty"` // Reusing Schema for items definition
	CollectionFormat string                 `yaml:"collectionFormat,omitempty" json:"collectionFormat,omitempty"`
	Default          interface{}            `yaml:"default,omitempty" json:"default,omitempty"`
	Enum             []interface{}          `yaml:"enum,omitempty" json:"enum,omitempty"`
	Minimum          *float64               `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum          *float64               `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	ExclusiveMinimum bool                   `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum bool                   `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64               `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`
	MinLength        *int                   `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength        *int                   `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	Pattern          string                 `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MinItems         *int                   `yaml:"minItems,omitempty" json:"minItems,omitempty"`
	MaxItems         *int                   `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`
	UniqueItems      bool                   `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`
	Extensions       map[string]interface{} `yaml:",inline" json:"-"`
	Examples         map[string]interface{} `yaml:"examples,omitempty" json:"examples,omitempty"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]MediaType   `yaml:"content,omitempty" json:"content,omitempty"`
	Required    bool                   `yaml:"required,omitempty" json:"required,omitempty"`
	Ref         string                 `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Extensions  map[string]interface{} `yaml:",inline" json:"-"`
}

// Response describes a single response from an API Operation.
type Response struct {
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Headers     map[string]Header    `yaml:"headers,omitempty" json:"headers,omitempty"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Links       map[string]Link      `yaml:"links,omitempty" json:"links,omitempty"`
	Ref         string               `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	// V2 specific fields
	Schema     *Schema                `yaml:"schema,omitempty" json:"schema,omitempty"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
}

// MediaType provides schema and examples for the media type identified by its key.
type MediaType struct {
	Schema     *Schema                `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example    interface{}            `yaml:"example,omitempty" json:"example,omitempty"`
	Examples   map[string]interface{} `yaml:"examples,omitempty" json:"examples,omitempty"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
	Encoding   map[string]Encoding    `yaml:"encoding,omitempty" json:"encoding,omitempty"`
}

// Example is an inline example or a reference to a reusable example.
type Example struct {
	Ref           string                 `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Summary       string                 `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description   string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Value         interface{}            `yaml:"value,omitempty" json:"value,omitempty"`
	ExternalValue string                 `yaml:"externalValue,omitempty" json:"externalValue,omitempty"`
	Extensions    map[string]interface{} `yaml:",inline" json:"-"`
}

// Encoding describes how a request body property is encoded.
type Encoding struct {
	ContentType   string                 `yaml:"contentType,omitempty" json:"contentType,omitempty"`
	Headers       map[string]Header      `yaml:"headers,omitempty" json:"headers,omitempty"`
	Style         string                 `yaml:"style,omitempty" json:"style,omitempty"`
	Explode       bool                   `yaml:"explode,omitempty" json:"explode,omitempty"`
	AllowReserved bool                   `yaml:"allowReserved,omitempty" json:"allowReserved,omitempty"`
	Extensions    map[string]interface{} `yaml:",inline" json:"-"`
}

// Schema allows the definition of input and output data types.
type Schema struct {
	Ref                  string             `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Type                 interface{}        `yaml:"type,omitempty" json:"type,omitempty"` // string or []string (3.1)
	Format               string             `yaml:"format,omitempty" json:"format,omitempty"`
	Title                string             `yaml:"title,omitempty" json:"title,omitempty"`
	Description          string             `yaml:"description,omitempty" json:"description,omitempty"`
	Default              interface{}        `yaml:"default,omitempty" json:"default,omitempty"`
	Enum                 []interface{}      `yaml:"enum,omitempty" json:"enum,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	Items                *Schema            `yaml:"items,omitempty" json:"items,omitempty"`
	Required             []string           `yaml:"required,omitempty" json:"required,omitempty"`
	MinLength            *int               `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength            *int               `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	Pattern              string             `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Minimum              *float64           `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum              *float64           `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	MultipleOf           *float64           `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`
	MinItems             *int               `yaml:"minItems,omitempty" json:"minItems,omitempty"`
	MaxItems             *int               `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`
	UniqueItems          bool               `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`
	MinProperties        *int               `yaml:"minProperties,omitempty" json:"minProperties,omitempty"`
	MaxProperties        *int               `yaml:"maxProperties,omitempty" json:"maxProperties,omitempty"`
	AdditionalProperties interface{}        `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
	ExclusiveMinimum     interface{}        `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     interface{}        `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"`
	Example              interface{}        `yaml:"example,omitempty" json:"example,omitempty"`
	ReadOnly             bool               `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
	WriteOnly            bool               `yaml:"writeOnly,omitempty" json:"writeOnly,omitempty"`
	Deprecated           bool               `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Nullable             bool               `yaml:"nullable,omitempty" json:"nullable,omitempty"`

	// Composition
	AllOf         []*Schema              `yaml:"allOf,omitempty" json:"allOf,omitempty"`
	AnyOf         []*Schema              `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	OneOf         []*Schema              `yaml:"oneOf,omitempty" json:"oneOf,omitempty"`
	Not           *Schema                `yaml:"not,omitempty" json:"not,omitempty"`
	Extensions    map[string]interface{} `yaml:",inline" json:"-"`
	ExternalDocs  *ExternalDocs          `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	Discriminator *Discriminator         `yaml:"discriminator,omitempty" json:"discriminator,omitempty"`
}

// Discriminator identifies the property used to select a schema.
type Discriminator struct {
	PropertyName string                 `yaml:"propertyName" json:"propertyName"`
	Mapping      map[string]string      `yaml:"mapping,omitempty" json:"mapping,omitempty"`
	Extensions   map[string]interface{} `yaml:",inline" json:"-"`
}

// UnmarshalYAML rejects schema keywords that the model cannot preserve.
// JSON Schema's const is representable as an intersecting single-value enum.
func (s *Schema) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("schema must be an object")
	}
	known := make(map[string]bool)
	typ := reflect.TypeOf(Schema{})
	for i := 0; i < typ.NumField(); i++ {
		known[strings.Split(typ.Field(i).Tag.Get("yaml"), ",")[0]] = true
	}
	var constNode *yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if key == "const" {
			constNode = value
			continue
		}
		if !known[key] && !strings.HasPrefix(key, "x-") {
			return fmt.Errorf("unsupported schema keyword %q", key)
		}
		if key == "additionalProperties" && value.Kind == yaml.MappingNode {
			var nested Schema
			if err := value.Decode(&nested); err != nil {
				return err
			}
		}
	}
	type plain Schema
	var decoded plain
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	*s = Schema(decoded)
	if constNode != nil {
		var value interface{}
		if err := constNode.Decode(&value); err != nil {
			return err
		}
		s.AllOf = append(s.AllOf, &Schema{Enum: []interface{}{value}})
	}
	return nil
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
	Callbacks       map[string]Callback       `yaml:"callbacks,omitempty" json:"callbacks,omitempty"`
	Extensions      map[string]interface{}    `yaml:",inline" json:"-"`
	Links           map[string]Link           `yaml:"links,omitempty" json:"links,omitempty"`
}

// Header follows the structure of the Parameter Object.
type Header struct {
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool    `yaml:"required,omitempty" json:"required,omitempty"`
	Deprecated  bool    `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`

	// V2 specific fields
	Type       string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Format     string                 `yaml:"format,omitempty" json:"format,omitempty"`
	Items      *Schema                `yaml:"items,omitempty" json:"items,omitempty"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
	Example    interface{}            `yaml:"example,omitempty" json:"example,omitempty"`
	Examples   map[string]interface{} `yaml:"examples,omitempty" json:"examples,omitempty"`
}

// SecurityScheme defines a security scheme that can be used by the operations.
type SecurityScheme struct {
	Ref              string      `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Type             string      `yaml:"type,omitempty" json:"type,omitempty"`
	Description      string      `yaml:"description,omitempty" json:"description,omitempty"`
	Name             string      `yaml:"name,omitempty" json:"name,omitempty"`
	In               string      `yaml:"in,omitempty" json:"in,omitempty"`
	Scheme           string      `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	BearerFormat     string      `yaml:"bearerFormat,omitempty" json:"bearerFormat,omitempty"`
	OpenIDConnectURL string      `yaml:"openIdConnectUrl,omitempty" json:"openIdConnectUrl,omitempty"`
	Flows            *OAuthFlows `yaml:"flows,omitempty" json:"flows,omitempty"`
	// Swagger 2.0 represents a single OAuth flow directly on the scheme.
	Flow             string                 `yaml:"flow,omitempty" json:"flow,omitempty"`
	AuthorizationURL string                 `yaml:"authorizationUrl,omitempty" json:"authorizationUrl,omitempty"`
	TokenURL         string                 `yaml:"tokenUrl,omitempty" json:"tokenUrl,omitempty"`
	Scopes           *map[string]string     `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	Extensions       map[string]interface{} `yaml:",inline" json:"-"`
}

// OAuthFlows contains the OAuth 2.0 flows available in an OpenAPI 3 security scheme.
type OAuthFlows struct {
	Implicit          *OAuthFlow             `yaml:"implicit,omitempty" json:"implicit,omitempty"`
	Password          *OAuthFlow             `yaml:"password,omitempty" json:"password,omitempty"`
	ClientCredentials *OAuthFlow             `yaml:"clientCredentials,omitempty" json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow             `yaml:"authorizationCode,omitempty" json:"authorizationCode,omitempty"`
	Extensions        map[string]interface{} `yaml:",inline" json:"-"`
}

// OAuthFlow describes an OAuth 2.0 flow.
type OAuthFlow struct {
	AuthorizationURL string                 `yaml:"authorizationUrl,omitempty" json:"authorizationUrl,omitempty"`
	TokenURL         string                 `yaml:"tokenUrl,omitempty" json:"tokenUrl,omitempty"`
	RefreshURL       string                 `yaml:"refreshUrl,omitempty" json:"refreshUrl,omitempty"`
	Scopes           map[string]string      `yaml:"scopes" json:"scopes"`
	Extensions       map[string]interface{} `yaml:",inline" json:"-"`
}

// Tag allows adding metadata to a single tag.
type Tag struct {
	Name         string                 `yaml:"name" json:"name"`
	Description  string                 `yaml:"description,omitempty" json:"description,omitempty"`
	ExternalDocs *ExternalDocs          `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	Extensions   map[string]interface{} `yaml:",inline" json:"-"`
}

// ExternalDocs allows referencing an external resource for extended documentation.
type ExternalDocs struct {
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	URL         string                 `yaml:"url" json:"url"`
	Extensions  map[string]interface{} `yaml:",inline" json:"-"`
}

// Link represents a possible design-time link for a response.
type Link struct {
	OperationRef string                 `yaml:"operationRef,omitempty" json:"operationRef,omitempty"`
	OperationID  string                 `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Description  string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Ref          string                 `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Parameters   map[string]interface{} `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody  interface{}            `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Server       *Server                `yaml:"server,omitempty" json:"server,omitempty"`
	Extensions   map[string]interface{} `yaml:",inline" json:"-"`
}
