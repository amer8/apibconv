// Package specschema validates API documents against pinned specification JSON Schemas.
package specschema

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	"github.com/amer8/apibconv/pkg/format"
)

//go:embed schemas/*.json
var schemaFS embed.FS

type schemaRef struct {
	name string
	file string
}

var (
	compileOnce sync.Once
	compileErr  error
	schemas     map[string]*jsonschema.Schema
)

var schemaRefs = map[string]schemaRef{
	"openapi:2.0":  {name: "OpenAPI 2.0", file: "openapi-2.0.json"},
	"openapi:3.0":  {name: "OpenAPI 3.0", file: "openapi-3.0.json"},
	"openapi:3.1":  {name: "OpenAPI 3.1", file: "openapi-3.1.json"},
	"asyncapi:2.6": {name: "AsyncAPI 2.6.0", file: "asyncapi-2.6.0.json"},
	"asyncapi:3.0": {name: "AsyncAPI 3.0.0", file: "asyncapi-3.0.0.json"},
}

// Validate validates a YAML or JSON API document against a pinned JSON Schema when
// one is available for the requested format and declared version.
func Validate(formatType format.Format, data []byte) []format.ValidationError {
	if formatType != format.FormatOpenAPI && formatType != format.FormatAsyncAPI {
		return nil
	}

	doc, err := decodeYAML(data)
	if err != nil {
		return []format.ValidationError{{
			Path:    "$",
			Message: fmt.Sprintf("failed to decode document for schema validation: %v", err),
			Level:   format.LevelError,
		}}
	}

	key := schemaKey(formatType, doc)
	if key == "" {
		if schemaUnavailableButSupported(formatType, doc) {
			return nil
		}
		return []format.ValidationError{unsupportedVersionError(formatType, doc)}
	}

	schema, err := getSchema(key)
	if err != nil {
		return []format.ValidationError{{
			Path:    "$",
			Message: fmt.Sprintf("failed to load %s schema: %v", schemaRefs[key].name, err),
			Level:   format.LevelError,
		}}
	}

	if err := schema.Validate(doc); err != nil {
		return validationErrors(schemaRefs[key].name, err)
	}
	return nil
}

func decodeYAML(data []byte) (any, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return normalizeYAML(doc), nil
}

func normalizeYAML(v any) any {
	switch value := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for k, child := range value {
			result[k] = normalizeYAML(child)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(value))
		for k, child := range value {
			result[fmt.Sprint(k)] = normalizeYAML(child)
		}
		return result
	case []any:
		for i, child := range value {
			value[i] = normalizeYAML(child)
		}
		return value
	default:
		return value
	}
}

func schemaKey(formatType format.Format, doc any) string {
	obj, ok := doc.(map[string]any)
	if !ok {
		return ""
	}

	switch formatType {
	case format.FormatOpenAPI:
		if swagger := stringField(obj, "swagger"); swagger == "2.0" {
			return "openapi:2.0"
		}
		switch majorMinor(stringField(obj, "openapi")) {
		case "3.0":
			return "openapi:3.0"
		case "3.1":
			return "openapi:3.1"
		}
	case format.FormatAsyncAPI:
		switch majorMinor(stringField(obj, "asyncapi")) {
		case "2.6":
			return "asyncapi:2.6"
		case "3.0":
			return "asyncapi:3.0"
		}
	}
	return ""
}

func schemaUnavailableButSupported(formatType format.Format, doc any) bool {
	obj, ok := doc.(map[string]any)
	if !ok {
		return false
	}

	switch formatType {
	case format.FormatAsyncAPI:
		version := stringField(obj, "asyncapi")
		return isUnpinnedAsyncAPI2Version(version)
	default:
		return false
	}
}

func isUnpinnedAsyncAPI2Version(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 || parts[0] != "2" || majorMinor(version) == "2.6" {
		return false
	}
	_, err := strconv.Atoi(parts[1])
	return err == nil
}

func unsupportedVersionError(formatType format.Format, doc any) format.ValidationError {
	obj, _ := doc.(map[string]any)
	var message string

	switch formatType {
	case format.FormatOpenAPI:
		version := firstNonEmpty(stringField(obj, "swagger"), stringField(obj, "openapi"))
		if version == "" {
			message = "missing OpenAPI version: expected swagger: 2.0 or openapi: 3.0/3.1"
		} else {
			message = fmt.Sprintf("unsupported OpenAPI version %q: expected 2.0, 3.0, or 3.1", version)
		}
	case format.FormatAsyncAPI:
		version := stringField(obj, "asyncapi")
		if version == "" {
			message = "missing AsyncAPI version: expected asyncapi: 2.6 or 3.0"
		} else {
			message = fmt.Sprintf("unsupported AsyncAPI version %q: expected 2.6 or 3.0", version)
		}
	default:
		message = "missing or unsupported specification version"
	}

	return format.ValidationError{
		Path:    "$",
		Message: message,
		Level:   format.LevelError,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringField(obj map[string]any, name string) string {
	value, ok := obj[name].(string)
	if !ok {
		return ""
	}
	return value
}

func majorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "." + parts[1]
}

func getSchema(key string) (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		schemas = make(map[string]*jsonschema.Schema, len(schemaRefs))
		compiler := jsonschema.NewCompiler()

		for key, ref := range schemaRefs {
			schemaPath := path.Join("schemas", ref.file)
			raw, err := schemaFS.ReadFile(schemaPath)
			if err != nil {
				compileErr = err
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				compileErr = fmt.Errorf("%s: %w", schemaPath, err)
				return
			}
			if err := compiler.AddResource(ref.file, doc); err != nil {
				compileErr = fmt.Errorf("%s: %w", schemaPath, err)
				return
			}
			compiled, err := compiler.Compile(ref.file)
			if err != nil {
				compileErr = fmt.Errorf("%s: %w", schemaPath, err)
				return
			}
			schemas[key] = compiled
		}
	})
	if compileErr != nil {
		return nil, compileErr
	}
	schema, ok := schemas[key]
	if !ok {
		return nil, fmt.Errorf("schema %q is not registered", key)
	}
	return schema, nil
}

func validationErrors(schemaName string, err error) []format.ValidationError {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return []format.ValidationError{{
			Path:    "$",
			Message: fmt.Sprintf("%s schema validation failed: %v", schemaName, err),
			Level:   format.LevelError,
		}}
	}

	var result []format.ValidationError
	collectValidationErrors(schemaName, validationErr, &result)
	if len(result) == 0 {
		result = append(result, format.ValidationError{
			Path:    jsonPointer(validationErr.InstanceLocation),
			Message: fmt.Sprintf("%s schema validation failed: %s", schemaName, validationErr.Error()),
			Level:   format.LevelError,
		})
	}
	return result
}

func collectValidationErrors(schemaName string, err *jsonschema.ValidationError, result *[]format.ValidationError) {
	if len(err.Causes) == 0 {
		*result = append(*result, format.ValidationError{
			Path:    jsonPointer(err.InstanceLocation),
			Message: fmt.Sprintf("%s schema validation failed: %s", schemaName, err.Error()),
			Level:   format.LevelError,
		})
		return
	}
	for _, cause := range err.Causes {
		collectValidationErrors(schemaName, cause, result)
	}
}

func jsonPointer(parts []string) string {
	if len(parts) == 0 {
		return "$"
	}
	escaped := make([]string, len(parts))
	for i, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		part = strings.ReplaceAll(part, "/", "~1")
		escaped[i] = part
	}
	return "/" + strings.Join(escaped, "/")
}
