// Package converter implements the core logic for converting between API formats.
package converter

import (
	"context"
	"fmt"
	"io"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/internal/fs"
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

// Convert converts the input from one format to another.
func (c *Converter) Convert(ctx context.Context, input io.Reader, output io.Writer, from, to format.Format) error {
	api, err := c.ParseToModel(ctx, input, from)
	if err != nil {
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
	parser, ok := c.parsers[from]
	if !ok {
		return nil, fmt.Errorf("no parser registered for format: %s", from)
	}

	if c.opts.OnProgress != nil {
		input = fs.NewProgressReader(input, c.opts.OnProgress)
	}

	api, err := parser.Parse(ctx, input)
	if err != nil {
		return nil, err
	}

	if c.opts.ValidateInput {
		if _, err := c.validator.Validate(ctx, api); err != nil {
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

	if c.opts.ValidateOutput {
		if _, err := c.validator.Validate(ctx, api); err != nil {
			return err
		}
	}

	return writer.Write(ctx, api, output)
}

// Validate validates the input specification.
func (c *Converter) Validate(ctx context.Context, input io.Reader, formatType format.Format) ([]format.ValidationError, error) {
	// This would parse then validate.
	api, err := c.ParseToModel(ctx, input, formatType)
	if err != nil {
		return nil, err
	}
	return c.validator.Validate(ctx, api)
}

// SupportedFormats returns the list of supported input and output formats.
func (c *Converter) SupportedFormats() (inputs, outputs []format.Format) {
	for k := range c.parsers {
		inputs = append(inputs, k)
	}
	for k := range c.writers {
		outputs = append(outputs, k)
	}
	return
}

// DetectFormat attempts to detect the format of the input.
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

