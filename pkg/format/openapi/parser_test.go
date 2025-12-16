package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAPI    *model.API
		wantErr    bool
		parserOpts []ParserOption
	}{
		{
			name: "OpenAPI 3.0 Simple API",
			input: `
openapi: 3.0.0
info:
  title: Sample API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: A list of users
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
`,
			wantAPI: &model.API{
				Version: "3.0.0",
				Info: model.Info{
					Title:   "Sample API",
					Version: "1.0.0",
				},
				Paths: map[string]model.PathItem{
					"/users": {
						Get: &model.Operation{
							Summary:    "Get all users",
							Parameters: make([]model.Parameter, 0),
							Responses: model.Responses{
								"200": model.Response{
									Description: "A list of users",
									Content:     map[string]model.MediaType{},
									Headers:     map[string]model.Header{},
								},
							},
						},
						Parameters: make([]model.Parameter, 0),
					},
				},
				Components: model.Components{
					Schemas: map[string]*model.Schema{
						"User": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"id": {
									Type: "integer",
								},
							},
						},
					},
					Responses:       map[string]model.Response{},
					Parameters:      map[string]model.Parameter{},
					Examples:        map[string]model.Example{},
					RequestBodies:   map[string]model.RequestBody{},
					Headers:         map[string]model.Header{},
					SecuritySchemes: map[string]model.SecurityScheme{},
					Links:           map[string]model.Link{},
					Callbacks:       map[string]model.Callback{},
				},
				Webhooks:   map[string]model.PathItem{},
				Extensions: map[string]interface{}{},
			},
		},
		{
			name: "OpenAPI 3.1 with nullable and webhooks",
			input: `
openapi: 3.1.0
info:
  title: Advanced API
  version: 1.0.0
paths:
  /items:
    post:
      summary: Create item
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                value:
                  type: [string, null]
      responses:
        '201':
          description: Item created
webhooks:
  newThing:
    post:
      summary: New thing created event
      responses:
        '200':
          description: Success
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string
`,
			wantAPI: &model.API{
				Version: "3.1.0",
				Info: model.Info{
					Title:   "Advanced API",
					Version: "1.0.0",
				},
				Paths: map[string]model.PathItem{
					"/items": {
						Post: &model.Operation{
							Summary:    "Create item",
							Parameters: make([]model.Parameter, 0),
							RequestBody: &model.RequestBody{
								Content: map[string]model.MediaType{
									"application/json": {
										Schema: &model.Schema{
											Type: "object",
											Properties: map[string]*model.Schema{
												"name": {
													Type: "string",
												},
												"value": {
													Type:     "string",
													Nullable: true,
												},
											},
										},
									},
								},
							},
							Responses: model.Responses{
								"201": model.Response{
									Description: "Item created",
									Content:     map[string]model.MediaType{},
									Headers:     map[string]model.Header{},
								},
							},
						},
						Parameters: make([]model.Parameter, 0),
					},
				},
				Webhooks: map[string]model.PathItem{
					"newThing": {
						Post: &model.Operation{
							Summary:    "New thing created event",
							Parameters: make([]model.Parameter, 0),
							Responses: model.Responses{
								"200": model.Response{
									Description: "Success",
									Content:     map[string]model.MediaType{},
									Headers:     map[string]model.Header{},
								},
							},
						},
						Parameters: make([]model.Parameter, 0),
					},
				},
				Components: model.Components{
					Schemas: map[string]*model.Schema{
						"Item": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"id": {
									Type: "string",
								},
							},
						},
					},
					Responses:       map[string]model.Response{},
					Parameters:      map[string]model.Parameter{},
					Examples:        map[string]model.Example{},
					RequestBodies:   map[string]model.RequestBody{},
					Headers:         map[string]model.Header{},
					SecuritySchemes: map[string]model.SecurityScheme{},
					Links:           map[string]model.Link{},
					Callbacks:       map[string]model.Callback{},
				},
				Extensions: map[string]interface{}{},
			},
		},
		{
			name: "OpenAPI 2.0 Simple API",
			input: `
swagger: "2.0"
info:
  title: Swagger Petstore
  version: 1.0.0
host: petstore.swagger.io
basePath: /v1
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: An array of pets
definitions:
  Pet:
    type: object
    properties:
      id:
        type: integer
`,
			wantAPI: &model.API{
				Version: "2.0", // Swagger 2.0 is effectively OpenAPI 2.0
				Info: model.Info{
					Title:   "Swagger Petstore",
					Version: "1.0.0",
				},
				Servers: []model.Server{
					{URL: "https://petstore.swagger.io/v1"}, // Default scheme https added
				},
				Paths: map[string]model.PathItem{
					"/pets": {
						Get: &model.Operation{
							Summary:    "List all pets",
							Parameters: nil,
							Responses: model.Responses{
								"200": model.Response{
									Description: "An array of pets",
									Content:     map[string]model.MediaType{},
									Headers:     map[string]model.Header{},
								},
							},
						},
						Parameters: make([]model.Parameter, 0),
					},
				},
				Components: model.Components{
					Schemas: map[string]*model.Schema{
						"Pet": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"id": {
									Type: "integer",
								},
							},
						},
					},
					Responses:       map[string]model.Response{},
					Parameters:      map[string]model.Parameter{},
					Examples:        map[string]model.Example{},
					RequestBodies:   map[string]model.RequestBody{},
					Headers:         map[string]model.Header{},
					SecuritySchemes: map[string]model.SecurityScheme{},
					Links:           map[string]model.Link{},
					Callbacks:       map[string]model.Callback{},
				},
				Webhooks:   map[string]model.PathItem{},
				Extensions: map[string]interface{}{},
			},
		},
		{
			name:    "Invalid YAML input",
			input:   "invalid: - yaml",
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.parserOpts...)
			gotAPI, err := p.Parse(context.Background(), strings.NewReader(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return // Skip deep comparison if we expect an error
			}

			// Marshal both to JSON for comparison, ignoring whitespace differences
			gotBytes, err := json.MarshalIndent(gotAPI, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal actual API to JSON: %v", err)
			}
			wantBytes, err := json.MarshalIndent(tt.wantAPI, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal expected API to JSON: %v", err)
			}

			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("Parse() mismatch\n--- WANT ---\n%s\n--- GOT ---\n%s", string(wantBytes), string(gotBytes))
			}
		})
	}
}

func TestFormat(t *testing.T) {
	p := NewParser()
	if p.Format() != format.FormatOpenAPI {
		t.Errorf("Format() = %v, want %v", p.Format(), format.FormatOpenAPI)
	}
}

func TestSupportsVersion(t *testing.T) {
	p := NewParser()
	tests := []struct {
		version string
		want    bool
	}{
		{"2.0", true},
		{"3.0", true},
		{"3.1", true},
		{"1.0", true}, // Parser should return true for any version since it attempts to parse all supported versions internally.
		{"unknown", true},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("Version_%s", tt.version)
		t.Run(name, func(t *testing.T) {
			if got := p.SupportsVersion(tt.version); got != tt.want {
				t.Errorf("SupportsVersion(%s) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
