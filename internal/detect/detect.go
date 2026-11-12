// Package detect provides functionality for detecting API specification formats.
package detect

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/pkg/format"
	"go.yaml.in/yaml/v3"
)

// Detector handles format detection for API specifications.
type Detector struct{}

type specEnvelope struct {
	OpenAPI string `yaml:"openapi"`
	Swagger string `yaml:"swagger"`
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
	if detectedFormat, ver, ok := detectStructuredSpec(data); ok {
		if detectedFormat == format.FormatUnknown {
			return format.FormatUnknown, "", fmt.Errorf("unsupported specification format")
		}
		return detectedFormat, ver, nil
	}
	if APIBlueprint(data) {
		return format.FormatAPIBlueprint, "1A", nil
	}
	return format.FormatUnknown, "", fmt.Errorf("unable to detect format")
}

func detectStructuredSpec(data []byte) (format.Format, string, bool) {
	var doc specEnvelope
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return format.FormatUnknown, "", false
	}

	// Do not mistake another structured API format for an OpenAPI document,
	// even if a document contains markers for multiple specifications.
	var fields map[string]interface{}
	if err := yaml.Unmarshal(data, &fields); err == nil {
		for key, value := range fields {
			if key != "openapi" && !strings.HasPrefix(key, "x-") && strings.HasSuffix(key, "api") && value != nil && fmt.Sprint(value) != "" {
				return format.FormatUnknown, "", true
			}
		}
	}

	switch {
	case strings.TrimSpace(doc.OpenAPI) != "":
		return format.FormatOpenAPI, normalizeOpenAPIVersion(doc.OpenAPI), true
	case strings.TrimSpace(doc.Swagger) != "":
		return format.FormatOpenAPI, "2.0", true
	default:
		return format.FormatUnknown, "", false
	}
}

func normalizeOpenAPIVersion(version string) string {
	version = strings.TrimSpace(version)
	switch {
	case strings.HasPrefix(version, "3.1"):
		return "3.1.x"
	case strings.HasPrefix(version, "3"):
		return "3.0.x"
	default:
		return "3.0.x"
	}
}

// APIBlueprint checks if the data represents an API Blueprint specification.
func APIBlueprint(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "FORMAT: 1A") {
		return true
	}

	for _, rawLine := range strings.Split(trimmed, "\n") {
		line := strings.TrimSpace(rawLine)
		// Treat a top-level heading as the API title, not a structure marker.
		if reBlueprintTitle.MatchString(line) {
			continue
		}
		switch {
		case reBlueprintGroup.MatchString(line),
			reBlueprintResource.MatchString(line),
			reBlueprintAction.MatchString(line),
			reBlueprintAttributes.MatchString(line),
			reBlueprintResponse.MatchString(line),
			reBlueprintDataStructures.MatchString(line):
			return true
		}
	}
	return false
}

var (
	reBlueprintTitle          = regexp.MustCompile(`^#\s+\S`)
	reBlueprintGroup          = regexp.MustCompile(`^#+\s+Group\s+\S`)
	reBlueprintResource       = regexp.MustCompile(`^#+\s+.+\[[^[\]]+\]$`)
	reBlueprintAction         = regexp.MustCompile(`^#+\s+.+\[[A-Z]+\]$`)
	reBlueprintAttributes     = regexp.MustCompile(`^\+\s+Attributes\b`)
	reBlueprintResponse       = regexp.MustCompile(`^\+\s+Response\b`)
	reBlueprintDataStructures = regexp.MustCompile(`^#+\s+Data Structures$`)
)
