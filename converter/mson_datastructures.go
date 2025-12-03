package converter

import (
	"bytes"
	"sort"
)

func writeDataStructures(buf *bytes.Buffer, schemas map[string]*Schema) {
	if len(schemas) == 0 {
		return
	}

	buf.WriteString("## Data Structures\n\n")

	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		schema := schemas[name]
		buf.WriteString("### ")
		buf.WriteString(name)

		typeStr := GetSchemaType(schema)
		if typeStr != "" {
			buf.WriteString(" (")
			buf.WriteString(typeStr)
			buf.WriteString(")")
		} else {
			buf.WriteString(" (object)")
		}
		buf.WriteString("\n")

		// Write properties
		if len(schema.Properties) > 0 {
			var propKeys []string
			for k := range schema.Properties {
				propKeys = append(propKeys, k)
			}
			sort.Strings(propKeys)

			for _, k := range propKeys {
				prop := schema.Properties[k]
				writeMSONProperty(buf, k, prop, schema.Required, 0)
			}
		}
		buf.WriteString("\n")
	}
}
