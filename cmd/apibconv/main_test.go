package main

import (
	"os"
	"strings"
	"testing"
)

// Test helper to create temporary files
func createTempFile(t *testing.T, content, suffix string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "apibconv-test-*"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpfile.Name()
}

func TestDetectInputFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "OpenAPI 3.0",
			content: `{
				"openapi": "3.0.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: "openapi",
		},
		{
			name: "OpenAPI 3.1",
			content: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: "openapi",
		},
		{
			name: "AsyncAPI 2.6",
			content: `{
				"asyncapi": "2.6.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: "asyncapi",
		},
		{
			name: "AsyncAPI 3.0",
			content: `{
				"asyncapi": "3.0.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: "asyncapi",
		},
		{
			name:     "Minified OpenAPI",
			content:  `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"}}`,
			expected: "openapi",
		},
		{
			name:     "Minified AsyncAPI",
			content:  `{"asyncapi":"2.6.0","info":{"title":"Test","version":"1.0.0"}}`,
			expected: "asyncapi",
		},
		{
			name: "Whitespace OpenAPI",
			content: `
			
			{
				"openapi": "3.0.0"
			}`,
			expected: "openapi",
		},
		{
			name: "API Blueprint by extension",
			content: `FORMAT: 1A
# Test API

## GET /test`,
			expected: "apib",
		},
		{
			name: "API Blueprint by content",
			content: `FORMAT: 1A
# Test API

## GET /test`,
			expected: "apib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suffix := ".json"
			if tt.name == "API Blueprint by extension" {
				suffix = ".apib"
			}
			tmpfile := createTempFile(t, tt.content, suffix)
			defer func() { _ = os.Remove(tmpfile) }()

			result, err := detectInputFormat(tmpfile)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectInputFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("detectInputFormat() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertOpenAPIToAPIBlueprint(t *testing.T) {
	openapiContent := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0",
			"description": "A test API"
		},
		"servers": [
			{"url": "https://api.example.com"}
		],
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {
										"type": "array",
										"items": {
											"type": "object"
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	inputFilePath := createTempFile(t, openapiContent, ".json")
	defer func() { _ = os.Remove(inputFilePath) }()

	// Use a unique temp file to avoid race conditions when tests run in parallel
	outputFile, err := os.CreateTemp("", "test-output-*.apib")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputFile.Name()
	_ = outputFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	// Open files
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputFile.Close() }()

	outputFile, err = os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputFile.Close() }()

	// Test conversion
	exitCode := convertOpenAPIToAPIBlueprint(inputFile, outputFile)
	if exitCode != 0 {
		t.Fatalf("convertOpenAPIToAPIBlueprint() exit code = %d, want 0", exitCode)
	}

	// Close files before reading
	_ = outputFile.Close()

	// Verify output file was created
	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and verify output content
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	if !strings.Contains(output, "FORMAT: 1A") {
		t.Error("Output should contain FORMAT: 1A")
	}
	if !strings.Contains(output, "# Test API") {
		t.Error("Output should contain API title")
	}
	if !strings.Contains(output, "HOST: https://api.example.com") {
		t.Error("Output should contain server URL")
	}
	if !strings.Contains(output, "## /users") {
		t.Error("Output should contain /users path")
	}
	if !strings.Contains(output, "### List users [GET]") {
		t.Error("Output should contain GET operation")
	}
}

func TestConvertAPIBlueprintToOpenAPI(t *testing.T) {
	apibContent := `FORMAT: 1A
# Test API

A simple test API

HOST: https://api.example.com

## /users [/users]

### List Users [GET]

+ Response 200 (application/json)

        {
            "users": []
        }
`

	inputFilePath := createTempFile(t, apibContent, ".apib")
	defer func() { _ = os.Remove(inputFilePath) }()

	// Use a unique temp file to avoid race conditions when tests run in parallel
	outputTempFile, err := os.CreateTemp("", "test-output-*.json")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	// Open files
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputFile.Close() }()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputFile.Close() }()

	// Test conversion
	exitCode := convertAPIBlueprintToOpenAPI(inputFile, outputFile)
	if exitCode != 0 {
		t.Fatalf("convertAPIBlueprintToOpenAPI() exit code = %d, want 0", exitCode)
	}

	// Close file before reading
	_ = outputFile.Close()

	// Read and verify output content
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	if !strings.Contains(output, `"openapi":`) {
		t.Error("Output should contain OpenAPI version")
	}
	if !strings.Contains(output, `"title": "Test API"`) {
		t.Error("Output should contain API title")
	}
	if !strings.Contains(output, `"url": "https://api.example.com"`) {
		t.Error("Output should contain server URL")
	}
	if !strings.Contains(output, `"/users"`) {
		t.Error("Output should contain /users path")
	}
}

func TestConvertAPIBlueprintToAsyncAPI(t *testing.T) {
	apibContent := `FORMAT: 1A
# Test API

A simple test API

HOST: wss://api.example.com

## /channel [/channel]

### Send Message [POST]

+ Request (application/json)

        {
            "message": "Hello"
        }
`

	tests := []struct {
		name         string
		protocol     string
		wantExitCode int
		wantOutput   bool
	}{
		{
			name:         "Missing protocol",
			protocol:     "",
			wantExitCode: 1,
			wantOutput:   false,
		},
		{
			name:         "Valid protocol",
			protocol:     "ws",
			wantExitCode: 0,
			wantOutput:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputFilePath := createTempFile(t, apibContent, ".apib")
			defer func() { _ = os.Remove(inputFilePath) }()

			outputTempFile, err := os.CreateTemp("", "test-output-*.json")
			if err != nil {
				t.Fatal(err)
			}
			outputFilePath := outputTempFile.Name()
			_ = outputTempFile.Close()
			defer func() { _ = os.Remove(outputFilePath) }()

			inputFile, err := os.Open(inputFilePath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = inputFile.Close() }()

			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = outputFile.Close() }()

			// Save and restore global protocol flag
			oldProtocol := *protocol
			defer func() { *protocol = oldProtocol }()
			*protocol = tt.protocol

			exitCode := convertAPIBlueprintToAsyncAPI(inputFile, outputFile)
			if exitCode != tt.wantExitCode {
				t.Errorf("convertAPIBlueprintToAsyncAPI() exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			if tt.wantOutput {
				_ = outputFile.Close()
				content, err := os.ReadFile(outputFilePath)
				if err != nil {
					t.Fatal(err)
				}
				output := string(content)
				if !strings.Contains(output, `"asyncapi":`) {
					t.Error("Output should contain AsyncAPI version")
				}
				if !strings.Contains(output, `"protocol": "ws"`) {
					t.Error("Output should contain specified protocol")
				}
			}
		})
	}
}

func TestDetectInputFormatNonExistentFile(t *testing.T) {
	_, err := detectInputFormat("/nonexistent/file.json")
	if err == nil {
		t.Error("detectInputFormat() should error on non-existent file")
	}
}

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name       string
		inputFile  string
		outputFile string
		format     string
		wantErr    bool
	}{
		{
			name:       "valid flags",
			inputFile:  "input.json",
			outputFile: "output.apib",
			format:     "json",
			wantErr:    false,
		},
		{
			name:       "missing input",
			inputFile:  "",
			outputFile: "output.apib",
			format:     "json",
			wantErr:    true,
		},
		{
			name:       "missing output",
			inputFile:  "input.json",
			outputFile: "",
			format:     "json",
			wantErr:    true,
		},
		{
			name:       "invalid encoding format",
			inputFile:  "input.json",
			outputFile: "output.apib",
			format:     "xml",
			wantErr:    true,
		},
		{
			name:       "yaml format",
			inputFile:  "input.json",
			outputFile: "output.yaml",
			format:     "yaml",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore flag values
			oldInput := *inputFile
			oldOutput := *outputFile
			oldFormat := *encodingFormat
			defer func() {
				*inputFile = oldInput
				*outputFile = oldOutput
				*encodingFormat = oldFormat
			}()

			*inputFile = tt.inputFile
			*outputFile = tt.outputFile
			*encodingFormat = tt.format

			err := validateFlags()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAutoDetectOutputFormat(t *testing.T) {
	tests := []struct {
		name        string
		outputFile  string
		inputFormat string
		expected    string
	}{
		{
			name:        "apib extension",
			outputFile:  "output.apib",
			inputFormat: "openapi",
			expected:    "apib",
		},
		{
			name:        "json from apib",
			outputFile:  "output.json",
			inputFormat: "apib",
			expected:    "openapi",
		},
		{
			name:        "json from openapi",
			outputFile:  "output.json",
			inputFormat: "openapi",
			expected:    "apib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldOutput := *outputFile
			defer func() { *outputFile = oldOutput }()

			*outputFile = tt.outputFile
			result := autoDetectOutputFormat(tt.inputFormat)
			if result != tt.expected {
				t.Errorf("autoDetectOutputFormat() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRunValidation(t *testing.T) {
	// Test with valid OpenAPI spec
	validSpec := `{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`

	tmpfile := createTempFile(t, validSpec, ".json")
	defer func() { _ = os.Remove(tmpfile) }()

	oldInput := *inputFile
	defer func() { *inputFile = oldInput }()

	*inputFile = tmpfile
	exitCode := runValidation()
	if exitCode != 0 {
		t.Errorf("runValidation() for valid spec = %d, want 0", exitCode)
	}
}

func TestRunValidationInvalid(t *testing.T) {
	// Test with invalid spec (missing required fields)
	invalidSpec := `{
		"openapi": "3.0.0",
		"info": {},
		"paths": {}
	}`

	tmpfile := createTempFile(t, invalidSpec, ".json")
	defer func() { _ = os.Remove(tmpfile) }()

	oldInput := *inputFile
	defer func() { *inputFile = oldInput }()

	*inputFile = tmpfile
	exitCode := runValidation()
	if exitCode != 1 {
		t.Errorf("runValidation() for invalid spec = %d, want 1", exitCode)
	}
}

func TestRunValidationMissingFile(t *testing.T) {
	oldInput := *inputFile
	defer func() { *inputFile = oldInput }()

	*inputFile = ""
	exitCode := runValidation()
	if exitCode != 1 {
		t.Errorf("runValidation() with no input = %d, want 1", exitCode)
	}
}

func TestConvertAsyncAPIToAPIBlueprint_CMD(t *testing.T) {
	asyncapiContent := `{
		"asyncapi": "2.6.0",
		"info": {
			"title": "Async Test",
			"version": "1.0.0"
		},
		"channels": {
			"test": {
				"subscribe": {
					"message": {
						"payload": {"type": "string"}
					}
				}
			}
		}
	}`

	inputFilePath := createTempFile(t, asyncapiContent, ".json")
	defer func() { _ = os.Remove(inputFilePath) }()

	outputTempFile, err := os.CreateTemp("", "test-output-*.apib")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	inputF, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputF.Close() }()

	outputF, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputF.Close() }()

	// Reset file pointers
	_, _ = inputF.Seek(0, 0)
	
	// Direct call to convertAsyncAPIToAPIBlueprint
	oldInput := *inputFile
	defer func() { *inputFile = oldInput }()
	*inputFile = inputFilePath // Needed for log message
	*outputFile = outputFilePath

	exitCode := convertAsyncAPIToAPIBlueprint(inputF, outputF)
	if exitCode != 0 {
		t.Errorf("convertAsyncAPIToAPIBlueprint exit code = %d, want 0", exitCode)
	}

	_ = outputF.Close()
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "FORMAT: 1A") {
		t.Error("Expected FORMAT: 1A in output")
	}
}

func TestPerformConversion_Unsupported(t *testing.T) {
	// Test unsupported conversion paths
	tests := []struct {
		inputFmt  string
		outputFmt string
	}{
		{"asyncapi", "openapi"},
		{"openapi", "asyncapi"},
		{"json", "yaml"}, // Invalid formats
		{"openapi", "openapi"},
	}

	for _, tt := range tests {
		t.Run(tt.inputFmt+"->"+tt.outputFmt, func(t *testing.T) {
			exitCode := performConversion(nil, nil, tt.inputFmt, tt.outputFmt)
			if exitCode != 1 {
				t.Errorf("performConversion(%s, %s) exit code = %d, want 1", tt.inputFmt, tt.outputFmt, exitCode)
			}
		})
	}
}

func TestConvertAPIBlueprintToOpenAPI_Version31(t *testing.T) {
	apibContent := `FORMAT: 1A
# Test API
## /test [/test]
### GET [GET]
+ Response 200
`
	inputFilePath := createTempFile(t, apibContent, ".apib")
	defer func() { _ = os.Remove(inputFilePath) }()
	
	outputTempFile, err := os.CreateTemp("", "test-output-*.json")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	inputF, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputF.Close() }()

	outputF, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputF.Close() }()

	// Set version flag
	oldVer := *openapiVersion
	defer func() { *openapiVersion = oldVer }()
	*openapiVersion = "3.1"
	*inputFile = inputFilePath
	*outputFile = outputFilePath

	exitCode := convertAPIBlueprintToOpenAPI(inputF, outputF)
	if exitCode != 0 {
		t.Fatalf("convertAPIBlueprintToOpenAPI (3.1) exit code = %d", exitCode)
	}

	_ = outputF.Close()
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"openapi": "3.1.0"`) {
		t.Errorf("Expected 3.1.0 version in output, got %s", string(content))
	}
}

func TestConvertAPIBlueprintToAsyncAPI_Version30(t *testing.T) {
	apibContent := `FORMAT: 1A
# Test API
HOST: ws://localhost
## /test [/test]
### POST [POST]
+ Request (application/json)
    + Body
            {"foo":"bar"}
`
	inputFilePath := createTempFile(t, apibContent, ".apib")
	defer func() { _ = os.Remove(inputFilePath) }()
	
	outputTempFile, err := os.CreateTemp("", "test-output-*.json")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	inputF, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputF.Close() }()

	outputF, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputF.Close() }()

	// Set flags
	oldProto := *protocol
	oldVer := *asyncapiVersion
	defer func() { 
		*protocol = oldProto
		*asyncapiVersion = oldVer
	}()
	*protocol = "ws"
	*asyncapiVersion = "3.0"
	*inputFile = inputFilePath
	*outputFile = outputFilePath

	exitCode := convertAPIBlueprintToAsyncAPI(inputF, outputF)
	if exitCode != 0 {
		t.Fatalf("convertAPIBlueprintToAsyncAPI (3.0) exit code = %d", exitCode)
	}

	_ = outputF.Close()
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"asyncapi": "3.0.0"`) {
		t.Errorf("Expected 3.0.0 version in output, got %s", string(content))
	}
}
