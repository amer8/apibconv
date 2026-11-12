package cli

import (
	"bytes"
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
	input, closeInput, err := openInput(cfg.InputPath)
	if err != nil {
		return err
	}
	defer closeInput()

	inputFormat := cfg.InputFormat
	if inputFormat == "" {
		// Detection consumes the input, so keep a copy for the parser.
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
		input = bytes.NewReader(data)

		if cfg.InputPath != "" {
			fmt.Fprintf(os.Stderr, "Detected input format: %s (%s)\n", fmtDetected, ver)
		}
	}

	outputFormat := cfg.OutputFormat
	outputEncoding := cfg.Encoding

	if cfg.OutputPath != "" && cfg.OutputPath != "-" {
		ext := strings.ToLower(filepath.Ext(cfg.OutputPath))

		// Derive output format from extension if not specified by --to
		if outputFormat == "" {
			switch ext {
			case ".apib", ".md":
				outputFormat = format.FormatAPIBlueprint
			case ".json", ".yaml", ".yml":
				outputFormat = format.FormatOpenAPI
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

	if outputFormat == "" {
		outputFormat = format.FormatOpenAPI
	}

	if outputEncoding != "" {
		ctx = converter.WithEncoding(ctx, outputEncoding)
	}
	if cfg.OpenAPIVersion != "" {
		ctx = converter.WithOpenAPIVersion(ctx, cfg.OpenAPIVersion)
	}

	if cfg.Verbose {
		r.conv.SetProgress(func(n int64) {
			fmt.Fprintf(os.Stderr, "\rProcessed %d bytes...", n)
		})
	}

	var output io.Writer = os.Stdout
	var pending bytes.Buffer
	var destination *atomicOutput
	if cfg.OutputPath != "" && cfg.OutputPath != "-" {
		var err error
		destination, err = newAtomicOutput(cfg.InputPath, cfg.OutputPath)
		if err != nil {
			return err
		}
		defer destination.cleanup()
		output = destination.file
	}
	if cfg.ValidateOutput {
		output = &pending
	}

	err = r.conv.Convert(ctx, input, output, inputFormat, outputFormat)
	if err != nil {
		return errors.Wrap("convert", string(outputFormat), fmt.Errorf("%w: %w", errors.ErrConversionFailed, err))
	}
	if cfg.ValidateOutput {
		issues, validationErr := r.conv.Validate(ctx, bytes.NewReader(pending.Bytes()), outputFormat)
		report := makeValidationReport(issues, validationErr)
		if err := writeTextIssues(os.Stderr, report.Issues); err != nil {
			return fmt.Errorf("write output validation diagnostics: %w", err)
		}
		if validationErr != nil {
			return fmt.Errorf("output validation: %w", validationErr)
		}
		if !report.Valid {
			return fmt.Errorf("output validation: %w", errors.ErrValidationFailed)
		}
		if destination != nil {
			output = destination.file
		} else {
			output = os.Stdout
		}
		if _, err := pending.WriteTo(output); err != nil {
			return fmt.Errorf("write validated output: %w", err)
		}
	}

	if destination != nil {
		if err := destination.commit(); err != nil {
			return err
		}
	}

	if cfg.Verbose {
		fmt.Fprintln(os.Stderr, "\nDone.")
	}

	return nil
}

// Validate executes the validation process based on the configuration.
func (r *Runner) Validate(ctx context.Context, cfg *Config) error {
	input, closeInput, err := openInput(cfg.InputPath)
	if err != nil {
		return err
	}
	defer closeInput()

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

	input = bytes.NewReader(data)

	issues, validationErr := r.conv.Validate(ctx, input, inputFormat)
	report := makeValidationReport(issues, validationErr)
	if err := writeValidationReport(report, cfg.Report, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("write validation report: %w", err)
	}
	if validationErr != nil {
		return validationErr
	}
	if !report.Valid {
		return errors.ErrValidationFailed
	}
	return nil
}

// openInput returns stdin for an empty path or "-". The caller closes files
// through the returned function; stdin is left open.
func openInput(path string) (io.Reader, func(), error) {
	if path == "" || path == "-" {
		return os.Stdin, func() {}, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open input file: %w", err)
	}
	closeInput := func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing input file %s: %v\n", path, err)
		}
	}
	return file, closeInput, nil
}
