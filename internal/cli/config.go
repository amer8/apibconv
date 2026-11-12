package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amer8/apibconv/pkg/format"
)

// Config holds the CLI configuration.
type Config struct {
	InputPath      string
	OutputPath     string
	InputFormat    format.Format
	OutputFormat   format.Format
	Encoding       string
	Verbose        bool
	ValidateOutput bool
	Report         string
	Mode           Mode
	OpenAPIVersion string
}

// Mode represents the operation mode of the application.
type Mode int

const (
	// ModeConvert indicates that the application should perform conversion.
	ModeConvert Mode = iota
	// ModeValidate indicates that the application should perform validation.
	ModeValidate
	// ModeInfo indicates that the application should display information (not used yet).
	ModeInfo
	// ModeVersion indicates that the application should print version information.
	ModeVersion
	// ModeHelp indicates that the application should print help information.
	ModeHelp
)

// ConfigFromFlags builds a configuration from flags and positional arguments.
func ConfigFromFlags(f *Flags, args []string) (*Config, error) {
	cfg := &Config{
		OutputPath:     f.Output,
		Encoding:       f.Encoding,
		Verbose:        f.Verbose,
		ValidateOutput: f.ValidateOutput,
		Report:         f.Report,
		OpenAPIVersion: f.OpenAPIVersion,
		Mode:           ModeConvert,
	}

	if f.Help {
		cfg.Mode = ModeHelp
		return cfg, nil
	}
	if f.Version {
		cfg.Mode = ModeVersion
		return cfg, nil
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("expected at most one input file, got %d", len(args))
	}
	if f.Validate {
		cfg.Mode = ModeValidate
		for _, opt := range []struct {
			name string
			set  bool
		}{
			{"--output", f.Output != "" || f.provided["o"] || f.provided["output"]},
			{"--to", f.OutputFormat != "" || f.provided["to"]},
			{"--encoding", f.Encoding != "" || f.provided["e"] || f.provided["encoding"]},
			{"--openapi-version", f.OpenAPIVersion != "" || f.provided["openapi-version"]},
			{"--validate-output", f.ValidateOutput || f.provided["validate-output"]},
		} {
			if opt.set {
				return nil, fmt.Errorf("%s cannot be used with --validate", opt.name)
			}
		}
	}
	if !f.Validate && (f.Report != "" || f.provided["report"]) {
		return nil, fmt.Errorf("--report requires --validate")
	}
	if cfg.Report == "" {
		cfg.Report = "text"
	}
	if cfg.Report != "text" && cfg.Report != "json" {
		return nil, fmt.Errorf("invalid report format %q: expected text or json", cfg.Report)
	}

	if f.InputFormat != "" {
		cfg.InputFormat = format.Format(f.InputFormat)
	}
	if f.OutputFormat != "" {
		cfg.OutputFormat = format.Format(f.OutputFormat)
	}

	if len(args) > 0 {
		cfg.InputPath = args[0]
	}
	if cfg.Mode == ModeValidate {
		return cfg, nil
	}

	if cfg.OutputFormat == format.FormatAPIBlueprint {
		return nil, fmt.Errorf("unsupported output format %q: CLI conversion supports only openapi", cfg.OutputFormat)
	}
	if cfg.OutputFormat == "" {
		switch strings.ToLower(filepath.Ext(cfg.OutputPath)) {
		case ".apib", ".md":
			return nil, fmt.Errorf("output file extension %q implies unsupported API Blueprint output; use an OpenAPI output file or --to openapi", filepath.Ext(cfg.OutputPath))
		}
	}

	if cfg.Encoding == "" {
		if envVal := os.Getenv("APIBCONV_DEFAULT_ENCODING"); envVal != "" {
			cfg.Encoding = envVal
		}
	}

	if cfg.OpenAPIVersion == "" {
		cfg.OpenAPIVersion = os.Getenv("APIBCONV_OPENAPI_VERSION")
		if cfg.OpenAPIVersion == "" {
			cfg.OpenAPIVersion = "3.1"
		}
	}
	if cfg.Encoding != "" && cfg.Encoding != "json" && cfg.Encoding != "yaml" {
		return nil, fmt.Errorf("invalid output encoding %q: expected json or yaml", cfg.Encoding)
	}
	switch cfg.OpenAPIVersion {
	case "2.0", "2.0.0", "3.0", "3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1", "3.1.0":
	default:
		return nil, fmt.Errorf("invalid OpenAPI output version %q: supported versions are 2.0, 3.0 through 3.0.3, and 3.1", cfg.OpenAPIVersion)
	}

	return cfg, nil
}
