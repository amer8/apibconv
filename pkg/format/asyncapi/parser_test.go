package asyncapi

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
			name: "AsyncAPI 2.6 Simple",
			input: `
asyncapi: 2.6.0
info:
  title: User API
  version: 1.0.0
channels:
  user/signedup:
    publish:
      operationId: userSignedUp
      message:
        payload:
          type: object
          properties:
            userId:
              type: string
`,
			wantAPI: &model.API{
				Version: "2.6.0",
				Info: model.Info{
					Title:   "User API",
					Version: "1.0.0",
				},
				Paths: map[string]model.PathItem{
					"user/signedup": {
						Name: "userSignedup", // toCamelCase logic
						Post: &model.Operation{
							OperationID: "userSignedUp",
							RequestBody: &model.RequestBody{
								Content: map[string]model.MediaType{
									"application/json": {
										Schema: &model.Schema{
											Type: "object",
											Properties: map[string]*model.Schema{
												"userId": {
													Type: "string",
												},
											},
										},
									},
								},
							},
							Responses: model.Responses{
								"200": model.Response{Description: "OK"},
							},
							Parameters: nil,
						},
						Parameters: []model.Parameter{},
					},
				},
				Servers: nil, // Initialized but empty if no servers
				Components: model.Components{
					Schemas:         make(map[string]*model.Schema),
					Responses:       make(map[string]model.Response),
					Parameters:      make(map[string]model.Parameter),
					Examples:        make(map[string]model.Example),
					RequestBodies:   make(map[string]model.RequestBody),
					Headers:         make(map[string]model.Header),
					SecuritySchemes: make(map[string]model.SecurityScheme),
					Links:           make(map[string]model.Link),
					Callbacks:       make(map[string]model.Callback),
				},
				Extensions: map[string]interface{}{}, Webhooks: map[string]model.PathItem{},
			},
		},
		{
			name: "AsyncAPI 3.0 Basic (Incomplete support expected)",
			input: `
asyncapi: 3.0.0
info:
  title: Streetlights API
  version: 1.0.0
channels:
  lightingMeasured:
    address: 'smartylighting/streetlights/1/0/event/lighting/measured'
operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
    summary: Inform about environmental lighting conditions
`,
			wantAPI: &model.API{
				Version: "3.0.0",
				Info: model.Info{
					Title:   "Streetlights API",
					Version: "1.0.0",
				},
				Paths: map[string]model.PathItem{
					"smartylighting/streetlights/1/0/event/lighting/measured": {
						Get: &model.Operation{
							OperationID: "receiveLightMeasurement",
							Summary:     "Inform about environmental lighting conditions",
						},
					},
				},
				Servers: nil,
				Components: model.Components{
					Schemas:         make(map[string]*model.Schema),
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
		{
			name:    "Invalid Input",
			input:   "invalid: - yaml",
			wantErr: true,
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
	if p.Format() != format.FormatAsyncAPI {
		t.Errorf("Format() = %v, want %v", p.Format(), format.FormatAsyncAPI)
	}
}

func TestSupportsVersion(t *testing.T) {
	p := NewParser()
	if !p.SupportsVersion("2.0.0") {
		t.Error("SupportsVersion(2.0.0) = false, want true")
	}
}
