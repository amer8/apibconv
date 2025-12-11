package converter

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
)

// BlueprintConvertible is an interface for types that can be converted to API Blueprint format.
type BlueprintConvertible interface {
	ToBlueprint() string
	WriteBlueprint(w io.Writer) error
}

// OpenAPI represents a minimal OpenAPI 3.0 specification structure.
//
// This is the root object for an OpenAPI document, containing all the metadata,
// server information, API paths, and reusable components.
//
// Example:
//
//	spec := &OpenAPI{
//	    OpenAPI: "3.0.0",
//	    Info: Info{
//	        Title:   "My API",
//	        Version: "1.0.0",
//	    },
//	    Paths: map[string]PathItem{
//	        "/users": {
//	            Get: &Operation{
//	                Summary: "List users",
//	                Responses: map[string]Response{
//	                    "200": {Description: "Success"},
//	                },
//	            },
//	        },
//	    },
//	}
//
//go:generate easyjson -all converter.go
type OpenAPI struct {
	OpenAPI           string              `json:"openapi"`                     // OpenAPI specification version (e.g., "3.0.0", "3.1.0")
	Info              Info                `json:"info"`                        // API metadata including title, description, and version
	Servers           []Server            `json:"servers,omitempty"`           // Array of Server objects providing connectivity information
	Paths             map[string]PathItem `json:"paths"`                       // Available paths and operations for the API
	Webhooks          map[string]PathItem `json:"webhooks,omitempty"`          // Webhooks for async callbacks (OpenAPI 3.1+)
	Components        *Components         `json:"components,omitempty"`        // Reusable components (schemas, parameters, etc.)
	JSONSchemaDialect string              `json:"jsonSchemaDialect,omitempty"` // JSON Schema dialect (OpenAPI 3.1+)
}

// Info represents the metadata about the API.
//
// This includes the API's title, description, version, and other informational fields
// that help users understand what the API does.
type Info struct {
	Title       string   `json:"title"`                 // The title of the API (required)
	Description string   `json:"description,omitempty"` // A description of the API
	Version     string   `json:"version"`               // The version of the API (required)
	License     *License `json:"license,omitempty"`     // License information for the API
}

// License represents license information for the API.
//
// OpenAPI 3.1 adds support for SPDX license identifiers via the Identifier field.
//
// Example:
//
//	license := &License{
//	    Name: "Apache 2.0",
//	    URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
//	}
//
// OpenAPI 3.1 with SPDX identifier:
//
//	license := &License{
//	    Name:       "Apache 2.0",
//	    Identifier: "Apache-2.0",  // SPDX identifier (3.1+)
//	}
type License struct {
	Name       string `json:"name"`                 // License name (required)
	URL        string `json:"url,omitempty"`        // URL to the license
	Identifier string `json:"identifier,omitempty"` // SPDX license identifier (OpenAPI 3.1+)
}

// Server represents a server URL and optional description.
//
// Servers define the base URLs where the API is available. Multiple servers can be
// defined for different environments (production, staging, development, etc.).
//
// Example:
//
//	server := Server{
//	    URL: "https://api.example.com/v1",
//	    Description: "Production server",
//	}
type Server struct {
	URL         string `json:"url"`                   // A URL to the target host
	Description string `json:"description,omitempty"` // An optional description of the server
}

// PathItem represents the operations available on a single API path.
//
// Each PathItem can contain operations for different HTTP methods (GET, POST, PUT, DELETE, PATCH).
// Not all methods need to be present; only define the methods your API supports for this path.
//
// Example:
//
//	pathItem := PathItem{
//	    Get: &Operation{
//	        Summary: "Retrieve user",
//	        Responses: map[string]Response{
//	            "200": {Description: "User found"},
//	        },
//	    },
//	    Delete: &Operation{
//	        Summary: "Delete user",
//	        Responses: map[string]Response{
//	            "204": {Description: "User deleted"},
//	        },
//	    },
//	}
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`    // GET operation for this path
	Post   *Operation `json:"post,omitempty"`   // POST operation for this path
	Put    *Operation `json:"put,omitempty"`    // PUT operation for this path
	Delete *Operation `json:"delete,omitempty"` // DELETE operation for this path
	Patch  *Operation `json:"patch,omitempty"`  // PATCH operation for this path
}

// Operation represents a single API operation (HTTP method) on a path.
//
// An operation describes what happens when a client makes a request to a specific
// path with a specific HTTP method. It includes a summary, description, parameters,
// request body, and possible responses.
//
// Example:
//
//	op := &Operation{
//	    Summary: "Create a new user",
//	    Description: "Creates a new user account with the provided information",
//	    Parameters: []Parameter{
//	        {Name: "api_key", In: "header", Required: true, Schema: &Schema{Type: "string"}},
//	    },
//	    RequestBody: &RequestBody{
//	        Required: true,
//	        Content: map[string]MediaType{
//	            "application/json": {
//	                Example: map[string]string{"name": "John", "email": "john@example.com"},
//	            },
//	        },
//	    },
//	    Responses: map[string]Response{
//	        "201": {Description: "User created successfully"},
//	        "400": {Description: "Invalid input"},
//	    },
//	}
type Operation struct {
	Summary     string              `json:"summary,omitempty"`     // A short summary of what the operation does
	Description string              `json:"description,omitempty"` // A detailed description of the operation
	Parameters  []Parameter         `json:"parameters,omitempty"`  // List of parameters for the operation
	RequestBody *RequestBody        `json:"requestBody,omitempty"` // The request body for the operation
	Responses   map[string]Response `json:"responses,omitempty"`   // Possible responses, keyed by HTTP status code
}

// Parameter represents a single operation parameter.
//
// Parameters can be in different locations: path, query, header, or cookie.
// Path parameters must be marked as required since they're part of the URL.
//
// Example:
//
//	// Path parameter
//	pathParam := Parameter{
//	    Name: "userId",
//	    In: "path",
//	    Required: true,
//	    Description: "The user ID",
//	    Schema: &Schema{Type: "string"},
//	}
//
//	// Query parameter
//	queryParam := Parameter{
//	    Name: "limit",
//	    In: "query",
//	    Required: false,
//	    Description: "Number of results to return",
//	    Schema: &Schema{Type: "integer"},
//	}
type Parameter struct {
	Name        string  `json:"name"`                  // The name of the parameter
	In          string  `json:"in"`                    // Location of the parameter: "path", "query", "header", or "cookie"
	Description string  `json:"description,omitempty"` // A description of the parameter
	Required    bool    `json:"required,omitempty"`    // Whether the parameter is required
	Schema      *Schema `json:"schema,omitempty"`      // The schema defining the type used for the parameter
}

// RequestBody represents the request body of an operation.
//
// The request body can contain different content types (e.g., JSON, XML, form data).
// Each content type can have its own schema and example.
//
// Example:
//
//	reqBody := &RequestBody{
//	    Description: "User object to create",
//	    Required: true,
//	    Content: map[string]MediaType{
//	        "application/json": {
//	            Schema: &Schema{
//	                Type: "object",
//	                Properties: map[string]*Schema{
//	                    "name": {Type: "string"},
//	                    "email": {Type: "string"},
//	                },
//	            },
//	            Example: map[string]string{
//	                "name": "Jane Doe",
//	                "email": "jane@example.com",
//	            },
//	        },
//	    },
//	}
type RequestBody struct {
	Description string               `json:"description,omitempty"` // A description of the request body
	Required    bool                 `json:"required,omitempty"`    // Whether the request body is required
	Content     map[string]MediaType `json:"content,omitempty"`     // Content types that can be sent, keyed by media type
}

// Response represents a single response from an API operation.
//
// A response includes a description and optional content for different media types.
// The description is required by the OpenAPI spec.
//
// Example:
//
//	resp := Response{
//	    Description: "Successful response",
//	    Content: map[string]MediaType{
//	        "application/json": {
//	            Example: map[string]any{
//	                "id": "123",
//	                "name": "John Doe",
//	            },
//	        },
//	    },
//	}
type Response struct {
	Description string               `json:"description"`       // A description of the response (required)
	Content     map[string]MediaType `json:"content,omitempty"` // Response content for different media types
}

// MediaType represents a media type and its schema.
//
// This defines what type of content is sent or received, along with optional
// examples to help users understand the expected format.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`  // The schema defining the content structure
	Example any     `json:"example,omitempty"` // An example of the media type content
}

// Schema represents a data type schema used in requests and responses.
//
// Schemas define the structure of data. They can be simple types (string, integer)
// or complex objects with properties and nested schemas.
//
// OpenAPI 3.0 vs 3.1 differences:
//   - Type: In 3.0, this is a string. In 3.1, can be string or []string for nullable types
//   - Nullable: Used in 3.0 for null support. In 3.1, use Type: []string{"string", "null"}
//   - Additional 3.1 fields: Const, DependentSchemas, PrefixItems, etc.
//
// Example of a simple schema (3.0):
//
//	stringSchema := &Schema{Type: "string"}
//
// Example of a nullable schema (3.0):
//
//	nullableSchema := &Schema{Type: "string", Nullable: true}
//
// Example of a nullable schema (3.1):
//
//	nullableSchema := &Schema{Type: []string{"string", "null"}}
//
// Example of an object schema:
//
//	userSchema := &Schema{
//	    Type: "object",
//	    Properties: map[string]*Schema{
//	        "id": {Type: "string"},
//	        "name": {Type: "string"},
//	        "email": {Type: "string"},
//	        "age": {Type: "integer"},
//	    },
//	    Example: map[string]any{
//	        "id": "123",
//	        "name": "John Doe",
//	        "email": "john@example.com",
//	        "age": 30,
//	    },
//	}
//
// Example of an array schema:
//
//	arraySchema := &Schema{
//	    Type: "array",
//	    Items: &Schema{Type: "string"},
//	    Example: []string{"apple", "banana", "orange"},
//	}
type Schema struct {
	// Common fields
	Ref         string   `json:"$ref,omitempty"`        // Reference to another schema
	Description string   `json:"description,omitempty"` // Description of the schema
	Required    []string `json:"required,omitempty"`    // List of required properties

	// Type is the data type. In OpenAPI 3.0, this is a string ("string", "number", etc.).
	// In OpenAPI 3.1, this can be a string or []string (e.g., []string{"string", "null"}).
	Type any `json:"type,omitempty"`

	// Nullable indicates if null is a valid value (OpenAPI 3.0 only).
	// In OpenAPI 3.1, use Type: []string{"type", "null"} instead.
	Nullable bool `json:"nullable,omitempty"`

	Properties map[string]*Schema `json:"properties,omitempty"` // Properties of an object schema
	Items      *Schema            `json:"items,omitempty"`      // Schema for array items (when Type is "array")
	Example    any                `json:"example,omitempty"`    // An example value for this schema

	// OpenAPI 3.1 / JSON Schema 2020-12 fields
	Const            any                `json:"const,omitempty"`            // Const value (3.1+)
	DependentSchemas map[string]*Schema `json:"dependentSchemas,omitempty"` // Dependent schemas (3.1+)
	PrefixItems      []*Schema          `json:"prefixItems,omitempty"`      // Tuple validation (3.1+)
}

// Components holds reusable schema components of the OpenAPI specification.
//
// Components allow you to define schemas once and reference them throughout
// your API specification, promoting reusability and maintainability.
//
// Example:
//
//	components := &Components{
//	    Schemas: map[string]*Schema{
//	        "User": {
//	            Type: "object",
//	            Properties: map[string]*Schema{
//	                "id": {Type: "string"},
//	                "name": {Type: "string"},
//	            },
//	        },
//	        "Error": {
//	            Type: "object",
//	            Properties: map[string]*Schema{
//	                "code": {Type: "integer"},
//	                "message": {Type: "string"},
//	            },
//	        },
//	    },
//	}
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"` // Reusable schemas keyed by name
}

// ToBlueprint converts the OpenAPI specification to API Blueprint format.
func (spec *OpenAPI) ToBlueprint() string {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAPIBlueprint(buf, spec)
	return buf.String()
}

// WriteBlueprint writes the OpenAPI specification in API Blueprint format to the writer.
func (spec *OpenAPI) WriteBlueprint(w io.Writer) error {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAPIBlueprint(buf, spec)
	_, err := w.Write(buf.Bytes())
	return err
}

// Convert converts OpenAPI JSON to API Blueprint format using streaming I/O.
//
// This is the primary conversion function for transforming OpenAPI specifications
// to API Blueprint format. It reads from the provided io.Reader, parses the OpenAPI JSON,
// and writes the formatted API Blueprint output to the provided io.Writer.
//
// The function uses zero allocations for buffer operations after initial parsing,
// thanks to internal buffer pooling with sync.Pool.
//
// Parameters:
//   - r: An io.Reader containing OpenAPI 3.0 JSON data
//   - w: An io.Writer where the API Blueprint output will be written
//
// Returns an error if:
//   - The input cannot be read
//   - The JSON is malformed or invalid
//   - The output cannot be written
//
// Example:
//
//	input, err := os.Open("openapi.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input.Close()
//
//	output, err := os.Create("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	if err := converter.Convert(input, output); err != nil {
//	    log.Fatal(err)
//	}
func Convert(r io.Reader, w io.Writer) error {
	spec, err := ParseReader(r)
	if err != nil {
		return err
	}
	return spec.WriteBlueprint(w)
}

// writeAPIBlueprint writes the API Blueprint format to the buffer
func writeAPIBlueprint(buf *bytes.Buffer, spec *OpenAPI) {
	// Pre-allocate buffer to reduce reallocations
	// Estimate: ~200 bytes per path * num paths + 1KB overhead
	estimatedSize := len(spec.Paths)*200 + 1024
	buf.Grow(estimatedSize)

	// Header
	buf.WriteString(APIBlueprintFormat + "\n\n")

	// Title
	buf.WriteString("# ")
	buf.WriteString(spec.Info.Title)
	buf.WriteString("\n\n")

	// Description
	if spec.Info.Description != "" {
		buf.WriteString(spec.Info.Description)
		buf.WriteString("\n\n")
	}

	// Host information
	if len(spec.Servers) > 0 {
		buf.WriteString("HOST: ")
		buf.WriteString(spec.Servers[0].URL)
		buf.WriteString("\n\n")
	}

	// Data Structures
	if spec.Components != nil && len(spec.Components.Schemas) > 0 {
		writeDataStructures(buf, spec.Components.Schemas)
	}

	// Sort paths for consistent output
	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Process each path
	for _, path := range paths {
		pathItem := spec.Paths[path]
		writePathItem(buf, path, &pathItem)
	}
}

func writePathItem(buf *bytes.Buffer, path string, item *PathItem) {
	// Group heading
	buf.WriteString("## Group ")
	// Extract group name from path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 {
		buf.WriteString(titleCase(parts[0]))
	}
	buf.WriteString("\n\n")

	// Resource
	buf.WriteString("## ")
	buf.WriteString(path)
	buf.WriteString(" [")
	buf.WriteString(path)
	buf.WriteString("]\n\n")

	// Operations
	if item.Get != nil {
		writeOperation(buf, http.MethodGet, path, item.Get)
	}
	if item.Post != nil {
		writeOperation(buf, http.MethodPost, path, item.Post)
	}
	if item.Put != nil {
		writeOperation(buf, http.MethodPut, path, item.Put)
	}
	if item.Delete != nil {
		writeOperation(buf, http.MethodDelete, path, item.Delete)
	}
	if item.Patch != nil {
		writeOperation(buf, http.MethodPatch, path, item.Patch)
	}
}

func writeOperation(buf *bytes.Buffer, method, path string, op *Operation) {
	// Action
	buf.WriteString("### ")
	if op.Summary != "" {
		buf.WriteString(op.Summary)
	} else {
		buf.WriteString(method)
		buf.WriteString(" ")
		buf.WriteString(path)
	}
	buf.WriteString(" [")
	buf.WriteString(method)
	buf.WriteString("]\n\n")

	// Description
	if op.Description != "" {
		buf.WriteString(op.Description)
		buf.WriteString("\n\n")
	}

	// Parameters
	if len(op.Parameters) > 0 {
		buf.WriteString("+ Parameters\n")
		for i := range op.Parameters {
			param := &op.Parameters[i]
			buf.WriteString("    + ")
			buf.WriteString(param.Name)
			if param.Required {
				buf.WriteString(" (required, ")
			} else {
				buf.WriteString(" (optional, ")
			}
			if param.Schema != nil {
				typeStr := SchemaType(param.Schema)
				if typeStr != "" {
					buf.WriteString(typeStr)
				} else {
					buf.WriteString("string")
				}
			} else {
				buf.WriteString("string")
			}
			buf.WriteString(")")
			if param.Description != "" {
				buf.WriteString(" - ")
				buf.WriteString(param.Description)
			}
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	// Request
	if op.RequestBody != nil {
		writeRequest(buf, op.RequestBody)
	}

	// Responses
	if len(op.Responses) > 0 {
		writeResponses(buf, op.Responses)
	}
}

func writeRequest(buf *bytes.Buffer, req *RequestBody) {
	buf.WriteString("+ Request (" + MediaTypeJSON + ")\n\n")

	// Get JSON content type if exists
	if content, ok := req.Content[MediaTypeJSON]; ok {
		if content.Schema != nil {
			writeMSON(buf, content.Schema, 1)
			buf.WriteString("\n")
		} else if content.Example != nil {
			exampleJSON, err := json.MarshalIndent(content.Example, "        ", "    ")
			if err == nil {
				buf.WriteString("        ")
				buf.Write(exampleJSON)
				buf.WriteString("\n\n")
			}
		}
	}
}

func writeResponses(buf *bytes.Buffer, responses map[string]Response) {
	// Sort response codes
	codes := make([]string, 0, len(responses))
	for code := range responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		resp := responses[code]
		buf.WriteString("+ Response ")
		buf.WriteString(code)

		// Check for JSON content
		if content, ok := resp.Content[MediaTypeJSON]; ok {
			buf.WriteString(" (" + MediaTypeJSON + ")\n\n")

			if content.Schema != nil {
				writeMSON(buf, content.Schema, 1)
				buf.WriteString("\n")
			} else if content.Example != nil {
				exampleJSON, err := json.MarshalIndent(content.Example, "        ", "    ")
				if err == nil {
					buf.WriteString("        ")
					buf.Write(exampleJSON)
					buf.WriteString("\n\n")
				}
			}
		} else {
			buf.WriteString("\n\n")
			if resp.Description != "" {
				buf.WriteString("    ")
				buf.WriteString(resp.Description)
				buf.WriteString("\n\n")
			}
		}
	}
}

// titleCase capitalizes the first letter of a string.
// This replaces the deprecated strings.Title for simple ASCII use cases.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// SchemaType returns the primary type of a schema as a string.
//
// In OpenAPI 3.0, Type is always a string.
// In OpenAPI 3.1, Type can be a string or []string.
// This helper extracts the primary (non-null) type.
func SchemaType(schema *Schema) string {
	if schema == nil || schema.Type == nil {
		return ""
	}

	// Handle string type
	if typeStr, ok := schema.Type.(string); ok {
		return typeStr
	}

	// Handle array type (3.1)
	if typeArr, ok := schema.Type.([]any); ok {
		for _, t := range typeArr {
			if tStr, ok := t.(string); ok && tStr != TypeNull {
				return tStr
			}
		}
	}

	return ""
}

// IsNullable returns true if a schema allows null values.
//
// In OpenAPI 3.0, this is indicated by nullable: true.
// In OpenAPI 3.1, this is indicated by type: [..., "null"].
func IsNullable(schema *Schema) bool {
	if schema == nil {
		return false
	}

	// Check 3.0 style nullable
	if schema.Nullable {
		return true
	}

	// Check 3.1 style type array
	if typeArr, ok := schema.Type.([]any); ok {
		for _, t := range typeArr {
			if tStr, ok := t.(string); ok && tStr == TypeNull {
				return true
			}
		}
	}

	return false
}

// NormalizeSchemaType ensures schema.Type is properly formatted for JSON marshaling.
//
// This is useful when you've manipulated the Type field and want to ensure
// it's in the correct format for the target OpenAPI version.
func NormalizeSchemaType(schema *Schema, version Version) {
	if schema == nil || schema.Type == nil {
		return
	}

	// For 3.0, ensure Type is a string
	if version == Version30 {
		typeStr := SchemaType(schema)
		if typeStr != "" {
			schema.Type = typeStr
		}
	}

	// For 3.1, convert to array if nullable
	if version == Version31 && schema.Nullable {
		typeStr := SchemaType(schema)
		if typeStr != "" {
			schema.Type = []any{typeStr, TypeNull}
			schema.Nullable = false
		}
	}

	// Recursively normalize nested schemas
	for _, prop := range schema.Properties {
		NormalizeSchemaType(prop, version)
	}

	if schema.Items != nil {
		NormalizeSchemaType(schema.Items, version)
	}
}
