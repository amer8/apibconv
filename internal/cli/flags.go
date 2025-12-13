package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/amer8/apibconv/pkg/format"
)

// Flags represents the parsed command-line flags
type Flags struct {
	Output         string
	InputFormat    string
	OutputFormat   string
	Encoding       string
	Validate       bool
	Verbose        bool
	Version        bool
	Help           bool
	Protocol       string // For AsyncAPI
	AsyncAPIVersion string
	OpenAPIVersion  string
}

// ParseFlags parses the command line arguments
func ParseFlags(args []string) (*Flags, []string, error) {
	f := &Flags{}
	fs := flag.NewFlagSet("apibconv", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&f.Output, "o", "", "Output file path (shorthand)")
	fs.StringVar(&f.Output, "output", "", "Output file path")
	
	fs.StringVar(&f.InputFormat, "from", "", "Input format (openapi, apiblueprint, asyncapi)")
	fs.StringVar(&f.OutputFormat, "to", "", "Target format (openapi, apiblueprint, asyncapi)")
	
	fs.StringVar(&f.Encoding, "e", "", "Output encoding (json, yaml) (shorthand)")	
	fs.StringVar(&f.Encoding, "encoding", "", "Output encoding (json, yaml)")
	
	fs.BoolVar(&f.Validate, "validate", false, "Validate input without converting")
	fs.BoolVar(&f.Verbose, "verbose", false, "Enable verbose output (including progress)")
	fs.BoolVar(&f.Version, "v", false, "Print version information (shorthand)")
	fs.BoolVar(&f.Version, "version", false, "Print version information")
	fs.BoolVar(&f.Help, "h", false, "Show help message (shorthand)")
	fs.BoolVar(&f.Help, "help", false, "Show help message")

	// AsyncAPI specific
	fs.StringVar(&f.Protocol, "protocol", "", "Protocol for AsyncAPI (ws, kafka, etc.) or 'auto'")
	fs.StringVar(&f.AsyncAPIVersion, "asyncapi-version", "2.6", "AsyncAPI version to generate")

	// OpenAPI specific
	fs.StringVar(&f.OpenAPIVersion, "openapi-version", "3.0", "OpenAPI version to generate")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	// Validate formats if provided
	if f.InputFormat != "" && !isValidFormat(f.InputFormat) {
		return nil, nil, fmt.Errorf("invalid input format: %s", f.InputFormat)
	}
	if f.OutputFormat != "" && !isValidFormat(f.OutputFormat) {
		return nil, nil, fmt.Errorf("invalid output format: %s", f.OutputFormat)
	}

	return f, fs.Args(), nil
}

func isValidFormat(f string) bool {
	switch format.Format(f) {
	case format.FormatOpenAPI, format.FormatAPIBlueprint, format.FormatAsyncAPI:
		return true
	default:
		return false
	}
}