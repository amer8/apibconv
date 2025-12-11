package cli

import (
	"flag"
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
			expected: formatOpenAPI,
		},
		{
			name: "OpenAPI 3.1",
			content: `{
				"openapi": "3.1.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: formatOpenAPI,
		},
		{
			name: "AsyncAPI 2.6",
			content: `{
				"asyncapi": "2.6.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: formatAsyncAPI,
		},
		{
			name: "AsyncAPI 3.0",
			content: `{
				"asyncapi": "3.0.0",
				"info": {"title": "Test", "version": "1.0.0"}
			}`,
			expected: formatAsyncAPI,
		},
		{
			name: "API Blueprint by extension",
			content: `FORMAT: 1A
# Test API

## GET /test`,
			expected: formatAPIB,
		},
		{
			name: "API Blueprint by content",
			content: `FORMAT: 1A
# Test API

## GET /test`,
			expected: formatAPIB,
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

func TestConvertAPIBlueprintToOpenAPI_V31(t *testing.T) {
	apibContent := `FORMAT: 1A
# Test API V31

A simple test API for OpenAPI 3.1 conversion.

HOST: https://api.example.com

## /items [/items]

### List Items [GET]

+ Response 200 (application/json)

        {
            "items": []
        }
`

	inputFilePath := createTempFile(t, apibContent, ".apib")
	defer func() { _ = os.Remove(inputFilePath) }()

	outputTempFile, err := os.CreateTemp("", "test-output-v31-*.json")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	oldInputFile := inputFile
	oldOutputFile := outputFile
	oldOpenAPIVersion := openapiVersion
	oldOutputFormat := outputFormat
	oldEncodingFormat := encodingFormat

	defer func() {
		inputFile = oldInputFile
		outputFile = oldOutputFile
		openapiVersion = oldOpenAPIVersion
		outputFormat = oldOutputFormat
		encodingFormat = oldEncodingFormat
	}()

	inputFile = inputFilePath
	outputFile = outputFilePath
	openapiVersion = "3.1"
	outputFormat = formatOpenAPI
	encodingFormat = encodingJSON // Ensure JSON output for easy string check

	// Open files
	inputFileRead, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputFileRead.Close() }()

	outputFileWrite, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputFileWrite.Close() }()

	// Test conversion
	exitCode := convertAPIBlueprintToOpenAPI(inputFileRead, outputFileWrite)
	if exitCode != 0 {
		t.Fatalf("convertAPIBlueprintToOpenAPI() for v3.1 exit code = %d, want 0", exitCode)
	}

	// Close file before reading
	_ = outputFileWrite.Close()

	// Read and verify output content
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	if !strings.Contains(output, `"openapi": "3.1.0"`) {
		t.Errorf("Output should contain OpenAPI 3.1.0 version, got: %s", output)
	}
	if !strings.Contains(output, `"title": "Test API V31"`) {
		t.Error("Output should contain API title")
	}
	if !strings.Contains(output, `"url": "https://api.example.com"`) {
		t.Error("Output should contain server URL")
	}
	if !strings.Contains(output, `"/items"`) {
		t.Error("Output should contain /items path")
	}
}

func TestConvertAPIBlueprintToAsyncAPI_V30(t *testing.T) {
	apibContent := `FORMAT: 1A
# Events API

A test API for AsyncAPI 3.0 conversion.

HOST: https://api.example.com

## /events [/events]

### Receive events [GET]
+ Response 200 (application/json)
    ` + "```json" + `
    {"message": "Event received"}
    ` + "```" + `

### Send events [POST]
+ Request (application/json)
    ` + "```json" + `
    {"data": "some data"}
    ` + "```" + `
+ Response 200 (application/json)
`

	inputFilePath := createTempFile(t, apibContent, ".apib")
	defer func() { _ = os.Remove(inputFilePath) }()

	outputTempFile, err := os.CreateTemp("", "test-output-asyncapi-v30-*.json")
	if err != nil {
		t.Fatal(err)
	}
	outputFilePath := outputTempFile.Name()
	_ = outputTempFile.Close()
	defer func() { _ = os.Remove(outputFilePath) }()

	oldInputFile := inputFile
	oldOutputFile := outputFile
	oldAsyncAPIVersion := asyncapiVersion
	oldOutputFormat := outputFormat
	oldProtocol := protocol
	oldEncodingFormat := encodingFormat

	defer func() {
		inputFile = oldInputFile
		outputFile = oldOutputFile
		asyncapiVersion = oldAsyncAPIVersion
		outputFormat = oldOutputFormat
		protocol = oldProtocol
		encodingFormat = oldEncodingFormat
	}()

	inputFile = inputFilePath
	outputFile = outputFilePath
	asyncapiVersion = "3.0"
	outputFormat = formatAsyncAPI
	protocol = "kafka"
	encodingFormat = encodingJSON // Ensure JSON output for easy string check

	// Open files
	inputFileRead, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inputFileRead.Close() }()

	outputFileWrite, err := os.Create(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outputFileWrite.Close() }()

	// Test conversion
	exitCode := convertAPIBlueprintToAsyncAPI(inputFileRead, outputFileWrite)
	if exitCode != 0 {
		t.Fatalf("convertAPIBlueprintToAsyncAPI() for v3.0 exit code = %d, want 0", exitCode)
	}

	// Close file before reading
	_ = outputFileWrite.Close()

	// Read and verify output content
	content, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	if !strings.Contains(output, `"asyncapi": "3.0.0"`) {
		t.Errorf("Output should contain AsyncAPI 3.0.0 version, got: %s", output)
	}
	if !strings.Contains(output, `"title": "Events API"`) {
		t.Error("Output should contain API title")
	}
	if !strings.Contains(output, `"protocol": "kafka"`) {
		t.Errorf("Output should contain \"protocol\": \"kafka\", got: %s", output)
	}
	if !strings.Contains(output, `"operations": {`) { // AsyncAPI 3.0 uses 'operations' at root
		t.Errorf("Output should contain \"operations\": { for AsyncAPI 3.0 operations, got: %s", output)
	}
	if !strings.Contains(output, `"action": "receive"`) {
		t.Errorf("Output should contain \"action\": \"receive\" for AsyncAPI 3.0, got: %s", output)
	}
	if !strings.Contains(output, `"action": "send"`) {
		t.Errorf("Output should contain \"action\": \"send\" for AsyncAPI 3.0, got: %s", output)
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
			format:     encodingJSON,
			wantErr:    false,
		},
		{
			name:       "missing input",
			inputFile:  "",
			outputFile: "output.apib",
			format:     encodingJSON,
			wantErr:    true,
		},
		{
			name:       "missing output",
			inputFile:  "input.json",
			outputFile: "",
			format:     encodingJSON,
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
			format:     encodingYAML,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new FlagSet for this test
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			configureFlags(fs)

			// Save and restore global flag variables that configureFlags sets
			oldInput := inputFile
			oldOutput := outputFile
			oldFormat := encodingFormat
			oldShowVersion := showVersion
			oldShowHelp := showHelp

			defer func() {
				inputFile = oldInput
				outputFile = oldOutput
				encodingFormat = oldFormat
				showVersion = oldShowVersion
				showHelp = oldShowHelp
			}()

			inputFile = tt.inputFile
			outputFile = tt.outputFile
			encodingFormat = tt.format
			showVersion = false // Ensure showVersion is false for these tests
			showHelp = false    // Ensure showHelp is false for these tests

			err := validateFlags()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunVersionWithoutOutputFlag(t *testing.T) {
	// Mock usage to suppress output
	origUsage := flag.Usage
	defer func() { flag.Usage = origUsage }()
	flag.Usage = func() {}

	// Save and restore global flag variables
	oldShowVersion := showVersion
	oldInputFile := inputFile
	oldOutputFile := outputFile

	defer func() {
		showVersion = oldShowVersion
		inputFile = oldInputFile
		outputFile = oldOutputFile
	}()

	// Simulate command-line arguments for "apibconv -v"
	oldArgs := os.Args
	os.Args = []string{"apibconv", "-v"}
	defer func() { os.Args = oldArgs }()

	// Reset flags before calling Run to ensure a clean state for internal flag parsing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Call Run directly with a dummy version
	exitCode := Run("test-version")

	if exitCode != 0 {
		t.Errorf("Run() with -v flag and no output file exit code = %d, want 0", exitCode)
	}
}

func TestRunHelpFlag(t *testing.T) {
	// Mock usage to suppress output
	origUsage := flag.Usage
	defer func() { flag.Usage = origUsage }()
	flag.Usage = func() {}

	// Save and restore global flag variables that Run modifies
	oldShowHelp := showHelp
	oldInputFile := inputFile
	oldOutputFile := outputFile

	defer func() {
		showHelp = oldShowHelp
		inputFile = oldInputFile
		outputFile = oldOutputFile
	}()

	// Simulate command-line arguments for "apibconv --help"
	oldArgs := os.Args
	os.Args = []string{"apibconv", "--help"}
	defer func() { os.Args = oldArgs }()

	// Reset flags before calling Run to ensure a clean state for internal flag parsing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Call Run directly with a dummy version (version string is not relevant for help)
	exitCode := Run("test-version")

	if exitCode != 0 {
		t.Errorf("Run() with --help flag exit code = %d, want 0", exitCode)
	}
}

func TestRunValidateWithPositionalArg(t *testing.T) {
	// Mock usage to suppress output
	origUsage := flag.Usage
	defer func() { flag.Usage = origUsage }()
	flag.Usage = func() {}

	// Create a dummy input file (valid OpenAPI spec to ensure exit code 0)
	tmpfile := createTempFile(t, `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"},"paths":{}}`, ".json")
	defer func() { _ = os.Remove(tmpfile) }()

	// Save and restore global flag variables
	oldInputFile := inputFile
	oldValidateOnly := validateOnly

	defer func() {
		inputFile = oldInputFile
		validateOnly = oldValidateOnly
	}()

	// Simulate command-line arguments: apibconv <file> --validate
	oldArgs := os.Args
	os.Args = []string{"apibconv", tmpfile, "--validate"}
	defer func() { os.Args = oldArgs }()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Call Run
	exitCode := Run("test-version")

	if exitCode != 0 {
		t.Errorf("Run() with positional file and --validate failed, exit code = %d", exitCode)
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
			inputFormat: formatOpenAPI,
			expected:    formatAPIB,
		},
		{
			name:        "json from apib",
			outputFile:  "output.json",
			inputFormat: formatAPIB,
			expected:    formatOpenAPI,
		},
		{
			name:        "json from openapi",
			outputFile:  "output.json",
			inputFormat: formatOpenAPI,
			expected:    formatAPIB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldOutput := outputFile
			defer func() { outputFile = oldOutput }()

			outputFile = tt.outputFile
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

	oldInput := inputFile
	defer func() { inputFile = oldInput }()

	inputFile = tmpfile
	exitCode := runValidation(func() {})
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

	oldInput := inputFile
	defer func() { inputFile = oldInput }()

	inputFile = tmpfile
	exitCode := runValidation(func() {})
	if exitCode != 1 {
		t.Errorf("runValidation() for invalid spec = %d, want 1", exitCode)
	}
}

func TestRunValidationMissingFile(t *testing.T) {
	// Mock usage to suppress output
	origUsage := flag.Usage
	defer func() { flag.Usage = origUsage }()
	flag.Usage = func() {}

	oldInput := inputFile
	defer func() { inputFile = oldInput }()

	inputFile = ""
	exitCode := runValidation(func() {})
	if exitCode != 1 {
		t.Errorf("runValidation() with no input = %d, want 1", exitCode)
	}
}
