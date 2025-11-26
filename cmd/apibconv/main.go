// Package main provides a command-line interface for converting between OpenAPI 3.0/3.1, AsyncAPI 2.6/3.0, and API Blueprint specifications.
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
//	apibconv -f api.apib -o openapi.yaml --format yaml
//
//	# Validate a specification without conversion
//	apibconv --validate -f openapi.json
//
//	# Convert AsyncAPI to API Blueprint (auto-detects v2.6 or v3.0)
//	apibconv -f asyncapi.json -o api.apib
//
//	# Convert API Blueprint to AsyncAPI 2.6 (default)
//	apibconv -f api.apib -o asyncapi.json --output-format asyncapi --protocol ws
//
//	# Convert API Blueprint to AsyncAPI 3.0
//	apibconv -f api.apib -o asyncapi-v3.json --output-format asyncapi --asyncapi-version 3.0 --protocol kafka
//
//	# Show version
//	apibconv -version
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
//   - YAML: Human-readable YAML format (use --format yaml)
//
// # Flags
//
//	-f string
//	     Input specification file (OpenAPI JSON, AsyncAPI JSON, or API Blueprint)
//	-o string
//	     Output file (API Blueprint, OpenAPI JSON/YAML, or AsyncAPI JSON/YAML)
//	--openapi-version string
//	     OpenAPI version for output (3.0 or 3.1) when converting from API Blueprint (default "3.0")
//	--asyncapi-version string
//	     AsyncAPI version for output (2.6 or 3.0) when converting from API Blueprint (default "2.6")
//	--output-format string
//	     Output format: openapi, asyncapi, or apib (auto-detected if not specified)
//	--format string
//	     Encoding format: json or yaml (default "json")
//	--protocol string
//	     Protocol for AsyncAPI output (ws, mqtt, kafka, amqp, http) (default "ws")
//	--validate
//	     Validate the input specification without conversion
//	-version
//	     Print version information
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/amer8/apibconv/converter"
)

var (
	inputFile       = flag.String("f", "", "Input specification file (OpenAPI JSON, AsyncAPI JSON, or API Blueprint)")
	outputFile      = flag.String("o", "", "Output file (API Blueprint, OpenAPI JSON/YAML, or AsyncAPI JSON/YAML)")
	openapiVersion  = flag.String("openapi-version", "3.0", "OpenAPI version for output (3.0 or 3.1) when converting from API Blueprint")
	asyncapiVersion = flag.String("asyncapi-version", "2.6", "AsyncAPI version for output (2.6 or 3.0) when converting from API Blueprint")
	outputFormat    = flag.String("output-format", "", "Output format: openapi, asyncapi, or apib (auto-detected if not specified)")
	encodingFormat  = flag.String("format", "json", "Encoding format: json or yaml")
	protocol        = flag.String("protocol", "ws", "Protocol for AsyncAPI output (ws, mqtt, kafka, amqp, http)")
	validateOnly    = flag.Bool("validate", false, "Validate the input specification without conversion")
	showVersion     = flag.Bool("version", false, "Print version information")
)

// version is set by GoReleaser at build time via -ldflags
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	if *showVersion {
		fmt.Printf("apibconv version %s\n", version)
		return 0
	}

	// Validation mode
	if *validateOnly {
		return runValidation()
	}

	if err := validateFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		return 1
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

// runValidation validates the input specification without conversion
func runValidation() int {
	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input file (-f) is required for validation")
		flag.Usage()
		return 1
	}

	// #nosec G304 - filename is provided by user via CLI flag
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		return 1
	}

	result := converter.ValidateBytes(data)

	// Print validation result
	fmt.Printf("File: %s\n", *inputFile)
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
	if *inputFile == "" {
		return fmt.Errorf("input file (-f) is required")
	}
	if *outputFile == "" {
		return fmt.Errorf("output file (-o) is required")
	}

	// Validate encoding format
	if *encodingFormat != "json" && *encodingFormat != "yaml" {
		return fmt.Errorf("invalid encoding format '%s' (must be json or yaml)", *encodingFormat)
	}

	return nil
}

// determineFormats detects input format and determines output format
func determineFormats() (inputFmt, outputFmt string, err error) {
	inputFmt, err = detectInputFormat(*inputFile)
	if err != nil {
		return "", "", fmt.Errorf("detecting input format: %w", err)
	}

	outputFmt = *outputFormat
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
	case strings.HasSuffix(strings.ToLower(*outputFile), ".apib"):
		return "apib"
	case inputFormat == "apib":
		return "openapi"
	default:
		return "apib"
	}
}

// openFiles opens input and output files
func openFiles() (input, output *os.File, err error) {
	input, err = os.Open(*inputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("opening input file: %w", err)
	}

	output, err = os.Create(*outputFile)
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

	// Check file content for JSON-based formats
	// #nosec G304 - filename is provided by user via CLI flag, this is expected behavior for a file conversion tool
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Check for API Blueprint format header
		if strings.HasPrefix(line, "FORMAT:") || strings.HasPrefix(line, "# ") {
			return "apib", nil
		}
		// Check for AsyncAPI
		if strings.Contains(line, "\"asyncapi\"") {
			return "asyncapi", nil
		}
		// Check for OpenAPI
		if strings.Contains(line, "\"openapi\"") {
			return "openapi", nil
		}
		// If we see JSON start, try to parse first few fields
		if strings.HasPrefix(line, "{") {
			// Read a bit more to detect format
			content := line
			for i := 0; i < 10 && scanner.Scan(); i++ {
				content += scanner.Text()
			}
			if strings.Contains(content, "\"asyncapi\"") {
				return "asyncapi", nil
			}
			if strings.Contains(content, "\"openapi\"") {
				return "openapi", nil
			}
			// Default to OpenAPI if uncertain
			return "openapi", nil
		}
	}

	return "openapi", nil // Default
}

// convertAPIBlueprintToOpenAPI converts API Blueprint to OpenAPI
func convertAPIBlueprintToOpenAPI(input, output *os.File) int {
	// Parse and validate the OpenAPI version
	var targetVersion converter.Version
	switch *openapiVersion {
	case "3.0":
		targetVersion = converter.Version30
	case "3.1":
		targetVersion = converter.Version31
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid OpenAPI version '%s' (must be 3.0 or 3.1)\n", *openapiVersion)
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

	fmt.Printf("Successfully converted API Blueprint %s to OpenAPI %s %s\n", *inputFile, targetVersion.ToFullVersion(), *outputFile)
	return 0
}

// convertAPIBlueprintToAsyncAPI converts API Blueprint to AsyncAPI
func convertAPIBlueprintToAsyncAPI(input, output *os.File) int {
	// Determine target AsyncAPI version
	var targetVersion string
	switch *asyncapiVersion {
	case "2", "2.6", "2.6.0":
		targetVersion = "2.6"
		if err := converter.ConvertAPIBlueprintToAsyncAPI(input, output, *protocol); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting API Blueprint to AsyncAPI 2.6: %v\n", err)
			return 1
		}
	case "3", "3.0", "3.0.0":
		targetVersion = "3.0"
		if err := converter.ConvertAPIBlueprintToAsyncAPIV3(input, output, *protocol); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting API Blueprint to AsyncAPI 3.0: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid AsyncAPI version '%s' (must be 2.6 or 3.0)\n", *asyncapiVersion)
		return 1
	}

	fmt.Printf("Successfully converted API Blueprint %s to AsyncAPI %s %s (protocol: %s)\n", *inputFile, targetVersion, *outputFile, *protocol)
	return 0
}

// convertOpenAPIToAPIBlueprint converts OpenAPI to API Blueprint
func convertOpenAPIToAPIBlueprint(input, output *os.File) int {
	if err := converter.Convert(input, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting OpenAPI to API Blueprint: %v\n", err)
		return 1
	}
	fmt.Printf("Successfully converted OpenAPI %s to API Blueprint %s\n", *inputFile, *outputFile)
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

	// Detect AsyncAPI version
	var versionCheck struct {
		AsyncAPI string `json:"asyncapi"`
	}
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting AsyncAPI version: %v\n", err)
		return 1
	}

	asyncVer := converter.DetectAsyncAPIVersion(versionCheck.AsyncAPI)

	// Convert based on detected version
	var blueprint string
	switch asyncVer {
	case 2:
		spec, err := converter.ParseAsyncAPI(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing AsyncAPI 2.x: %v\n", err)
			return 1
		}
		blueprint = converter.AsyncAPIToAPIBlueprint(spec)
	case 3:
		spec, err := converter.ParseAsyncAPIV3(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing AsyncAPI 3.x: %v\n", err)
			return 1
		}
		blueprint = converter.AsyncAPIV3ToAPIBlueprint(spec)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported AsyncAPI version: %s\n", versionCheck.AsyncAPI)
		return 1
	}

	// Write output
	if _, err := output.WriteString(blueprint); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully converted AsyncAPI %s %s to API Blueprint %s\n", versionCheck.AsyncAPI, *inputFile, *outputFile)
	return 0
}

// writeSpec writes a specification to the output in the configured format (JSON or YAML)
func writeSpec(output *os.File, spec interface{}) error {
	if *encodingFormat == "yaml" {
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
