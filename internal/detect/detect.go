// Package detect provides functionality for detecting API specification formats.
package detect

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/pkg/format"
)

// Detector handles format detection for API specifications.
type Detector struct{}

// FormatReader interface for types that can inspect data to determine format.
type FormatReader interface {
	CanRead(data []byte) bool
	GetFormat() format.Format
	GetVersion() string
}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect reads from the reader and attempts to determine the API format and version.
func (d *Detector) Detect(ctx context.Context, r io.Reader) (format.Format, string, error) {
	if err := ctx.Err(); err != nil {
		return format.FormatUnknown, "", err
	}
	data, err := fs.ReadAll(ctx, r)
	if err != nil {
		return format.FormatUnknown, "", err
	}
	return d.DetectFromBytes(data)
}

// DetectFromBytes attempts to determine the API format and version from a byte slice.
func (d *Detector) DetectFromBytes(data []byte) (format.Format, string, error) {
	if ok, ver := OpenAPI(data); ok {
		return format.FormatOpenAPI, ver, nil
	}
	if ok, ver := AsyncAPI(data); ok {
		return format.FormatAsyncAPI, ver, nil
	}
	if APIBlueprint(data) {
		return format.FormatAPIBlueprint, "1A", nil
	}
	return format.FormatUnknown, "", fmt.Errorf("unable to detect format")
}

// OpenAPI checks if the data represents an OpenAPI specification.
func OpenAPI(data []byte) (isFormat bool, version string) {
	s := string(data)
	if strings.Contains(s, `"openapi"`) || strings.Contains(s, "openapi:") {
		return true, "3.x"
	}
	if strings.Contains(s, `"swagger"`) || strings.Contains(s, "swagger:") {
		return true, "2.0"
	}
	return false, ""
}

// APIBlueprint checks if the data represents an API Blueprint specification.
func APIBlueprint(data []byte) bool {
	s := string(data)
	return strings.Contains(s, "FORMAT: 1A") || strings.HasPrefix(strings.TrimSpace(s), "#")
}

// AsyncAPI checks if the data represents an AsyncAPI specification.
func AsyncAPI(data []byte) (isFormat bool, version string) {
	s := string(data)
	if strings.Contains(s, `"asyncapi"`) || strings.Contains(s, "asyncapi:") {
		return true, "2.x"
	}
	return false, ""
}
