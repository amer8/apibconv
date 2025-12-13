package model

// Schema represents an OpenAPI Schema object.
type Schema struct {
	// Type and format
	Type   SchemaType `json:"type,omitempty" yaml:"type,omitempty"`
	Format string     `json:"format,omitempty" yaml:"format,omitempty"`

	// Constraints
	Title       string      `json:"title,omitempty" yaml:"title,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`

	// String constraints
	MinLength *int   `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	// Number constraints
	Minimum          *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMinimum bool     `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum bool     `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`

	// Array constraints
	Items       *Schema `json:"items,omitempty" yaml:"items,omitempty"`
	MinItems    *int    `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems    *int    `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	UniqueItems bool    `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

	// Object constraints
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"` // bool or *Schema
	MinProperties        *int               `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties        *int               `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`

	// Composition
	AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Not   *Schema   `json:"not,omitempty" yaml:"not,omitempty"`

	// Reference
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// Validation
	Enum []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Metadata
	Nullable   bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly   bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly  bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// Discriminator for polymorphism
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`

	// External documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// SchemaType defines the type of a schema.
type SchemaType string

const (
	// TypeString indicates a string schema type.
	TypeString  SchemaType = "string"
	// TypeNumber indicates a number schema type.
	TypeNumber  SchemaType = "number"
	// TypeInteger indicates an integer schema type.
	TypeInteger SchemaType = "integer"
	// TypeBoolean indicates a boolean schema type.
	TypeBoolean SchemaType = "boolean"
	// TypeArray indicates an array schema type.
	TypeArray   SchemaType = "array"
	// TypeObject indicates an object schema type.
	TypeObject  SchemaType = "object"
)

// Discriminator object for polymorphism.
type Discriminator struct {
	PropertyName string            `json:"propertyName,omitempty" yaml:"propertyName,omitempty"`
	Mapping      map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// Components holds a set of reusable objects for different aspects of the API.
type Components struct {
	Schemas         map[string]*Schema
	Responses       map[string]Response
	Parameters      map[string]Parameter
	Examples        map[string]Example
	RequestBodies   map[string]RequestBody
	Headers         map[string]Header
	SecuritySchemes map[string]SecurityScheme
	Links           map[string]Link
	Callbacks       map[string]Callback
}

// NewComponents creates a new Components instance
func NewComponents() Components {
	return Components{
		Schemas:         make(map[string]*Schema),
		Responses:       make(map[string]Response),
		Parameters:      make(map[string]Parameter),
		Examples:        make(map[string]Example),
		RequestBodies:   make(map[string]RequestBody),
		Headers:         make(map[string]Header),
		SecuritySchemes: make(map[string]SecurityScheme),
		Links:           make(map[string]Link),
		Callbacks:       make(map[string]Callback),
	}
}

// NewSchema creates a new Schema of the given type
func NewSchema(typ SchemaType) *Schema {
	return &Schema{
		Type:       typ,
		Properties: make(map[string]*Schema),
	}
}

// AddProperty adds a property to the schema (assuming it's an object)
func (s *Schema) AddProperty(name string, schema *Schema) {
	if s.Properties == nil {
		s.Properties = make(map[string]*Schema)
	}
	s.Properties[name] = schema
}

// SetRequired sets the required fields
func (s *Schema) SetRequired(fields ...string) {
	s.Required = fields
}

// Validate validates the schema
func (s *Schema) Validate() error {
	// Validation is primarily handled by the `validator` package.
	return nil
}

// ResolveRef resolves a reference within the components. Full implementation is pending.
func (s *Schema) ResolveRef(components *Components) (*Schema, error) {
	return nil, nil
}

// AddSchema adds a schema to components
func (c *Components) AddSchema(name string, schema *Schema) {
	if c.Schemas == nil {
		c.Schemas = make(map[string]*Schema)
	}
	c.Schemas[name] = schema
}

// GetSchema retrieves a schema from components
func (c *Components) GetSchema(name string) (*Schema, bool) {
	s, ok := c.Schemas[name]
	return s, ok
}