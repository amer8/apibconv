package converter

import "github.com/amer8/apibconv/pkg/model"

// Options holds configuration settings for the Converter.
type Options struct {
	Strict             bool
	ValidateInput      bool
	ValidateOutput     bool
	PreserveExtensions bool
	Transform          TransformFunc
	OnWarning          WarningFunc
	OnProgress         func(int64)
}

// TransformFunc defines a function signature for modifying the API model.
type TransformFunc func(*model.API) error

// WarningFunc defines a function signature for handling warning messages.
type WarningFunc func(warning string)

// Option defines a functional option for configuring the Converter.
type Option func(*Options)

func defaultOptions() Options {
	return Options{
		Strict:             false,
		ValidateInput:      false,
		ValidateOutput:     false,
		PreserveExtensions: true,
	}
}

// WithStrict enables strict mode, where warnings are treated as errors.
func WithStrict(strict bool) Option {
	return func(o *Options) {
		o.Strict = strict
	}
}

// WithValidation enables input and/or output validation.
func WithValidation(input, output bool) Option {
	return func(o *Options) {
		o.ValidateInput = input
		o.ValidateOutput = output
	}
}

// WithExtensions enables or disables preservation of vendor extensions.
func WithExtensions(preserve bool) Option {
	return func(o *Options) {
		o.PreserveExtensions = preserve
	}
}

// WithTransform adds a custom transformation function to be applied during conversion.
func WithTransform(fn TransformFunc) Option {
	return func(o *Options) {
		o.Transform = fn
	}
}

// WithWarningHandler sets a custom handler for warning messages.
func WithWarningHandler(fn WarningFunc) Option {
	return func(o *Options) {
		o.OnWarning = fn
	}
}

// WithProgress sets a callback function to track conversion progress (bytes read).
func WithProgress(fn func(int64)) Option {
	return func(o *Options) {
		o.OnProgress = fn
	}
}
