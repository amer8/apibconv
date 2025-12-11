package converter_test

import (
	"fmt"
	"log"

	"github.com/amer8/apibconv/converter"
)

// Example demonstrates parsing OpenAPI JSON into a structure.
func ExampleParse() {
	data := []byte(`{
		"openapi": "3.0.0",
		"info": {
			"title": "Example API",
			"version": "1.0.0"
		},
		"paths": {}
	}`)

	s, err := converter.Parse(data)
	if err != nil {
		log.Fatal(err)
	}

	spec := s.(*converter.OpenAPI)

	fmt.Printf("API Title: %s\n", spec.Info.Title)
	fmt.Printf("API Version: %s\n", spec.Info.Version)
	// Output:
	// API Title: Example API
	// API Version: 1.0.0
}

// Example demonstrates converting an OpenAPI structure to API Blueprint.
func ExampleOpenAPI_ToBlueprint() {
	spec := &converter.OpenAPI{
		OpenAPI: "3.0.0",
		Info: converter.Info{
			Title:       "Simple API",
			Version:     "1.0.0",
			Description: "A simple API example",
		},
		Servers: []converter.Server{
			{URL: "https://api.example.com"},
		},
		Paths: map[string]converter.PathItem{
			"/hello": {
				Get: &converter.Operation{
					Summary: "Say hello",
					Responses: map[string]converter.Response{
						"200": {
							Description: "Success",
						},
					},
				},
			},
		},
	}

	apiBlueprint, err := spec.ToBlueprint()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apiBlueprint)
	// Output will contain API Blueprint format
}

// Example demonstrates converting an OpenAPI structure to AsyncAPI 2.6.
func ExampleOpenAPI_ToAsyncAPI() {
	spec := &converter.OpenAPI{
		OpenAPI: "3.0.0",
		Info: converter.Info{
			Title:   "My API",
			Version: "1.0.0",
		},
		Servers: []converter.Server{
			{URL: "https://api.example.com"},
		},
		Paths: map[string]converter.PathItem{
			"/messages": {
				Post: &converter.Operation{
					Summary: "Send message",
				},
			},
		},
	}

	asyncSpec, err := spec.ToAsyncAPI(converter.ProtocolWS)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AsyncAPI Version: %s\n", asyncSpec.AsyncAPI)
	if server, ok := asyncSpec.Servers["server0"]; ok {
		fmt.Printf("Protocol: %s\n", server.Protocol)
	}

	// Check if channel exists
	if _, ok := asyncSpec.Channels["messages"]; ok {
		fmt.Println("Channel 'messages' exists")
	}

	// Output:
	// AsyncAPI Version: 2.6.0
	// Protocol: ws
	// Channel 'messages' exists
}

// Example demonstrates converting an OpenAPI structure to AsyncAPI 3.0.
func ExampleOpenAPI_ToAsyncAPIV3() {
	spec := &converter.OpenAPI{
		OpenAPI: "3.0.0",
		Info: converter.Info{
			Title:   "My API",
			Version: "1.0.0",
		},
		Servers: []converter.Server{
			{URL: "https://api.example.com"},
		},
		Paths: map[string]converter.PathItem{
			"/messages": {
				Post: &converter.Operation{
					Summary: "Send message",
				},
			},
		},
	}

	asyncSpec, err := spec.ToAsyncAPIV3(converter.ProtocolKafka)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AsyncAPI Version: %s\n", asyncSpec.AsyncAPI)
	if server, ok := asyncSpec.Servers["server0"]; ok {
		fmt.Printf("Protocol: %s\n", server.Protocol)
	}

	// Check if channel and operation exist
	if _, ok := asyncSpec.Channels["messages"]; ok {
		fmt.Println("Channel 'messages' exists")
	}
	// Operations are at root in v3
	for _, op := range asyncSpec.Operations {
		if op.Action == "send" { // Post -> Send
			fmt.Println("Send operation exists")
			break
		}
	}

	// Output:
	// AsyncAPI Version: 3.0.0
	// Protocol: kafka
	// Channel 'messages' exists
	// Send operation exists
}

// Example demonstrates converting an AsyncAPI structure to OpenAPI 3.0.
func ExampleAsyncAPI_ToOpenAPI() {
	spec := &converter.AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: converter.Info{
			Title:   "Event API",
			Version: "1.0.0",
		},
		Channels: map[string]converter.Channel{
			"events": {
				Subscribe: &converter.AsyncAPIOperation{
					Summary: "Receive events",
				},
			},
		},
	}

	openapiSpec, err := spec.ToOpenAPI()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("OpenAPI Version: %s\n", openapiSpec.OpenAPI)
	if pathItem, ok := openapiSpec.Paths["/events"]; ok {
		// Subscribe -> GET
		fmt.Printf("Operation: %s\n", pathItem.Get.Summary)
	}

	// Output:
	// OpenAPI Version: 3.0.0
	// Operation: Receive events
}

// Example demonstrates converting an AsyncAPI structure to API Blueprint.
func ExampleAsyncAPI_ToBlueprint() {
	spec := &converter.AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: converter.Info{
			Title:   "Event API",
			Version: "1.0.0",
		},
		Channels: map[string]converter.Channel{
			"events": {
				Subscribe: &converter.AsyncAPIOperation{
					Summary: "Receive events",
				},
			},
		},
	}

	bp, err := spec.ToBlueprint()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bp)

	// Output:
	// FORMAT: 1A
	//
	// # Event API
	//
	// ## /events [/events]
	//
	// ### Receive events [GET]
}

// Example demonstrates converting AsyncAPI 2.6 to AsyncAPI 3.0.
func ExampleAsyncAPI_ToAsyncAPIV3() {
	spec := &converter.AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: converter.Info{
			Title:   "Event API",
			Version: "1.0.0",
		},
		Channels: map[string]converter.Channel{
			"events": {
				Subscribe: &converter.AsyncAPIOperation{
					Summary: "Receive events",
				},
			},
		},
	}

	// Note: protocol is needed for server conversion, but we have no servers here.
	v3Spec, err := spec.ToAsyncAPIV3(converter.ProtocolWS)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AsyncAPI Version: %s\n", v3Spec.AsyncAPI)

	// v3 has operations at root
	for _, op := range v3Spec.Operations {
		fmt.Printf("Action: %s\n", op.Action)
		fmt.Printf("Summary: %s\n", op.Summary)
	}

	// Output:
	// AsyncAPI Version: 3.0.0
	// Action: receive
	// Summary: Receive events
}
