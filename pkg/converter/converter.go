// Package converter implements the core logic for converting between API formats.
package converter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/internal/specschema"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
	"github.com/amer8/apibconv/pkg/validator"
)

// Converter handles the conversion between different API specification formats.
type Converter struct {
	parsers   map[format.Format]format.Parser
	writers   map[format.Format]format.Writer
	validator *validator.Validator
	opts      Options
}

// New creates a new Converter with the given options.
func New(opts ...Option) (*Converter, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	v := validator.New()
	v.AddRule(&validator.PathValidationRule{})
	v.AddRule(&validator.SchemaValidationRule{})
	v.AddRule(&validator.ReferenceValidationRule{})
	if options.Strict || options.OnWarning != nil {
		v.AddRule(&validator.APIVersionWarningRule{})
	}

	return &Converter{
		parsers:   make(map[format.Format]format.Parser),
		writers:   make(map[format.Format]format.Writer),
		validator: v,
		opts:      options,
	}, nil
}

// RegisterParser registers a parser for a specific format.
func (c *Converter) RegisterParser(p format.Parser) {
	c.parsers[p.Format()] = p
}

// RegisterWriter registers a writer for a specific format.
func (c *Converter) RegisterWriter(w format.Writer) {
	c.writers[w.Format()] = w
}

// Convert migrates built-in formats in the supported directions: API Blueprint
// to OpenAPI 3.1, OpenAPI 2.0/3.0 to 3.1, or OpenAPI
// within the same major/minor version. Custom formats are unrestricted.
func (c *Converter) Convert(ctx context.Context, input io.Reader, output io.Writer, from, to format.Format) error {
	api, err := c.ParseToModel(ctx, input, from)
	if err != nil {
		return err
	}
	writer, ok := c.writers[to]
	if !ok {
		return fmt.Errorf("no writer registered for format: %s", to)
	}
	if err := checkMigration(ctx, from, to, api.Version, writer.Version()); err != nil {
		return err
	}

	if c.opts.Transform != nil {
		if err := c.opts.Transform(api); err != nil {
			return err
		}
	}

	return c.WriteFromModel(ctx, api, output, to)
}

// ParseToModel parses the input into the unified model.
func (c *Converter) ParseToModel(ctx context.Context, input io.Reader, from format.Format) (*model.API, error) {
	return c.parseToModel(ctx, input, from, c.opts.ValidateInput)
}

func (c *Converter) parseToModel(ctx context.Context, input io.Reader, from format.Format, validate bool) (*model.API, error) {
	parser, ok := c.parsers[from]
	if !ok {
		return nil, fmt.Errorf("no parser registered for format: %s", from)
	}

	validationCtx := validator.WithFormat(ctx, from)

	if c.opts.OnProgress != nil {
		input = fs.NewProgressReader(input, c.opts.OnProgress)
	}

	api, err := parser.Parse(ctx, input)
	if err != nil {
		return nil, err
	}

	if !c.opts.PreserveExtensions {
		api = api.WithoutExtensions()
	}

	if validate || c.opts.Strict || c.opts.OnWarning != nil {
		if err := c.validateModel(validationCtx, api, "input"); err != nil {
			return nil, err
		}
	}

	return api, nil
}

// WriteFromModel writes the unified model to the output format.
func (c *Converter) WriteFromModel(ctx context.Context, api *model.API, output io.Writer, to format.Format) error {
	writer, ok := c.writers[to]
	if !ok {
		return fmt.Errorf("no writer registered for format: %s", to)
	}

	validationCtx := validator.WithFormat(ctx, to)

	if !c.opts.PreserveExtensions {
		api = api.WithoutExtensions()
	}

	if c.opts.ValidateOutput || c.opts.Strict || c.opts.OnWarning != nil {
		if err := c.validateModel(validationCtx, api, "output"); err != nil {
			return err
		}
	}

	return writer.Write(ctx, api, output)
}

func (c *Converter) validateModel(ctx context.Context, api *model.API, stage string) error {
	issues, err := c.validator.Validate(ctx, api)
	if err != nil {
		return fmt.Errorf("%s validation: %w", stage, err)
	}
	issues = addStructureIssue(api, issues)
	var failures []error
	for _, issue := range issues {
		if issue.Level == format.LevelWarning && c.opts.OnWarning != nil {
			c.opts.OnWarning(issue.Error())
		}
		if issue.Level == format.LevelError || (c.opts.Strict && issue.Level == format.LevelWarning) {
			failures = append(failures, issue)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s validation: %w: %w", stage, apierrors.ErrValidationFailed, errors.Join(failures...))
	}
	return nil
}

// Validate validates the input specification.
func (c *Converter) Validate(ctx context.Context, input io.Reader, formatType format.Format) ([]format.ValidationError, error) {
	data, err := fs.ReadAll(ctx, input)
	if err != nil {
		return nil, err
	}

	allErrors := specschema.Validate(formatType, data)

	// Validate collects the issues itself so enabled conversion validation does
	// not turn the same issues into an early error or discard schema diagnostics.
	api, err := c.parseToModel(ctx, bytes.NewReader(data), formatType, false)
	if err != nil {
		return allErrors, err
	}

	modelErrors, err := c.validator.Validate(validator.WithFormat(ctx, formatType), api)
	if err != nil {
		return allErrors, err
	}
	modelErrors = addStructureIssue(api, modelErrors)
	allErrors = append(allErrors, modelErrors...)
	if c.opts.OnWarning != nil {
		for _, issue := range allErrors {
			if issue.Level == format.LevelWarning {
				c.opts.OnWarning(issue.Error())
			}
		}
	}
	return allErrors, nil
}

// SupportedFormats returns registered input and output formats in unspecified order.
func (c *Converter) SupportedFormats() (inputs, outputs []format.Format) {
	for k := range c.parsers {
		inputs = append(inputs, k)
	}
	for k := range c.writers {
		outputs = append(outputs, k)
	}
	return
}

// DetectFormat attempts to detect the format of the input. It reads the entire
// input and does not currently use ctx to interrupt the read.
func (c *Converter) DetectFormat(ctx context.Context, input io.Reader) (format.Format, string, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return format.FormatUnknown, "", fmt.Errorf("failed to read input for detection: %w", err)
	}

	detector := detect.NewDetector()
	return detector.DetectFromBytes(data)
}

// SetProgress sets the progress callback function.
func (c *Converter) SetProgress(fn func(int64)) {
	c.opts.OnProgress = fn
}

// addStructureIssue supplements the detailed rules when they found no errors.
// API.Validate reports one structural issue; the other rules can report several.
func addStructureIssue(api *model.API, issues []format.ValidationError) []format.ValidationError {
	for _, issue := range issues {
		if issue.Level == format.LevelError {
			return issues
		}
	}
	if err := api.Validate(); err != nil {
		return append(issues, format.ValidationError{Path: "$", Message: err.Error(), Level: format.LevelError})
	}
	return issues
}
