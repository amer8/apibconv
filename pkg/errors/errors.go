// Package errors defines standard error types used throughout the application.
package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrUnsupportedFormat indicates that the requested format is not supported.
	ErrUnsupportedFormat   = errors.New("unsupported format")
	// ErrInvalidSpec indicates that the specification is invalid.
	ErrInvalidSpec         = errors.New("invalid specification")
	// ErrParseFailure indicates a failure during parsing of the specification.
	ErrParseFailure        = errors.New("parse failure")
	// ErrWriteFailure indicates a failure during writing of the specification.
	ErrWriteFailure        = errors.New("write failure")
	// ErrConversionFailed indicates a general failure during conversion.
	ErrConversionFailed    = errors.New("conversion failed")
	// ErrValidationFailed indicates that the validation of the specification failed.
	ErrValidationFailed    = errors.New("validation failed")
	// ErrFormatDetection indicates a failure during automatic format detection.
	ErrFormatDetection     = errors.New("format detection failed")
	// ErrIncompatibleVersion indicates that the specification version is incompatible.
	ErrIncompatibleVersion = errors.New("incompatible version")
)

// ConversionError represents an error that occurred during a conversion process.
type ConversionError struct {
	Op     string
	Format string
	Err    error
}

// Error returns the string representation of the conversion error.
func (e *ConversionError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Format, e.Err)
}

// Unwrap returns the underlying error.
func (e *ConversionError) Unwrap() error {
	return e.Err
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error returns the string representation of the validation error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s (value: %v)", e.Field, e.Message, e.Value)
}

// ParseError represents an error that occurred during parsing, with location information.
type ParseError struct {
	Line   int
	Column int
	Offset int
	Err    error
}

// Error returns the string representation of the parse error.
func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at line %d, col %d: %v", e.Line, e.Column, e.Err)
}

// Wrap creates a ConversionError
func Wrap(op, format string, err error) error {
	return &ConversionError{
		Op:     op,
		Format: format,
		Err:    err,
	}
}

// Is wraps errors.Is
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As wraps errors.As
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
