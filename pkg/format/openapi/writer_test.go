package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"gopkg.in/yaml.v3"
)

// A helper to remove dynamic default 404 response added by writer for testing.
func removeDefault404(doc map[string]interface{}) {
	if paths, ok := doc["paths"].(map[string]interface{}); ok {
		for _, pathItem := range paths {
			if piMap, ok := pathItem.(map[string]interface{}); ok {
				for _, operation := range piMap {
					if opMap, ok := operation.(map[string]interface{}); ok {
						if responses, ok := opMap["responses"].(map[string]interface{}); ok {
							delete(responses, "404")
						}
					}
				}
			}
		}
	}
	if swgPaths, ok := doc["swagger"].(map[string]interface{}); ok {
		if paths, ok := swgPaths["paths"].(map[string]interface{}); ok {
			for _, pathItem := range paths {
				if piMap, ok := pathItem.(map[string]interface{}); ok {
					for _, operation := range piMap {
						if opMap, ok := operation.(map[string]interface{}); ok {
							if responses, ok := opMap["responses"].(map[string]interface{}); ok {
								delete(responses, "404")
							}
						}
					}
				}
			}
		}
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name       string
		api        *model.API
		writerOpts []WriterOption
		ctx        context.Context
		wantOutput string
		wantErr    bool
	}{
		{
			name: "OpenAPI 3.0 YAML Output",
			api: &model.API{
				Version: "3.0.0",
				Info: model.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: map[string]model.PathItem{
					"/test": {
						Get: &model.Operation{
							Responses: model.Responses{
								"200": model.Response{
									Description: "OK",
								},
							},
						},
					},
				},
			},
			writerOpts: []WriterOption{WithYAML(true), WithWriterVersion("3.0")},
			ctx:        context.Background(),
			wantOutput: strings.TrimSpace(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
        "404":
          description: Not Found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
components:
  schemas:
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
        message:
          type: string
`),
		},
		{
			name: "OpenAPI 3.1 JSON Output with nullable schema",
			api: &model.API{
				Version: "3.1.0",
				Info: model.Info{
					Title:   "Test API 3.1",
					Version: "1.0.0",
				},
				Components: model.Components{
					Schemas: map[string]*model.Schema{
						"NullableProperty": {
							Type:     "string",
							Nullable: true,
						},
					},
				},
			},
			writerOpts: []WriterOption{WithJSONOutput(true), WithWriterVersion("3.1")},
			ctx:        context.Background(),
			wantOutput: strings.TrimSpace(`
{
  "openapi": "3.1.0",
  "info": {
    "title": "Test API 3.1",
    "version": "1.0.0"
  },
  "paths": {},
  "components": {
    "schemas": {
      "Error": {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "type": "integer"
          },
          "message": {
            "type": "string"
          }
        }
      },
      "NullableProperty": {
        "type": [
          "string",
          "null"
        ]
      }
    }
  }
}`),
		},
		{
			name: "OpenAPI 2.0 JSON Output",
			api: &model.API{
				Version: "2.0",
				Info: model.Info{
					Title:   "Swagger API",
					Version: "1.0.0",
				},
				Servers: []model.Server{{URL: "http://api.example.com"}},
				Paths: map[string]model.PathItem{
					"/users": {
						Post: &model.Operation{
							OperationID: "createUser",
							Responses: model.Responses{
								"201": model.Response{
									Description: "User created",
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
								"name": {Type: "string"},
							},
						},
					},
				},
			},
			writerOpts: []WriterOption{WithJSONOutput(true), WithWriterVersion("2.0")},
			ctx:        context.Background(),
			wantOutput: strings.TrimSpace(`
{
  "swagger": "2.0",
  "info": {
    "title": "Swagger API",
    "version": "1.0.0"
  },
  "host": "http://api.example.com",
  "paths": {
    "/users": {
      "post": {
        "operationId": "createUser",
        "responses": {
          "201": {
            "description": "User created"
          },
          "404": {
            "description": "Not Found",
            "schema": {
              "$ref": "#/definitions/Error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Error": {
      "type": "object",
      "required": [
        "code",
        "message"
      ],
      "properties": {
        "code": {
          "type": "integer"
        },
        "message": {
          "type": "string"
        }
      }
    },
    "User": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        }
      }
    }
  }
}`),
		},
		{
			name: "Context Encoding Override - YAML",
			api: &model.API{
				Version: "3.0.0",
				Info:    model.Info{Title: "Context Test"},
				Paths: map[string]model.PathItem{
					"/": {
						Get: &model.Operation{Responses: model.Responses{"200": {Description: "OK"}}},
					},
				},
			},
			writerOpts: []WriterOption{WithJSONOutput(true)},                 // Default to JSON
			ctx:        converter.WithEncoding(context.Background(), "yaml"), // Context forces YAML
			wantOutput: strings.TrimSpace(`
openapi: 3.0.0
info:
  title: Context Test
  version: ""
paths:
  /:
    get:
      responses:
        "200":
          description: OK
        "404":
          description: Not Found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
components:
  schemas:
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
        message:
          type: string
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(tt.writerOpts...)
			var buf bytes.Buffer
			err := w.Write(tt.ctx, tt.api, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Unmarshal actual and expected to ignore whitespace/ordering differences
			var gotDoc, wantDoc map[string]interface{}
			if strings.Contains(tt.wantOutput, "openapi") || strings.Contains(tt.wantOutput, "info") {
				// Assume YAML for OpenAPI/info in root
				err = yaml.Unmarshal(buf.Bytes(), &gotDoc)
				if err != nil {
					t.Fatalf("Failed to unmarshal actual YAML: %v\n%s", err, buf.String())
				}
				err = yaml.Unmarshal([]byte(tt.wantOutput), &wantDoc)
				if err != nil {
					t.Fatalf("Failed to unmarshal expected YAML: %v\n%s", err, tt.wantOutput)
				}
			} else {
				// Assume JSON
				err = json.Unmarshal(buf.Bytes(), &gotDoc)
				if err != nil {
					t.Fatalf("Failed to unmarshal actual JSON: %v\n%s", err, buf.String())
				}
				err = json.Unmarshal([]byte(tt.wantOutput), &wantDoc)
				if err != nil {
					t.Fatalf("Failed to unmarshal expected JSON: %v\n%s", err, tt.wantOutput)
				}
			}

			removeDefault404(gotDoc)
			removeDefault404(wantDoc)

			gotBytes, err := json.MarshalIndent(gotDoc, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal actual doc to JSON: %v", err)
			}
			wantBytes, err := json.MarshalIndent(wantDoc, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal expected doc to JSON: %v", err)
			}

			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("Write() mismatch\n--- WANT ---\n%s\n--- GOT ---\n%s", string(wantBytes), string(gotBytes))
			}
		})
	}
}

func TestWriterFormat(t *testing.T) {
	w := NewWriter()
	if w.Format() != format.FormatOpenAPI {
		t.Errorf("Format() = %v, want %v", w.Format(), format.FormatOpenAPI)
	}
}

func TestWriterVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"Empty", "", ""},
		{"Explicit 3.0", "3.0", "3.0"},
		{"Explicit 3.1", "3.1", "3.1"},
		{"Full 3.0.1", "3.0.1", "3.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(WithWriterVersion(tt.version))
			if got := w.Version(); got != tt.want {
				t.Errorf("Version() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriterOptions(t *testing.T) {
	// Test WithIndent
	w := NewWriter(WithIndent(4), WithYAML(true))
	var buf bytes.Buffer
	api := &model.API{Info: model.Info{Title: "Indent Test"}, Version: "3.0.0"}
	err := w.Write(context.Background(), api, &buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !strings.Contains(buf.String(), "    title: Indent Test") {
		t.Errorf("WithIndent failed, got:\n%s", buf.String())
	}

	// Test WithJSONOutput
	w = NewWriter(WithJSONOutput(true))
	buf.Reset()
	err = w.Write(context.Background(), api, &buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !strings.Contains(buf.String(), "  \"title\": \"Indent Test\"") {
		t.Errorf("WithJSONOutput failed, got:\n%s", buf.String())
	}
}
