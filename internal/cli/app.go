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
	version   string
}

// NewApp creates a new CLI application instance.
func NewApp(conv *converter.Converter, version string) *App {
	return &App{
		converter: conv,
		runner:    NewRunner(conv),
		version:   version,
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
	fmt.Println(`apibconv migrates API specifications to newer versions.

Supported conversions:
    API Blueprint -> OpenAPI 3.1
    OpenAPI 2.0 or 3.0 -> OpenAPI 3.1
    AsyncAPI 2.x -> AsyncAPI 3.0
    OpenAPI and AsyncAPI -> the same version (JSON/YAML)

Conversion can be lossy; retain and review the original specification.
Cross-format conversions other than API Blueprint to OpenAPI, and downgrades,
are not supported.

Usage: apibconv [OPTIONS] [INPUT_FILE]
    Omit INPUT_FILE or use - to read stdin.

Options:
    -o, --output FILE          Output file; - or omitted writes to stdout
    --from FORMAT              Input: apiblueprint, openapi, asyncapi
    --to FORMAT                Output: openapi, asyncapi
    -e, --encoding FORMAT      json or yaml (default: output extension)
    --openapi-version VERSION  Output version (default: 3.1)
    --asyncapi-version VERSION Output version (default: 3.0)
    --protocol PROTO           Default AsyncAPI server protocol
    --validate                 Validate input without converting
    --verbose                  Report progress
    -v, --version              Print version
    -h, --help                 Show help

JSON/YAML output defaults to the input format, or OpenAPI for API Blueprint.
Use --to when writing to stdout. Output versions other than the default are
accepted only for same-version OpenAPI or AsyncAPI conversion.

Environment variables: APIBCONV_DEFAULT_ENCODING,
APIBCONV_OPENAPI_VERSION, APIBCONV_ASYNCAPI_VERSION.

Examples:
    apibconv -o openapi.yaml api.apib
    apibconv -o openapi.json swagger.yaml
    apibconv --to asyncapi -o asyncapi3.yaml asyncapi2.yaml
    apibconv --to openapi --openapi-version 3.0 -o output.json input.yaml
    apibconv --validate spec.yaml`)
	return nil
}

func (a *App) printVersion() error {
	fmt.Printf("apibconv %s\n", a.version)
	return nil
}

// isStdinTerminal checks if stdin is a terminal
func isStdinTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
