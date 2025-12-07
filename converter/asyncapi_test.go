package converter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAsyncAPI(t *testing.T) {
	asyncapiJSON := []byte(`{
		"asyncapi": "2.6.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0",
			"description": "A test async API"
		},
		"servers": {
			"production": {
				"url": "mqtt://test.mosquitto.org:1883",
				"protocol": "mqtt",
				"description": "Test broker"
			}
		},
		"channels": {
			"user/signedup": {
				"description": "User signup events",
				"subscribe": {
					"operationId": "onUserSignedUp",
					"summary": "User signed up",
					"message": {
						"name": "UserSignedUp",
						"contentType": "application/json",
						"payload": {
							"type": "object",
							"properties": {
								"userId": {"type": "string"},
								"email": {"type": "string"}
							}
						}
					}
				}
			}
		}
	}`)

	spec, err := ParseAsyncAPI(asyncapiJSON)
	if err != nil {
		t.Fatalf("ParseAsyncAPI failed: %v", err)
	}

	if spec.AsyncAPI != "2.6.0" {
		t.Errorf("Expected AsyncAPI version '2.6.0', got '%s'", spec.AsyncAPI)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}

	if len(spec.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(spec.Servers))
	}

	server, ok := spec.Servers["production"]
	if !ok {
		t.Fatal("Expected 'production' server")
	}

	if server.Protocol != "mqtt" {
		t.Errorf("Expected protocol 'mqtt', got '%s'", server.Protocol)
	}

	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}

	channel, ok := spec.Channels["user/signedup"]
	if !ok {
		t.Fatal("Expected 'user/signedup' channel")
	}

	if channel.Subscribe == nil {
		t.Fatal("Expected subscribe operation")
	}

	if channel.Subscribe.OperationID != "onUserSignedUp" {
		t.Errorf("Expected operationId 'onUserSignedUp', got '%s'", channel.Subscribe.OperationID)
	}
}

func TestAsyncAPIToAPIBlueprint(t *testing.T) {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:       "Chat API",
			Version:     "1.0.0",
			Description: "A simple chat API",
		},
		Servers: map[string]AsyncAPIServer{
			"testing": {
				URL:         "ws://localhost:8080",
				Protocol:    "ws",
				Description: "WebSocket server",
			},
		},
		Channels: map[string]Channel{
			"chat": {
				Description: "Chat channel",
				Subscribe: &AsyncAPIOperation{
					Summary:     "Receive chat messages",
					Description: "Subscribe to receive chat messages",
					Message: &Message{
						Name:        "ChatMessage",
						ContentType: "application/json",
						Payload: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"message": {Type: "string"},
								"user":    {Type: "string"},
							},
							Example: map[string]interface{}{
								"message": "Hello, world!",
								"user":    "john",
							},
						},
					},
				},
				Publish: &AsyncAPIOperation{
					Summary: "Send chat messages",
					Message: &Message{
						Name:        "ChatMessage",
						ContentType: "application/json",
						Payload: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"message": {Type: "string"},
								"user":    {Type: "string"},
							},
							Example: map[string]interface{}{
								"message": "Hello!",
								"user":    "jane",
							},
						},
					},
				},
			},
		},
	}

	blueprint := AsyncAPIToAPIBlueprint(spec)

	// Check for required API Blueprint elements
	if !strings.Contains(blueprint, "FORMAT: 1A") {
		t.Error("Expected 'FORMAT: 1A' in output")
	}

	if !strings.Contains(blueprint, "# Chat API") {
		t.Error("Expected '# Chat API' title in output")
	}

	if !strings.Contains(blueprint, "A simple chat API") {
		t.Error("Expected description in output")
	}

	if !strings.Contains(blueprint, "HOST: ws://localhost:8080") {
		t.Error("Expected HOST in output")
	}

	if !strings.Contains(blueprint, "## /chat") {
		t.Error("Expected '/chat' path in output")
	}

	if !strings.Contains(blueprint, "### Receive chat messages [GET]") {
		t.Error("Expected GET operation for subscribe")
	}

	if !strings.Contains(blueprint, "### Send chat messages [POST]") {
		t.Error("Expected POST operation for publish")
	}

	if !strings.Contains(blueprint, "+ Response 200") {
		t.Error("Expected response for subscribe operation")
	}

	if !strings.Contains(blueprint, "+ Request") {
		t.Error("Expected request for publish operation")
	}
}

// createTestEventOpenAPISpec creates a test OpenAPI spec for event-based APIs
func createTestEventOpenAPISpec(title, description string) *OpenAPI {
	return &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       title,
			Version:     "1.0.0",
			Description: description,
		},
		Servers: []Server{
			{
				URL:         "https://api.example.com",
				Description: "Production server",
			},
		},
		Paths: map[string]PathItem{
			"/events": {
				Get: &Operation{
					Summary:     "Subscribe to events",
					Description: "Receive events stream",
					Responses: map[string]Response{
						"200": {
							Description: "Event received",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"event": {Type: "string"},
											"data":  {Type: "string"},
										},
									},
									Example: map[string]interface{}{
										"event": "user.created",
										"data":  "user123",
									},
								},
							},
						},
					},
				},
				Post: &Operation{
					Summary:     "Publish event",
					Description: "Send an event",
					RequestBody: &RequestBody{
						Required: true,
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"event": {Type: "string"},
										"data":  {Type: "string"},
									},
								},
								Example: map[string]interface{}{
									"event": "user.updated",
									"data":  "user456",
								},
							},
						},
					},
					Responses: map[string]Response{
						"201": {Description: "Event published"},
					},
				},
			},
		},
	}
}

func TestAPIBlueprintToAsyncAPI(t *testing.T) {
	openAPISpec := createTestEventOpenAPISpec("Events API", "Event streaming API")

	asyncSpec := APIBlueprintToAsyncAPI(openAPISpec, "kafka")

	if asyncSpec.AsyncAPI != "2.6.0" {
		t.Errorf("Expected AsyncAPI version '2.6.0', got '%s'", asyncSpec.AsyncAPI)
	}

	if asyncSpec.Info.Title != "Events API" {
		t.Errorf("Expected title 'Events API', got '%s'", asyncSpec.Info.Title)
	}

	if len(asyncSpec.Servers) == 0 {
		t.Error("Expected at least one server")
	}

	// Check that server has kafka protocol
	for _, server := range asyncSpec.Servers {
		if server.Protocol != "kafka" {
			t.Errorf("Expected protocol 'kafka', got '%s'", server.Protocol)
		}
	}

	if len(asyncSpec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(asyncSpec.Channels))
	}

	channel, ok := asyncSpec.Channels["events"]
	if !ok {
		t.Fatal("Expected 'events' channel")
	}

	if channel.Subscribe == nil {
		t.Error("Expected subscribe operation from GET")
	}

	if channel.Publish == nil {
		t.Error("Expected publish operation from POST")
	}

	if channel.Subscribe.Summary != "Subscribe to events" {
		t.Errorf("Expected subscribe summary 'Subscribe to events', got '%s'", channel.Subscribe.Summary)
	}

	if channel.Publish.Summary != "Publish event" {
		t.Errorf("Expected publish summary 'Publish event', got '%s'", channel.Publish.Summary)
	}
}

func TestConvertAsyncAPIToAPIBlueprint(t *testing.T) {
	asyncapiJSON := `{
		"asyncapi": "2.6.0",
		"info": {
			"title": "Notification API",
			"version": "1.0.0"
		},
		"channels": {
			"notifications": {
				"subscribe": {
					"summary": "Receive notifications",
					"message": {
						"payload": {
							"type": "object"
						}
					}
				}
			}
		}
	}`

	input := bytes.NewBufferString(asyncapiJSON)
	output := &bytes.Buffer{}

	err := ConvertAsyncAPIToAPIBlueprint(input, output)
	if err != nil {
		t.Fatalf("ConvertAsyncAPIToAPIBlueprint failed: %v", err)
	}

	blueprint := output.String()

	if !strings.Contains(blueprint, "FORMAT: 1A") {
		t.Error("Expected 'FORMAT: 1A' in output")
	}

	if !strings.Contains(blueprint, "# Notification API") {
		t.Error("Expected '# Notification API' title in output")
	}

	if !strings.Contains(blueprint, "## /notifications") {
		t.Error("Expected '/notifications' path in output")
	}
}

func TestConvertAPIBlueprintToAsyncAPI(t *testing.T) {
	apiblueprintContent := `FORMAT: 1A

# Webhook API

Webhook event delivery API

## /webhooks [/webhooks]

### Receive webhook [GET]

+ Response 200 (application/json)

    + Body

            {
                "event": "order.created",
                "payload": {}
            }

### Send webhook [POST]

+ Request (application/json)

    + Body

            {
                "event": "order.updated",
                "payload": {}
            }

+ Response 201
`

	input := bytes.NewBufferString(apiblueprintContent)
	output := &bytes.Buffer{}

	err := ConvertAPIBlueprintToAsyncAPI(input, output, "http")
	if err != nil {
		t.Fatalf("ConvertAPIBlueprintToAsyncAPI failed: %v", err)
	}

	// Parse the output JSON
	var asyncSpec AsyncAPI
	err = json.Unmarshal(output.Bytes(), &asyncSpec)
	if err != nil {
		t.Fatalf("Failed to parse AsyncAPI output: %v", err)
	}

	if asyncSpec.AsyncAPI != "2.6.0" {
		t.Errorf("Expected AsyncAPI version '2.6.0', got '%s'", asyncSpec.AsyncAPI)
	}

	if asyncSpec.Info.Title != "Webhook API" {
		t.Errorf("Expected title 'Webhook API', got '%s'", asyncSpec.Info.Title)
	}

	// Check servers have http protocol
	for _, server := range asyncSpec.Servers {
		if server.Protocol != "http" {
			t.Errorf("Expected protocol 'http', got '%s'", server.Protocol)
		}
	}

	// Check channel exists
	channel, ok := asyncSpec.Channels["webhooks"]
	if !ok {
		t.Fatal("Expected 'webhooks' channel")
	}

	if channel.Subscribe == nil {
		t.Error("Expected subscribe operation")
	}

	if channel.Publish == nil {
		t.Error("Expected publish operation")
	}
}

func TestAsyncAPIWithMultipleChannels(t *testing.T) {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:   "Multi-Channel API",
			Version: "1.0.0",
		},
		Channels: map[string]Channel{
			"users/created": {
				Subscribe: &AsyncAPIOperation{
					Summary: "User created events",
					Message: &Message{Payload: &Schema{Type: "object"}},
				},
			},
			"orders/placed": {
				Subscribe: &AsyncAPIOperation{
					Summary: "Order placed events",
					Message: &Message{Payload: &Schema{Type: "object"}},
				},
			},
			"payments/completed": {
				Subscribe: &AsyncAPIOperation{
					Summary: "Payment completed events",
					Message: &Message{Payload: &Schema{Type: "object"}},
				},
			},
		},
	}

	blueprint := AsyncAPIToAPIBlueprint(spec)

	// All channels should be present
	if !strings.Contains(blueprint, "## /users/created") {
		t.Error("Expected '/users/created' channel")
	}

	if !strings.Contains(blueprint, "## /orders/placed") {
		t.Error("Expected '/orders/placed' channel")
	}

	if !strings.Contains(blueprint, "## /payments/completed") {
		t.Error("Expected '/payments/completed' channel")
	}

	// Check operations
	if !strings.Contains(blueprint, "User created events") {
		t.Error("Expected 'User created events' operation")
	}

	if !strings.Contains(blueprint, "Order placed events") {
		t.Error("Expected 'Order placed events' operation")
	}

	if !strings.Contains(blueprint, "Payment completed events") {
		t.Error("Expected 'Payment completed events' operation")
	}
}

// ============================================================================
// AsyncAPI 3.0 Tests
// ============================================================================

func TestDetectAsyncAPIVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected int
	}{
		{"AsyncAPI 2.0.0", "2.0.0", 2},
		{"AsyncAPI 2.6.0", "2.6.0", 2},
		{"AsyncAPI 3.0.0", "3.0.0", 3},
		{"AsyncAPI 3.1.0", "3.1.0", 3},
		{"Empty version", "", 0},
		{"Invalid version", "1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAsyncAPIVersion(tt.version)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d for version %s", tt.expected, result, tt.version)
			}
		})
	}
}

func TestParseAsyncAPIV3(t *testing.T) {
	asyncapiJSON := []byte(`{
		"asyncapi": "3.0.0",
		"info": {
			"title": "Test API v3",
			"version": "1.0.0",
			"description": "A test async API v3"
		},
		"servers": {
			"production": {
				"url": "ws://localhost:8080",
				"protocol": "ws",
				"description": "WebSocket server"
			}
		},
		"channels": {
			"userSignup": {
				"address": "user/signedup",
				"description": "User signup events",
				"messages": {
					"UserSignedUp": {
						"contentType": "application/json",
						"payload": {
							"type": "object",
							"properties": {
								"userId": {"type": "string"},
								"email": {"type": "string"}
							}
						}
					}
				}
			}
		},
		"operations": {
			"onUserSignup": {
				"action": "receive",
				"summary": "Receive user signup events",
				"channel": {
					"$ref": "#/channels/userSignup"
				}
			}
		}
	}`)

	spec, err := ParseAsyncAPIV3(asyncapiJSON)
	if err != nil {
		t.Fatalf("ParseAsyncAPIV3 failed: %v", err)
	}

	if spec.AsyncAPI != "3.0.0" {
		t.Errorf("Expected AsyncAPI version '3.0.0', got '%s'", spec.AsyncAPI)
	}

	if spec.Info.Title != "Test API v3" {
		t.Errorf("Expected title 'Test API v3', got '%s'", spec.Info.Title)
	}

	if len(spec.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(spec.Servers))
	}

	server, ok := spec.Servers["production"]
	if !ok {
		t.Fatal("Expected 'production' server")
	}

	if server.Protocol != "ws" {
		t.Errorf("Expected protocol 'ws', got '%s'", server.Protocol)
	}

	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}

	channel, ok := spec.Channels["userSignup"]
	if !ok {
		t.Fatal("Expected 'userSignup' channel")
	}

	if channel.Address != "user/signedup" {
		t.Errorf("Expected address 'user/signedup', got '%s'", channel.Address)
	}

	if len(spec.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(spec.Operations))
	}

	op, ok := spec.Operations["onUserSignup"]
	if !ok {
		t.Fatal("Expected 'onUserSignup' operation")
	}

	if op.Action != "receive" {
		t.Errorf("Expected action 'receive', got '%s'", op.Action)
	}

	if op.Channel == nil || op.Channel.Ref != "#/channels/userSignup" {
		t.Error("Expected channel reference '#/channels/userSignup'")
	}
}

func TestAsyncAPIV3ToAPIBlueprint(t *testing.T) {
	spec := &AsyncAPIV3{
		AsyncAPI: "3.0.0",
		Info: Info{
			Title:       "Chat API v3",
			Version:     "1.0.0",
			Description: "A simple chat API using AsyncAPI 3.0",
		},
		Servers: map[string]AsyncAPIServer{
			"testing": {
				URL:         "ws://localhost:8080",
				Protocol:    "ws",
				Description: "WebSocket server",
			},
		},
		Channels: map[string]ChannelV3{
			"chatChannel": {
				Address:     "chat",
				Description: "Chat channel",
				Messages: map[string]*Message{
					"ChatMessage": {
						Name:        "ChatMessage",
						ContentType: "application/json",
						Payload: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"message": {Type: "string"},
								"user":    {Type: "string"},
							},
							Example: map[string]interface{}{
								"message": "Hello, world!",
								"user":    "john",
							},
						},
					},
				},
			},
		},
		Operations: map[string]OperationV3{
			"receiveMessages": {
				Action:      "receive",
				Summary:     "Receive chat messages",
				Description: "Subscribe to receive chat messages",
				Channel: &ChannelReference{
					Ref: "#/channels/chatChannel",
				},
			},
			"sendMessages": {
				Action:  "send",
				Summary: "Send chat messages",
				Channel: &ChannelReference{
					Ref: "#/channels/chatChannel",
				},
			},
		},
	}

	blueprint := AsyncAPIV3ToAPIBlueprint(spec)

	// Check for required API Blueprint elements
	if !strings.Contains(blueprint, "FORMAT: 1A") {
		t.Error("Expected 'FORMAT: 1A' in output")
	}

	if !strings.Contains(blueprint, "# Chat API v3") {
		t.Error("Expected '# Chat API v3' title in output")
	}

	if !strings.Contains(blueprint, "A simple chat API using AsyncAPI 3.0") {
		t.Error("Expected description in output")
	}

	if !strings.Contains(blueprint, "HOST: ws://localhost:8080") {
		t.Error("Expected HOST in output")
	}

	if !strings.Contains(blueprint, "## /chat") {
		t.Error("Expected '/chat' path in output")
	}

	if !strings.Contains(blueprint, "### Receive chat messages [GET]") {
		t.Error("Expected GET operation for receive")
	}

	if !strings.Contains(blueprint, "### Send chat messages [POST]") {
		t.Error("Expected POST operation for send")
	}

	if !strings.Contains(blueprint, "+ Response 200") {
		t.Error("Expected response for receive operation")
	}

	if !strings.Contains(blueprint, "+ Request") {
		t.Error("Expected request for send operation")
	}
}

func TestAPIBlueprintToAsyncAPIV3(t *testing.T) {
	openAPISpec := createTestEventOpenAPISpec("Events API v3", "Event streaming API for AsyncAPI 3.0")

	asyncSpec := APIBlueprintToAsyncAPIV3(openAPISpec, "kafka")

	if asyncSpec.AsyncAPI != "3.0.0" {
		t.Errorf("Expected AsyncAPI version '3.0.0', got '%s'", asyncSpec.AsyncAPI)
	}

	if asyncSpec.Info.Title != "Events API v3" {
		t.Errorf("Expected title 'Events API v3', got '%s'", asyncSpec.Info.Title)
	}

	if len(asyncSpec.Servers) == 0 {
		t.Error("Expected at least one server")
	}

	// Check that server has kafka protocol
	for _, server := range asyncSpec.Servers {
		if server.Protocol != "kafka" {
			t.Errorf("Expected protocol 'kafka', got '%s'", server.Protocol)
		}
	}

	if len(asyncSpec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(asyncSpec.Channels))
	}

	// Check for operations
	receiveOpFound := false
	sendOpFound := false
	for _, op := range asyncSpec.Operations {
		if op.Action == "receive" {
			receiveOpFound = true
			if op.Summary != "Subscribe to events" {
				t.Errorf("Expected receive operation summary 'Subscribe to events', got '%s'", op.Summary)
			}
		}
		if op.Action == "send" {
			sendOpFound = true
			if op.Summary != "Publish event" {
				t.Errorf("Expected send operation summary 'Publish event', got '%s'", op.Summary)
			}
		}
	}

	if !receiveOpFound {
		t.Error("Expected receive operation (from GET)")
	}

	if !sendOpFound {
		t.Error("Expected send operation (from POST)")
	}
}

func TestConvertAsyncAPIV3ToAPIBlueprint(t *testing.T) {
	asyncapiJSON := `{
		"asyncapi": "3.0.0",
		"info": {
			"title": "Notification API v3",
			"version": "1.0.0"
		},
		"channels": {
			"notifications": {
				"address": "notifications",
				"messages": {
					"Notification": {
						"payload": {
							"type": "object"
						}
					}
				}
			}
		},
		"operations": {
			"receiveNotifications": {
				"action": "receive",
				"summary": "Receive notifications",
				"channel": {
					"$ref": "#/channels/notifications"
				}
			}
		}
	}`

	input := bytes.NewBufferString(asyncapiJSON)
	output := &bytes.Buffer{}

	err := ConvertAsyncAPIV3ToAPIBlueprint(input, output)
	if err != nil {
		t.Fatalf("ConvertAsyncAPIV3ToAPIBlueprint failed: %v", err)
	}

	blueprint := output.String()

	if !strings.Contains(blueprint, "FORMAT: 1A") {
		t.Error("Expected 'FORMAT: 1A' in output")
	}

	if !strings.Contains(blueprint, "# Notification API v3") {
		t.Error("Expected '# Notification API v3' title in output")
	}

	if !strings.Contains(blueprint, "## /notifications") {
		t.Error("Expected '/notifications' path in output")
	}
}

func TestConvertAPIBlueprintToAsyncAPIV3(t *testing.T) {
	apiblueprintContent := `FORMAT: 1A

# Webhook API v3

Webhook event delivery API for AsyncAPI 3.0

## /webhooks [/webhooks]

### Receive webhook [GET]

+ Response 200 (application/json)

    + Body

            {
                "event": "order.created",
                "payload": {}
            }

### Send webhook [POST]

+ Request (application/json)

    + Body

            {
                "event": "order.updated",
                "payload": {}
            }

+ Response 201
`

	input := bytes.NewBufferString(apiblueprintContent)
	output := &bytes.Buffer{}

	err := ConvertAPIBlueprintToAsyncAPIV3(input, output, "http")
	if err != nil {
		t.Fatalf("ConvertAPIBlueprintToAsyncAPIV3 failed: %v", err)
	}

	// Parse the output JSON
	var asyncSpec AsyncAPIV3
	err = json.Unmarshal(output.Bytes(), &asyncSpec)
	if err != nil {
		t.Fatalf("Failed to parse AsyncAPI v3 output: %v", err)
	}

	if asyncSpec.AsyncAPI != "3.0.0" {
		t.Errorf("Expected AsyncAPI version '3.0.0', got '%s'", asyncSpec.AsyncAPI)
	}

	if asyncSpec.Info.Title != "Webhook API v3" {
		t.Errorf("Expected title 'Webhook API v3', got '%s'", asyncSpec.Info.Title)
	}

	// Check servers have http protocol
	for _, server := range asyncSpec.Servers {
		if server.Protocol != "http" {
			t.Errorf("Expected protocol 'http', got '%s'", server.Protocol)
		}
	}

	// Check channel exists
	if len(asyncSpec.Channels) == 0 {
		t.Fatal("Expected at least one channel")
	}

	// Check operations
	receiveOpFound := false
	sendOpFound := false
	for _, op := range asyncSpec.Operations {
		if op.Action == "receive" {
			receiveOpFound = true
		}
		if op.Action == "send" {
			sendOpFound = true
		}
	}

	if !receiveOpFound {
		t.Error("Expected receive operation")
	}

	if !sendOpFound {
		t.Error("Expected send operation")
	}
}

func TestParseAsyncAPIAny(t *testing.T) {
	// Test v2
	v2JSON := []byte(`{
		"asyncapi": "2.6.0",
		"info": {"title": "Test API", "version": "1.0.0"},
		"channels": {}
	}`)

	spec, version, err := ParseAsyncAPIAny(v2JSON)
	if err != nil {
		t.Fatalf("ParseAsyncAPIAny failed for v2: %v", err)
	}

	if version != 2 {
		t.Errorf("Expected version 2, got %d", version)
	}

	v2Spec, ok := spec.(*AsyncAPI)
	if !ok {
		t.Error("Expected *AsyncAPI type for v2")
	}

	if v2Spec.AsyncAPI != "2.6.0" {
		t.Errorf("Expected version '2.6.0', got '%s'", v2Spec.AsyncAPI)
	}

	// Test v3
	v3JSON := []byte(`{
		"asyncapi": "3.0.0",
		"info": {"title": "Test API", "version": "1.0.0"},
		"channels": {},
		"operations": {}
	}`)

	spec, version, err = ParseAsyncAPIAny(v3JSON)
	if err != nil {
		t.Fatalf("ParseAsyncAPIAny failed for v3: %v", err)
	}

	if version != 3 {
		t.Errorf("Expected version 3, got %d", version)
	}

	v3Spec, ok := spec.(*AsyncAPIV3)
	if !ok {
		t.Error("Expected *AsyncAPIV3 type for v3")
	}

	if v3Spec.AsyncAPI != "3.0.0" {
		t.Errorf("Expected version '3.0.0', got '%s'", v3Spec.AsyncAPI)
	}

	// Test invalid version
	invalidJSON := []byte(`{
		"asyncapi": "1.0.0",
		"info": {"title": "Test API", "version": "1.0.0"}
	}`)

	_, _, err = ParseAsyncAPIAny(invalidJSON)
	if err == nil {
		t.Error("Expected error for unsupported version")
	}
}

func TestAsyncAPIV3WithMultipleChannelsAndOperations(t *testing.T) {
	spec := &AsyncAPIV3{
		AsyncAPI: "3.0.0",
		Info: Info{
			Title:   "Multi-Channel API v3",
			Version: "1.0.0",
		},
		Channels: map[string]ChannelV3{
			"userCreated": {
				Address: "users/created",
				Messages: map[string]*Message{
					"UserCreated": {Payload: &Schema{Type: "object"}},
				},
			},
			"orderPlaced": {
				Address: "orders/placed",
				Messages: map[string]*Message{
					"OrderPlaced": {Payload: &Schema{Type: "object"}},
				},
			},
			"paymentCompleted": {
				Address: "payments/completed",
				Messages: map[string]*Message{
					"PaymentCompleted": {Payload: &Schema{Type: "object"}},
				},
			},
		},
		Operations: map[string]OperationV3{
			"onUserCreated": {
				Action:  "receive",
				Summary: "User created events",
				Channel: &ChannelReference{Ref: "#/channels/userCreated"},
			},
			"onOrderPlaced": {
				Action:  "receive",
				Summary: "Order placed events",
				Channel: &ChannelReference{Ref: "#/channels/orderPlaced"},
			},
			"onPaymentCompleted": {
				Action:  "receive",
				Summary: "Payment completed events",
				Channel: &ChannelReference{Ref: "#/channels/paymentCompleted"},
			},
		},
	}

	blueprint := AsyncAPIV3ToAPIBlueprint(spec)

	// All channels should be present
	if !strings.Contains(blueprint, "## /users/created") {
		t.Error("Expected '/users/created' channel")
	}

	if !strings.Contains(blueprint, "## /orders/placed") {
		t.Error("Expected '/orders/placed' channel")
	}

	if !strings.Contains(blueprint, "## /payments/completed") {
		t.Error("Expected '/payments/completed' channel")
	}

	// Check operations
	if !strings.Contains(blueprint, "User created events") {
		t.Error("Expected 'User created events' operation")
	}

	if !strings.Contains(blueprint, "Order placed events") {
		t.Error("Expected 'Order placed events' operation")
	}

	if !strings.Contains(blueprint, "Payment completed events") {
		t.Error("Expected 'Payment completed events' operation")
	}
}

func TestParseAsyncAPIReader(t *testing.T) {
	jsonContent := `{
		"asyncapi": "2.6.0",
		"info": {"title": "Reader Test", "version": "1.0.0"},
		"channels": {}
	}`
	reader := strings.NewReader(jsonContent)
	spec, err := ParseAsyncAPIReader(reader)
	if err != nil {
		t.Fatalf("ParseAsyncAPIReader failed: %v", err)
	}
	if spec.Info.Title != "Reader Test" {
		t.Errorf("Expected title 'Reader Test', got '%s'", spec.Info.Title)
	}
}

func TestParseAsyncAPIV3Reader(t *testing.T) {
	jsonContent := `{
		"asyncapi": "3.0.0",
		"info": {"title": "Reader Test V3", "version": "1.0.0"},
		"channels": {},
		"operations": {}
	}`
	reader := strings.NewReader(jsonContent)
	spec, err := ParseAsyncAPIV3Reader(reader)
	if err != nil {
		t.Fatalf("ParseAsyncAPIV3Reader failed: %v", err)
	}
	if spec.Info.Title != "Reader Test V3" {
		t.Errorf("Expected title 'Reader Test V3', got '%s'", spec.Info.Title)
	}
}

func TestParseAsyncAPI_Errors(t *testing.T) {
	// Invalid JSON - Malformed
	_, err := ParseAsyncAPI([]byte(`{invalid`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Invalid YAML - Type mismatch (asyncapi field should be string)
	// The simplified parser is permissive, so we use a type mismatch to trigger unmarshal error
	_, err = ParseAsyncAPI([]byte(`asyncapi: ["2.6.0"]`))
	if err == nil {
		t.Error("Expected error for invalid YAML type mismatch")
	}
}

func TestParseAsyncAPIV3_Errors(t *testing.T) {
	// Invalid JSON
	_, err := ParseAsyncAPIV3([]byte(`{invalid`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Invalid YAML - Type mismatch
	_, err = ParseAsyncAPIV3([]byte(`asyncapi: ["3.0.0"]`))
	if err == nil {
		t.Error("Expected error for invalid YAML type mismatch")
	}
}

func TestSanitizeIDs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/{id}", "users_id"},
		{"/my-channel", "my-channel"},
		{"/a/b/c", "a_b_c"},
		{"///", "channel"}, // Default case
		{"User ID", "UserID"},
	}

	for _, tt := range tests {
		got := sanitizeChannelID(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeChannelID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
	
	// Test sanitizeOperationID capitalizes first letter
	opID := sanitizeOperationID("/users")
	if opID != "Users" {
		t.Errorf("sanitizeOperationID(/users) = %s, want Users", opID)
	}
}

func TestExtractMessageFromOperation_EdgeCases(t *testing.T) {
	// Test nil operation/message
	msg := extractMessageFromOperation(&Operation{}, false)
	if msg == nil || msg.Payload != nil {
		t.Error("Expected empty message for empty operation")
	}
	
	// Test subscribe with no 200 response
	op := &Operation{
		Responses: map[string]Response{
			"400": {Description: "Error"},
		},
	}
	msg = extractMessageFromOperation(op, false)
	if msg == nil || msg.Payload != nil {
		t.Error("Expected empty message for op without 200 response")
	}
}
