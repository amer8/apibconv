package converter_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/amer8/apibconv/converter"
)

// ExampleFromJSONString demonstrates converting an OpenAPI JSON string to API Blueprint format.
func ExampleFromJSONString() {
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

	apiBlueprint, err := converter.FromJSONString(openapiJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(apiBlueprint)
	// Output will contain API Blueprint format with FORMAT: 1A header
}

// ExampleToOpenAPIString demonstrates converting API Blueprint to OpenAPI JSON format.
func ExampleToOpenAPIString() {
	apibContent := `FORMAT: 1A

# My API

A simple API example

HOST: https://api.example.com

## /users [/users]

### List Users [GET]

Get a list of all users

+ Response 200 (application/json)

        {"users": []}`

	openapiJSON, err := converter.ToOpenAPIString(apibContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(openapiJSON)
	// Output will contain OpenAPI 3.0 JSON specification
}

// ExampleParse demonstrates parsing OpenAPI JSON into a structure.
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

// ExampleFormat demonstrates formatting an OpenAPI structure to API Blueprint.
func ExampleFormat() {
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

// ExampleConvert demonstrates streaming conversion from OpenAPI to API Blueprint.
func ExampleConvert() {
	openapiJSON := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Streaming Example",
			"version": "1.0.0"
		},
		"paths": {
			"/data": {
				"get": {
					"summary": "Get data",
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	reader := strings.NewReader(openapiJSON)
	writer := os.Stdout

	err := converter.Convert(reader, writer)
	if err != nil {
		log.Fatal(err)
	}
	// Output will be written to stdout in API Blueprint format
}

// ExampleConvertToOpenAPI demonstrates streaming conversion from API Blueprint to OpenAPI.
func ExampleConvertToOpenAPI() {
	apibContent := `FORMAT: 1A

# Streaming API

HOST: https://api.stream.com

## /stream [/stream]

### Get Stream [GET]

+ Response 200 (application/json)

        {"status": "ok"}`

	reader := strings.NewReader(apibContent)
	writer := os.Stdout

	err := converter.ConvertToOpenAPI(reader, writer)
	if err != nil {
		log.Fatal(err)
	}
	// Output will be written to stdout in OpenAPI JSON format
}

// ExampleParseBlueprint demonstrates parsing API Blueprint to OpenAPI structure.
func ExampleParseBlueprint() {
	apibContent := []byte(`FORMAT: 1A

# Parse Example API

HOST: https://api.parse.com

## /data [/data]

### Get Data [GET]

+ Response 200 (application/json)

        {"result": "success"}`)

	spec, err := converter.ParseBlueprint(apibContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("API Title: %s\n", spec.Info.Title)
	fmt.Printf("Server URL: %s\n", spec.Servers[0].URL)
	// Output:
	// API Title: Parse Example API
	// Server URL: https://api.parse.com
}

// Example demonstrates a complete workflow: parse OpenAPI, modify it, and format to API Blueprint.
func Example() {
	// Parse OpenAPI JSON
	openapiJSON := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Original API",
			"version": "1.0.0"
		},
		"servers": [{"url": "https://api.example.com"}],
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

	s, err := converter.Parse([]byte(openapiJSON))
	if err != nil {
		log.Fatal(err)
	}
	
	spec := s.(*converter.OpenAPI)

	// Modify the spec programmatically
	spec.Info.Title = "Modified API"
	spec.Info.Description = "This API has been modified"

	// Format to API Blueprint
	apiBlueprint, err := spec.ToBlueprint()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apiBlueprint)
	// Output will contain API Blueprint with modified title
}
