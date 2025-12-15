package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
)

// Runner manages the execution of conversion and validation tasks.
type Runner struct {
	conv *converter.Converter
}

// NewRunner creates a new Runner instance.
func NewRunner(conv *converter.Converter) *Runner {
	return &Runner{conv: conv}
}

// Convert executes the conversion process based on the configuration.
func (r *Runner) Convert(ctx context.Context, cfg *Config) error {
	// 1. Prepare Input
	var input io.Reader

	if cfg.InputPath == "" || cfg.InputPath == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(cfg.InputPath)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "Error closing input file %s: %v\n", cfg.InputPath, cerr)
			}
		}()
		input = f
	}

	// 2. Prepare Output
	var output io.Writer
	if cfg.OutputPath == "" || cfg.OutputPath == "-" {
		output = os.Stdout
	} else {
		f, err := os.Create(cfg.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file %s: %v\n", cfg.OutputPath, cerr)
			}
		}()
		output = f
	}

	// 3. Determine Formats
	inputFormat := cfg.InputFormat
	if inputFormat == "" {
		// Auto-detect input format
		// We need to read the stream to detect.
		// If input is stdin, we might need to buffer it.
		// For simplicity, let's assume detect.Detector handles buffering or we read into memory.
		data, err := io.ReadAll(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		detector := detect.NewDetector()
		fmtDetected, ver, err := detector.DetectFromBytes(data)
		if err != nil {
			return fmt.Errorf("format detection failed: %w", err)
		}
		inputFormat = fmtDetected
		// Re-create reader from bytes since we consumed the stream
		input = strings.NewReader(string(data))

		if cfg.InputPath != "" {
			fmt.Fprintf(os.Stderr, "Detected input format: %s (%s)\n", fmtDetected, ver)
		}
	}

	outputFormat := cfg.OutputFormat
	outputEncoding := cfg.Encoding // Already holds value from flag or env var

	if cfg.OutputPath != "" && cfg.OutputPath != "-" {
		ext := strings.ToLower(filepath.Ext(cfg.OutputPath))

		// Derive output format from extension if not specified by --to
		if outputFormat == "" {
			switch ext {
			case ".apib", ".md":
				outputFormat = format.FormatAPIBlueprint
			case ".json", ".yaml", ".yml":
				outputFormat = format.FormatOpenAPI // Default to OpenAPI for JSON/YAML
			}
		}

		// Derive output encoding from extension if outputEncoding is still empty
		if outputEncoding == "" {
			switch ext {
			case ".json":
				outputEncoding = "json"
			case ".yaml", ".yml":
				outputEncoding = "yaml"
			}
		}
	} else if outputFormat == "" {
		return fmt.Errorf("output format must be specified via --to when writing to stdout")
	}

	// If outputFormat is still empty, default to OpenAPI
	// (This would happen if no --to flag, and no discernible extension)
	// This is already covered above with the JSON/YAML default if no apib/md
	if outputFormat == "" {
		outputFormat = format.FormatOpenAPI
	}

	// 4. Run Conversion
	// Inject configuration into context
	if outputEncoding != "" {
		ctx = converter.WithEncoding(ctx, outputEncoding)
	}
	if cfg.Protocol != "" {
		ctx = converter.WithProtocol(ctx, cfg.Protocol)
	}
	if cfg.OpenAPIVersion != "" {
		ctx = converter.WithOpenAPIVersion(ctx, cfg.OpenAPIVersion)
	}
	if cfg.AsyncAPIVersion != "" {
		ctx = converter.WithAsyncAPIVersion(ctx, cfg.AsyncAPIVersion)
	}

	if cfg.Verbose {
		r.conv.SetProgress(func(n int64) {
			// Print to stderr, using carriage return to update line
			fmt.Fprintf(os.Stderr, "\rProcessed %d bytes...", n)
		})
	}

	err := r.conv.Convert(ctx, input, output, inputFormat, outputFormat)
	if err != nil {
		return errors.Wrap("convert", string(outputFormat), errors.ErrConversionFailed)
	}

	if cfg.Verbose {
		fmt.Fprintln(os.Stderr, "\nDone.")
	}

	return nil
}

// Validate executes the validation process based on the configuration.
func (r *Runner) Validate(ctx context.Context, cfg *Config) error {
	// similar logic to Convert but calls r.conv.Validate
	var input io.Reader
	var err error

	if cfg.InputPath == "" || cfg.InputPath == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(cfg.InputPath)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "Error closing input file %s: %v\n", cfg.InputPath, cerr)
			}
		}()
		input = f
	}

	// Read content for detection
	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	inputFormat := cfg.InputFormat
	if inputFormat == "" {
		detector := detect.NewDetector()
		fmtDetected, ver, err := detector.DetectFromBytes(data)
		if err != nil {
			return fmt.Errorf("format detection failed: %w", err)
		}
		inputFormat = fmtDetected
		fmt.Fprintf(os.Stderr, "Detected format: %s (%s)\n", fmtDetected, ver)
	}

	input = strings.NewReader(string(data))

	errs, err := r.conv.Validate(ctx, input, inputFormat)
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s: %s\n", e.Level, e.Message)
		}
		return errors.ErrValidationFailed
	}

	fmt.Println("Validation successful")
	return nil
}
