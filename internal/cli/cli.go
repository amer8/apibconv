// Package cli provides the command-line interface logic for converting between OpenAPI 3.0/3.1, AsyncAPI 2.6/3.0, and API Blueprint specifications.
//
// # Usage
//
// The tool automatically detects the input format based on file extension and content,
// then converts to the appropriate output format:
//
//	# Convert OpenAPI to API Blueprint
//	apibconv -f openapi.json -o api.apib
//
//	# Convert API Blueprint to OpenAPI (default: 3.0)
//	apibconv -f api.apib -o openapi.json
//
//	# Convert API Blueprint to OpenAPI 3.1
//	apibconv -f api.apib -o openapi.json --openapi-version 3.1
//
//	# Output as YAML instead of JSON
//	apibconv -f api.apib -o openapi.yaml -e yaml
//
//	# Validate a specification without conversion
//	apibconv --validate -f openapi.json
//
//	# Convert AsyncAPI to API Blueprint (auto-detects v2.6 or v3.0)
//	apibconv -f asyncapi.json -o api.apib
//
//	# Convert API Blueprint to AsyncAPI 2.6 (default)
//	apibconv -f api.apib -o asyncapi.json --to asyncapi --protocol ws
//
//	# Convert API Blueprint to AsyncAPI 3.0
//	apibconv -f api.apib -o asyncapi-v3.json --to asyncapi --asyncapi-version 3.0 --protocol kafka
//
//	# Show version
//	apibconv -v
//
// # Format Detection
//
// The input format is automatically detected by:
//   - File extension (.apib for API Blueprint, .json/.yaml for OpenAPI/AsyncAPI)
//   - Content inspection (looks for "FORMAT:" or "#" for API Blueprint, "openapi" or "asyncapi" fields in JSON)
//   - AsyncAPI version detection (2.x or 3.x)
//
// # Output Formats
//
// For JSON-based specifications (OpenAPI, AsyncAPI), output can be:
//   - JSON (default): Pretty-printed with 2-space indentation
//   - YAML: Human-readable YAML format (use -e yaml)
//
// # Flags
//
//	-f, --file string
//	     Input specification file
//	-o, --output string
//	     Output file path
//	-e, --encoding string
//	     Output encoding: json, yaml (default "json")
//	--to string
//	     Target specification format: openapi, asyncapi, apib
//	     (auto-detected from output file extension if not specified)
//	--openapi-version string
//	     OpenAPI target version: 3.0, 3.1 (default "3.0")
//	--asyncapi-version string
//	     AsyncAPI target version: 2.6, 3.0 (default "2.6")
//	--protocol string
//	     Protocol for AsyncAPI: ws, mqtt, kafka, amqp, http (default "ws")
//	--validate
//	     Validate input without converting
//	-v, --version
//	     Print version information
//	-h, --help
//	     Show this help message
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/amer8/apibconv/converter"
)

var (
	inputFile       string
	outputFile      string
	openapiVersion  = "3.0"
	asyncapiVersion = "2.6"
	outputFormat    string
	encodingFormat  string
	protocol        string
	validateOnly    bool
	showVersion     bool
	showHelp        bool
)

// Run is the entry point for the CLI logic.
// It accepts the version string (usually set by linker flags in main package).
func Run(version string) int {
	configureFlags()
	flag.Parse()

	if showHelp {
		flag.Usage()
		return 0
	}

	if showVersion {
		fmt.Printf("apibconv version %s\n", version)
		return 0
	}

	// Handle positional arguments
	args := flag.Args()
	if len(args) > 0 {
		inputFile = args[0]
	}

	// Handle stdin if no input file specified
	if inputFile == "" {
		// If stdin has data, read it to a temp file
		// Check if we are reading from pipe
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Read from stdin to temp file
			tmpFile, err := os.CreateTemp("", "apibconv-stdin-*")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating temp file for stdin: %v\n", err)
				return 1
			}
			defer func() {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
			}()

			if _, err := io.Copy(tmpFile, os.Stdin); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				return 1
			}
			inputFile = tmpFile.Name()
		}
	}

	// Validation mode
	if validateOnly {
		return runValidation()
	}

	if err := validateFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		return 1
	}

	// Auto-detect encoding if not specified
	if encodingFormat == "" {
		if strings.HasSuffix(strings.ToLower(outputFile), ".yaml") || strings.HasSuffix(strings.ToLower(outputFile), ".yml") {
			encodingFormat = "yaml"
		} else {
			encodingFormat = "json"
		}
	}

	inputFormatType, outputFormatType, err := determineFormats()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	input, output, err := openFiles()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		_ = output.Close()
		_ = input.Close()
	}()

	return performConversion(input, output, inputFormatType, outputFormatType)
}

func configureFlags() {
	// Reset flag.CommandLine to avoid double registration in tests or if Run is called multiple times
	// although for this refactoring it's not strictly necessary, it's safer.
	// However, we can't easily reset flag.CommandLine without assigning a NewFlagSet.
	// But standard flag usage relies on init() or main() running once.
	// We'll assume Run is called once.
	
	// We need to check if flags are already defined to avoid panic "flag redefined"
	if flag.Lookup("f") != nil {
		return
	}

	flag.StringVar(&inputFile, "f", "", "Input specification file")
	flag.StringVar(&inputFile, "file", "", "Input specification file")

	flag.StringVar(&outputFile, "o", "", "Output file path")
	flag.StringVar(&outputFile, "output", "", "Output file path")

	flag.StringVar(&encodingFormat, "e", "", "Output encoding: json, yaml")
	flag.StringVar(&encodingFormat, "encoding", "", "Output encoding: json, yaml")

	flag.StringVar(&outputFormat, "to", "", "Target specification format: openapi, asyncapi, apib")

	flag.StringVar(&openapiVersion, "openapi-version", "3.0", "OpenAPI target version: 3.0, 3.1")

	flag.StringVar(&asyncapiVersion, "asyncapi-version", "2.6", "AsyncAPI target version: 2.6, 3.0")

	flag.StringVar(&protocol, "protocol", "", "Protocol for AsyncAPI: ws, mqtt, kafka, amqp, http")

	flag.BoolVar(&validateOnly, "validate", false, "Validate input without converting")

	flag.BoolVar(&showVersion, "v", false, "Print version information")
	flag.BoolVar(&showVersion, "version", false, "Print version information")

	flag.BoolVar(&showHelp, "h", false, "Show this help message")
	flag.BoolVar(&showHelp, "help", false, "Show this help message")

	flag.Usage = func() {
		w := os.Stderr

		p := func(s string) {
			_, _ = fmt.Fprintln(w, s)
		}

		p("Usage: apibconv [INPUT_FILE] [OPTIONS]")
		p("")
		p("Arguments:")
		p("  INPUT_FILE")
		p("      Input specification file (OpenAPI, AsyncAPI, or API Blueprint)")
		p("")
		p("Options:")
		p("  -o, --output FILE")
		p("      Output file path (required for conversion)")
		p("  ")
		p("  --to FORMAT")
		p("      Target format: openapi, asyncapi, apib")
		p("      Auto-detected from --output extension if not specified")
		p("  ")
		p("  -e, --encoding FORMAT")
		p("      Encoding: json, yaml (default: auto-detect from output extension)")
		p("  ")
		p("  --validate")
		p("      Validate input without converting")
		p("  ")
		p("  -v, --version")
		p("      Print version information")
		p("  ")
		p("  -h, --help")
		p("      Show this help message")
		p("")
		p("AsyncAPI Options:")
		p("  --protocol PROTO")
		p("      Protocol: ws, mqtt, kafka, amqp, http (required)")
		p("  ")
		p("  --asyncapi-version VERSION")
		p("      Version: 2.6, 3.0 (default: \"2.6\")")
		p("")
		p("OpenAPI Options:")
		p("  --openapi-version VERSION")
		p("      Version: 3.0, 3.1 (default: \"3.0\")")
		p("")
		p("Examples:")
		p("  apibconv api.apib -o openapi.json")
		p("  apibconv api.apib -o asyncapi.yaml --protocol ws")
		p("  apibconv -o openapi.json --to openapi --openapi-version 3.1 < api.apib")
		p("  apibconv openapi.json --validate")
	}
}

// runValidation validates the input specification without conversion
func runValidation() int {
	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input file is required for validation (provide as argument or pipe via stdin)")
		flag.Usage()
		return 1
	}

	// #nosec G304 - filename is provided by user via CLI flag
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		return 1
	}

	result := converter.ValidateBytes(data)

	// Print validation result
	fmt.Printf("File: %s\n", inputFile)
	fmt.Printf("Format: %s\n", result.Format)
	if result.Version != "" {
		fmt.Printf("Version: %s\n", result.Version)
	}
	fmt.Println()

	if result.Valid {
		fmt.Println("Status: VALID")
	} else {
		fmt.Println("Status: INVALID")
	}

	// Print errors
	if result.Errors != nil && result.Errors.HasErrors() {
		fmt.Printf("\nErrors (%d):\n", result.Errors.Count())
		for _, err := range result.Errors.Errors {
			if err.Field != "" {
				fmt.Printf("  - %s: %s\n", err.Field, err.Message)
			} else {
				fmt.Printf("  - %s\n", err.Message)
			}
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	if !result.Valid {
		return 1
	}
	return 0
}

// validateFlags validates required command-line flags
func validateFlags() error {
	if inputFile == "" {
		return fmt.Errorf("input file is required (provide as argument or pipe via stdin)")
	}
	if outputFile == "" {
		return fmt.Errorf("output file (-o) is required")
	}

	// Validate encoding format if specified
	if encodingFormat != "" && encodingFormat != "json" && encodingFormat != "yaml" {
		return fmt.Errorf("invalid encoding format '%s' (must be json or yaml)", encodingFormat)
	}

	return nil
}

// determineFormats detects input format and determines output format
func determineFormats() (inputFmt, outputFmt string, err error) {
	inputFmt, err = detectInputFormat(inputFile)
	if err != nil {
		return "", "", fmt.Errorf("detecting input format: %w", err)
	}

	outputFmt = outputFormat
	if outputFmt == "" {
		outputFmt = autoDetectOutputFormat(inputFmt)
	}

	if outputFmt != "openapi" && outputFmt != "asyncapi" && outputFmt != "apib" {
		return "", "", fmt.Errorf("invalid output format '%s' (must be openapi, asyncapi, or apib)", outputFmt)
	}

	return inputFmt, outputFmt, nil
}

// autoDetectOutputFormat determines output format based on file extension and input format
func autoDetectOutputFormat(inputFormat string) string {
	switch {
	case strings.HasSuffix(strings.ToLower(outputFile), ".apib"):
		return "apib"
	case inputFormat == "apib":
		return "openapi"
	default:
		return "apib"
	}
}

// openFiles opens input and output files
func openFiles() (input, output *os.File, err error) {
	// #nosec G304 - filename is provided by user via CLI flag
	input, err = os.Open(inputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("opening input file: %w", err)
	}

	// #nosec G304 - filename is provided by user via CLI flag
	output, err = os.Create(outputFile)
	if err != nil {
		_ = input.Close()
		return nil, nil, fmt.Errorf("creating output file: %w", err)
	}

	return input, output, nil
}

// performConversion performs the actual conversion based on formats
func performConversion(input, output *os.File, inputFormat, outputFormat string) int {
	switch {
	case inputFormat == "apib" && outputFormat == "openapi":
		return convertAPIBlueprintToOpenAPI(input, output)
	case inputFormat == "apib" && outputFormat == "asyncapi":
		return convertAPIBlueprintToAsyncAPI(input, output)
	case inputFormat == "openapi" && outputFormat == "apib":
		return convertOpenAPIToAPIBlueprint(input, output)
	case inputFormat == "asyncapi" && outputFormat == "apib":
		return convertAsyncAPIToAPIBlueprint(input, output)
	case inputFormat == "asyncapi" && outputFormat == "openapi":
		fmt.Fprintln(os.Stderr, "Error: AsyncAPI to OpenAPI conversion is not directly supported. Convert to API Blueprint first.")
		return 1
	case inputFormat == "openapi" && outputFormat == "asyncapi":
		fmt.Fprintln(os.Stderr, "Error: OpenAPI to AsyncAPI conversion is not directly supported. Convert to API Blueprint first.")
		return 1
	default:
		fmt.Fprintf(os.Stderr, "Error: conversion from %s to %s is not supported or input equals output\n", inputFormat, outputFormat)
		return 1
	}
}

// detectInputFormat detects the input file format: "apib", "openapi", or "asyncapi"
func detectInputFormat(filename string) (string, error) {
	// Check file extension
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".apib") {
		return "apib", nil
	}

	// Check file content for JSON/YAML-based formats
	// #nosec G304 - filename is provided by user via CLI flag, this is expected behavior for a file conversion tool
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	// Read first 1KB to detect format
	// This is more robust than bufio.Scanner for minified files which might be one huge line
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	content := string(buf[:n])

	// Check for API Blueprint signatures
	// We check line-based signatures within the buffer
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "FORMAT:") || strings.HasPrefix(line, "# ") {
			return "apib", nil
		}
		// If we see a JSON start object, stop checking for APIB line headers to avoid false positives in JSON strings
		if strings.HasPrefix(line, "{") {
			break
		}
	}

	// Check for AsyncAPI (JSON or YAML)
	if strings.Contains(content, "\"asyncapi\"") || strings.Contains(content, "asyncapi:") {
		return "asyncapi", nil
	}
	// Check for OpenAPI (JSON or YAML)
	if strings.Contains(content, "\"openapi\"") || strings.Contains(content, "openapi:") {
		return "openapi", nil
	}

	return "openapi", nil // Default
}

// convertAPIBlueprintToOpenAPI converts API Blueprint to OpenAPI
func convertAPIBlueprintToOpenAPI(input, output *os.File) int {
	// Parse and validate the OpenAPI version
	var targetVersion converter.Version
	switch openapiVersion {
	case "3.0":
		targetVersion = converter.Version30
	case "3.1":
		targetVersion = converter.Version31
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid OpenAPI version '%s' (must be 3.0 or 3.1)\n", openapiVersion)
		return 1
	}

	// Create conversion options
	opts := &converter.ConversionOptions{
		OutputVersion: targetVersion,
	}

	// Parse API Blueprint with options
	spec, err := converter.ParseAPIBlueprintReader(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing API Blueprint: %v\n", err)
		return 1
	}

	// Convert to the target version if needed
	spec, err = converter.ConvertToVersion(spec, targetVersion, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to OpenAPI %s: %v\n", targetVersion.ToFullVersion(), err)
		return 1
	}

	// Write output based on encoding format
	if err := writeSpec(output, spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing OpenAPI spec: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully converted API Blueprint %s to OpenAPI %s %s\n", inputFile, targetVersion.ToFullVersion(), outputFile)
	return 0
}

// convertAPIBlueprintToAsyncAPI converts API Blueprint to AsyncAPI
func convertAPIBlueprintToAsyncAPI(input, output *os.File) int {
	// Protocol is required for AsyncAPI
	if protocol == "" {
		fmt.Fprintln(os.Stderr, "Error: protocol is required for AsyncAPI conversion (use --protocol)")
		return 1
	}

	// Determine target AsyncAPI version
	var targetVersion string
	switch asyncapiVersion {
	case "2", "2.6", "2.6.0":
		targetVersion = "2.6"
		if err := converter.ConvertAPIBlueprintToAsyncAPI(input, output, protocol); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting API Blueprint to AsyncAPI 2.6: %v\n", err)
			return 1
		}
	case "3", "3.0", "3.0.0":
		targetVersion = "3.0"
		if err := converter.ConvertAPIBlueprintToAsyncAPIV3(input, output, protocol); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting API Blueprint to AsyncAPI 3.0: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid AsyncAPI version '%s' (must be 2.6 or 3.0)\n", asyncapiVersion)
		return 1
	}

	fmt.Printf("Successfully converted API Blueprint %s to AsyncAPI %s %s (protocol: %s)\n", inputFile, targetVersion, outputFile, protocol)
	return 0
}

// convertOpenAPIToAPIBlueprint converts OpenAPI to API Blueprint
func convertOpenAPIToAPIBlueprint(input, output *os.File) int {
	if err := converter.Convert(input, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting OpenAPI to API Blueprint: %v\n", err)
		return 1
	}
	fmt.Printf("Successfully converted OpenAPI %s to API Blueprint %s\n", inputFile, outputFile)
	return 0
}

// convertAsyncAPIToAPIBlueprint converts AsyncAPI to API Blueprint
func convertAsyncAPIToAPIBlueprint(input, output *os.File) int {
	// Read the input to detect version
	data, err := io.ReadAll(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		return 1
	}

	// Use ParseAsyncAPIAny which handles JSON and YAML detection and versioning
	spec, ver, err := converter.ParseAsyncAPIAny(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AsyncAPI: %v\n", err)
		return 1
	}

	// Convert based on detected version
	var blueprint string
	switch ver {
	case 2:
		blueprint = converter.AsyncAPIToAPIBlueprint(spec.(*converter.AsyncAPI))
	case 3:
		blueprint = converter.AsyncAPIV3ToAPIBlueprint(spec.(*converter.AsyncAPIV3))
	default:
		// Should be caught by ParseAsyncAPIAny, but safety first
		fmt.Fprintf(os.Stderr, "Error: unsupported AsyncAPI version\n")
		return 1
	}

	// Write output
	if _, err := output.WriteString(blueprint); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		return 1
	}

	// Get AsyncAPI version string for logging
	asyncVerStr := "2.6.0"
	if ver == 3 {
		asyncVerStr = "3.0.0"
	}

	fmt.Printf("Successfully converted AsyncAPI %s %s to API Blueprint %s\n", asyncVerStr, inputFile, outputFile)
	return 0
}

// writeSpec writes a specification to the output in the configured format (JSON or YAML)
func writeSpec(output *os.File, spec interface{}) error {
	if encodingFormat == "yaml" {
		yamlBytes, err := converter.MarshalYAML(spec)
		if err != nil {
			return err
		}
		_, err = output.Write(yamlBytes)
		return err
	}

	// Default to JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(spec)
}
