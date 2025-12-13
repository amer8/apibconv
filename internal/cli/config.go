package cli

import (
	"os"

	"github.com/amer8/apibconv/pkg/format"
)

// Config holds the runtime configuration for the application
type Config struct {
	InputPath      string
	OutputPath     string
	InputFormat    format.Format
	OutputFormat   format.Format
	Encoding       string
	Verbose        bool
	Mode           Mode
	Protocol       string
	AsyncAPIVersion string
	OpenAPIVersion  string
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

// ConfigFromFlags creates a Config object from parsed flags and positional arguments
func ConfigFromFlags(f *Flags, args []string) (*Config, error) {
	cfg := &Config{
		OutputPath:      f.Output,
		Encoding:        f.Encoding,
		Verbose:         f.Verbose,
		Protocol:        f.Protocol,
		AsyncAPIVersion: f.AsyncAPIVersion,
		OpenAPIVersion:  f.OpenAPIVersion,
		Mode:            ModeConvert, // Default
	}

	// Determine Mode
	if f.Help {
		cfg.Mode = ModeHelp
		return cfg, nil
	}
	if f.Version {
		cfg.Mode = ModeVersion
		return cfg, nil
	}
	if f.Validate {
		cfg.Mode = ModeValidate
	}

	// Parse formats
	if f.InputFormat != "" {
		cfg.InputFormat = format.Format(f.InputFormat)
	}
	if f.OutputFormat != "" {
		cfg.OutputFormat = format.Format(f.OutputFormat)
	}

	// Handle positional arguments (Input file)
	if len(args) > 0 {
		cfg.InputPath = args[0]
	}

	// Apply Environment Variable Defaults if flags are not set
	if cfg.Encoding == "" {
		if envVal := os.Getenv("APIBCONV_DEFAULT_ENCODING"); envVal != "" {
			cfg.Encoding = envVal
		}
	}

	if cfg.AsyncAPIVersion == "2.6" { // Default set in flags, check if it was explicitly changed
		if envVal := os.Getenv("APIBCONV_ASYNCAPI_VERSION"); envVal != "" {
			cfg.AsyncAPIVersion = envVal
		}
	}

	if cfg.OpenAPIVersion == "3.0" { // Default set in flags, check if it was explicitly changed
		if envVal := os.Getenv("APIBCONV_OPENAPI_VERSION"); envVal != "" {
			cfg.OpenAPIVersion = envVal
		}
	}

	return cfg, nil
}