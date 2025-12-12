package converter

import (
	"io"
)

// APIBlueprint represents the Abstract Syntax Tree (AST) for an API Blueprint specification.
type APIBlueprint struct {
	Version     string            `json:"_version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Groups      []*ResourceGroup  `json:"groups,omitempty"`
	Components  *Components       `json:"components,omitempty"` // Reused from OpenAPI for schema storage
}

// ResourceGroup represents a logical grouping of resources.
type ResourceGroup struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Resources   []*Resource `json:"resources,omitempty"`
}

// Resource represents an API resource identified by a URI template.
type Resource struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	URITemplate string      `json:"uriTemplate"`
	Parameters  []Parameter `json:"parameters,omitempty"`
	Actions     []*Action   `json:"actions,omitempty"`
}

// Action represents an HTTP operation on a resource.
type Action struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Method      string         `json:"method"`
	Parameters  []Parameter    `json:"parameters,omitempty"`
	Attributes  *Schema        `json:"attributes,omitempty"` // MSON attributes
	Examples    []*Transaction `json:"examples,omitempty"`
}

// Transaction represents a request-response pair example.
type Transaction struct {
	Request  *Request           `json:"request,omitempty"`
	Response *BlueprintResponse `json:"response,omitempty"`
}

// BlueprintResponse represents an HTTP response in API Blueprint.
// It extends the standard Response with a Name field for the status code.
type BlueprintResponse struct {
	Name string `json:"name,omitempty"` // Status Code (e.g. "200")
	Response
}

// Request represents an HTTP request.
type Request struct {
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	Headers     map[string]string    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"` // Content-Type -> Schema/Example
}

// ToBlueprint converts the API Blueprint AST back to its string representation.
func (spec *APIBlueprint) ToBlueprint() (string, error) {
	// TODO: Implement proper serialization from this AST
	// For now, convert to OpenAPI and then to Blueprint string as a fallback/bridge
	openapi, err := spec.ToOpenAPI()
	if err != nil {
		return "", err
	}
	return openapi.ToBlueprint()
}

// WriteBlueprint writes the API Blueprint to a writer.
func (spec *APIBlueprint) WriteBlueprint(w io.Writer) error {
	s, err := spec.ToBlueprint()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

// ToOpenAPI converts the API Blueprint AST to an OpenAPI specification.
func (spec *APIBlueprint) ToOpenAPI() (*OpenAPI, error) {
	openapi := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       spec.Name,
			Description: spec.Description,
			Version:     spec.GetVersion(),
		},
		Paths:      make(map[string]PathItem),
		Components: spec.Components,
	}

	if openapi.Components == nil {
		openapi.Components = &Components{Schemas: make(map[string]*Schema)}
	}

	if openapi.Info.Version == "" {
		openapi.Info.Version = "1.0.0"
	}

	if host, ok := spec.Metadata["HOST"]; ok {
		openapi.Servers = []Server{{URL: host}}
	}

	for _, group := range spec.Groups {
		for _, resource := range group.Resources {
			pathItem := PathItem{}
			for _, action := range resource.Actions {
				convertActionToOperation(action, &pathItem)
			}
			openapi.Paths[resource.URITemplate] = pathItem
		}
	}

	return openapi, nil
}

func convertActionToOperation(action *Action, pathItem *PathItem) {
	op := &Operation{
		Summary:     action.Name,
		Description: action.Description,
		Parameters:  action.Parameters,
		Responses:   make(map[string]Response), // Basic init
	}
	// Restore Request/Response from Examples
	for _, example := range action.Examples {
		if example.Request != nil {
			// Only set request body if not already set (or merge?)
			// Usually first transaction has the request
			if op.RequestBody == nil {
				op.RequestBody = &RequestBody{
					Content: example.Request.Content,
				}
			}
		}
		if example.Response != nil {
			code := example.Response.Name
			if code == "" {
				code = "200" // Default
			}
			op.Responses[code] = example.Response.Response
		}
	}

	switch action.Method {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	}
}

// ToAsyncAPI converts the API Blueprint to AsyncAPI.
func (spec *APIBlueprint) ToAsyncAPI(protocol Protocol) (*AsyncAPI, error) {
	openapi, err := spec.ToOpenAPI()
	if err != nil {
		return nil, err
	}
	return openapi.ToAsyncAPI(protocol)
}

// ToAsyncAPIV3 converts the API Blueprint to AsyncAPI 3.0.
func (spec *APIBlueprint) ToAsyncAPIV3(protocol Protocol) (*AsyncAPI, error) {
	openapi, err := spec.ToOpenAPI()
	if err != nil {
		return nil, err
	}
	return openapi.ToAsyncAPIV3(protocol)
}

// GetTitle returns the title.
func (spec *APIBlueprint) GetTitle() string {
	return spec.Name
}

// GetVersion returns the version.
func (spec *APIBlueprint) GetVersion() string {
	// Check metadata for version
	if v, ok := spec.Metadata["VERSION"]; ok {
		return v
	}
	return spec.Version
}

// AsOpenAPI returns nil.
func (spec *APIBlueprint) AsOpenAPI() (*OpenAPI, bool) {
	return nil, false
}

// AsAsyncAPI returns nil.
func (spec *APIBlueprint) AsAsyncAPI() (*AsyncAPI, bool) {
	return nil, false
}

// AsAsyncAPIV3 returns nil.
func (spec *APIBlueprint) AsAsyncAPIV3() (*AsyncAPI, bool) {
	return nil, false
}

// AsAPIBlueprint returns the spec itself.
func (spec *APIBlueprint) AsAPIBlueprint() (*APIBlueprint, bool) {
	return spec, true
}
