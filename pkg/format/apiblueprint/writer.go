package apiblueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/amer8/apibconv/internal/security"
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
	if err := security.CheckUnsupported(api, "API Blueprint"); err != nil {
		return err
	}
	if err := security.CheckServerOverrides(api, "API Blueprint"); err != nil {
		return err
	}
	for path := range api.Paths {
		item := api.Paths[path]
		for _, op := range []*model.Operation{item.Get, item.Post, item.Put, item.Delete, item.Patch, item.Head, item.Options, item.Trace} {
			if op == nil {
				continue
			}
			for code, response := range op.Responses {
				if len(response.Content) > 1 {
					return fmt.Errorf("path %q response %s has multiple content types that API Blueprint cannot represent without loss", path, code)
				}
				for _, media := range response.Content {
					if media.Schema != nil {
						if _, err := json.Marshal(media.Schema); err != nil {
							return fmt.Errorf("path %q response %s schema: %w", path, code, err)
						}
					}
				}
			}
		}
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
		case item.Head != nil:
			op = item.Head
		case item.Options != nil:
			op = item.Options
		case item.Trace != nil:
			op = item.Trace
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
		fmt.Fprintf(&builder, "\n# Group %s\n", group)
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
		fmt.Fprintf(&sb, "HOST: %s\n", api.Servers[0].URL)
	}
	sb.WriteString("\n")

	title := api.Info.Title
	if title == "" {
		title = "API Documentation"
	}
	fmt.Fprintf(&sb, "# %s\n", title)

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

	fmt.Fprintf(&sb, "\n## %s [%s]\n", summary, path)
	if item.Description != "" {
		sb.WriteString(item.Description + "\n")
	}

	// Write actions
	sb.WriteString(w.writeAction("GET", item.Get))
	sb.WriteString(w.writeAction("POST", item.Post))
	sb.WriteString(w.writeAction("PUT", item.Put))
	sb.WriteString(w.writeAction("DELETE", item.Delete))
	sb.WriteString(w.writeAction("PATCH", item.Patch))
	sb.WriteString(w.writeAction("HEAD", item.Head))
	sb.WriteString(w.writeAction("OPTIONS", item.Options))
	sb.WriteString(w.writeAction("TRACE", item.Trace))

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

	fmt.Fprintf(&sb, "\n### %s [%s]\n", summary, method)
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
		contentTypes := make([]string, 0, len(op.RequestBody.Content))
		for contentType := range op.RequestBody.Content {
			contentTypes = append(contentTypes, contentType)
		}
		sort.Strings(contentTypes)
		for _, contentType := range contentTypes {
			mediaType := op.RequestBody.Content[contentType]
			fmt.Fprintf(&sb, "\n+ Request (%s)\n", contentType)
			if mediaType.Schema != nil {
				w.writePayloadAttributes(&sb, mediaType.Schema)
			}
			if mediaType.Example != nil {
				sb.WriteString("    + Body\n")
				for _, line := range strings.Split(strings.TrimSpace(fmt.Sprint(mediaType.Example)), "\n") {
					fmt.Fprintf(&sb, "\n        %s", line)
				}
				sb.WriteString("\n")
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

		fmt.Fprintf(&sb, "\n+ Response %s (%s)\n", code, ct)
		if mt, ok := resp.Content[ct]; ok && mt.Schema != nil {
			w.writePayloadAttributes(&sb, mt.Schema)
			data, _ := json.Marshal(mt.Schema) // Checked before writing.
			fmt.Fprintf(&sb, "\n    + Schema\n\n        %s\n", data)
		}
		if resp.Description != "" {
			fmt.Fprintf(&sb, "\n    %s\n", resp.Description)
		}

		// Write response body example if available
		if mt, ok := resp.Content[ct]; ok && mt.Example != nil {
			if mt.Schema != nil {
				sb.WriteString("\n    + Body\n")
			}
			sb.WriteString("\n        ") // Indentation for body
			fmt.Fprint(&sb, mt.Example)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (w *Writer) writePayloadAttributes(sb *strings.Builder, schema *model.Schema) {
	typ := string(schema.Type)
	if schema.Ref != "" {
		typ = strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	} else if schema.Type == model.TypeArray && schema.Items != nil && schema.Items.Ref != "" {
		typ = "array[" + strings.TrimPrefix(schema.Items.Ref, "#/components/schemas/") + "]"
	}
	if typ == "" {
		typ = "object"
	}
	fmt.Fprintf(sb, "    + Attributes (%s)\n", typ)
	if schema.Ref == "" {
		sb.WriteString(w.writeMSONSchema(schema, 3))
	}
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
			typ := string(schema.Type)
			if typ == "" {
				typ = "object"
			}
			fmt.Fprintf(&sb, "\n### %s (%s)\n", name, typ)
			if schema.Description != "" {
				fmt.Fprintf(&sb, "    %s\n", schema.Description)
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
		fmt.Fprintf(&sb, "%s+ Attributes (array[%s])\n", indent, refName)
		return sb.String()
	}

	// Handle AllOf
	for _, allOfSchema := range schema.AllOf {
		if allOfSchema.Ref != "" {
			refName := strings.TrimPrefix(allOfSchema.Ref, "#/components/schemas/")
			fmt.Fprintf(&sb, "%s+ Attributes (array[%s])\n", indent, refName) // Assuming allOf with ref is like extending
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

			fmt.Fprintf(&sb, "%s+ %s (%s)%s\n", indent, propName, propType, requiredStr)
			if propSchema.Description != "" {
				fmt.Fprintf(&sb, "%s    %s\n", indent, propSchema.Description)
			}

			// Recursively write nested properties for objects or items for arrays
			if propSchema.Type == model.TypeObject || propSchema.Type == model.TypeArray {
				sb.WriteString(w.writeMSONSchema(propSchema, indentLevel+1))
			}
		}
	}

	return sb.String()
}
