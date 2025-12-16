package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
)

func TestDetectFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFormat  format.Format
		wantVersion string
		wantErr     bool
	}{
		// OpenAPI Cases
		{
			name:        "OpenAPI 2.0 JSON",
			input:       `{"swagger": "2.0", "info": {}}`,
			wantFormat:  format.FormatOpenAPI,
			wantVersion: "2.0",
		},
		{
			name:        "OpenAPI 2.0 YAML",
			input:       `swagger: "2.0"`,
			wantFormat:  format.FormatOpenAPI,
			wantVersion: "2.0",
		},
		{
			name:        "OpenAPI 3.0.0 YAML",
			input:       `openapi: 3.0.0`,
			wantFormat:  format.FormatOpenAPI,
			wantVersion: "3.0.x",
		},
		{
			name:        "OpenAPI 3.0.1 JSON",
			input:       `{"openapi": "3.0.1"}`,
			wantFormat:  format.FormatOpenAPI,
			wantVersion: "3.0.x",
		},
		{
			name:        "OpenAPI 3.1.0 YAML",
			input:       `openapi: 3.1.0`,
			wantFormat:  format.FormatOpenAPI,
			wantVersion: "3.1.x",
		},

		// AsyncAPI Cases
		{
			name:        "AsyncAPI 2.0.0",
			input:       `asyncapi: 2.0.0`,
			wantFormat:  format.FormatAsyncAPI,
			wantVersion: "2.0",
		},
		{
			name:        "AsyncAPI 2.6.0",
			input:       `asyncapi: '2.6.0'`,
			wantFormat:  format.FormatAsyncAPI,
			wantVersion: "2.6",
		},
		{
			name:        "AsyncAPI 3.0.0",
			input:       `asyncapi: 3.0.0`,
			wantFormat:  format.FormatAsyncAPI,
			wantVersion: "3.0",
		},
		{
			name:        "AsyncAPI JSON",
			input:       `{"asyncapi": "2.6.0"}`,
			wantFormat:  format.FormatAsyncAPI,
			wantVersion: "2.6",
		},

		// API Blueprint Cases
		{
			name:        "API Blueprint Format Header",
			input:       `FORMAT: 1A`,
			wantFormat:  format.FormatAPIBlueprint,
			wantVersion: "1A",
		},
		{
			name:        "API Blueprint Metadata",
			input:       `# My API`,
			wantFormat:  format.FormatAPIBlueprint,
			wantVersion: "1A",
		},

		// Unknown/Invalid Cases
		{
			name:       "Empty Input",
			input:      ``,
			wantFormat: format.FormatUnknown,
			wantErr:    true,
		},
		{
			name:       "Random Text",
			input:      `Hello World`,
			wantFormat: format.FormatUnknown,
			wantErr:    true,
		},
		{
			name:       "Invalid JSON",
			input:      `{"invalid": }`,
			wantFormat: format.FormatUnknown,
			wantErr:    true,
		},
	}

	detector := NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFormat, gotVersion, err := detector.DetectFromBytes([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFormat != tt.wantFormat {
				t.Errorf("DetectFromBytes() format = %v, want %v", gotFormat, tt.wantFormat)
			}
			if gotVersion != tt.wantVersion {
				t.Errorf("DetectFromBytes() version = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}
}

func TestDetect_Reader(t *testing.T) {
	// Simple test to ensure the Reader wrapper works, leveraging one of the simple cases
	input := `openapi: 3.0.0`
	detector := NewDetector()

	gotFormat, gotVersion, err := detector.Detect(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if gotFormat != format.FormatOpenAPI {
		t.Errorf("Detect() format = %v, want %v", gotFormat, format.FormatOpenAPI)
	}
	if gotVersion != "3.0.x" {
		t.Errorf("Detect() version = %v, want %v", gotVersion, "3.0.x")
	}
}
