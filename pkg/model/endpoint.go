package model

// PathItem represents an OpenAPI Path Item object.
type PathItem struct {
	Name        string // Original name/key if available (e.g. AsyncAPI channel name)
	Summary     string
	Description string
	Get         *Operation
	Put         *Operation
	Post        *Operation
	Delete      *Operation
	Options     *Operation
	Head        *Operation
	Patch       *Operation
	Trace       *Operation
	Servers     []Server
	Parameters  []Parameter
}

// Operation represents an OpenAPI Operation object.
type Operation struct {
	Tags         []string
	Summary      string
	Description  string
	OperationID  string
	Parameters   []Parameter
	RequestBody  *RequestBody
	Responses    Responses
	Callbacks    map[string]Callback
	Deprecated   bool
	Security     []SecurityRequirement
	Servers      []Server
	ExternalDocs *ExternalDocs
	Bindings     map[string]interface{} // Protocol-specific operation bindings
}

// Parameter represents an OpenAPI Parameter object.
type Parameter struct {
	Name            string
	In              ParameterLocation // path, query, header, cookie
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Style           string
	Explode         bool
	Schema          *Schema
	Example         interface{}
	Examples        map[string]Example
	Content         map[string]MediaType
}

// ParameterLocation defines the location of a parameter.
type ParameterLocation string

const (
	// ParameterInPath indicates a parameter in the path.
	ParameterInPath   ParameterLocation = "path"
	// ParameterInQuery indicates a parameter in the query string.
	ParameterInQuery  ParameterLocation = "query"
	// ParameterInHeader indicates a parameter in the header.
	ParameterInHeader ParameterLocation = "header"
	// ParameterInCookie indicates a parameter in a cookie.
	ParameterInCookie ParameterLocation = "cookie"
)

// RequestBody represents an OpenAPI Request Body object.
type RequestBody struct {
	Description string
	Content     map[string]MediaType
	Required    bool
}

// Responses is a map of status codes to Response objects.
type Responses map[string]Response

// Response represents an OpenAPI Response object.
type Response struct {
	Description string
	Headers     map[string]Header
	Content     map[string]MediaType
	Links       map[string]Link
}

// MediaType represents an OpenAPI Media Type object.
type MediaType struct {
	Schema   *Schema
	Example  interface{}
	Examples map[string]Example
	Encoding map[string]Encoding
}

// Encoding represents an OpenAPI Encoding object.
type Encoding struct {
	ContentType   string
	Headers       map[string]Header
	Style         string
	Explode       bool
	AllowReserved bool
}

// Header represents an OpenAPI Header object.
type Header struct {
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Schema          *Schema
	Example         interface{}
}

// Example represents an OpenAPI Example object.
type Example struct {
	Summary       string
	Description   string
	Value         interface{}
	ExternalValue string
}

// Link represents an OpenAPI Link object.
type Link struct {
	OperationRef string
	OperationID  string
	Parameters   map[string]interface{}
	RequestBody  interface{}
	Description  string
	Server       *Server
}

// Callback represents an OpenAPI Callback object.
type Callback map[string]PathItem

// GetOperation retrieves an operation by method
func (p *PathItem) GetOperation(method string) *Operation {
	switch method {
	case "GET":
		return p.Get
	case "PUT":
		return p.Put
	case "POST":
		return p.Post
	case "DELETE":
		return p.Delete
	case "OPTIONS":
		return p.Options
	case "HEAD":
		return p.Head
	case "PATCH":
		return p.Patch
	case "TRACE":
		return p.Trace
	default:
		return nil
	}
}

// SetOperation sets an operation by method
func (p *PathItem) SetOperation(method string, op *Operation) {
	switch method {
	case "GET":
		p.Get = op
	case "PUT":
		p.Put = op
	case "POST":
		p.Post = op
	case "DELETE":
		p.Delete = op
	case "OPTIONS":
		p.Options = op
	case "HEAD":		p.Head = op
	case "PATCH":
		p.Patch = op
	case "TRACE":
		p.Trace = op
	}
}

// AddParameter adds a parameter to the operation
func (o *Operation) AddParameter(param *Parameter) {
	o.Parameters = append(o.Parameters, *param)
}

// AddResponse adds a response to the operation
func (o *Operation) AddResponse(code string, resp Response) {
	if o.Responses == nil {
		o.Responses = make(Responses)
	}
	o.Responses[code] = resp
}

// GetDefault retrieves the default response
func (r *Responses) GetDefault() *Response {
	if resp, ok := (*r)["default"]; ok {
		return &resp
	}
	return nil
}

// GetByCode retrieves a response by status code
func (r *Responses) GetByCode(code string) *Response {
	if resp, ok := (*r)[code]; ok {
		return &resp
	}
	return nil
}