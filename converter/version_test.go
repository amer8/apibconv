package converter

import (
	"testing"
)

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Version
	}{
		{
			name:     "OpenAPI 3.0.0",
			input:    "3.0.0",
			expected: Version30,
		},
		{
			name:     "OpenAPI 3.0.3",
			input:    "3.0.3",
			expected: Version30,
		},
		{
			name:     "OpenAPI 3.1.0",
			input:    "3.1.0",
			expected: Version31,
		},
		{
			name:     "OpenAPI 3.1.1",
			input:    "3.1.1",
			expected: Version31,
		},
		{
			name:     "Invalid version defaults to 3.0",
			input:    "2.0.0",
			expected: Version30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectVersion(tt.input)
			if result != tt.expected {
				t.Errorf("DetectVersion(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVersionToFullVersion(t *testing.T) {
	tests := []struct {
		version  Version
		expected string
	}{
		{Version30, "3.0.0"},
		{Version31, "3.1.0"},
	}

	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			result := tt.version.ToFullVersion()
			if result != tt.expected {
				t.Errorf("%v.ToFullVersion() = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

func TestConversionOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *ConversionOptions
		wantErr bool
	}{
		{
			name: "Valid 3.0",
			opts: &ConversionOptions{
				OutputVersion: Version30,
			},
			wantErr: false,
		},
		{
			name: "Valid 3.1",
			opts: &ConversionOptions{
				OutputVersion: Version31,
			},
			wantErr: false,
		},
		{
			name: "Invalid version",
			opts: &ConversionOptions{
				OutputVersion: "4.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ConversionOptions.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConversionOptions(t *testing.T) {
	opts := DefaultConversionOptions()
	if opts.OutputVersion != Version30 {
		t.Errorf("DefaultConversionOptions().OutputVersion = %v, want %v", opts.OutputVersion, Version30)
	}
	if opts.StrictMode != false {
		t.Errorf("DefaultConversionOptions().StrictMode = %v, want false", opts.StrictMode)
	}
}
