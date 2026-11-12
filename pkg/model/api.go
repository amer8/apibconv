// Package model defines the unified internal representation of API specifications.
package model

import "maps"

// API is the shared representation used by parsers and writers.
type API struct {
	Version string   // Spec version
	Info    Info     // API metadata
	Servers []Server // Server configurations
	// PathsExtensions contains extensions on the paths object itself.
	PathsExtensions map[string]interface{}
	Paths           map[string]PathItem    // API paths/endpoints
	Webhooks        map[string]PathItem    // OpenAPI 3.1 Webhooks
	Components      Components             // Reusable components
	Security        []SecurityRequirement  // Global security
	Tags            []Tag                  // API tags
	ExternalDocs    *ExternalDocs          // External documentation
	Extensions      map[string]interface{} // Vendor extensions (x-*)
}

// Info provides metadata about the API.
type Info struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string
	Contact        *Contact
	License        *License
	Extensions     map[string]interface{}
}

// Contact information for the exposed API.
type Contact struct {
	Name       string
	URL        string
	Email      string
	Extensions map[string]interface{}
}

// License information for the exposed API.
type License struct {
	Name       string
	URL        string
	Extensions map[string]interface{}
}

// Server object represents a server for the API.
type Server struct {
	Name        string // Server name/ID (e.g. "production")
	URL         string
	Description string
	Protocol    string                // Protocol used by a custom format
	Security    []SecurityRequirement // Security required by this server, independent of operations
	Variables   map[string]ServerVariable
	Bindings    map[string]interface{} // Protocol-specific server bindings
	Extensions  map[string]interface{}
}

// ServerVariable object represents a variable for server URL template substitution.
type ServerVariable struct {
	Default     string
	Description string
	Enum        []string
	Extensions  map[string]interface{}
}

// Tag object for organizing operations.
type Tag struct {
	Name         string
	Description  string
	ExternalDocs *ExternalDocs
	Extensions   map[string]interface{} `json:"-" yaml:"-"`
}

// ExternalDocs object for external documentation.
type ExternalDocs struct {
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string                 `json:"url,omitempty" yaml:"url,omitempty"`
	Extensions  map[string]interface{} `json:"-" yaml:"-"`
}

// NewAPI creates an API with initialized path and component maps.
func NewAPI() *API {
	return &API{
		Paths:      make(map[string]PathItem),
		Webhooks:   make(map[string]PathItem),
		Components: NewComponents(),
		Extensions: make(map[string]interface{}),
	}
}

// AddPath adds or replaces a path item.
func (a *API) AddPath(path string, item *PathItem) {
	if a.Paths == nil {
		a.Paths = make(map[string]PathItem)
	}
	a.Paths[path] = *item
}

// GetPath returns the path item and whether it exists.
func (a *API) GetPath(path string) (PathItem, bool) {
	item, ok := a.Paths[path]
	return item, ok
}

// Validate checks model structure, local schema references, security, and schemas
// across components, paths, webhooks, and callbacks. It does not resolve external
// references or replace version-specific specification schema validation.
func (a *API) Validate() error {
	return validateAPI(a)
}

// Clone copies the API and its Paths, Webhooks, Schemas and Extensions maps.
// Nested pointers, slices and maps remain shared with the original.
func (a *API) Clone() *API {
	clone := *a
	clone.Paths = maps.Clone(a.Paths)
	clone.Webhooks = maps.Clone(a.Webhooks)
	clone.Components.Schemas = maps.Clone(a.Components.Schemas)
	clone.Extensions = maps.Clone(a.Extensions)
	return &clone
}
