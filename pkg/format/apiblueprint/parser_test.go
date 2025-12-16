package apiblueprint

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAPI *model.API
		wantErr bool
	}{
		{
			name: "Simple API Blueprint",
			input: `
FORMAT: 1A
HOST: https://api.example.com

# My API
Description of My API.

## Data Structures

## User (object)
+ id (number, required)
+ name (string)

# Group Users

## User [/users/{id}]

### Get User [GET]
+ Parameters
    + id (number) - User ID

+ Response 200 (application/json)
        {
            "id": 1,
            "name": "John Doe"
        }
`,
			wantAPI: &model.API{
				Version: "1A",
				Info: model.Info{
					Title: "My API",
				},
				Servers: []model.Server{
					{URL: "https://api.example.com"},
				},
				Paths: map[string]model.PathItem{
					"/users/{id}": {
						Summary: "User",
						Get: &model.Operation{
							Summary:    "Get User",
							Tags:       []string{"Users"},
							Parameters: nil,
							Responses: model.Responses{
								"200": model.Response{
									Content: map[string]model.MediaType{
										"application/json": {
											Example: "{\n\"id\": 1,\n\"name\": \"John Doe\"\n}\n\n",
										},
									},
								},
							},
						},
					},
				},
				Components: model.Components{
					Schemas: map[string]*model.Schema{
						"User": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"id":   {Type: "number"},
								"name": {Type: "string"},
							},
							Required: []string{"id"},
						},
					},
					// Initialize other maps as empty to match NewComponents()
					Responses:       make(map[string]model.Response),
					Parameters:      make(map[string]model.Parameter),
					Examples:        make(map[string]model.Example),
					RequestBodies:   make(map[string]model.RequestBody),
					Headers:         make(map[string]model.Header),
					SecuritySchemes: make(map[string]model.SecurityScheme),
					Links:           make(map[string]model.Link),
					Callbacks:       make(map[string]model.Callback),
				},
				Extensions: map[string]interface{}{},
				Webhooks:   map[string]model.PathItem{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			gotAPI, err := p.Parse(context.Background(), strings.NewReader(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Marshal for comparison
			gotBytes, err := json.MarshalIndent(gotAPI, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal actual API: %v", err)
			}
			wantBytes, err := json.MarshalIndent(tt.wantAPI, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal expected API: %v", err)
			}

			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("Parse() mismatch\n--- WANT ---\n%s\n--- GOT ---\n%s", string(wantBytes), string(gotBytes))
			}
		})
	}
}

func TestFormat(t *testing.T) {
	p := NewParser()
	if p.Format() != format.FormatAPIBlueprint {
		t.Errorf("Format() = %v, want %v", p.Format(), format.FormatAPIBlueprint)
	}
}
