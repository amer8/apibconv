// Package detect provides functionality for detecting API specification formats.
package detect

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/internal/yamlguard"
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
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err == nil {
		if err := yamlguard.Check(&root); err != nil {
			// Markdown headings are YAML comments, and prose can look like
			// duplicate keys. Never apply a typed decoder to a rejected graph.
			if couldBeBlueprintProse(&root) && APIBlueprint(data) {
				return format.FormatAPIBlueprint, "1A", nil
			}
			return format.FormatUnknown, "", err
		}
		if detectedFormat, ver, ok := detectStructuredSpec(&root); ok {
			if detectedFormat == format.FormatUnknown {
				return format.FormatUnknown, "", fmt.Errorf("unsupported specification format")
			}
			return detectedFormat, ver, nil
		}
	}
	if APIBlueprint(data) {
		return format.FormatAPIBlueprint, "1A", nil
	}
	return format.FormatUnknown, "", fmt.Errorf("unable to detect format")
}

// Only ordinary string labels qualify for the prose exception. A rejected
// graph cannot safely be decoded to determine what an indirect key means.
func couldBeBlueprintProse(root *yaml.Node) bool {
	if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return false
	}
	fields := root.Content[0].Content
	for i := 0; i < len(fields); i += 2 {
		key := fields[i]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return false
		}
		if key.Value == "swagger" || key.Value == "<<" || (!strings.HasPrefix(key.Value, "x-") && strings.HasSuffix(key.Value, "api")) {
			return false
		}
	}
	return true
}

func detectStructuredSpec(root *yaml.Node) (format.Format, string, bool) {
	var doc specEnvelope
	if err := root.Decode(&doc); err != nil {
		return format.FormatUnknown, "", false
	}

	// Do not mistake another structured API format for an OpenAPI document,
	// even if a document contains markers for multiple specifications.
	var fields map[string]interface{}
	if err := root.Decode(&fields); err == nil {
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
	case strings.HasPrefix(version, "3.1.") || version == "3.1":
		return "3.1.x"
	case strings.HasPrefix(version, "3.0.") || version == "3.0":
		return "3.0.x"
	default:
		return version
	}
}

// APIBlueprint checks if the data represents an API Blueprint specification.
func APIBlueprint(data []byte) bool {
	trimmed := strings.TrimSpace(strings.TrimPrefix(string(data), "\uFEFF"))
	if trimmed == "" {
		return false
	}
	if reBlueprintFormat.MatchString(trimmed) {
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
	reBlueprintFormat         = regexp.MustCompile(`(?m)^FORMAT:[ \t]*1A[ \t]*\r?$`)
	reBlueprintTitle          = regexp.MustCompile(`^#\s+\S`)
	reBlueprintGroup          = regexp.MustCompile(`^#+\s+Group\s+\S`)
	reBlueprintResource       = regexp.MustCompile(`^#+\s+.+\[[^[\]]+\]$`)
	reBlueprintAction         = regexp.MustCompile(`^#+\s+.+\[[A-Z]+\]$`)
	reBlueprintAttributes     = regexp.MustCompile(`^\+\s+Attributes\b`)
	reBlueprintResponse       = regexp.MustCompile(`^\+\s+Response\b`)
	reBlueprintDataStructures = regexp.MustCompile(`^#+\s+Data Structures$`)
)
