package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// UnmarshalYAML parses YAML data into the value pointed to by v.
// It uses a simplified YAML parser suitable for API specifications.
//
// Limitations:
//   - Supports common YAML features used in OpenAPI/AsyncAPI.
//   - Does not support complex keys, anchors, or tags.
//   - Assumes consistent indentation (spaces only).
func UnmarshalYAML(data []byte, v interface{}) error {
	// 1. Parse YAML to generic interface{} (map/slice/scalar)
	parsed, err := parseYAML(data)
	if err != nil {
		return err
	}

	// 2. Convert generic structure to JSON bytes
	// This allows us to use json.Unmarshal to populate the target struct,
	// handling all the struct tag mapping for us.
	jsonBytes, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("failed to convert parsed YAML to JSON: %w", err)
	}

	// 3. Unmarshal JSON into the target struct
	return json.Unmarshal(jsonBytes, v)
}

// UnmarshalYAMLReader parses YAML from an io.Reader.
func UnmarshalYAMLReader(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return UnmarshalYAML(data, v)
}

// ============================================================================
// Simplified YAML Parser
// ============================================================================

func parseYAML(data []byte) (interface{}, error) {
	parser := newYamlParser(data)
	return parser.parse()
}

type yamlParser struct {
	lines       []string
	currentLine int
}

func newYamlParser(data []byte) *yamlParser {
	// Normalize line endings
	s := string(data)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	return &yamlParser{
		lines:       lines,
		currentLine: 0,
	}
}

func (p *yamlParser) parse() (interface{}, error) {
	// Detect if root is array or map based on first meaningful line
	line, indent, _, ok := p.peek()
	if !ok {
		return make(map[string]interface{}), nil
	}

	// If starts with dash, it's an array
	if strings.HasPrefix(strings.TrimSpace(line), "-") {
		return p.parseArray(indent)
	}

	// Default to map
	return p.parseMap(indent)
}

// parseMap parses a YAML map at the given indentation level.
func (p *yamlParser) parseMap(minIndent int) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for {
		_, indent, trimmed, ok := p.peek()
		// Stop if EOF or indentation decreases (end of block)
		if !ok || indent < minIndent {
			break
		}

		// Skip comments and empty lines
		if shouldSkipLine(trimmed) {
			p.advance()
			continue
		}

		// Check for array item marker at map level (potential mixed content or error)
		if strings.HasPrefix(trimmed, "-") {
			if indent == minIndent {
				// Treat as end of map if strictly indented
				return result, nil
			}
			break
		}

		// Parse key-value pair
		key, val, err := p.parseMapEntry(indent, trimmed)
		if err != nil {
			return nil, err
		}
		if key != "" {
			result[key] = val
		}
	}

	return result, nil
}

// parseMapEntry parses a single key-value pair in a map.
func (p *yamlParser) parseMapEntry(indent int, trimmed string) (key string, val interface{}, err error) {
	// Expect "key: value"
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) < 1 {
		p.advance()
		return "", nil, nil
	}

	// Handle keys
	key = unquoteYAMLString(strings.TrimSpace(parts[0]))

	// Move past this line
	p.advance()

	// Check for inline value
	if len(parts) == 2 {
		valueStr := strings.TrimSpace(parts[1])
		if valueStr != "" && !strings.HasPrefix(valueStr, "#") {
			val, err = p.parseInlineValue(indent, valueStr)
			if err != nil {
				return "", nil, err
			}
			return key, val, nil
		}
	}

	// If no inline value, look for nested content on next lines
	val, err = p.parseNestedValue(indent)
	if err != nil {
		return "", nil, err
	}

	return key, val, nil
}

// parseInlineValue parses a value found on the same line as the key.
func (p *yamlParser) parseInlineValue(indent int, valueStr string) (interface{}, error) {
	// Check for block scalar indicators
	if isBlockScalar(valueStr) {
		return p.parseBlockScalar(indent+1, valueStr) // Indent must be > key
	}
	// Inline scalar or flow structure
	return parseScalarOrFlow(valueStr)
}

// parseNestedValue parses a value starting on the next line (indented).
func (p *yamlParser) parseNestedValue(parentIndent int) (interface{}, error) {
	_, nextIndent, nextTrimmed, nextOk := p.peek()
	if !nextOk {
		return nil, nil
	}

	if nextIndent > parentIndent {
		// Nested content
		if strings.HasPrefix(nextTrimmed, "-") {
			return p.parseArray(nextIndent)
		}
		return p.parseMap(nextIndent)
	}

	return nil, nil
}

// parseArray parses a YAML array at the given indentation level.
func (p *yamlParser) parseArray(minIndent int) ([]interface{}, error) {
	result := make([]interface{}, 0)

	for {
		_, indent, trimmed, ok := p.peek()
		if !ok || indent < minIndent {
			break
		}

		if !strings.HasPrefix(trimmed, "-") {
			break
		}

		val, err := p.parseArrayItem(indent, trimmed)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}

	return result, nil
}

// parseArrayItem parses a single item in an array.
func (p *yamlParser) parseArrayItem(indent int, trimmed string) (interface{}, error) {
	content := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))

	// Case 1: "- value" (Scalar or Flow) or "- key: value" (Compact Map)
	if content != "" {
		// Check for compact map "- key: value"
		if isCompactMap(content) {
			return p.parseCompactMapItem(indent, content)
		}

		// Simple scalar or flow object
		p.advance()
		return parseScalarOrFlow(content)
	}

	// Case 2: "-" (followed by newline, content indented)
	p.advance()
	return p.parseNestedValue(indent)
}

// parseCompactMapItem parses an array item that is a compact map (starts on the same line as dash).
func (p *yamlParser) parseCompactMapItem(indent int, content string) (map[string]interface{}, error) {
	p.advance() // Consume dash line

	// Parse the first key-value pair from the dash line
	parts := strings.SplitN(content, ":", 2)
	key := unquoteYAMLString(strings.TrimSpace(parts[0]))

	mapItem := make(map[string]interface{})

	// Value of the first key
	valStr := strings.TrimSpace(parts[1])
	if valStr != "" {
		val, err := parseScalarOrFlow(valStr)
		if err != nil {
			return nil, err
		}
		mapItem[key] = val
	} else {
		// Nested value for this key?
		// Simply set nil for now as placeholder, ignoring recursion for simplicity in this specific structure
		mapItem[key] = nil
	}

	// Look for more keys for this map item
	// They must align with the first key (indent + 2 spaces usually)
	itemIndent := indent + 2

	// Try to parse more map entries at this level
	moreMap, err := p.parseMap(itemIndent)
	if err == nil {
		for k, v := range moreMap {
			mapItem[k] = v
		}
	}

	return mapItem, nil
}

// parseBlockScalar handles | and > blocks.
func (p *yamlParser) parseBlockScalar(minIndent int, indicator string) (string, error) {
	var buf bytes.Buffer
	isFolded := strings.HasPrefix(indicator, ">")

	first := true
	for {
		_, indent, trimmed, ok := p.peek()
		if !ok {
			break
		}

		// Empty lines are preserved
		if trimmed == "" {
			buf.WriteString("\n")
			p.advance()
			continue
		}

		// If indentation matches or is less than key's indent, block ended
		if indent < minIndent {
			break
		}

		content := trimmed
		p.advance()

		if !first {
			if isFolded {
				buf.WriteString(" ")
			} else {
				buf.WriteString("\n")
			}
		}
		buf.WriteString(content)

		first = false
	}

	return buf.String(), nil
}

// Helpers

func (p *yamlParser) peek() (line string, indent int, trimmed string, ok bool) {
	for p.currentLine < len(p.lines) {
		line = p.lines[p.currentLine]
		trimmed = strings.TrimSpace(line)

		if shouldSkipLine(trimmed) {
			p.currentLine++
			continue
		}

		indent = 0
		for i, c := range line {
			if c != ' ' {
				indent = i
				break
			}
		}
		return line, indent, trimmed, true
	}
	return "", 0, "", false
}

func (p *yamlParser) advance() {
	p.currentLine++
}

func shouldSkipLine(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

func isBlockScalar(s string) bool {
	return s == "|" || s == ">" || strings.HasPrefix(s, "|") || strings.HasPrefix(s, ">")
}

func isCompactMap(s string) bool {
	return strings.Contains(s, ":") && !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[")
}

func parseScalarOrFlow(s string) (interface{}, error) {
	// Check for flow-style map/array (JSON-like)
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		var v interface{}
		// Try parsing as JSON
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v, nil
		}
	}

	return parseScalar(s), nil
}

func parseScalar(s string) interface{} {
	// 1. Check for quoted string
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) ||
		(strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		return unquoteYAMLString(s)
	}

	// 2. Check keywords
	switch s {
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	case "null", "Null", "NULL", "~":
		return nil
	}

	// 3. Check numbers
	// Try int first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(i) // Use float64 to match JSON default unmarshal
	}
	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// 4. Default to string
	return s
}

func unquoteYAMLString(s string) string {
		if len(s) >= 2 {
			if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
				// Remove quotes
				inner := s[1 : len(s)-1]
				// Handle escapes if double quoted
				if s[0] == '"' {
					// Simple unescape
					inner = strings.ReplaceAll(inner, `\"`, `"`)
					inner = strings.ReplaceAll(inner, `\n`, "\n")
					inner = strings.ReplaceAll(inner, `\r`, "\r")
					inner = strings.ReplaceAll(inner, `\t`, "\t")
					inner = strings.ReplaceAll(inner, `\\`, `\`)
				}
				return inner
			}
		}
		return s
}
