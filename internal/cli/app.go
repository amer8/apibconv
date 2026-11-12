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
    OpenAPI -> the same version (JSON/YAML)

Conversion can be lossy; retain and review the original specification.
Cross-format conversions other than API Blueprint to OpenAPI, and downgrades,
are not supported.

Usage: apibconv [OPTIONS] [INPUT_FILE]
    Omit INPUT_FILE or use - to read stdin.
    Options may appear before or after INPUT_FILE; use -- before a filename starting with -.

Options:
    -o, --output FILE          Output file; - or omitted writes to stdout
    --from FORMAT              Input: apiblueprint, openapi
    --to FORMAT                Output: openapi
    -e, --encoding FORMAT      json or yaml
    --openapi-version VERSION  Output version (default: 3.1)
    --validate                 Validate input without converting
    --validate-output          Validate generated OpenAPI before writing it
    --report FORMAT            Validation report: text or json (requires --validate)
    --verbose                  Report progress
    -v, --version              Print version
    -h, --help                 Show help

JSON/YAML output defaults to the input format, or OpenAPI for API Blueprint.
Use --to when writing to stdout. Output versions other than the default are
accepted only for same-version OpenAPI conversion.

Environment variables: APIBCONV_DEFAULT_ENCODING, APIBCONV_OPENAPI_VERSION.
Precedence: explicit option, then environment variable, then default.
Encoding defaults to the output extension, or YAML when no encoding is inferred.
--validate-output checks the generated document against its OpenAPI schema and
the built-in model rules; it does not guarantee a lossless conversion or a
non-empty info.version.

Examples:
    apibconv -o openapi.yaml api.apib
    apibconv -o openapi.json swagger.yaml
    apibconv --to openapi --openapi-version 3.0 -o output.json input.yaml
    apibconv --to openapi -e json < api.apib > openapi.json
    apibconv --validate --report json spec.yaml
    apibconv --validate-output -o openapi.yaml api.apib`)
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
