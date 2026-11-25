package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	inputfs "github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/pkg/format"
)

// Flags holds the parsed command-line flags.
type Flags struct {
	MaxInputBytes  int64
	Output         string
	InputFormat    string
	OutputFormat   string
	Encoding       string
	Validate       bool
	ValidateOutput bool
	Report         string
	Verbose        bool
	Version        bool
	Licenses       bool
	Help           bool
	OpenAPIVersion string
	APIInfoVersion string
	provided       map[string]bool
}

// ParseFlags parses the command-line arguments.
func ParseFlags(args []string) (*Flags, []string, error) {
	f := &Flags{}
	fs := flag.NewFlagSet("apibconv", flag.ContinueOnError)
	// main formats the returned diagnostic. The flag package would also echo
	// the untrusted argument verbatim before printing its usage.
	fs.SetOutput(io.Discard)

	fs.StringVar(&f.Output, "o", "", "Output file path (shorthand)")
	fs.StringVar(&f.Output, "output", "", "Output file path")

	fs.StringVar(&f.InputFormat, "from", "", "Input format (openapi, apiblueprint)")
	fs.StringVar(&f.OutputFormat, "to", "", "Target format (openapi)")

	fs.StringVar(&f.Encoding, "e", "", "Output encoding (json, yaml) (shorthand)")
	fs.StringVar(&f.Encoding, "encoding", "", "Output encoding (json, yaml)")

	fs.Int64Var(&f.MaxInputBytes, "max-input-bytes", inputfs.DefaultMaxInputBytes, "Maximum input document size in bytes")
	fs.BoolVar(&f.Validate, "validate", false, "Validate input without converting")
	fs.BoolVar(&f.ValidateOutput, "validate-output", true, "Validate generated output before writing it (default true)")
	fs.StringVar(&f.Report, "report", "", "Validation report format (text, json; requires --validate)")
	fs.BoolVar(&f.Verbose, "verbose", false, "Enable verbose output (including progress)")
	fs.BoolVar(&f.Version, "v", false, "Print version information (shorthand)")
	fs.BoolVar(&f.Version, "version", false, "Print version information")
	fs.BoolVar(&f.Licenses, "licenses", false, "Print third-party licenses and notices")
	fs.BoolVar(&f.Help, "h", false, "Show help message (shorthand)")
	fs.BoolVar(&f.Help, "help", false, "Show help message")

	fs.StringVar(&f.OpenAPIVersion, "openapi-version", "", "OpenAPI version to generate")
	fs.StringVar(&f.APIInfoVersion, "api-version", "", "Version of the API described by the output")

	// The standard flag parser stops at the first positional argument. Separate
	// options from positional arguments while preserving flag values and --.
	var options, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}
		options = append(options, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		opt := fs.Lookup(name)
		if opt == nil || hasValue {
			continue
		}
		if b, ok := opt.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			options = append(options, args[i])
		}
	}
	if err := fs.Parse(options); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return nil, nil, err
	}
	f.provided = make(map[string]bool)
	fs.Visit(func(opt *flag.Flag) { f.provided[opt.Name] = true })

	if f.InputFormat != "" && !format.Format(f.InputFormat).IsValid() {
		return nil, nil, fmt.Errorf("invalid input format: %s", f.InputFormat)
	}
	if f.OutputFormat != "" && !format.Format(f.OutputFormat).IsValid() {
		return nil, nil, fmt.Errorf("invalid output format: %s", f.OutputFormat)
	}

	return f, positional, nil
}
