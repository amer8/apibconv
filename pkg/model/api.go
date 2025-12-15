// Package model defines the unified internal representation of API specifications.
package model

// API represents the unified API specification model
type API struct {
	Version      string                 // Spec version
	Info         Info                   // API metadata
	Servers      []Server               // Server configurations
	Paths        map[string]PathItem    // API paths/endpoints
	Webhooks     map[string]PathItem    // OpenAPI 3.1 Webhooks
	Components   Components             // Reusable components
	Security     []SecurityRequirement  // Global security
	Tags         []Tag                  // API tags
	ExternalDocs *ExternalDocs          // External documentation
	Extensions   map[string]interface{} // Vendor extensions (x-*)
}

// Info provides metadata about the API.
type Info struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string
	Contact        *Contact
	License        *License
}

// Contact information for the exposed API.
type Contact struct {
	Name  string
	URL   string
	Email string
}

// License information for the exposed API.
type License struct {
	Name string
	URL  string
}

// Server object represents a server for the API.
type Server struct {
	Name        string // Server name/ID (e.g. "production")
	URL         string
	Description string
	Protocol    string // Protocol (e.g. "kafka", "https") - primarily for AsyncAPI
	Variables   map[string]ServerVariable
	Bindings    map[string]interface{} // Protocol-specific server bindings
}

// ServerVariable object represents a variable for server URL template substitution.
type ServerVariable struct {
	Default     string
	Description string
	Enum        []string
}

// Tag object for organizing operations.
type Tag struct {
	Name         string
	Description  string
	ExternalDocs *ExternalDocs
}

// ExternalDocs object for external documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

// NewAPI creates a new API instance
func NewAPI() *API {
	return &API{
		Paths:      make(map[string]PathItem),
		Webhooks:   make(map[string]PathItem),
		Components: NewComponents(),
		Extensions: make(map[string]interface{}),
	}
}

// AddPath adds a path item to the API
func (a *API) AddPath(path string, item *PathItem) {
	if a.Paths == nil {
		a.Paths = make(map[string]PathItem)
	}
	a.Paths[path] = *item // Dereference here
}

// GetPath retrieves a path item from the API
func (a *API) GetPath(path string) (PathItem, bool) {
	item, ok := a.Paths[path]
	return item, ok
}

// Validate validates the API structure
func (a *API) Validate() error {
	// Validation is primarily handled by the `validator` package.
	// This method can be expanded for basic structural validation if needed.
	return nil
}

// Clone creates a deep copy of the API structure
func (a *API) Clone() *API {
	newAPI := *a
	
	// Copy Paths
	if a.Paths != nil {
		newAPI.Paths = make(map[string]PathItem)
		for k := range a.Paths {
			newAPI.Paths[k] = a.Paths[k]
		}
	}

	// Copy Webhooks
	if a.Webhooks != nil {
		newAPI.Webhooks = make(map[string]PathItem)
		for k := range a.Webhooks {
			newAPI.Webhooks[k] = a.Webhooks[k]
		}
	}

	// Copy Components (Simplified shallow copy of maps, deep copy would require more traversal)
	newAPI.Components = a.Components
	if newAPI.Components.Schemas != nil {
		newAPI.Components.Schemas = make(map[string]*Schema)
		for k, v := range a.Components.Schemas {
			newAPI.Components.Schemas[k] = v // Pointer copy, schema itself isn't cloned yet
		}
	}

	// Copy Extensions
	if a.Extensions != nil {
		newAPI.Extensions = make(map[string]interface{})
		for k, v := range a.Extensions {
			newAPI.Extensions[k] = v
		}
	}

	return &newAPI
}
