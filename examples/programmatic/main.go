package main

import (
	"fmt"
	"log"

	"github.com/amer8/apibconv/converter"
)

func main() {
	// Example 1: Simple string conversion
	fmt.Println("=== Example 1: Simple String Conversion ===")
	exampleSimpleConversion()

	fmt.Println("\n=== Example 2: Parse and Modify ===")
	exampleParseAndModify()

	fmt.Println("\n=== Example 3: Working with Bytes ===")
	exampleWithBytes()
}

func exampleSimpleConversion() {
	openapiJSON := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Pet Store API",
			"version": "1.0.0",
			"description": "A simple pet store API"
		},
		"servers": [
			{"url": "https://api.petstore.com"}
		],
		"paths": {
			"/pets": {
				"get": {
					"summary": "List all pets",
					"responses": {
						"200": {
							"description": "Success"
						}
					}
				}
			}
		}
	}`

	// Convert directly from JSON string to API Blueprint
	apiBlueprint, err := converter.FromJSONString(openapiJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(apiBlueprint)
}

func exampleParseAndModify() {
	openapiJSON := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Original API",
			"version": "1.0.0"
		},
		"servers": [
			{"url": "https://api.example.com"}
		],
		"paths": {
			"/users": {
				"get": {
					"summary": "Get users",
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	// Parse the OpenAPI spec
	spec, err := converter.Parse([]byte(openapiJSON))
	if err != nil {
		log.Fatal(err)
	}

	// Modify the spec programmatically
	spec.Info.Title = "Modified API Title"
	spec.Info.Description = "This title was changed programmatically"

	// Format back to API Blueprint
	apiBlueprint, err := converter.Format(spec)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(apiBlueprint)
}

func exampleWithBytes() {
	openapiJSON := []byte(`{
		"openapi": "3.0.0",
		"info": {
			"title": "Bytes API",
			"version": "1.0.0"
		},
		"servers": [
			{"url": "https://api.bytes.com"}
		],
		"paths": {
			"/convert": {
				"post": {
					"summary": "Convert data",
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`)

	// Convert to bytes (useful for storing or transmitting)
	resultBytes, err := converter.ToBytes(openapiJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result size: %d bytes\n", len(resultBytes))
	fmt.Println(string(resultBytes))
}
