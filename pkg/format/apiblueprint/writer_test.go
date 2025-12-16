package apiblueprint

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name       string
		api        *model.API
		wantOutput string
		wantErr    bool
	}{
		{
			name: "Simple API Blueprint Writer",
			api: &model.API{
				Info: model.Info{
					Title:       "My API",
					Description: "Description of My API.",
				},
				Servers: []model.Server{
					{URL: "https://api.example.com"},
				},
				Paths: map[string]model.PathItem{
					"/users/{id}": {
						Summary: "User",
						Get: &model.Operation{
							Summary: "Get User",
							Tags:    []string{"Users"},
							Parameters: []model.Parameter{
								{
									Name:        "id",
									Description: "User ID",
									Schema:      &model.Schema{Type: "number"},
									Required:    true,
								},
							},
							Responses: model.Responses{
								"200": model.Response{
									Content: map[string]model.MediaType{
										"application/json": {
											Example: "{\"id\": 1}",
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
								"id": {Type: "number"},
							},
							Required: []string{"id"},
						},
					},
				},
			},
			wantOutput: `FORMAT: 1A
HOST: https://api.example.com

# My API
Description of My API.

# Group Users

## User [/users/{id}]

### Get User [GET]

+ Parameters
    + id (number) (required) - User ID

+ Response 200 (application/json)

        {"id": 1}

## Data Structures

### User (object)
    + id (number) (required)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter()
			var buf bytes.Buffer
			err := w.Write(context.Background(), tt.api, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Normalize line endings and trim space for comparison
			got := strings.TrimSpace(strings.ReplaceAll(buf.String(), "\r\n", "\n"))
			want := strings.TrimSpace(strings.ReplaceAll(tt.wantOutput, "\r\n", "\n"))

			if got != want {
				t.Errorf("Write() mismatch\n--- WANT ---\n%s\n--- GOT ---\n%s", want, got)
			}
		})
	}
}

func TestWriterFormat(t *testing.T) {
	w := NewWriter()
	if w.Format() != format.FormatAPIBlueprint {
		t.Errorf("Format() = %v, want %v", w.Format(), format.FormatAPIBlueprint)
	}
}
