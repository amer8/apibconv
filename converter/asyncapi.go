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
	AsyncAPI   string                       `json:"asyncapi"`             // AsyncAPI specification version (e.g., "2.6.0", "3.0.0")
	Info       Info                         `json:"info"`                 // API metadata including title, description, and version
	Servers    map[string]AsyncAPIServer    `json:"servers,omitempty"`    // Server definitions with protocol information
	Channels   map[string]Channel           `json:"channels,omitempty"`   // Channel definitions
	Operations map[string]AsyncAPIOperation `json:"operations,omitempty"` // Operations (v3)
	Components *AsyncAPIComponents          `json:"components,omitempty"` // Reusable components (messages, schemas)
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
// A channel is a communication endpoint (topic, queue, routing key, etc.).
// In v2, key is address. In v3, key is ID and Address field is used.
type Channel struct {
	Description string             `json:"description,omitempty"` // Description of the channel
	// v2 fields
	Subscribe   *AsyncAPIOperation `json:"subscribe,omitempty"`   // Client subscribes to receive messages
	Publish     *AsyncAPIOperation `json:"publish,omitempty"`     // Client publishes messages to this channel
	// v3 fields
	Address     string              `json:"address,omitempty"`     // Channel address/topic/queue name
	Messages    map[string]*Message `json:"messages,omitempty"`    // Messages that can be sent on this channel
}

// AsyncAPIOperation represents an operation in AsyncAPI specification.
//
// Operations describe the action that a consumer or producer performs.
type AsyncAPIOperation struct {
	// Common / v2
	OperationID string   `json:"operationId,omitempty"` // Unique operation identifier
	Summary     string   `json:"summary,omitempty"`     // Short summary of the operation
	Description string   `json:"description,omitempty"` // Detailed description
	// v2 specific
	Message     *Message `json:"message,omitempty"`     // Message definition for this operation
	// v3 specific
	Action      string            `json:"action,omitempty"`      // Action: "send" or "receive"
	Channel     *ChannelReference `json:"channel,omitempty"`     // Reference to a channel
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

// parseAsync parses AsyncAPI JSON or YAML data into an AsyncAPI struct.
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
func parseAsync(data []byte) (*AsyncAPI, error) {
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
//   - error: Error if conversion fails
//
// Example:
//
//	asyncSpec := &AsyncAPI{...}
//	blueprint, err := asyncSpec.ToBlueprint()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(blueprint)
func (spec *AsyncAPI) ToBlueprint() (string, error) {
	buf := getBuffer()
	defer putBuffer(buf)
	if strings.HasPrefix(spec.AsyncAPI, "3.") {
		writeAsyncAPIV3Blueprint(buf, spec)
	} else {
		writeAsyncAPIBlueprint(buf, spec)
	}
	return buf.String(), nil
}

// WriteBlueprint writes the AsyncAPI specification in API Blueprint format to the writer.
func (spec *AsyncAPI) WriteBlueprint(w io.Writer) error {
	buf := getBuffer()
	defer putBuffer(buf)
	writeAsyncAPIBlueprint(buf, spec)
	_, err := w.Write(buf.Bytes())
	return err
}

// ToOpenAPI converts the AsyncAPI specification to OpenAPI format.
func (spec *AsyncAPI) ToOpenAPI() (*OpenAPI, error) {
	bp, err := spec.ToBlueprint()
	if err != nil {
		return nil, err
	}
	blueprintSpec, err := ParseBlueprint([]byte(bp))
	if err != nil {
		return nil, err
	}
	return blueprintSpec.ToOpenAPI()
}

// ToAsyncAPI returns the AsyncAPI specification itself.
func (spec *AsyncAPI) ToAsyncAPI(protocol Protocol) (*AsyncAPI, error) {
	return spec, nil
}

// ToAsyncAPIV3 converts the AsyncAPI specification to AsyncAPI 3.0 format.
func (spec *AsyncAPI) ToAsyncAPIV3(protocol Protocol) (*AsyncAPI, error) {
	// If already v3, return self (maybe clone?)
	if strings.HasPrefix(spec.AsyncAPI, "3.") {
		return spec, nil
	}
	// Convert via OpenAPI
	openapi, err := spec.ToOpenAPI()
	if err != nil {
		return nil, err
	}
	return openapi.ToAsyncAPIV3(protocol)
}

// GetTitle returns the title of the AsyncAPI specification.
func (spec *AsyncAPI) GetTitle() string {
	if spec != nil {
		return spec.Info.Title
	}
	return ""
}

// GetVersion returns the version of the AsyncAPI specification.
func (spec *AsyncAPI) GetVersion() string {
	if spec != nil {
		return spec.AsyncAPI
	}
	return ""
}

// AsOpenAPI returns nil and false for an AsyncAPI spec.
func (spec *AsyncAPI) AsOpenAPI() (*OpenAPI, bool) {
	return nil, false
}

// AsAsyncAPI returns the *AsyncAPI spec itself, and true.
func (spec *AsyncAPI) AsAsyncAPI() (*AsyncAPI, bool) {
	return spec, true
}

// AsAsyncAPIV3 returns the *AsyncAPI spec itself if it's v3, or converts.
func (spec *AsyncAPI) AsAsyncAPIV3() (*AsyncAPI, bool) {
	if strings.HasPrefix(spec.AsyncAPI, "3.") {
		return spec, true
	}
	specV3, err := spec.ToAsyncAPIV3(ProtocolAuto)
	if err != nil {
		return nil, false
	}
	return specV3, true
}

// AsAPIBlueprint returns nil and false for an AsyncAPI spec.
func (spec *AsyncAPI) AsAPIBlueprint() (*APIBlueprint, bool) {
	return nil, false
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

// ToAsyncAPI converts an API Blueprint (OpenAPI struct) to AsyncAPI format.
//
// This function converts OpenAPI/API Blueprint paths to AsyncAPI channels:
//   - GET operations -> Subscribe operations (receiving messages)
//   - POST operations -> Publish operations (sending messages)
//   - Response bodies -> Message payloads for GET
//   - Request bodies -> Message payloads for POST
//
// Parameters:
//   - protocol: Protocol to use for AsyncAPI servers (e.g., ProtocolHTTP, ProtocolWS)
//
// Returns:
//   - *AsyncAPI: Converted AsyncAPI specification
//   - error: Error if conversion fails
//
// Example:
//
//	openAPISpec := converter.ParseAPIBlueprint(data)
//	asyncSpec, err := openAPISpec.ToAsyncAPI(converter.ProtocolWS)
func (spec *OpenAPI) ToAsyncAPI(protocol Protocol) (*AsyncAPI, error) {
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
				Protocol:    string(protocol),
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
			channel.Subscribe = pathItem.Get.ToAsyncAPI(false)
		}

		// Convert POST to Publish (client sends messages)
		if pathItem.Post != nil {
			channel.Publish = pathItem.Post.ToAsyncAPI(true)
		}

		// If we have any operations, add the channel
		if channel.Subscribe != nil || channel.Publish != nil {
			asyncSpec.Channels[channelName] = channel
		}
	}

	return asyncSpec, nil
}

// ToAsyncAPI converts an OpenAPI operation to AsyncAPI operation
func (op *Operation) ToAsyncAPI(isPublish bool) *AsyncAPIOperation {
	asyncOp := &AsyncAPIOperation{
		Summary:     op.Summary,
		Description: op.Description,
	}

	asyncOp.Message = op.ExtractMessage(isPublish)
	return asyncOp
}

// ============================================================================
// AsyncAPI 3.0 Support
// ============================================================================

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

// detectAsyncAPIVersion detects the AsyncAPI version from the version string.
//
// Returns:
//   - 2: for AsyncAPI 2.x versions
//   - 3: for AsyncAPI 3.x versions
//   - 0: for unknown or invalid versions
//
// Example:
//
//	version := detectAsyncAPIVersion("2.6.0") // returns 2
//	version := detectAsyncAPIVersion("3.0.0") // returns 3
func detectAsyncAPIVersion(asyncapiVersion string) int {
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

// parseAsyncV3 parses AsyncAPI 3.0 JSON or YAML data into an AsyncAPI struct.
func parseAsyncV3(data []byte) (*AsyncAPI, error) {
	var spec AsyncAPI

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
//	asyncSpec, err := openAPISpec.ToAsyncAPIV3(converter.ProtocolWS)
func (spec *OpenAPI) ToAsyncAPIV3(protocol Protocol) (*AsyncAPI, error) {
	asyncSpec := &AsyncAPI{
		AsyncAPI:   AsyncAPIVersion30,
		Info:       spec.Info,
		Channels:   make(map[string]Channel),
		Operations: make(map[string]AsyncAPIOperation),
	}

	// Convert servers
	if len(spec.Servers) > 0 {
		asyncSpec.Servers = make(map[string]AsyncAPIServer)
		for i, server := range spec.Servers {
			serverName := fmt.Sprintf("server%d", i)
			asyncSpec.Servers[serverName] = AsyncAPIServer{
				URL:         server.URL,
				Protocol:    string(protocol),
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
		channel := Channel{
			Address:  path[1:], // Remove leading slash
			Messages: make(map[string]*Message),
		}

		// Convert GET to Receive operation
		if pathItem.Get != nil {
			operationCounter++
			opID := fmt.Sprintf("receive%s%d", sanitizeOperationID(path), operationCounter)

			// Add message to channel
			msgID := "Message"
			message := pathItem.Get.ExtractMessage(false)
			if message != nil {
				channel.Messages[msgID] = message
			}

			// Create receive operation
			asyncSpec.Operations[opID] = AsyncAPIOperation{
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
			message := pathItem.Post.ExtractMessage(true)
			if message != nil {
				if len(channel.Messages) == 0 {
					channel.Messages[msgID] = message
				} else {
					// If we already have a message from GET, use a different ID
					channel.Messages["SendMessage"] = message
				}
			}

			// Create send operation
			asyncSpec.Operations[opID] = AsyncAPIOperation{
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

	return asyncSpec, nil
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

// ExtractMessage extracts a message from an OpenAPI operation
func (op *Operation) ExtractMessage(isPublish bool) *Message {
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

// parseAsyncAPIAny parses AsyncAPI JSON or YAML (any version) and returns the appropriate struct.
//
// Returns:
//   - *AsyncAPI (v2 or v3)
//   - version (int): 2 or 3
//   - error if version is unsupported or parsing fails
func parseAsyncAPIAny(data []byte) (*AsyncAPI, int, error) {
	// First, detect version by parsing just the version field
	var versionCheck struct {
		AsyncAPI string `json:"asyncapi"`
	}

	// Try JSON first
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		// If JSON fails, try YAML
		if yamlErr := UnmarshalYAML(data, &versionCheck); yamlErr != nil {
			// If both fail, return an error
			return nil, 0, fmt.Errorf("failed to detect AsyncAPI version from JSON or YAML: %w", err)
		}
	}

	version := detectAsyncAPIVersion(versionCheck.AsyncAPI)
	var spec *AsyncAPI
	var err error

	switch version {
	case 2:
		spec, err = parseAsync(data)
	case 3:
		spec, err = parseAsyncV3(data)
	default:
		return nil, 0, fmt.Errorf("unsupported AsyncAPI version: %s", versionCheck.AsyncAPI)
	}

	if err != nil {
		return nil, version, err
	}
	return spec, version, nil
}

// writeAsyncAPIV3Blueprint writes the API Blueprint format to the buffer
func writeAsyncAPIV3Blueprint(buf *bytes.Buffer, spec *AsyncAPI) {
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
		op   AsyncAPIOperation
	})

	for opID, op := range spec.Operations {
		if op.Channel != nil && op.Channel.Ref != "" {
			// Extract channel ID from reference (e.g., "#/channels/userSignup" -> "userSignup")
			channelID := extractChannelID(op.Channel.Ref)
			channelOps[channelID] = append(channelOps[channelID], struct {
				opID string
				op   AsyncAPIOperation
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
func writeAsyncAPIV3Channel(buf *bytes.Buffer, channelID string, channel Channel, ops []struct {
	opID string
	op   AsyncAPIOperation
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
func writeAsyncAPIV3Operation(buf *bytes.Buffer, method string, op AsyncAPIOperation, msg *Message, defaultSummary string) {
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

