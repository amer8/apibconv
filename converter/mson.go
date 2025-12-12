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
	if schema.IsObject() {
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
	if schema.IsArray() {
		buf.WriteString(indent)
		buf.WriteString("+ Attributes (array")
		if schema.Items != nil {
			itemType := schema.Items.TypeName()
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

// writeDataStructures writes the "## Data Structures" section and its content.
func writeDataStructures(buf *bytes.Buffer, schemas map[string]*Schema) {
	if len(schemas) == 0 {
		return
	}

	buf.WriteString("## Data Structures\n\n")

	// Sort schemas by name for deterministic output
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		schema := schemas[name]
		// Write the named type definition
		fmt.Fprintf(buf, "### %s (%s)\n", name, schema.TypeName()) // Example: ### User (object)
		if schema.Description != "" {
			buf.WriteString(schema.Description)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
		
		// If it's an object or array, write its attributes
		if schema.IsObject() || schema.IsArray() {
			writeMSON(buf, schema, 1) // Indent by 1 level
		}
		buf.WriteString("\n") // Extra newline after each data structure
	}
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
	typeStr := prop.TypeName()
	if prop.Ref != "" {
		typeStr = getRefName(prop.Ref)
	}
	if typeStr == "" {
		typeStr = "string" // default
	}
	buf.WriteString(typeStr)

	// Add typed array info: array[User]
	if typeStr == "array" && prop.Items != nil {
		itemType := prop.Items.TypeName()
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
	if prop.IsObject() && len(prop.Properties) > 0 {
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

// IsObject checks if a schema is an object type or has properties.
func (s *Schema) IsObject() bool {
	return s.TypeName() == TypeObject || len(s.Properties) > 0
}

// IsArray checks if a schema is an array type or has items.
func (s *Schema) IsArray() bool {
	return s.TypeName() == TypeArray || s.Items != nil
}

// getRefName extracts the simple name from a reference string (e.g. "#/components/schemas/User" -> "User").
func getRefName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}
