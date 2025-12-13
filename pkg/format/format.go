// Package format defines interfaces and common types for API format parsers and writers.
package format

import (
	"context"
	"fmt"
	"io"

	"github.com/amer8/apibconv/pkg/model"
)

// Format represents a supported API specification format.
type Format string

const (
	// FormatOpenAPI indicates the OpenAPI specification format.
	FormatOpenAPI      Format = "openapi"
	// FormatAPIBlueprint indicates the API Blueprint specification format.
	FormatAPIBlueprint Format = "apiblueprint"
	// FormatAsyncAPI indicates the AsyncAPI specification format.
	FormatAsyncAPI     Format = "asyncapi"
	// FormatUnknown indicates an unknown or unrecognised format.
	FormatUnknown      Format = "unknown"
)

// Parser defines the interface for parsing an API specification into a unified model.
type Parser interface {
	Parse(ctx context.Context, r io.Reader) (*model.API, error)
	Format() Format
	SupportsVersion(version string) bool
}

// Writer defines the interface for writing a unified API model to a specific format.
type Writer interface {
	Write(ctx context.Context, api *model.API, w io.Writer) error
	Format() Format
	Version() string
}

// Validator defines the interface for validating an API specification.
type Validator interface {
	Validate(ctx context.Context, r io.Reader) ([]ValidationError, error)
	Format() Format
}

// ValidationError represents a single validation issue.
type ValidationError struct {
	Path    string
	Message string
	Level   ValidationLevel
}

// ValidationLevel indicates the severity of a validation error.
type ValidationLevel string

const (
	// LevelError indicates a critical validation error.
	LevelError   ValidationLevel = "error"
	// LevelWarning indicates a non-critical validation warning.
	LevelWarning ValidationLevel = "warning"
	// LevelInfo provides informational messages from validation.
	LevelInfo    ValidationLevel = "info"
)

// Detector defines the interface for detecting the format of an API specification.
type Detector interface {
	Detect(ctx context.Context, r io.Reader) (Format, string, error)
}

// String returns the string representation of the Format.
func (f Format) String() string {
	return string(f)
}

// IsValid checks if the Format is one of the known supported formats.
func (f Format) IsValid() bool {
	switch f {
	case FormatOpenAPI, FormatAPIBlueprint, FormatAsyncAPI:
		return true
	default:
		return false
	}
}

// Error returns the string representation of the validation error.
func (v ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", v.Level, v.Path, v.Message)
}
