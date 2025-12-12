package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// APIBlueprint represents the Abstract Syntax Tree (AST) for an API Blueprint specification.
type APIBlueprint struct {
	VersionField string            `json:"_version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	TitleField   string            `json:"name,omitempty"`
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

// Title returns the name of the API Blueprint specification.
func (spec *APIBlueprint) Title() string {
	if spec != nil {
		return spec.TitleField
	}
	return ""
}

// Version returns the version of the API Blueprint specification.
func (spec *APIBlueprint) Version() string {
	if spec != nil {
		if v, ok := spec.Metadata["VERSION"]; ok && v != "" {
			return v
		}
		return spec.VersionField
	}
	return ""
}

// String returns the API Blueprint string representation.
func (spec *APIBlueprint) String() string {
	buf := getBuffer()
	defer putBuffer(buf)

	writeAPIBlueprintHeader(buf, spec)

	if spec.Components != nil && len(spec.Components.Schemas) > 0 {
		writeDataStructures(buf, spec.Components.Schemas)
	}

	for _, group := range spec.Groups {
		writeResourceGroup(buf, group)
		for _, resource := range group.Resources {
			writeResource(buf, resource)
			for _, action := range resource.Actions {
				writeAction(buf, action)
			}
		}
	}
	return buf.String()
}

// WriteTo writes the API Blueprint to a writer.
func (spec *APIBlueprint) WriteTo(w io.Writer) (int64, error) {
	s := spec.String()
	n, err := io.WriteString(w, s)
	return int64(n), err
}

// ToOpenAPI converts the API Blueprint AST to an OpenAPI specification.
func (spec *APIBlueprint) ToOpenAPI() (*OpenAPI, error) {
	openapi := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       spec.TitleField,
			Description: spec.Description,
			Version:     spec.VersionField,
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
				convertActionToOperation(action, resource.Parameters, &pathItem)
			}
			openapi.Paths[resource.URITemplate] = pathItem
		}
	}

	return openapi, nil
}

func convertActionToOperation(action *Action, resourceParameters []Parameter, pathItem *PathItem) {
	op := &Operation{
		Summary:     action.Name,
		Description: action.Description,
		Parameters:  mergeParameters(action.Parameters, resourceParameters),
		Responses:   make(map[string]Response),
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

func mergeParameters(actionParams, resourceParams []Parameter) []Parameter {
	merged := make([]Parameter, 0, len(resourceParams)+len(actionParams))
	
	// Add action-level parameters first (they take precedence)
	merged = append(merged, actionParams...)

	// Add resource-level parameters, but only if not already defined by action-level
	for _, resParam := range resourceParams {
		alreadyDefined := false
		for _, opParam := range merged {
			if opParam.Name == resParam.Name && opParam.In == resParam.In {
				alreadyDefined = true
				break
			}
		}
		if !alreadyDefined {
			merged = append(merged, resParam)
		}
	}
	return merged
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
	return spec.TitleField
}

// GetVersion returns the version.
func (spec *APIBlueprint) GetVersion() string {
	// Check metadata for version
	    if v, ok := spec.Metadata["VERSION"]; ok {
	        return v
	    }
	    return spec.VersionField}

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

// writeAPIBlueprintHeader writes the API Blueprint header information to the buffer.
func writeAPIBlueprintHeader(buf *bytes.Buffer, spec *APIBlueprint) {
	buf.WriteString(APIBlueprintFormat + "\n")
	if v := spec.VersionField; v != "" {
		fmt.Fprintf(buf, "FORMAT: %s\n", v)
	}
	// Host from metadata
	if host, ok := spec.Metadata["HOST"]; ok {
		fmt.Fprintf(buf, "HOST: %s\n", host)
	}
	buf.WriteString("\n")

	// Name
	if spec.TitleField != "" {
		fmt.Fprintf(buf, "# %s\n", spec.TitleField)
	}

	// Description
	if spec.Description != "" {
		buf.WriteString(spec.Description)
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
}

// writeResourceGroup writes a resource group to the buffer.
func writeResourceGroup(buf *bytes.Buffer, group *ResourceGroup) {
	if group.Name != "" {
		fmt.Fprintf(buf, "## Group %s\n", group.Name)
	}
	if group.Description != "" {
		buf.WriteString(group.Description)
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
}

// writeResource writes a resource to the buffer.
func writeResource(buf *bytes.Buffer, resource *Resource) {
	fmt.Fprintf(buf, "### %s [%s]\n", resource.Name, resource.URITemplate)
	if resource.Description != "" {
		buf.WriteString(resource.Description)
		buf.WriteString("\n")
	}

	// Parameters
	if len(resource.Parameters) > 0 {
		buf.WriteString("+ Parameters\n")
		writeParameters(buf, resource.Parameters, "\t") // Add a tab prefix for parameters
	}
	buf.WriteString("\n")
}

// writeParameters writes a list of parameters to the buffer.
// prefix is used for indentation.
func writeParameters(buf *bytes.Buffer, params []Parameter, prefix string) {
	for _, p := range params {
		fmt.Fprintf(buf, "%s+ %s ", prefix, p.Name)

		// Type and required status
		parts := []string{}
		if p.Schema != nil && p.Schema.Type != nil {
			parts = append(parts, p.Schema.Type.(string))
		}
		if p.Required {
			parts = append(parts, "required")
		} else {
			parts = append(parts, "optional")
		}
		fmt.Fprintf(buf, "(%s)", strings.Join(parts, ", "))

		// Description
		if p.Description != "" {
			fmt.Fprintf(buf, " - %s", p.Description)
		}
		buf.WriteString("\n")
	}
}

// writeAction writes an action to the buffer.
func writeAction(buf *bytes.Buffer, action *Action) {
	fmt.Fprintf(buf, "#### %s [%s]\n", action.Name, action.Method)
	if action.Description != "" {
		buf.WriteString(action.Description)
		buf.WriteString("\n")
	}

	if action.Attributes != nil {
		writeMSON(buf, action.Attributes, 0)
		buf.WriteString("\n")
	}

	// Parameters for action
	// TODO: Check if action.Parameters should be nested under a different header if resource had parameters
	if len(action.Parameters) > 0 {
		buf.WriteString("+ Parameters\n")
		writeParameters(buf, action.Parameters, "\t")
	}

	// Examples (transactions)
	for _, example := range action.Examples {
		writeTransaction(buf, example)
	}
	buf.WriteString("\n")
}

// writeTransaction writes a transaction (request/response example) to the buffer.
func writeTransaction(buf *bytes.Buffer, transaction *Transaction) {
	if transaction.Request != nil {
		writeTransactionRequest(buf, transaction.Request)
	}
	if transaction.Response != nil {
		writeTransactionResponse(buf, transaction.Response)
	}
}



func writeTransactionRequest(buf *bytes.Buffer, req *Request) {
	buf.WriteString("\t+ Request")
	if req.Name != "" {
		fmt.Fprintf(buf, " %s", req.Name)
	}
	// Content-Type
	for contentType, mediaType := range req.Content {
		fmt.Fprintf(buf, " (%s)", contentType)
		writeTransactionContent(buf, req.Headers, mediaType)
		break // Only take the first content type for now
	}
	buf.WriteString("\n")
}



func writeTransactionResponse(buf *bytes.Buffer, resp *BlueprintResponse) {
	fmt.Fprintf(buf, "\t+ Response %s", resp.Name)
	// Content-Type
	for contentType, mediaType := range resp.Content {
		fmt.Fprintf(buf, " (%s)", contentType)
		writeTransactionContent(buf, resp.Headers, mediaType)
		break // Only take the first content type for now
	}
	buf.WriteString("\n")
}

func writeTransactionContent(buf *bytes.Buffer, headers map[string]string, mediaType MediaType) {
	// Headers
	if len(headers) > 0 {
		buf.WriteString("\n\t\t+ Headers\n")
		for k, v := range headers {
			fmt.Fprintf(buf, "\t\t\t%s: %s\n", k, v)
		}
	}
	// Attributes
	if mediaType.Schema != nil {
		buf.WriteString("\n")
		writeMSON(buf, mediaType.Schema, 2)
	}
	// Body
	if mediaType.Example != nil {
		jsonBytes, err := json.MarshalIndent(mediaType.Example, "\t\t\t", "    ")
		if err == nil {
			buf.WriteString("\n\t\t+ Body\n")
			fmt.Fprintf(buf, "\t\t\t%s\n", string(jsonBytes))
		}
	}
}
