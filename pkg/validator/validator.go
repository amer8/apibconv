// Package validator provides rules and logic for validating API models.
package validator

import (
	"context"
	"fmt"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Validator provides functionality to validate API specifications against a set of rules.
type Validator struct {
	rules []Rule
	opts  Options
}

// Rule defines an interface for a validation rule.
type Rule interface {
	Name() string
	Validate(api *model.API) []format.ValidationError
	Level() format.ValidationLevel
}

// Options configures the behavior of the Validator.
type Options struct {
	StopOnFirstError bool
	IncludeWarnings  bool
}

// Option is a functional option for configuring the Validator.
type Option func(*Options)

// WithStopOnError configures the validator to stop on the first error encountered.
func WithStopOnError(stop bool) Option {
	return func(o *Options) {
		o.StopOnFirstError = stop
	}
}

// WithWarnings configures the validator to include warnings in the results.
func WithWarnings(include bool) Option {
	return func(o *Options) {
		o.IncludeWarnings = include
	}
}

// New creates a new Validator instance with the given options.
func New(opts ...Option) *Validator {
	options := Options{
		StopOnFirstError: false,
		IncludeWarnings:  true,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &Validator{
		opts: options,
	}
}

// AddRule adds a validation rule to the validator.
func (v *Validator) AddRule(rule Rule) {
	v.rules = append(v.rules, rule)
}

// Validate runs all registered validation rules against the API model.
func (v *Validator) Validate(ctx context.Context, api *model.API) ([]format.ValidationError, error) {
	var allErrors []format.ValidationError

	for _, rule := range v.rules {
		errs := rule.Validate(api)
		allErrors = append(allErrors, errs...)

		if v.opts.StopOnFirstError && len(errs) > 0 {
			// Check if any are errors
			for _, e := range errs {
				if e.Level == format.LevelError {
					return allErrors, nil
				}
			}
		}
	}
	return allErrors, nil
}

// ValidateSchema performs validation on a specific schema.
func (v *Validator) ValidateSchema(schema *model.Schema) []format.ValidationError {
	return ValidateSchemaNode(schema, "schema")
}

// ValidateOperation performs validation on a specific operation.
func (v *Validator) ValidateOperation(op *model.Operation) []format.ValidationError {
	var errors []format.ValidationError
	if op == nil {
		return errors
	}

	// Validate Parameters
	for i, p := range op.Parameters {
		if p.Schema != nil {
			errs := ValidateSchemaNode(p.Schema, fmt.Sprintf("parameters[%d]", i))
			errors = append(errors, errs...)
		}
	}

	// Validate RequestBody
	if op.RequestBody != nil {
		for ct, mt := range op.RequestBody.Content {
			if mt.Schema != nil {
				errs := ValidateSchemaNode(mt.Schema, fmt.Sprintf("requestBody/content[%s]", ct))
				errors = append(errors, errs...)
			}
		}
	}

	// Validate Responses
	for status, resp := range op.Responses {
		for ct, mt := range resp.Content {
			if mt.Schema != nil {
				errs := ValidateSchemaNode(mt.Schema, fmt.Sprintf("responses[%s]/content[%s]", status, ct))
				errors = append(errors, errs...)
			}
		}
	}

	return errors
}