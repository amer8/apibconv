package converter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// writeMSON writes the MSON representation of a schema to the buffer.
// indentLevel is the number of indentation levels (4 spaces each).
func writeMSON(buf *bytes.Buffer, schema *Schema, indentLevel int) {
	if schema == nil {
		return
	}

	indent := strings.Repeat("    ", indentLevel)

	// If it's a reference, we just print Attributes (RefName)
	if schema.Ref != "" {
		refName := getRefName(schema.Ref)
		buf.WriteString(indent)
		buf.WriteString("+ Attributes (")
		buf.WriteString(refName)
		buf.WriteString(")\n")
		return
	}

	// If it's an object with properties
	if isObject(schema) {
		buf.WriteString(indent)
		buf.WriteString("+ Attributes")

		// Add type info if it's explicitly an object
		// buf.WriteString(" (object)")
		// Actually "Attributes" implies object usually, or we list properties.

		buf.WriteString("\n")

		// Sort properties for deterministic output
		var keys []string
		for k := range schema.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			prop := schema.Properties[k]
			writeMSONProperty(buf, k, prop, schema.Required, indentLevel+1)
		}
		return
	}

	// If it's an array
	if isArray(schema) {
		buf.WriteString(indent)
		buf.WriteString("+ Attributes (array")
		if schema.Items != nil {
			itemType := SchemaType(schema.Items)
			if schema.Items.Ref != "" {
				buf.WriteString("[")
				buf.WriteString(getRefName(schema.Items.Ref))
				buf.WriteString("]")
			} else if itemType != "" {
				buf.WriteString("[")
				buf.WriteString(itemType)
				buf.WriteString("]")
			}
		}
		buf.WriteString(")\n")
		return
	}

	// Fallback for other types if they appear at root level (rare for Attributes)
	// Usually root is object or array
}

// writeMSONProperty writes a single MSON property definition to the buffer.
// It handles property name, type, required/optional status, description, default/example values,
// and recursively writes nested properties for object types.
func writeMSONProperty(buf *bytes.Buffer, name string, prop *Schema, required []string, indentLevel int) {
	indent := strings.Repeat("    ", indentLevel)
	buf.WriteString(indent)
	buf.WriteString("+ ")
	buf.WriteString(name)

	// Add example value if present
	if prop.Example != nil {
		buf.WriteString(": `")
		fmt.Fprintf(buf, "%v", prop.Example)
		buf.WriteString("`")
	}

	buf.WriteString(" (")

	// Type
	typeStr := SchemaType(prop)
	if prop.Ref != "" {
		typeStr = getRefName(prop.Ref)
	}
	if typeStr == "" {
		typeStr = "string" // default
	}
	buf.WriteString(typeStr)

	// Add typed array info: array[User]
	if typeStr == "array" && prop.Items != nil {
		itemType := SchemaType(prop.Items)
		if prop.Items.Ref != "" {
			itemType = getRefName(prop.Items.Ref)
		}
		if itemType != "" {
			buf.WriteString("[")
			buf.WriteString(itemType)
			buf.WriteString("]")
		}
	}

	// Required/Optional
	if isPropRequired(name, required) {
		buf.WriteString(", required")
	} else {
		buf.WriteString(", optional")
	}

	buf.WriteString(")")

	// Description
	if prop.Description != "" {
		buf.WriteString(" - ")
		buf.WriteString(prop.Description)
	}

	buf.WriteString("\n")

	// Nested properties (if object)
	if isObject(prop) && len(prop.Properties) > 0 {
		var keys []string
		for k := range prop.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			subProp := prop.Properties[k]
			writeMSONProperty(buf, k, subProp, prop.Required, indentLevel+1)
		}
	}
}

// isPropRequired checks if a property name is in the required list.
func isPropRequired(name string, required []string) bool {
	for _, r := range required {
		if r == name {
			return true
		}
	}
	return false
}

// isObject checks if a schema is an object type or has properties.
func isObject(s *Schema) bool {
	return SchemaType(s) == TypeObject || len(s.Properties) > 0
}

// isArray checks if a schema is an array type or has items.
func isArray(s *Schema) bool {
	return SchemaType(s) == TypeArray || s.Items != nil
}

// getRefName extracts the simple name from a reference string (e.g. "#/components/schemas/User" -> "User").
func getRefName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}
