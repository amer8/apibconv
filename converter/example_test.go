package converter_test

import (
	"fmt"
	"log"

	"github.com/amer8/apibconv/converter"
)

// Example demonstrates parsing OpenAPI and AsyncAPI JSON into a structure.
func ExampleParse() {
	// OpenAPI Example
	openapiData := []byte(`{
		"openapi": "3.0.0",
		"info": {
			"title": "Example API",
			"version": "1.0.0"
		},
		"paths": {}
	}`)

	s, err := converter.Parse(openapiData)
	if err != nil {
		log.Fatal(err)
	}

	if spec, ok := s.AsOpenAPI(); ok {
		fmt.Printf("API Title: %s\n", spec.Info.Title)
		fmt.Printf("API Version: %s\n", spec.Info.Version)
	}

	// AsyncAPI Example
	asyncapiData := []byte(`{
		"asyncapi": "2.6.0",
		"info": {
			"title": "Example AsyncAPI",
			"version": "1.0.0"
		},
		"channels": {}
	}`)

	s2, err := converter.Parse(asyncapiData)
	if err != nil {
		log.Fatal(err)
	}

	if spec, ok := s2.AsAsyncAPI(0); ok {
		fmt.Printf("AsyncAPI Title: %s\n", spec.Info.Title)
		fmt.Printf("AsyncAPI Version: %s\n", spec.Info.Version)
	}

	// YAML Example
	yamlData := []byte(`
openapi: 3.0.0
info:
  title: YAML API
  version: 1.0.0
paths: {}
`)

	s3, err := converter.Parse(yamlData)
	if err != nil {
		log.Fatal(err)
	}

	if spec, ok := s3.AsOpenAPI(); ok {
		fmt.Printf("YAML API Title: %s\n", spec.Info.Title)
	}

	// API Blueprint Example
	apibData := []byte(`FORMAT: 1A
# My API Blueprint
## Group Users
## /users [GET]
+ Response 200 (text/plain)

    Hello Users!
`)

	s4, err := converter.Parse(apibData)
	if err != nil {
		log.Fatal(err)
	}

	if spec, ok := s4.AsAPIBlueprint(); ok {
		fmt.Printf("API Blueprint Name: %s\n", spec.TitleField)
	} else {
		log.Fatal("Expected OpenAPI spec from API Blueprint")
	}

	// Output:
	// API Title: Example API
	// API Version: 1.0.0
	// AsyncAPI Title: Example AsyncAPI
	// AsyncAPI Version: 1.0.0
	// YAML API Title: YAML API
	// API Blueprint Name: My API Blueprint
}

// Example demonstrates converting an OpenAPI structure to AsyncAPI 2.6.
func ExampleOpenAPI_ToAsyncAPI() {
	yamlData := []byte(`
openapi: 3.0.0
info:
  title: YAML API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /messages:
    post:
      summary: Send message
`)

	s3, err := converter.Parse(yamlData)
	if err != nil {
		log.Fatal(err)
	}

	openAPISpec, ok := s3.AsOpenAPI()
	if !ok {
		log.Fatal("Expected OpenAPI spec")
	}

	asyncSpec, err := openAPISpec.ToAsyncAPI(converter.ProtocolWS, 2)
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
func ExampleOpenAPI_ToAsyncAPI_v3() {
	yamlData := []byte(`
openapi: 3.0.0
info:
  title: YAML API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /messages:
    post:
      summary: Send message
`)

	s3, err := converter.Parse(yamlData)
	if err != nil {
		log.Fatal(err)
	}

	openAPISpec, ok := s3.AsOpenAPI()
	if !ok {
		log.Fatal("Expected OpenAPI spec")
	}

	asyncSpec, err := openAPISpec.ToAsyncAPI(converter.ProtocolKafka, 3)
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
func ExampleAsyncAPI_ToAPIBlueprint() {
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

	bpObj, err := spec.ToAPIBlueprint()
	if err != nil {
		log.Fatal(err)
	}
	bp := bpObj.String()

	fmt.Println(bp)

	// Output:
	// FORMAT: 1A
	//
	// # Event API
	//
	// ## Group Events
	//
	// ### /events [/events]
	//
	// #### Receive events [GET]
}

// Example demonstrates converting AsyncAPI 2.6 to AsyncAPI 3.0.
func ExampleAsyncAPI_ToAsyncAPI_v3() {
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
	v3Spec, err := spec.ToAsyncAPI(converter.ProtocolWS, 3)
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

// Example demonstrates using ProtocolAuto for automatic protocol handling.
func ExampleProtocolAuto() {
	spec := &converter.OpenAPI{
		OpenAPI: "3.0.0",
		Info: converter.Info{
			Title:   "Auto Protocol API",
			Version: "1.0.0",
		},
		Servers: []converter.Server{
			{URL: "https://api.example.com"},
		},
		Paths: map[string]converter.PathItem{
			"/notifications": {
				Get: &converter.Operation{
					Summary: "Subscribe to notifications",
				},
			},
		},
	}

	// Use ProtocolAuto to infer or set a generic protocol
	asyncSpec, err := spec.ToAsyncAPI(converter.ProtocolAuto, 2)
	if err != nil {
		log.Fatal(err)
	}

	if server, ok := asyncSpec.Servers["server0"]; ok {
		fmt.Printf("Protocol: %s\n", server.Protocol)
	}

	// Output:
	// Protocol: auto
}
