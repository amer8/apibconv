package asyncapi

// AsyncAPI2 represents the root of an AsyncAPI 2.x document.
type AsyncAPI2 struct {
	AsyncAPI   string                 `yaml:"asyncapi" json:"asyncapi"`
	Info       Info                   `yaml:"info" json:"info"`
	Channels   map[string]Channel2    `yaml:"channels" json:"channels"`
	Servers    map[string]Server      `yaml:"servers,omitempty" json:"servers,omitempty"`
	Components map[string]interface{} `yaml:"components,omitempty" json:"components,omitempty"`
}

// Channel2 represents a channel in AsyncAPI 2.x.
type Channel2 struct {
	Publish    *Operation           `yaml:"publish,omitempty" json:"publish,omitempty"`
	Subscribe  *Operation           `yaml:"subscribe,omitempty" json:"subscribe,omitempty"`
	Parameters map[string]Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Bindings   map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Channel Bindings
}

// Parameter represents a channel parameter.
type Parameter struct {
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Schema      interface{} `yaml:"schema,omitempty" json:"schema,omitempty"` // simplified schema
	Location    string      `yaml:"location,omitempty" json:"location,omitempty"`
}

// AsyncAPI3 represents the root of an AsyncAPI 3.0 document.
type AsyncAPI3 struct {
	AsyncAPI   string                 `yaml:"asyncapi" json:"asyncapi"`
	Info       Info                   `yaml:"info" json:"info"`
	Channels   map[string]Channel3    `yaml:"channels,omitempty" json:"channels,omitempty"`
	Operations map[string]Operation3  `yaml:"operations,omitempty" json:"operations,omitempty"`
	Servers    map[string]ServerV3    `yaml:"servers,omitempty" json:"servers,omitempty"`
	Components map[string]interface{} `yaml:"components,omitempty" json:"components,omitempty"`
}

// ServerV3 object represents a message broker or server in AsyncAPI 3.0.
type ServerV3 struct {
	Host     string                 `yaml:"host" json:"host"`
	Protocol string                 `yaml:"protocol" json:"protocol"`
	Bindings map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Server Bindings
}

// Channel3 represents a channel in AsyncAPI 3.0.
type Channel3 struct {
	Address  string                 `yaml:"address,omitempty" json:"address,omitempty"`
	Messages map[string]interface{} `yaml:"messages,omitempty" json:"messages,omitempty"`	
	Bindings map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Channel Bindings
}

// Operation3 represents an operation in AsyncAPI 3.0.
type Operation3 struct {
	Action      string                 `yaml:"action" json:"action"` // "send" or "receive"
	Channel     interface{}            `yaml:"channel" json:"channel"` // Ref or Object
	Summary     string                 `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	OperationID string                 `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Messages    []interface{}          `yaml:"messages,omitempty" json:"messages,omitempty"`
	Bindings    map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Operation Bindings
}

// Info object provides metadata about the API.
type Info struct {
	Title       string `yaml:"title" json:"title"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Operation object describes a publish or subscribe operation.
type Operation struct {
	OperationID string   `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Summary     string   `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Message     *Message `yaml:"message,omitempty" json:"message,omitempty"`	
	Bindings    map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Operation Bindings
}

// Server object represents a message broker or server.
type Server struct {
	URL      string `yaml:"url" json:"url"`
	Protocol string `yaml:"protocol" json:"protocol"`	
	Bindings map[string]interface{} `yaml:"bindings,omitempty" json:"bindings,omitempty"` // Server Bindings
}

// Message object represents a message exchanged via a channel.
type Message struct {
	Ref         string      `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Name        string      `yaml:"name,omitempty" json:"name,omitempty"`
	Title       string      `yaml:"title,omitempty" json:"title,omitempty"`
	Summary     string      `yaml:"summary,omitempty" json:"summary,omitempty"`
	ContentType string      `yaml:"contentType,omitempty" json:"contentType,omitempty"`
	Payload     interface{} `yaml:"payload,omitempty" json:"payload,omitempty"`
}