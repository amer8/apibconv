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
	OpenAPI  string `yaml:"openapi"`
	Swagger  string `yaml:"swagger"`
	AsyncAPI string `yaml:"asyncapi"`
}

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
	if detectedFormat, ver, ok := detectStructuredSpec(data); ok {
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

	switch {
	case strings.TrimSpace(doc.AsyncAPI) != "":
		return format.FormatAsyncAPI, normalizeAsyncAPIVersion(doc.AsyncAPI), true
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

func normalizeAsyncAPIVersion(version string) string {
	version = strings.TrimSpace(version)
	switch {
	case strings.HasPrefix(version, "3"):
		return "3.0"
	case strings.HasPrefix(version, "2.6"):
		return "2.6"
	case strings.HasPrefix(version, "2"):
		return "2.0"
	default:
		return "2.6"
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

	lines := strings.Split(trimmed, "\n")
	hasTitle := false
	hasBlueprintStructure := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		switch {
		case reBlueprintTitle.MatchString(line):
			hasTitle = true
		case reBlueprintGroup.MatchString(line),
			reBlueprintResource.MatchString(line),
			reBlueprintAction.MatchString(line),
			reBlueprintAttributes.MatchString(line),
			reBlueprintResponse.MatchString(line),
			reBlueprintDataStructures.MatchString(line):
			hasBlueprintStructure = true
		}

		if hasTitle && hasBlueprintStructure {
			return true
		}
	}

	return hasBlueprintStructure
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
