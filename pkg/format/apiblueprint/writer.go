package apiblueprint

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Writer implements the format.Writer interface for API Blueprint.
type Writer struct {
	format string
}

// WriterOption configures the Writer.
type WriterOption func(*Writer)

// WithFormat sets the output format for the writer (currently unused for API Blueprint).
func WithFormat(f string) WriterOption {
	return func(w *Writer) {
		w.format = f
	}
}

// NewWriter creates a new API Blueprint writer with optional configurations.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Write writes the unified API model to an API Blueprint document.
func (w *Writer) Write(ctx context.Context, api *model.API, wr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var builder strings.Builder

	builder.WriteString(w.writeHeader(api))

	// Group paths by Tag
	groups := make(map[string][]string) // tag -> paths

	// Collect and sort all paths
	paths := make([]string, 0, len(api.Paths))
	for k := range api.Paths {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := api.Paths[path]
		tag := "Resources"

		// Heuristic: check first available operation for a tag
		var op *model.Operation
		switch {
		case item.Get != nil:
			op = item.Get
		case item.Post != nil:
			op = item.Post
		case item.Put != nil:
			op = item.Put
		case item.Delete != nil:
			op = item.Delete
		case item.Patch != nil:
			op = item.Patch
		}

		if op != nil && len(op.Tags) > 0 {
			tag = op.Tags[0]
		}

		groups[tag] = append(groups[tag], path)
	}

	// Sort group names
	groupNames := make([]string, 0, len(groups))
	for k := range groups {
		groupNames = append(groupNames, k)
	}
	sort.Strings(groupNames)

	for _, group := range groupNames {
		builder.WriteString(fmt.Sprintf("\n# Group %s\n", group))
		for _, path := range groups[group] {
			item := api.Paths[path]
			displayPath := path
			if !strings.HasPrefix(displayPath, "/") {
				displayPath = "/" + displayPath
			}
			builder.WriteString(w.writeResource(displayPath, &item))
		}
	}

	// Write Data Structures
	builder.WriteString(w.writeSchemas(api))

	_, err := wr.Write([]byte(builder.String()))
	return err
}

// Format returns the format type for the writer.
func (w *Writer) Format() format.Format {
	return format.FormatAPIBlueprint
}

// Version returns the API Blueprint version being written.
func (w *Writer) Version() string {
	return "1A"
}

func (w *Writer) writeHeader(api *model.API) string {
	var sb strings.Builder
	sb.WriteString("FORMAT: 1A\n")
	if len(api.Servers) > 0 {
		sb.WriteString(fmt.Sprintf("HOST: %s\n", api.Servers[0].URL))
	}
	sb.WriteString("\n")

	title := api.Info.Title
	if title == "" {
		title = "API Documentation"
	}
	sb.WriteString(fmt.Sprintf("# %s\n", title))

	if api.Info.Description != "" {
		sb.WriteString(api.Info.Description + "\n")
	}

	return sb.String()
}

func (w *Writer) writeResource(path string, item *model.PathItem) string {
	var sb strings.Builder

	summary := item.Summary
	if summary == "" {
		summary = "Resource"
	}

	sb.WriteString(fmt.Sprintf("\n## %s [%s]\n", summary, path))
	if item.Description != "" {
		sb.WriteString(item.Description + "\n")
	}

	// Write actions
	sb.WriteString(w.writeAction("GET", item.Get))
	sb.WriteString(w.writeAction("POST", item.Post))
	sb.WriteString(w.writeAction("PUT", item.Put))
	sb.WriteString(w.writeAction("DELETE", item.Delete))
	sb.WriteString(w.writeAction("PATCH", item.Patch))

	return sb.String()
}

func (w *Writer) writeAction(method string, op *model.Operation) string {
	if op == nil {
		return ""
	}
	var sb strings.Builder

	summary := op.Summary
	if summary == "" {
		summary = method // Default summary
	}

	sb.WriteString(fmt.Sprintf("\n### %s [%s]\n", summary, method))
	if op.Description != "" {
		sb.WriteString(op.Description + "\n")
	}

	// Parameters
	if len(op.Parameters) > 0 {
		sb.WriteString("\n+ Parameters\n")
		for _, param := range op.Parameters {
			// Parameter Name (type, format) - Description (required)
			paramLine := fmt.Sprintf("    + %s", param.Name)

			if param.Schema != nil {
				typeInfo := ""
				if param.Schema.Type != "" {
					typeInfo = string(param.Schema.Type)
				}
				if param.Schema.Format != "" {
					if typeInfo != "" {
						typeInfo += ", "
					}
					typeInfo += param.Schema.Format
				}
				if typeInfo != "" {
					paramLine += fmt.Sprintf(" (%s)", typeInfo)
				}
			}

			if param.Required {
				paramLine += " (required)"
			}

			if param.Description != "" {
				paramLine += fmt.Sprintf(" - %s", param.Description)
			}
			sb.WriteString(paramLine + "\n")
		}
	}

	// Request Body
	if op.RequestBody != nil {
		sb.WriteString("\n+ Request\n")
		// Assuming JSON content for simplicity based on previous examples
		for contentType, mediaType := range op.RequestBody.Content {
			sb.WriteString(fmt.Sprintf("    + Body (%s)\n", contentType))
			if mediaType.Example != nil {
				sb.WriteString("            ")
				sb.WriteString(fmt.Sprintf("%v\n", mediaType.Example))
				sb.WriteString("\n")
			}
			// If schema is present, we should reference it if Data Structures are implemented
			if mediaType.Schema != nil && mediaType.Schema.Ref != "" {
				sb.WriteString(fmt.Sprintf("            + Attributes (array[%s])\n", strings.TrimPrefix(mediaType.Schema.Ref, "#/components/schemas/")))
			}
		}
	}

	// Responses
	// Sort responses
	codes := make([]string, 0, len(op.Responses))
	for k := range op.Responses {
		codes = append(codes, k)
	}
	sort.Strings(codes)

	for _, code := range codes {
		resp := op.Responses[code]
		ct := "application/json" // Default
		// Pick first content type
		for k := range resp.Content {
			ct = k
			break
		}

		sb.WriteString(fmt.Sprintf("\n+ Response %s (%s)\n", code, ct))
		if resp.Description != "" {
			sb.WriteString(fmt.Sprintf("\n    %s\n", resp.Description))
		}

		// Write response body example if available
		if mt, ok := resp.Content[ct]; ok && mt.Example != nil {
			sb.WriteString("\n        ") // Indentation for body
			sb.WriteString(fmt.Sprintf("%v", mt.Example))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (w *Writer) writeSchemas(api *model.API) string {
	var sb strings.Builder
	if len(api.Components.Schemas) > 0 {
		sb.WriteString("\n## Data Structures\n")

		// Sort schema names for deterministic output
		schemaNames := make([]string, 0, len(api.Components.Schemas))
		for k := range api.Components.Schemas {
			schemaNames = append(schemaNames, k)
		}
		sort.Strings(schemaNames)

		for _, name := range schemaNames {
			schema := api.Components.Schemas[name]
			sb.WriteString(fmt.Sprintf("\n### %s (object)\n", name))
			if schema.Description != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", schema.Description))
			}
			sb.WriteString(w.writeMSONSchema(schema, 1)) // Start with 1 indentation level
		}
	}
	return sb.String()
}

func (w *Writer) writeMSONSchema(schema *model.Schema, indentLevel int) string {
	var sb strings.Builder
	indent := strings.Repeat("    ", indentLevel)

	// Handle $ref
	if schema.Ref != "" {
		// If it's a local reference, just use the name
		refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		sb.WriteString(fmt.Sprintf("%s+ Attributes (array[%s])\n", indent, refName))
		return sb.String()
	}

	// Handle AllOf
	for _, allOfSchema := range schema.AllOf {
		if allOfSchema.Ref != "" {
			refName := strings.TrimPrefix(allOfSchema.Ref, "#/components/schemas/")
			sb.WriteString(fmt.Sprintf("%s+ Attributes (array[%s])\n", indent, refName)) // Assuming allOf with ref is like extending
		} else {
			// Inline allOf schema properties
			sb.WriteString(w.writeMSONSchema(allOfSchema, indentLevel)) // Recursive call for inline schemas
		}
	}

	// Properties
	if len(schema.Properties) > 0 {
		// Sort properties for deterministic output
		propNames := make([]string, 0, len(schema.Properties))
		for k := range schema.Properties {
			propNames = append(propNames, k)
		}
		sort.Strings(propNames)

		for _, propName := range propNames {
			propSchema := schema.Properties[propName]
			propType := string(propSchema.Type)
			if propType == "" {
				propType = "string"
			} // Default to string if type is unknown

			// Check if property is required
			isRequired := false
			for _, reqProp := range schema.Required {
				if reqProp == propName {
					isRequired = true
					break
				}
			}

			requiredStr := ""
			if isRequired {
				requiredStr = " (required)"
			}

			line := fmt.Sprintf("%s+ %s (%s)%s\n", indent, propName, propType, requiredStr)
			sb.WriteString(line)
			if propSchema.Description != "" {
				sb.WriteString(fmt.Sprintf("%s    %s\n", indent, propSchema.Description))
			}

			// Recursively write nested properties for objects or items for arrays
			if propSchema.Type == model.TypeObject || propSchema.Type == model.TypeArray {
				sb.WriteString(w.writeMSONSchema(propSchema, indentLevel+1))
			}
		}
	}

	return sb.String()
}
