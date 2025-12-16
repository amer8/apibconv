// Package cli implements the command-line interface logic for apibconv.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/amer8/apibconv/pkg/converter"
)

// App is the main application struct.
type App struct {
	converter *converter.Converter
	runner    *Runner
}

// NewApp creates a new CLI application instance.
func NewApp(conv *converter.Converter) *App {
	return &App{
		converter: conv,
		runner:    NewRunner(conv),
	}
}

// Run executes the CLI application.
func (a *App) Run(ctx context.Context, args []string) error {
	// 1. Parse Flags
	flags, posArgs, err := ParseFlags(args)
	if err != nil {
		return err // Flag package already prints errors to stderr usually, but we bubble up
	}

	// 2. Create Config
	cfg, err := ConfigFromFlags(flags, posArgs)
	if err != nil {
		return err
	}

	// Check if we are waiting for stdin in a terminal environment
	if cfg.InputPath == "" && (cfg.Mode == ModeConvert || cfg.Mode == ModeValidate) {
		if isStdinTerminal() {
			return a.printUsage()
		}
	}

	// 3. Execute based on Mode
	switch cfg.Mode {
	case ModeHelp:
		return a.printUsage()
	case ModeVersion:
		return a.printVersion()
	case ModeValidate:
		return a.runner.Validate(ctx, cfg)
	case ModeConvert:
		return a.runner.Convert(ctx, cfg)
	default:
		return fmt.Errorf("unknown mode")
	}
}

func (a *App) printUsage() error {
	fmt.Println(`apibconv is a command-line tool for converting between API specification formats.

Supported formats:

  - OpenAPI 2.0 (Swagger)
  - OpenAPI 3.0.x
  - OpenAPI 3.1.x
  - API Blueprint
  - AsyncAPI 2.x, 3.0

Usage:

    apibconv [OPTIONS] [INPUT_FILE]

Arguments:

    INPUT_FILE
        Input specification file (OpenAPI, AsyncAPI, or API Blueprint)
        If omitted or "-", reads from stdin

Options:

    -o, --output FILE
        Output file path (required for conversion)
        Use "-" to write to stdout

    --to FORMAT
        Target format: openapi, asyncapi, apib
        Auto-detected from --output extension if not specified

    -e, --encoding FORMAT
        Output encoding: json, yaml (default: auto-detected from output extension)

    --validate
        Validate input without converting
        Performs format detection and structural validation

    -v, --version
        Print version information

    -h, --help
        Show this help message

AsyncAPI Options:

    --protocol PROTO
        Protocol: ws, wss, mqtt, kafka, amqp, http, https, auto (required)
        Required when converting to AsyncAPI format
        "auto" attempts to detect from input specification

    --asyncapi-version VERSION
        AsyncAPI version: 2.6, 3.0 (default: "2.6")
        Determines which AsyncAPI schema version to generate

OpenAPI Options:

    --openapi-version VERSION
        OpenAPI version: 3.0, 3.1 (default: "3.0")
        Determines which OpenAPI schema version to generate

Format Detection:

    Input format is automatically detected by analyzing file content:
    - OpenAPI: Checks for "openapi" or "swagger" version field
    - AsyncAPI: Checks for "asyncapi" version field
    - API Blueprint: Checks for Markdown structure and FORMAT directive

    Output format is determined in this order:
    1. --to flag (if specified)
    2. File extension of --output flag
    3. Error if cannot determine

Encoding Detection:

    Output encoding (JSON or YAML) is determined in this order:
    1. -e/--encoding flag (if specified)
    2. File extension of --output flag (.json → JSON, .yaml/.yml → YAML)
    3. Default to JSON

Examples:

    # Convert API Blueprint to OpenAPI JSON
    apibconv -o output.json spec.apib

    # Convert API Blueprint to AsyncAPI YAML with WebSocket protocol
    apibconv -o output.yaml --protocol ws spec.apib

    # Convert OpenAPI to specific version
    apibconv -o output.yaml --to openapi --openapi-version 3.1 spec.json

    # Read from stdin, write to stdout
    apibconv -o output.json --to openapi < spec.apib
    cat spec.apib | apibconv --to openapi > output.json

    # Validate a specification
    apibconv --validate spec.json
    cat spec.yaml | apibconv --validate

    # Specify encoding explicitly
    apibconv -o output.txt --to openapi -e json spec.apib

    # Convert to AsyncAPI with auto-detected protocol
    apibconv -o output.yaml --to asyncapi --protocol auto spec.json

Exit Codes:

    0    Success
    1    General error (invalid arguments, file not found, etc.)
    2    Validation error (invalid specification)
    3    Conversion error (format incompatibility)

Environment Variables:

    APIBCONV_DEFAULT_ENCODING
        Default encoding when not specified: json, yaml

    APIBCONV_ASYNCAPI_VERSION
        Default AsyncAPI version: 2.6, 3.0

    APIBCONV_OPENAPI_VERSION
        Default OpenAPI version: 3.0, 3.1

File Extensions:

    Input (auto-detected):
    - .json, .yaml, .yml    → Content-based detection
    - .apib, .md            → API Blueprint
    - Any other             → Content-based detection

    Output (determines format/encoding):
    - .json                 → JSON encoding
    - .yaml, .yml           → YAML encoding
    - .apib, .md            → API Blueprint format

Validation:

    The --validate flag performs basic structural validation:
    - Format detection and syntax validation
    - Basic path, schema type, and reference validation

    Validation output shows errors and warnings with:
    - Path to the problematic field
    - Description of the issue
    - Severity level (error/warning)

Conversion Notes:

    OpenAPI ↔ API Blueprint:
    - Basic bidirectional conversion support
    - Maps paths, operations, and basic schema structures including MSON attributes
    - API Blueprint groups are mapped to operation tags

    OpenAPI → AsyncAPI:
    - HTTP operations become publish/subscribe operations
    - Request/response become message schemas
    - Protocol derived from --protocol flag, environment variable, or server URL

    AsyncAPI → OpenAPI:
    - Publish/subscribe become POST/GET operations
    - Channel names become path segments
    - Message schemas become request/response bodies

    API Blueprint → AsyncAPI:
    - Similar to OpenAPI → AsyncAPI conversion
    - Resource groups map to channel groups
    - Actions map to operations

Protocol Specification (AsyncAPI):

    When converting to AsyncAPI, the protocol determines how the API
    operates. Common protocols:

    - ws/wss: WebSocket (real-time bidirectional)
    - mqtt: MQTT (pub/sub messaging)
    - kafka: Apache Kafka (distributed streaming)
    - amqp: Advanced Message Queuing Protocol
    - http/https: HTTP-based webhooks/polling

    The --protocol flag sets the default protocol for generated servers.
    "auto" detection is currently experimental.`)
	return nil
}

func (a *App) printVersion() error {
	fmt.Println("apibconv v1.0.0")
	return nil
}

// isStdinTerminal checks if stdin is a terminal
func isStdinTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
