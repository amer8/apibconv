package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// OutputFormat represents the output format for specification encoding.
type OutputFormat string

const (
	// FormatJSON outputs the specification as JSON.
	FormatJSON OutputFormat = "json"
	// FormatYAML outputs the specification as YAML.
	FormatYAML OutputFormat = "yaml"
)

// MarshalYAML converts any value to YAML format.
// This is a zero-dependency YAML encoder suitable for API specifications.
//
// The encoder supports:
//   - Primitive types (string, number, bool, null)
//   - Arrays/slices
//   - Maps and structs (via JSON marshaling)
//   - Nested structures
//
// Example:
//
//	spec := &OpenAPI{
//	    OpenAPI: "3.0.0",
//	    Info: Info{Title: "My API", Version: "1.0.0"},
//	}
//	yamlBytes, err := MarshalYAML(spec)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(string(yamlBytes))
func MarshalYAML(v any) ([]byte, error) {
	// First convert to JSON to normalize the structure
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Parse JSON into a generic structure
	var data any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert to YAML
	var buf bytes.Buffer
	if err := writeYAML(&buf, data, 0); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// MarshalYAMLIndent converts any value to YAML format with custom indentation.
//
// Parameters:
//   - v: The value to marshal
//   - indent: Number of spaces per indentation level (default 2 if <= 0)
//
// Example:
//
//	yamlBytes, err := MarshalYAMLIndent(spec, 4)
func MarshalYAMLIndent(v any, indent int) ([]byte, error) {
	if indent <= 0 {
		indent = 2
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	var data any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var buf bytes.Buffer
	if err := writeYAMLWithIndent(&buf, data, 0, indent); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// EncodeYAML writes the value as YAML to the provided writer.
//
// Example:
//
//	file, _ := os.Create("openapi.yaml")
//	defer file.Close()
//	if err := EncodeYAML(file, spec); err != nil {
//	    log.Fatal(err)
//	}
func encodeYAML(w io.Writer, v any) error {
	yamlBytes, err := MarshalYAML(v)
	if err != nil {
		return err
	}
	_, err = w.Write(yamlBytes)
	return err
}

// writeYAML writes a value as YAML with the given indentation level.
func writeYAML(buf *bytes.Buffer, v any, level int) error {
	return writeYAMLWithIndent(buf, v, level, 2)
}

// writeYAMLWithIndent writes a value as YAML with custom indentation.
func writeYAMLWithIndent(buf *bytes.Buffer, v any, level, indent int) error {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		writeYAMLBool(buf, val)
	case float64:
		writeYAMLNumber(buf, val)
	case string:
		writeYAMLString(buf, val)
	case []any:
		return writeYAMLArray(buf, val, level, indent)
	case map[string]any:
		return writeYAMLMapValue(buf, val, level, indent)
	default:
		return writeYAMLFallback(buf, val)
	}
	return nil
}

// writeYAMLBool writes a boolean value.
func writeYAMLBool(buf *bytes.Buffer, val bool) {
	if val {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
}

// writeYAMLNumber writes a numeric value.
func writeYAMLNumber(buf *bytes.Buffer, val float64) {
	if val == float64(int64(val)) {
		buf.WriteString(strconv.FormatInt(int64(val), 10))
	} else {
		buf.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
	}
}

// writeYAMLArray writes an array value.
func writeYAMLArray(buf *bytes.Buffer, val []any, level, indent int) error {
	if len(val) == 0 {
		buf.WriteString("[]")
		return nil
	}
	for i, item := range val {
		if i > 0 {
			buf.WriteString(strings.Repeat(" ", level*indent))
		}
		buf.WriteString("- ")
		if isComplexType(item) {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat(" ", (level+1)*indent))
			if err := writeYAMLMapInline(buf, item, level+1, indent); err != nil {
				return err
			}
		} else {
			if err := writeYAMLWithIndent(buf, item, level+1, indent); err != nil {
				return err
			}
		}
		buf.WriteString("\n")
	}
	return nil
}

// writeYAMLMapValue writes a map value, handling empty maps.
func writeYAMLMapValue(buf *bytes.Buffer, val map[string]any, level, indent int) error {
	if len(val) == 0 {
		buf.WriteString("{}")
		return nil
	}
	return writeYAMLMap(buf, val, level, indent)
}

// writeYAMLFallback handles unknown types by converting to JSON.
func writeYAMLFallback(buf *bytes.Buffer, val any) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	buf.Write(jsonBytes)
	return nil
}

// writeYAMLMap writes a map as YAML.
func writeYAMLMap(buf *bytes.Buffer, m map[string]any, level, indent int) error {
	keys := sortedKeys(m)
	return writeYAMLMapEntries(buf, m, keys, level, indent)
}

// writeYAMLMapInline writes map contents after an array item marker.
func writeYAMLMapInline(buf *bytes.Buffer, v any, level, indent int) error {
	m, ok := v.(map[string]any)
	if !ok {
		return writeYAMLWithIndent(buf, v, level, indent)
	}
	keys := sortedKeys(m)
	return writeYAMLMapEntries(buf, m, keys, level, indent)
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeYAMLMapEntries writes map entries with the given keys.
func writeYAMLMapEntries(buf *bytes.Buffer, m map[string]any, keys []string, level, indent int) error {
	first := true
	for _, key := range keys {
		val := m[key]
		if !first {
			buf.WriteString(strings.Repeat(" ", level*indent))
		}
		first = false

		writeYAMLKey(buf, key)
		buf.WriteString(":")

		if err := writeYAMLMapEntry(buf, val, level, indent); err != nil {
			return err
		}
	}
	return nil
}

// writeYAMLMapEntry writes a single map entry value.
func writeYAMLMapEntry(buf *bytes.Buffer, val any, level, indent int) error {
	switch v := val.(type) {
	case map[string]any:
		if len(v) == 0 {
			buf.WriteString(" {}\n")
		} else {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat(" ", (level+1)*indent))
			if err := writeYAMLMap(buf, v, level+1, indent); err != nil {
				return err
			}
		}
	case []any:
		if len(v) == 0 {
			buf.WriteString(" []\n")
		} else {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat(" ", (level+1)*indent))
			if err := writeYAMLWithIndent(buf, v, level+1, indent); err != nil {
				return err
			}
		}
	default:
		buf.WriteString(" ")
		if err := writeYAMLWithIndent(buf, val, level+1, indent); err != nil {
			return err
		}
		buf.WriteString("\n")
	}
	return nil
}

// writeYAMLKey writes a YAML key, quoting if necessary.
func writeYAMLKey(buf *bytes.Buffer, key string) {
	if needsQuoting(key) {
		buf.WriteString("\"")
		buf.WriteString(escapeString(key))
		buf.WriteString("\"")
	} else {
		buf.WriteString(key)
	}
}

// writeYAMLString writes a string value, handling multiline and special characters.
func writeYAMLString(buf *bytes.Buffer, s string) {
	if s == "" {
		buf.WriteString("\"\"")
		return
	}

	if needsQuoting(s) {
		buf.WriteString("\"")
		buf.WriteString(escapeString(s))
		buf.WriteString("\"")
		return
	}

	buf.WriteString(s)
}

// needsQuoting returns true if the string needs to be quoted in YAML.
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	if isYAMLSpecialValue(s) {
		return true
	}
	if needsQuotingFirstChar(s[0]) {
		return true
	}
	if containsSpecialChar(s) {
		return true
	}
	return needsQuotingAsNumber(s)
}

// isYAMLSpecialValue checks if the string is a YAML special value.
func isYAMLSpecialValue(s string) bool {
	lower := strings.ToLower(s)
	switch lower {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}
	return false
}

// needsQuotingFirstChar checks if the first character requires quoting.
func needsQuotingFirstChar(first byte) bool {
	switch first {
	case '-', ':', '?', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`', '#', '{', '[', ',', ' ':
		return true
	}
	return false
}

// containsSpecialChar checks if the string contains characters that require quoting.
func containsSpecialChar(s string) bool {
	for _, r := range s {
		switch r {
		case ':', '#', '\n', '\r', '\t', '\\', '"', '\'':
			return true
		}
	}
	return false
}

// needsQuotingAsNumber checks if a numeric-looking string needs quoting.
func needsQuotingAsNumber(s string) bool {
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		// Quote strings with leading zeros (except "0" and "0.x")
		if strings.HasPrefix(s, "0") && len(s) > 1 && s[1] != '.' {
			return true
		}
	}
	return false
}

// escapeString escapes special characters in a string for YAML.
func escapeString(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// isComplexType returns true if the value is a map or non-empty slice.
func isComplexType(v any) bool {
	switch val := v.(type) {
	case map[string]any:
		return len(val) > 0
	case []any:
		return false // Arrays within arrays use inline format
	default:
		return false
	}
}

// FormatOpenAPIAsYAML formats an OpenAPI spec as YAML.
//
// Example:
//
//	spec, _ := Parse(jsonData)
//	yamlBytes, err := FormatOpenAPIAsYAML(spec)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("openapi.yaml", yamlBytes, 0644)
func FormatOpenAPIAsYAML(spec *OpenAPI) ([]byte, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}
	return MarshalYAML(spec)
}

// FormatAsyncAPIAsYAML formats an AsyncAPI spec as YAML.
func FormatAsyncAPIAsYAML(spec *AsyncAPI) ([]byte, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}
	return MarshalYAML(spec)
}

// FormatAsyncAPIV3AsYAML formats an AsyncAPI 3.0 spec as YAML.
func FormatAsyncAPIV3AsYAML(spec *AsyncAPIV3) ([]byte, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}
	return MarshalYAML(spec)
}
