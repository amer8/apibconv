package asyncapi

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name       string
		api        *model.API
		writerOpts []WriterOption
		wantOutput string
		wantErr    bool
	}{
		{
			name: "AsyncAPI 2.0 Writer",
			api: &model.API{
				Version: "2.6.0",
				Info:    model.Info{Title: "Async Service", Version: "1.0"},
				Paths: map[string]model.PathItem{
					"events": {
						Post: &model.Operation{
							Summary: "Publish event",
							RequestBody: &model.RequestBody{
								Content: map[string]model.MediaType{
									"application/json": {
										Schema: &model.Schema{
											Type: "object",
										},
									},
								},
							},
						},
					},
				},
			},
			writerOpts: []WriterOption{WithAsyncWriterVersion("2.6"), WithJSONOutput(true)},
			wantOutput: `{
  "asyncapi": "2.6.0",
  "channels": {
    "events": {
      "publish": {
        "message": {
          "payload": {
            "type": "object"
          }
        },
        "summary": "Publish event"
      }
    }
  },
  "info": {
    "title": "Async Service",
    "version": "1.0"
  }
}`,
		},
		{
			name: "AsyncAPI 3.0 Writer (Basic)",
			api: &model.API{
				Version: "3.0.0",
				Info:    model.Info{Title: "Async V3", Version: "1.0"},
				Paths: map[string]model.PathItem{
					"my/channel": {
						Name: "myChannel",
						Post: &model.Operation{
							Summary: "Send Message",
						},
					},
				},
			},
			writerOpts: []WriterOption{WithAsyncWriterVersion("3.0"), WithJSONOutput(true)},
			wantOutput: `{
  "asyncapi": "3.0.0",
  "info": {
    "title": "Async V3",
    "version": "1.0"
  },
  "channels": {
    "myChannel": {
      "address": "my/channel"
    }
  },
  "operations": {
    "send_myChannel": {
      "action": "send",
      "channel": {
        "$ref": "#/channels/myChannel"
      },
      "summary": "Send Message"
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(tt.writerOpts...)
			var buf bytes.Buffer
			err := w.Write(context.Background(), tt.api, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare JSON structure
			var gotDoc, wantDoc map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &gotDoc); err != nil {
				t.Fatalf("Failed to unmarshal got JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantOutput), &wantDoc); err != nil {
				t.Fatalf("Failed to unmarshal want JSON: %v", err)
			}

			gotBytes, _ := json.MarshalIndent(gotDoc, "", "  ")
			wantBytes, _ := json.MarshalIndent(wantDoc, "", "  ")

			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("Write() mismatch\n--- WANT ---\n%s\n--- GOT ---\n%s", string(wantBytes), string(gotBytes))
			}
		})
	}
}

func TestWriterFormat(t *testing.T) {
	w := NewWriter()
	if w.Format() != format.FormatAsyncAPI {
		t.Errorf("Format() = %v, want %v", w.Format(), format.FormatAsyncAPI)
	}
}

func TestWriterVersion(t *testing.T) {
	w := NewWriter(WithAsyncWriterVersion("2.4.0"))
	if w.Version() != "2.4.0" {
		t.Errorf("Version() = %v, want 2.4.0", w.Version())
	}
}
