package model

// PathItem represents an OpenAPI Path Item object.
type PathItem struct {
	Name        string // Original name/key if available to custom formats
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
	// ChannelServers lists server names selected by a custom channel-based format.
	// An empty list means all servers.
	ChannelServers []string
	Parameters     []Parameter
	Extensions     map[string]interface{}
}

// Operation represents an OpenAPI Operation object.
type Operation struct {
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Parameters  []Parameter
	RequestBody *RequestBody
	// ResponsesExtensions contains extensions on the responses object itself.
	ResponsesExtensions map[string]interface{}
	Responses           Responses
	CallbackExtensions  map[string]map[string]interface{}
	Callbacks           map[string]Callback
	Deprecated          bool
	Security            []SecurityRequirement
	Servers             []Server
	ExternalDocs        *ExternalDocs
	Bindings            map[string]interface{} // Protocol-specific operation bindings
	Extensions          map[string]interface{}
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
	ExplodeSet      bool // Explode was explicitly specified, including false.
	Schema          *Schema
	Example         interface{}
	Examples        map[string]Example
	Content         map[string]MediaType
	Extensions      map[string]interface{}
}

// ParameterLocation defines the location of a parameter.
type ParameterLocation string

const (
	// ParameterInPath indicates a parameter in the path.
	ParameterInPath ParameterLocation = "path"
	// ParameterInQuery indicates a parameter in the query string.
	ParameterInQuery ParameterLocation = "query"
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
	Extensions  map[string]interface{}
}

// Responses is a map of status codes to Response objects.
type Responses map[string]Response

// Response represents an OpenAPI Response object.
type Response struct {
	Description string
	Headers     map[string]Header
	Content     map[string]MediaType
	Links       map[string]Link
	Extensions  map[string]interface{}
}

// MediaType represents an OpenAPI Media Type object.
type MediaType struct {
	Schema     *Schema
	Example    interface{}
	Examples   map[string]Example
	Encoding   map[string]Encoding
	Extensions map[string]interface{}
}

// Encoding represents an OpenAPI Encoding object.
type Encoding struct {
	ContentType   string
	Headers       map[string]Header
	Style         string
	Explode       bool
	AllowReserved bool
	Extensions    map[string]interface{}
}

// Header represents an OpenAPI Header object.
type Header struct {
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Schema          *Schema
	Example         interface{}
	Extensions      map[string]interface{}
	Examples        map[string]Example
}

// Example represents an OpenAPI Example object.
type Example struct {
	Summary       string
	Description   string
	Value         interface{}
	ExternalValue string
	Ref           string
	Extensions    map[string]interface{}
}

// Link represents an OpenAPI Link object.
type Link struct {
	OperationRef string
	OperationID  string
	Parameters   map[string]interface{}
	RequestBody  interface{}
	Description  string
	Server       *Server
	Ref          string
	Extensions   map[string]interface{}
}

// Callback represents an OpenAPI Callback object.
type Callback map[string]PathItem

// GetOperation returns the operation for an uppercase HTTP method, or nil.
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

// SetOperation sets the operation for an uppercase HTTP method. Unknown methods are ignored.
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
	case "HEAD":
		p.Head = op
	case "PATCH":
		p.Patch = op
	case "TRACE":
		p.Trace = op
	}
}

// AddParameter adds a parameter to the operation.
func (o *Operation) AddParameter(param *Parameter) {
	o.Parameters = append(o.Parameters, *param)
}

// AddResponse adds or replaces a response for the given status code.
func (o *Operation) AddResponse(code string, resp Response) {
	if o.Responses == nil {
		o.Responses = make(Responses)
	}
	o.Responses[code] = resp
}

// GetDefault returns a shallow copy of the default response, or nil if absent.
func (r *Responses) GetDefault() *Response {
	if resp, ok := (*r)["default"]; ok {
		return &resp
	}
	return nil
}

// GetByCode returns a shallow copy of the response, or nil if absent.
func (r *Responses) GetByCode(code string) *Response {
	if resp, ok := (*r)[code]; ok {
		return &resp
	}
	return nil
}
