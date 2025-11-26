package converter

import (
	"errors"
	"fmt"
)

// Error sentinel values for common error conditions.
// These can be used with errors.Is() for programmatic error handling.
var (
	// ErrNilSpec is returned when a nil specification is provided to a function that requires a non-nil spec.
	ErrNilSpec = errors.New("specification cannot be nil")

	// ErrInvalidJSON is returned when JSON parsing fails due to malformed JSON.
	ErrInvalidJSON = errors.New("invalid JSON format")

	// ErrInvalidYAML is returned when YAML parsing fails due to malformed YAML.
	ErrInvalidYAML = errors.New("invalid YAML format")

	// ErrUnsupportedVersion is returned when an unsupported API specification version is encountered.
	ErrUnsupportedVersion = errors.New("unsupported specification version")

	// ErrUnsupportedFormat is returned when an unsupported output format is requested.
	ErrUnsupportedFormat = errors.New("unsupported output format")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrConversionFailed is returned when a conversion operation fails.
	ErrConversionFailed = errors.New("conversion failed")

	// ErrValidationFailed is returned when specification validation fails.
	ErrValidationFailed = errors.New("validation failed")
)

// ParseError represents an error that occurred during parsing of a specification.
// It wraps the underlying error and provides context about where the error occurred.
//
// Example:
//
//	err := &ParseError{
//	    Format: "OpenAPI",
//	    Cause:  jsonErr,
//	}
//	fmt.Println(err) // "failed to parse OpenAPI: <json error>"
//
//	if errors.Is(err, ErrInvalidJSON) {
//	    // Handle JSON parsing error
//	}
type ParseError struct {
	// Format is the specification format being parsed (e.g., "OpenAPI", "AsyncAPI", "API Blueprint").
	Format string
	// Cause is the underlying error that caused the parse failure.
	Cause error
}

// Error returns the error message.
func (e *ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("failed to parse %s: %v", e.Format, e.Cause)
	}
	return fmt.Sprintf("failed to parse %s", e.Format)
}

// Unwrap returns the underlying cause for use with errors.Is and errors.As.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// ConversionError represents an error that occurred during specification conversion.
// It provides context about the source and target formats.
//
// Example:
//
//	err := &ConversionError{
//	    From:  "OpenAPI 3.0",
//	    To:    "API Blueprint",
//	    Cause: someErr,
//	}
//	fmt.Println(err) // "failed to convert from OpenAPI 3.0 to API Blueprint: <error>"
type ConversionError struct {
	// From is the source format.
	From string
	// To is the target format.
	To string
	// Cause is the underlying error that caused the conversion failure.
	Cause error
}

// Error returns the error message.
func (e *ConversionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("failed to convert from %s to %s: %v", e.From, e.To, e.Cause)
	}
	return fmt.Sprintf("failed to convert from %s to %s", e.From, e.To)
}

// Unwrap returns the underlying cause for use with errors.Is and errors.As.
func (e *ConversionError) Unwrap() error {
	return e.Cause
}

// ValidationError represents a validation error with details about what failed.
// It can contain multiple validation issues.
//
// Example:
//
//	err := &ValidationError{
//	    Field:   "info.title",
//	    Message: "title is required",
//	}
//	fmt.Println(err) // "validation error at info.title: title is required"
type ValidationError struct {
	// Field is the JSON path to the field that failed validation (e.g., "info.title", "paths./users.get").
	Field string
	// Message describes the validation failure.
	Message string
	// Cause is an optional underlying error.
	Cause error
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error at %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Unwrap returns the underlying cause for use with errors.Is and errors.As.
func (e *ValidationError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrValidationFailed
}

// ValidationErrors represents a collection of validation errors.
// This is useful when multiple validation issues are found at once.
//
// Example:
//
//	errs := &ValidationErrors{
//	    Errors: []*ValidationError{
//	        {Field: "info.title", Message: "title is required"},
//	        {Field: "paths", Message: "at least one path is required"},
//	    },
//	}
//	fmt.Println(errs.Count()) // 2
type ValidationErrors struct {
	// Errors is the list of validation errors.
	Errors []*ValidationError
}

// Error returns a summary of all validation errors.
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed with no specific errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("validation failed with %d errors: %s (and %d more)",
		len(e.Errors), e.Errors[0].Error(), len(e.Errors)-1)
}

// Unwrap returns ErrValidationFailed for use with errors.Is.
func (e *ValidationErrors) Unwrap() error {
	return ErrValidationFailed
}

// Count returns the number of validation errors.
func (e *ValidationErrors) Count() int {
	return len(e.Errors)
}

// Add adds a validation error to the collection.
func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, &ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// VersionError represents an error related to specification version handling.
//
// Example:
//
//	err := &VersionError{
//	    Version: "4.0.0",
//	    Message: "OpenAPI 4.0 is not supported",
//	}
type VersionError struct {
	// Version is the problematic version string.
	Version string
	// Message describes the version-related issue.
	Message string
}

// Error returns the error message.
func (e *VersionError) Error() string {
	return fmt.Sprintf("version error for %s: %s", e.Version, e.Message)
}

// Unwrap returns ErrUnsupportedVersion for use with errors.Is.
func (e *VersionError) Unwrap() error {
	return ErrUnsupportedVersion
}

// NewParseError creates a new ParseError with the given format and cause.
func NewParseError(format string, cause error) *ParseError {
	return &ParseError{
		Format: format,
		Cause:  cause,
	}
}

// NewConversionError creates a new ConversionError with the given details.
func NewConversionError(from, to string, cause error) *ConversionError {
	return &ConversionError{
		From:  from,
		To:    to,
		Cause: cause,
	}
}

// NewValidationError creates a new ValidationError with the given field and message.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewVersionError creates a new VersionError with the given version and message.
func NewVersionError(version, message string) *VersionError {
	return &VersionError{
		Version: version,
		Message: message,
	}
}
