package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// AsyncAPI represents an AsyncAPI 2.6 specification structure.
//
// AsyncAPI is used to describe event-driven architectures and asynchronous APIs,
// including protocols like WebSocket, MQTT, Kafka, AMQP, etc.
//
// Example:
//
//	spec := &AsyncAPI{
//	    AsyncAPI: "2.6.0",
//	    Info: Info{
//	        Title:   "My Event API",
//	        Version: "1.0.0",
//	    },
//	    Channels: map[string]Channel{
//	        "user/signedup": {
//	            Subscribe: &AsyncAPIOperation{
//	                Message: &Message{
//	                    Payload: &Schema{Type: "object"},
//	                },
//	            },
//	        },
//	    },
//	}
type AsyncAPI struct {
	AsyncAPI   string                    `json:"asyncapi"`             // AsyncAPI specification version (e.g., "2.6.0")
	Info       Info                      `json:"info"`                 // API metadata including title, description, and version
	Servers    map[string]AsyncAPIServer `json:"servers,omitempty"`    // Server definitions with protocol information
	Channels   map[string]Channel        `json:"channels"`             // Channel definitions for publish/subscribe operations
	Components *AsyncAPIComponents       `json:"components,omitempty"` // Reusable components (messages, schemas)
}

// AsyncAPIServer represents a server in AsyncAPI specification.
//
// Unlike OpenAPI servers which are URL-based for REST APIs, AsyncAPI servers
// include protocol information for event-driven communication.
//
// Example:
//
//	server := AsyncAPIServer{
//	    URL: "mqtt://test.mosquitto.org:1883",
//	    Protocol: "mqtt",
//	    Description: "Test MQTT broker",
//	}
type AsyncAPIServer struct {
	URL         string `json:"url"`                   // Server URL including protocol and port
	Protocol    string `json:"protocol"`              // Protocol: mqtt, ws, kafka, amqp, http, etc.
	Description string `json:"description,omitempty"` // Description of the server
}

// Channel represents a channel in AsyncAPI specification.
//
// A channel is a communication endpoint (topic, queue, routing key, etc.) where
// messages can be published to or subscribed from. In AsyncAPI 2.x, the channel
// key represents the actual address/topic.
//
// Example:
//
//	channel := Channel{
//	    Description: "User signup events",
//	    Subscribe: &AsyncAPIOperation{
//	        Summary: "User signed up notification",
//	        Message: &Message{
//	            Name: "UserSignedUp",
//	            Payload: &Schema{Type: "object"},
//	        },
//	    },
//	}
type Channel struct {
	Description string             `json:"description,omitempty"` // Description of the channel
	Subscribe   *AsyncAPIOperation `json:"subscribe,omitempty"`   // Client subscribes to receive messages
	Publish     *AsyncAPIOperation `json:"publish,omitempty"`     // Client publishes messages to this channel
}

// AsyncAPIOperation represents an operation on a channel (publish or subscribe).
//
// Operations in AsyncAPI describe the action that a consumer or producer performs
// on a channel.
//
// Example:
//
//	operation := &AsyncAPIOperation{
//	    OperationID: "onUserSignedUp",
//	    Summary: "Handle user signup events",
//	    Message: &Message{
//	        Name: "UserSignedUp",
//	        ContentType: "application/json",
//	        Payload: &Schema{Type: "object"},
//	    },
//	}
type AsyncAPIOperation struct {
	OperationID string   `json:"operationId,omitempty"` // Unique operation identifier
	Summary     string   `json:"summary,omitempty"`     // Short summary of the operation
	Description string   `json:"description,omitempty"` // Detailed description
	Message     *Message `json:"message,omitempty"`     // Message definition for this operation
}

// Message represents a message in AsyncAPI specification.
//
// Messages are the data structures sent over channels. They contain a payload
// schema and metadata about the message format.
//
// Example:
//
//	message := &Message{
//	    Name: "UserSignedUp",
//	    Title: "User Signed Up Event",
//	    Summary: "Notification when a user signs up",
//	    ContentType: "application/json",
//	    Payload: &Schema{
//	        Type: "object",
//	        Properties: map[string]*Schema{
//	            "userId": {Type: "string"},
//	            "email": {Type: "string"},
//	        },
//	    },
//	}
type Message struct {
	Name        string  `json:"name,omitempty"`        // Message name/identifier
	Title       string  `json:"title,omitempty"`       // Human-readable title
	Summary     string  `json:"summary,omitempty"`     // Short summary of the message
	Description string  `json:"description,omitempty"` // Detailed description
	ContentType string  `json:"contentType,omitempty"` // Content type (e.g., "application/json")
	Payload     *Schema `json:"payload,omitempty"`     // Message payload schema
}

// AsyncAPIComponents holds reusable components for AsyncAPI specification.
//
// Components allow you to define messages and schemas once and reference them
// throughout your AsyncAPI specification.
//
// Example:
//
//	components := &AsyncAPIComponents{
//	    Messages: map[string]*Message{
//	        "UserSignedUp": {
//	            Payload: &Schema{Type: "object"},
//	        },
//	    },
//	    Schemas: map[string]*Schema{
//	        "User": {
//	            Type: "object",
//	            Properties: map[string]*Schema{
//	                "id": {Type: "string"},
//	                "email": {Type: "string"},
//	            },
//	        },
//	    },
//	}
type AsyncAPIComponents struct {
	Messages map[string]*Message `json:"messages,omitempty"` // Reusable message definitions
	Schemas  map[string]*Schema  `json:"schemas,omitempty"`  // Reusable schema definitions
}

// ParseAsync parses AsyncAPI JSON or YAML data into an AsyncAPI struct.
//
// This function reads AsyncAPI specification in JSON or YAML format and unmarshals it
// into the AsyncAPI Go struct.
//
// Parameters:
//   - data: JSON or YAML byte array containing AsyncAPI specification
//
// Returns:
//   - *AsyncAPI: Parsed AsyncAPI structure
//   - error: Error if parsing fails
//
// Example:
//
//	data := []byte(`{"asyncapi": "2.6.0", "info": {"title": "My API", "version": "1.0.0"}, "channels": {}}`)
//	spec, err := converter.ParseAsync(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseAsync(data []byte) (*AsyncAPI, error) {
	var spec AsyncAPI

	// Try JSON first
	if isJSON(data) {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse AsyncAPI JSON: %w", err)
		}
		return &spec, nil
	}

	// Try YAML
	if err := UnmarshalYAML(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse AsyncAPI YAML: %w", err)
	}

	return &spec, nil
}

// ParseAsyncAPI is a deprecated alias for ParseAsync.
//
// Deprecated: Use ParseAsync instead.
func ParseAsyncAPI(data []byte) (*AsyncAPI, error) {
	return ParseAsync(data)
}

// ParseAsyncReader parses AsyncAPI JSON or YAML from an io.Reader.
//
// This is a streaming version of ParseAsync that reads from an io.Reader
// instead of a byte slice.
//
// Example:
//
//	file, err := os.Open("asyncapi.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	spec, err := converter.ParseAsyncReader(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseAsyncReader(r io.Reader) (*AsyncAPI, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseAsync(data)
}

// ParseAsyncAPIReader is a deprecated alias for ParseAsyncReader.
//
// Deprecated: Use ParseAsyncReader instead.
func ParseAsyncAPIReader(r io.Reader) (*AsyncAPI, error) {
	return ParseAsyncReader(r)
}

// ToBlueprint converts an AsyncAPI specification to API Blueprint format.
//
// This method converts AsyncAPI channels and operations into API Blueprint paths
// and operations. The conversion maps:
//   - AsyncAPI channels -> API Blueprint paths
//   - Subscribe operations -> GET operations (receiving messages)
//   - Publish operations -> POST operations (sending messages)
//   - Message payloads -> Request/Response bodies
//
// Returns:
//   - string: API Blueprint formatted output
//
// Example:
//
//	asyncSpec := &AsyncAPI{...}
//	blueprint := asyncSpec.ToBlueprint()
//	fmt.Println(blueprint)
func (spec *AsyncAPI) ToBlueprint() string {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAsyncAPIBlueprint(buf, spec)
	return buf.String()
}

// WriteBlueprint writes the AsyncAPI specification in API Blueprint format to the writer.
func (spec *AsyncAPI) WriteBlueprint(w io.Writer) error {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAsyncAPIBlueprint(buf, spec)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeAsyncAPIBlueprint writes the API Blueprint format to the buffer
func writeAsyncAPIBlueprint(buf *bytes.Buffer, spec *AsyncAPI) {
	// Write header
	buf.WriteString(APIBlueprintFormat + "\n\n")

	// Write title
	if spec.Info.Title != "" {
		buf.WriteString("# ")
		buf.WriteString(spec.Info.Title)
		buf.WriteString("\n\n")
	}

	// Write description
	if spec.Info.Description != "" {
		buf.WriteString(spec.Info.Description)
		buf.WriteString("\n\n")
	}

	// Write host (use first server if available)
	if len(spec.Servers) > 0 {
		// Get first server (map iteration order is undefined, so we sort keys)
		var serverKeys []string
		for k := range spec.Servers {
			serverKeys = append(serverKeys, k)
		}
		sort.Strings(serverKeys)
		firstServer := spec.Servers[serverKeys[0]]
		buf.WriteString("HOST: ")
		buf.WriteString(firstServer.URL)
		buf.WriteString("\n\n")
	}

	// Sort channel names for consistent output
	channelNames := make([]string, 0, len(spec.Channels))
	for name := range spec.Channels {
		channelNames = append(channelNames, name)
	}
	sort.Strings(channelNames)

	// Write channels as paths
	for _, channelName := range channelNames {
		channel := spec.Channels[channelName]
		writeAsyncAPIChannel(buf, channelName, channel)
	}
}

// AsyncAPIToAPIBlueprint is a deprecated alias for AsyncAPI.ToBlueprint.
//
// Deprecated: Use AsyncAPI.ToBlueprint instead.
func AsyncAPIToAPIBlueprint(spec *AsyncAPI) string {
	return spec.ToBlueprint()
}

// writeAsyncAPIChannel writes a single AsyncAPI channel as API Blueprint path
func writeAsyncAPIChannel(buf *bytes.Buffer, channelName string, channel Channel) {
	// Convert channel name to path format (replace / with /)
	path := "/" + channelName

	// Write channel header
	buf.WriteString("## ")
	buf.WriteString(path)
	buf.WriteString(" [")
	buf.WriteString(path)
	buf.WriteString("]\n\n")

	if channel.Description != "" {
		buf.WriteString(channel.Description)
		buf.WriteString("\n\n")
	}

	// Write subscribe operation as GET (receiving messages)
	if channel.Subscribe != nil {
		writeAsyncAPIOperation(buf, http.MethodGet, channel.Subscribe, "Subscribe to receive messages")
	}

	// Write publish operation as POST (sending messages)
	if channel.Publish != nil {
		writeAsyncAPIOperation(buf, http.MethodPost, channel.Publish, "Publish a message")
	}
}

// writeAsyncAPIOperation writes an AsyncAPI operation as API Blueprint operation
func writeAsyncAPIOperation(buf *bytes.Buffer, method string, op *AsyncAPIOperation, defaultSummary string) {
	summary := op.Summary
	if summary == "" {
		summary = defaultSummary
	}

	buf.WriteString("### ")
	buf.WriteString(summary)
	buf.WriteString(" [")
	buf.WriteString(method)
	buf.WriteString("]\n\n")

	if op.Description != "" {
		buf.WriteString(op.Description)
		buf.WriteString("\n\n")
	}

	// Write message as request (for POST) or response (for GET)
	if op.Message != nil {
		if method == http.MethodPost {
			writeAsyncAPIMessageAsRequest(buf, op.Message)
		} else {
			writeAsyncAPIMessageAsResponse(buf, op.Message)
		}
	}
}

// writeAsyncAPIMessageAsRequest writes a message as API Blueprint request
func writeAsyncAPIMessageAsRequest(buf *bytes.Buffer, msg *Message) {
	contentType := msg.ContentType
	if contentType == "" {
		contentType = MediaTypeJSON
	}

	buf.WriteString("+ Request (")
	buf.WriteString(contentType)
	buf.WriteString(")\n\n")

	if msg.Payload != nil && msg.Payload.Example != nil {
		exampleJSON, err := json.MarshalIndent(msg.Payload.Example, "        ", "    ")
		if err == nil {
			buf.WriteString("    + Body\n\n")
			buf.WriteString("            ")
			buf.Write(exampleJSON)
			buf.WriteString("\n\n")
		}
	}
}

// writeAsyncAPIMessageAsResponse writes a message as API Blueprint response
func writeAsyncAPIMessageAsResponse(buf *bytes.Buffer, msg *Message) {
	contentType := msg.ContentType
	if contentType == "" {
		contentType = MediaTypeJSON
	}

	buf.WriteString("+ Response 200 (")
	buf.WriteString(contentType)
	buf.WriteString(")\n\n")

	if msg.Payload != nil && msg.Payload.Example != nil {
		exampleJSON, err := json.MarshalIndent(msg.Payload.Example, "        ", "    ")
		if err == nil {
			buf.WriteString("    + Body\n\n")
			buf.WriteString("            ")
			buf.Write(exampleJSON)
			buf.WriteString("\n\n")
		}
	}
}

// ConvertAsyncAPIToAPIBlueprint converts AsyncAPI JSON to API Blueprint format using streaming I/O.
//
// This function reads AsyncAPI JSON from an io.Reader, converts it to API Blueprint
// format, and writes the result to an io.Writer.
//
// Example:
//
//	input, err := os.Open("asyncapi.json")
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
//	if err := converter.ConvertAsyncAPIToAPIBlueprint(input, output); err != nil {
//	    log.Fatal(err)
//	}
func ConvertAsyncAPIToAPIBlueprint(r io.Reader, w io.Writer) error {
	spec, err := ParseAsyncReader(r)
	if err != nil {
		return err
	}

	blueprint := spec.ToBlueprint()
	_, err = w.Write([]byte(blueprint))
	return err
}

// ToAsyncAPI converts an API Blueprint (OpenAPI struct) to AsyncAPI format.
//
// This function converts OpenAPI/API Blueprint paths to AsyncAPI channels:
//   - GET operations -> Subscribe operations (receiving messages)
//   - POST operations -> Publish operations (sending messages)
//   - Response bodies -> Message payloads for GET
//   - Request bodies -> Message payloads for POST
//
// Parameters:
//   - protocol: Protocol to use for AsyncAPI servers (e.g., "http", "ws", "mqtt", "kafka")
//
// Returns:
//   - *AsyncAPI: Converted AsyncAPI specification
//
// Example:
//
//	openAPISpec := converter.ParseAPIBlueprint(data)
//	asyncSpec := openAPISpec.ToAsyncAPI("ws")
func (spec *OpenAPI) ToAsyncAPI(protocol string) *AsyncAPI {
	asyncSpec := &AsyncAPI{
		AsyncAPI: AsyncAPIVersion26,
		Info:     spec.Info,
		Channels: make(map[string]Channel),
	}

	// Convert servers
	if len(spec.Servers) > 0 {
		asyncSpec.Servers = make(map[string]AsyncAPIServer)
		for i, server := range spec.Servers {
			serverName := fmt.Sprintf("server%d", i)
			asyncSpec.Servers[serverName] = AsyncAPIServer{
				URL:         server.URL,
				Protocol:    protocol,
				Description: server.Description,
			}
		}
	}

	// Convert paths to channels
	for path, pathItem := range spec.Paths {
		// Remove leading slash for channel name
		channelName := path
		if channelName != "" && channelName[0] == '/' {
			channelName = channelName[1:]
		}

		channel := Channel{}

		// Convert GET to Subscribe (client receives messages)
		if pathItem.Get != nil {
			channel.Subscribe = convertOperationToAsyncAPI(pathItem.Get, false)
		}

		// Convert POST to Publish (client sends messages)
		if pathItem.Post != nil {
			channel.Publish = convertOperationToAsyncAPI(pathItem.Post, true)
		}

		// If we have any operations, add the channel
		if channel.Subscribe != nil || channel.Publish != nil {
			asyncSpec.Channels[channelName] = channel
		}
	}

	return asyncSpec
}

// APIBlueprintToAsyncAPI is a deprecated alias for OpenAPI.ToAsyncAPI.
//
// Deprecated: Use OpenAPI.ToAsyncAPI instead.
func APIBlueprintToAsyncAPI(spec *OpenAPI, protocol string) *AsyncAPI {
	return spec.ToAsyncAPI(protocol)
}

// convertOperationToAsyncAPI converts an OpenAPI operation to AsyncAPI operation
func convertOperationToAsyncAPI(op *Operation, isPublish bool) *AsyncAPIOperation {
	asyncOp := &AsyncAPIOperation{
		Summary:     op.Summary,
		Description: op.Description,
	}

	// Create message from request body (for publish) or response (for subscribe)
	message := &Message{}

	if isPublish && op.RequestBody != nil {
		// Use request body for publish operations
		for contentType, mediaType := range op.RequestBody.Content {
			message.ContentType = contentType
			message.Payload = mediaType.Schema
			if mediaType.Example != nil {
				if message.Payload == nil {
					message.Payload = &Schema{}
				}
				message.Payload.Example = mediaType.Example
			}
			break // Use first content type
		}
	} else if !isPublish && op.Responses != nil {
		// Use response body for subscribe operations (typically 200 response)
		if resp, ok := op.Responses["200"]; ok {
			for contentType, mediaType := range resp.Content {
				message.ContentType = contentType
				message.Payload = mediaType.Schema
				if mediaType.Example != nil {
					if message.Payload == nil {
						message.Payload = &Schema{}
					}
					message.Payload.Example = mediaType.Example
				}
				break // Use first content type
			}
		}
	}

	asyncOp.Message = message
	return asyncOp
}

// ConvertAPIBlueprintToAsyncAPI converts API Blueprint format to AsyncAPI JSON using streaming I/O.
//
// This function reads API Blueprint from an io.Reader, converts it to AsyncAPI format,
// and writes the JSON result to an io.Writer.
//
// Parameters:
//   - r: Reader containing API Blueprint format
//   - w: Writer for AsyncAPI JSON output
//   - protocol: Protocol to use in AsyncAPI servers (e.g., "ws", "mqtt", "kafka")
//
// Example:
//
//	input, err := os.Open("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input.Close()
//
//	output, err := os.Create("asyncapi.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	if err := converter.ConvertAPIBlueprintToAsyncAPI(input, output, "ws"); err != nil {
//	    log.Fatal(err)
//	}
func ConvertAPIBlueprintToAsyncAPI(r io.Reader, w io.Writer, protocol string) error {
	// First parse API Blueprint to OpenAPI structure
	spec, err := ParseBlueprintReader(r)
	if err != nil {
		return fmt.Errorf("failed to parse API Blueprint: %w", err)
	}

	// Convert to AsyncAPI
	asyncSpec := APIBlueprintToAsyncAPI(spec, protocol)

	// Marshal to JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(asyncSpec); err != nil {
		return fmt.Errorf("failed to encode AsyncAPI: %w", err)
	}

	return nil
}

// ============================================================================
// AsyncAPI 3.0 Support
// ============================================================================

// AsyncAPIV3 represents an AsyncAPI 3.0 specification structure.
//
// AsyncAPI 3.0 introduces significant changes from 2.x:
//   - Operations are at root level instead of nested in channels
//   - Operations use 'action' field with 'send'/'receive' instead of 'publish'/'subscribe'
//   - Channels have separate 'address' field
//   - Operations reference channels via $ref
//
// Example:
//
//	spec := &AsyncAPIV3{
//	    AsyncAPI: "3.0.0",
//	    Info: Info{
//	        Title:   "My Event API",
//	        Version: "1.0.0",
//	    },
//	    Channels: map[string]ChannelV3{
//	        "userSignup": {
//	            Address: "user/signedup",
//	            Messages: map[string]*Message{
//	                "UserSignedUp": {Payload: &Schema{Type: "object"}},
//	            },
//	        },
//	    },
//	    Operations: map[string]OperationV3{
//	        "onUserSignup": {
//	            Action: "receive",
//	            Channel: &ChannelReference{Ref: "#/channels/userSignup"},
//	        },
//	    },
//	}
type AsyncAPIV3 struct {
	AsyncAPI   string                    `json:"asyncapi"`             // AsyncAPI specification version (e.g., "3.0.0")
	Info       Info                      `json:"info"`                 // API metadata including title, description, and version
	Servers    map[string]AsyncAPIServer `json:"servers,omitempty"`    // Server definitions with protocol information
	Channels   map[string]ChannelV3      `json:"channels,omitempty"`   // Channel definitions (address + messages)
	Operations map[string]OperationV3    `json:"operations"`           // Operations with send/receive actions
	Components *AsyncAPIComponents       `json:"components,omitempty"` // Reusable components (messages, schemas)
}

// ChannelV3 represents a channel in AsyncAPI 3.0 specification.
//
// In v3, the channel key is an arbitrary identifier, and the actual
// address/topic is specified in the Address field.
//
// Example:
//
//	channel := ChannelV3{
//	    Address: "user/signedup",
//	    Description: "User signup events",
//	    Messages: map[string]*Message{
//	        "UserSignedUp": {
//	            Payload: &Schema{Type: "object"},
//	        },
//	    },
//	}
type ChannelV3 struct {
	Address     string              `json:"address,omitempty"`     // Channel address/topic/queue name
	Description string              `json:"description,omitempty"` // Description of the channel
	Messages    map[string]*Message `json:"messages,omitempty"`    // Messages that can be sent on this channel
}

// OperationV3 represents an operation in AsyncAPI 3.0 specification.
//
// Operations in v3 are at the root level and reference channels.
// The 'action' field specifies whether the application sends or receives.
//
// Example:
//
//	operation := OperationV3{
//	    Action: "receive",
//	    Summary: "Receive user signup notifications",
//	    Channel: &ChannelReference{Ref: "#/channels/userSignup"},
//	}
type OperationV3 struct {
	Action      string            `json:"action"`                // Action: "send" or "receive"
	Summary     string            `json:"summary,omitempty"`     // Short summary of the operation
	Description string            `json:"description,omitempty"` // Detailed description
	Channel     *ChannelReference `json:"channel,omitempty"`     // Reference to a channel
}

// ChannelReference represents a reference to a channel in AsyncAPI 3.0.
//
// Example:
//
//	ref := &ChannelReference{
//	    Ref: "#/channels/userSignup",
//	}
type ChannelReference struct {
	Ref string `json:"$ref"` // JSON reference to a channel
}

// DetectAsyncAPIVersion detects the AsyncAPI version from the version string.
//
// Returns:
//   - 2: for AsyncAPI 2.x versions
//   - 3: for AsyncAPI 3.x versions
//   - 0: for unknown or invalid versions
//
// Example:
//
//	version := DetectAsyncAPIVersion("2.6.0") // returns 2
//	version := DetectAsyncAPIVersion("3.0.0") // returns 3
func DetectAsyncAPIVersion(asyncapiVersion string) int {
	if asyncapiVersion == "" {
		return 0
	}
	// Check first character of version
	if asyncapiVersion[0] == '2' {
		return 2
	}
	if asyncapiVersion[0] == '3' {
		return 3
	}
	return 0
}

// ParseAsyncV3 parses AsyncAPI 3.0 JSON or YAML data into an AsyncAPIV3 struct.
//
// Example:
//
//	data := []byte(`{"asyncapi": "3.0.0", "info": {"title": "My API", "version": "1.0.0"}, "channels": {}, "operations": {}}`)
//	spec, err := converter.ParseAsyncV3(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseAsyncV3(data []byte) (*AsyncAPIV3, error) {
	var spec AsyncAPIV3

	// Try JSON first
	if isJSON(data) {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse AsyncAPI v3 JSON: %w", err)
		}
		return &spec, nil
	}

	// Try YAML
	if err := UnmarshalYAML(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse AsyncAPI v3 YAML: %w", err)
	}

	return &spec, nil
}

// ParseAsyncAPIV3 is a deprecated alias for ParseAsyncV3.
//
// Deprecated: Use ParseAsyncV3 instead.
func ParseAsyncAPIV3(data []byte) (*AsyncAPIV3, error) {
	return ParseAsyncV3(data)
}

// ParseAsyncV3Reader parses AsyncAPI 3.0 JSON or YAML from an io.Reader.
//
// Example:
//
//	file, err := os.Open("asyncapi-v3.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	spec, err := converter.ParseAsyncV3Reader(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseAsyncV3Reader(r io.Reader) (*AsyncAPIV3, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseAsyncV3(data)
}

// ParseAsyncAPIV3Reader is a deprecated alias for ParseAsyncV3Reader.
//
// Deprecated: Use ParseAsyncV3Reader instead.
func ParseAsyncAPIV3Reader(r io.Reader) (*AsyncAPIV3, error) {
	return ParseAsyncV3Reader(r)
}

// ToBlueprint converts an AsyncAPI 3.0 specification to API Blueprint format.
//
// The conversion maps:
//   - AsyncAPI channels -> API Blueprint paths (using channel address)
//   - Receive operations -> GET operations (receiving messages)
//   - Send operations -> POST operations (sending messages)
//   - Channel messages -> Request/Response bodies
//
// Example:
//
//	asyncSpec := &AsyncAPIV3{...}
//	blueprint := asyncSpec.ToBlueprint()
//	fmt.Println(blueprint)
func (spec *AsyncAPIV3) ToBlueprint() string {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAsyncAPIV3Blueprint(buf, spec)
	return buf.String()
}

// WriteBlueprint writes the AsyncAPI 3.0 specification in API Blueprint format to the writer.
func (spec *AsyncAPIV3) WriteBlueprint(w io.Writer) error {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAsyncAPIV3Blueprint(buf, spec)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeAsyncAPIV3Blueprint writes the API Blueprint format to the buffer
func writeAsyncAPIV3Blueprint(buf *bytes.Buffer, spec *AsyncAPIV3) {
	// Write header
	buf.WriteString(APIBlueprintFormat + "\n\n")

	// Write title
	if spec.Info.Title != "" {
		buf.WriteString("# ")
		buf.WriteString(spec.Info.Title)
		buf.WriteString("\n\n")
	}

	// Write description
	if spec.Info.Description != "" {
		buf.WriteString(spec.Info.Description)
		buf.WriteString("\n\n")
	}

	// Write host (use first server if available)
	if len(spec.Servers) > 0 {
		var serverKeys []string
		for k := range spec.Servers {
			serverKeys = append(serverKeys, k)
		}
		sort.Strings(serverKeys)
		firstServer := spec.Servers[serverKeys[0]]
		buf.WriteString("HOST: ")
		buf.WriteString(firstServer.URL)
		buf.WriteString("\n\n")
	}

	// Group operations by channel
	channelOps := make(map[string][]struct {
		opID string
		op   OperationV3
	})

	for opID, op := range spec.Operations {
		if op.Channel != nil && op.Channel.Ref != "" {
			// Extract channel ID from reference (e.g., "#/channels/userSignup" -> "userSignup")
			channelID := extractChannelID(op.Channel.Ref)
			channelOps[channelID] = append(channelOps[channelID], struct {
				opID string
				op   OperationV3
			}{opID, op})
		}
	}

	// Sort channel IDs for consistent output
	channelIDs := make([]string, 0, len(spec.Channels))
	for channelID := range spec.Channels {
		channelIDs = append(channelIDs, channelID)
	}
	sort.Strings(channelIDs)

	// Write channels as paths
	for _, channelID := range channelIDs {
		channel := spec.Channels[channelID]
		ops := channelOps[channelID]
		writeAsyncAPIV3Channel(buf, channelID, channel, ops)
	}
}

// AsyncAPIV3ToAPIBlueprint is a deprecated alias for AsyncAPIV3.ToBlueprint.
//
// Deprecated: Use AsyncAPIV3.ToBlueprint instead.
func AsyncAPIV3ToAPIBlueprint(spec *AsyncAPIV3) string {
	return spec.ToBlueprint()
}

// extractChannelID extracts the channel ID from a JSON reference
// e.g., "#/channels/userSignup" -> "userSignup"
func extractChannelID(ref string) string {
	// Simple extraction - assumes format #/channels/<id>
	prefix := "#/channels/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ref
}

// writeAsyncAPIV3Channel writes a single AsyncAPI v3 channel as API Blueprint path
func writeAsyncAPIV3Channel(buf *bytes.Buffer, channelID string, channel ChannelV3, ops []struct {
	opID string
	op   OperationV3
},
) {
	// Use channel address for the path
	path := "/" + channel.Address
	if channel.Address == "" {
		path = "/" + channelID
	}

	// Write channel header
	buf.WriteString("## ")
	buf.WriteString(path)
	buf.WriteString(" [")
	buf.WriteString(path)
	buf.WriteString("]\n\n")

	if channel.Description != "" {
		buf.WriteString(channel.Description)
		buf.WriteString("\n\n")
	}

	// Get first message from channel for payload
	var firstMessage *Message
	for _, msg := range channel.Messages {
		firstMessage = msg
		break
	}

	// Write operations
	for _, opInfo := range ops {
		switch opInfo.op.Action {
		case ActionReceive:
			// Receive -> GET (receiving messages)
			writeAsyncAPIV3Operation(buf, http.MethodGet, opInfo.op, firstMessage, "Receive messages")
		case ActionSend:
			// Send -> POST (sending messages)
			writeAsyncAPIV3Operation(buf, http.MethodPost, opInfo.op, firstMessage, "Send a message")
		}
	}
}

// writeAsyncAPIV3Operation writes an AsyncAPI v3 operation as API Blueprint operation
func writeAsyncAPIV3Operation(buf *bytes.Buffer, method string, op OperationV3, msg *Message, defaultSummary string) {
	summary := op.Summary
	if summary == "" {
		summary = defaultSummary
	}

	buf.WriteString("### ")
	buf.WriteString(summary)
	buf.WriteString(" [")
	buf.WriteString(method)
	buf.WriteString("]\n\n")

	if op.Description != "" {
		buf.WriteString(op.Description)
		buf.WriteString("\n\n")
	}

	// Write message as request (for POST) or response (for GET)
	if msg != nil {
		if method == http.MethodPost {
			writeAsyncAPIMessageAsRequest(buf, msg)
		} else {
			writeAsyncAPIMessageAsResponse(buf, msg)
		}
	}
}

// ConvertAsyncAPIV3ToAPIBlueprint converts AsyncAPI 3.0 JSON to API Blueprint format using streaming I/O.
//
// Example:
//
//	input, err := os.Open("asyncapi-v3.json")
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
//	if err := converter.ConvertAsyncAPIV3ToAPIBlueprint(input, output); err != nil {
//	    log.Fatal(err)
//	}
func ConvertAsyncAPIV3ToAPIBlueprint(r io.Reader, w io.Writer) error {
	spec, err := ParseAsyncV3Reader(r)
	if err != nil {
		return err
	}

	blueprint := spec.ToBlueprint()
	_, err = w.Write([]byte(blueprint))
	return err
}

// ToAsyncAPIV3 converts an API Blueprint (OpenAPI struct) to AsyncAPI 3.0 format.
//
// This function converts OpenAPI/API Blueprint paths to AsyncAPI v3 channels and operations:
//   - GET operations -> Receive operations (receiving messages)
//   - POST operations -> Send operations (sending messages)
//   - Response bodies -> Message payloads for GET
//   - Request bodies -> Message payloads for POST
//
// Example:
//
//	openAPISpec := converter.ParseAPIBlueprint(data)
//	asyncSpec := openAPISpec.ToAsyncAPIV3("ws")
func (spec *OpenAPI) ToAsyncAPIV3(protocol string) *AsyncAPIV3 {
	asyncSpec := &AsyncAPIV3{
		AsyncAPI:   AsyncAPIVersion30,
		Info:       spec.Info,
		Channels:   make(map[string]ChannelV3),
		Operations: make(map[string]OperationV3),
	}

	// Convert servers
	if len(spec.Servers) > 0 {
		asyncSpec.Servers = make(map[string]AsyncAPIServer)
		for i, server := range spec.Servers {
			serverName := fmt.Sprintf("server%d", i)
			asyncSpec.Servers[serverName] = AsyncAPIServer{
				URL:         server.URL,
				Protocol:    protocol,
				Description: server.Description,
			}
		}
	}

	// Convert paths to channels and operations
	operationCounter := 0
	for path, pathItem := range spec.Paths {
		// Remove leading slash for channel ID
		channelID := path
		if channelID != "" && channelID[0] == '/' {
			channelID = channelID[1:]
		}
		// Replace slashes with underscores for channel ID
		channelID = sanitizeChannelID(channelID)

		// Create channel with address
		channel := ChannelV3{
			Address:  path[1:], // Remove leading slash
			Messages: make(map[string]*Message),
		}

		// Convert GET to Receive operation
		if pathItem.Get != nil {
			operationCounter++
			opID := fmt.Sprintf("receive%s%d", sanitizeOperationID(path), operationCounter)

			// Add message to channel
			msgID := "Message"
			message := extractMessageFromOperation(pathItem.Get, false)
			if message != nil {
				channel.Messages[msgID] = message
			}

			// Create receive operation
			asyncSpec.Operations[opID] = OperationV3{
				Action:      ActionReceive,
				Summary:     pathItem.Get.Summary,
				Description: pathItem.Get.Description,
				Channel: &ChannelReference{
					Ref: fmt.Sprintf("#/channels/%s", channelID),
				},
			}
		}

		// Convert POST to Send operation
		if pathItem.Post != nil {
			operationCounter++
			opID := fmt.Sprintf("send%s%d", sanitizeOperationID(path), operationCounter)

			// Add message to channel
			msgID := "Message"
			message := extractMessageFromOperation(pathItem.Post, true)
			if message != nil {
				if len(channel.Messages) == 0 {
					channel.Messages[msgID] = message
				} else {
					// If we already have a message from GET, use a different ID
					channel.Messages["SendMessage"] = message
				}
			}

			// Create send operation
			asyncSpec.Operations[opID] = OperationV3{
				Action:      ActionSend,
				Summary:     pathItem.Post.Summary,
				Description: pathItem.Post.Description,
				Channel: &ChannelReference{
					Ref: fmt.Sprintf("#/channels/%s", channelID),
				},
			}
		}

		// Add channel if it has messages or operations
		if len(channel.Messages) > 0 {
			asyncSpec.Channels[channelID] = channel
		}
	}

	return asyncSpec
}

// APIBlueprintToAsyncAPIV3 is a deprecated alias for OpenAPI.ToAsyncAPIV3.
//
// Deprecated: Use OpenAPI.ToAsyncAPIV3 instead.
func APIBlueprintToAsyncAPIV3(spec *OpenAPI, protocol string) *AsyncAPIV3 {
	return spec.ToAsyncAPIV3(protocol)
}

// sanitizeChannelID converts a path to a valid channel ID.
// Uses strings.Builder for efficient string concatenation and preserves
// hyphens vs underscores to avoid collisions (e.g., /user-id vs /user_id).
func sanitizeChannelID(path string) string {
	// Build a sanitized channel ID with strings.Builder for performance
	var builder strings.Builder
	builder.Grow(len(path)) // Pre-allocate capacity

	for _, ch := range path {
		//nolint:gocritic // if-else chain is clearer than switch for character ranges
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
		} else if ch == '/' {
			// Preserve path separators as underscores
			builder.WriteRune('_')
		} else if ch == '-' {
			// Preserve hyphens (different from underscores to avoid collisions)
			builder.WriteRune('-')
		} else if ch == '_' {
			// Keep underscores as-is
			builder.WriteRune('_')
		}
		// Other special characters are dropped
	}

	result := builder.String()

	// If completely empty after sanitization, use default
	if result == "" {
		return "channel"
	}

	// Trim leading/trailing underscores for cleaner IDs
	result = strings.Trim(result, "_")
	if result == "" {
		return "channel"
	}

	return result
}

// sanitizeOperationID converts a path to a valid operation ID suffix
func sanitizeOperationID(path string) string {
	result := sanitizeChannelID(path)
	// Capitalize first letter
	if result != "" && result[0] >= 'a' && result[0] <= 'z' {
		result = string(result[0]-32) + result[1:]
	}
	return result
}

// extractMessageFromOperation extracts a message from an OpenAPI operation
func extractMessageFromOperation(op *Operation, isPublish bool) *Message {
	message := &Message{}

	if isPublish && op.RequestBody != nil {
		// Use request body for send operations
		for contentType, mediaType := range op.RequestBody.Content {
			message.ContentType = contentType
			message.Payload = mediaType.Schema
			if mediaType.Example != nil {
				if message.Payload == nil {
					message.Payload = &Schema{}
				}
				message.Payload.Example = mediaType.Example
			}
			return message
		}
	} else if !isPublish && op.Responses != nil {
		// Use response body for receive operations (typically 200 response)
		if resp, ok := op.Responses["200"]; ok {
			for contentType, mediaType := range resp.Content {
				message.ContentType = contentType
				message.Payload = mediaType.Schema
				if mediaType.Example != nil {
					if message.Payload == nil {
						message.Payload = &Schema{}
					}
					message.Payload.Example = mediaType.Example
				}
				return message
			}
		}
	}

	return message
}

// ConvertAPIBlueprintToAsyncAPIV3 converts API Blueprint format to AsyncAPI 3.0 JSON using streaming I/O.
//
// Example:
//
//	input, err := os.Open("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input.Close()
//
//	output, err := os.Create("asyncapi-v3.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	if err := converter.ConvertAPIBlueprintToAsyncAPIV3(input, output, "ws"); err != nil {
//	    log.Fatal(err)
//	}
func ConvertAPIBlueprintToAsyncAPIV3(r io.Reader, w io.Writer, protocol string) error {
	// First parse API Blueprint to OpenAPI structure
	spec, err := ParseBlueprintReader(r)
	if err != nil {
		return fmt.Errorf("failed to parse API Blueprint: %w", err)
	}

	// Convert to AsyncAPI v3
	asyncSpec := APIBlueprintToAsyncAPIV3(spec, protocol)

	// Marshal to JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(asyncSpec); err != nil {
		return fmt.Errorf("failed to encode AsyncAPI v3: %w", err)
	}

	return nil
}

// ParseAsyncAPIAny parses AsyncAPI JSON or YAML (any version) and returns the appropriate struct.
//
// Returns:
//   - *AsyncAPI (v2) if version is 2.x
//   - *AsyncAPIV3 if version is 3.x
//   - error if version is unsupported or parsing fails
//
// Example:
//
//	data := []byte(`{"asyncapi": "3.0.0", ...}`)
//	spec, version, err := converter.ParseAsyncAPIAny(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	switch version {
//	case 2:
//	    v2Spec := spec.(*AsyncAPI)
//	case 3:
//	    v3Spec := spec.(*AsyncAPIV3)
//	}
func ParseAsyncAPIAny(data []byte) (spec any, version int, err error) {
	// First, detect version by parsing just the version field
	var versionCheck struct {
		AsyncAPI string `json:"asyncapi"`
	}

	// Try JSON first
	if err = json.Unmarshal(data, &versionCheck); err != nil {
		// If JSON fails, try YAML
		if yamlErr := UnmarshalYAML(data, &versionCheck); yamlErr != nil {
			// If both fail, return an error
			return nil, 0, fmt.Errorf("failed to detect AsyncAPI version from JSON or YAML: %w", err)
		}
	}

	version = DetectAsyncAPIVersion(versionCheck.AsyncAPI)

	switch version {
	case 2:
		spec, err = ParseAsyncAPI(data)
		return spec, 2, err
	case 3:
		spec, err = ParseAsyncAPIV3(data)
		return spec, 3, err
	default:
		return nil, 0, fmt.Errorf("unsupported AsyncAPI version: %s", versionCheck.AsyncAPI)
	}
}
