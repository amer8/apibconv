// Package apiblueprint implements the API Blueprint format parser and writer.
package apiblueprint

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strings"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Parser implements the format.Parser interface for API Blueprint.
type Parser struct {
	parseMarkdown bool
}

// ParserOption configures the Parser.
type ParserOption func(*Parser)

// WithMarkdownParsing enables or disables Markdown parsing for descriptions.
func WithMarkdownParsing(enable bool) ParserOption {
	return func(p *Parser) {
		p.parseMarkdown = enable
	}
}

// NewParser creates a new API Blueprint parser with optional configurations.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Parse parses an API Blueprint document from the given reader into a unified model.API.
func (p *Parser) Parse(ctx context.Context, r io.Reader) (*model.API, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(r)
	var content strings.Builder
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		content.WriteString(scanner.Text() + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return p.parseDocument(content.String())
}

// Format returns the format type for the parser.
func (p *Parser) Format() format.Format {
	return format.FormatAPIBlueprint
}

// SupportsVersion returns true for any version as API Blueprint is less strict about versions.
func (p *Parser) SupportsVersion(version string) bool {
	return true
}

// Regex patterns for API Blueprint parsing
var (
	reMetadata       = regexp.MustCompile(`^(\w+):\s*(.*)$`)
	reHeader         = regexp.MustCompile(`^#\s+(.*)$`)
	reGroup          = regexp.MustCompile(`^#+\s+Group\s+(.*)$`)
	reResource       = regexp.MustCompile(`^#+\s+(.*)\[(.*)\]$`)
	reAction         = regexp.MustCompile(`^#+\s+(.*)\[([A-Z]+)\]$`)
	reResponse       = regexp.MustCompile(`^\+\s+Response\s+(\d+)\s*\(?([^)]*)\)?$`)
	reAttributes     = regexp.MustCompile(`^\+\s+Attributes\s*\(?([^)]*)\)?$`)
	reDataStructures = regexp.MustCompile(`^#+\s+Data Structures$`)
	reNamedType      = regexp.MustCompile(`^##\s+(.*)\s*\((.*)\)$`) // ## Task (object)
)

func (p *Parser) parseDocument(content string) (*model.API, error) {
	api := model.NewAPI()
	api.Version = "1A" // Default version for API Blueprint

	lines := strings.Split(content, "\n")
	var currentPath string
	var currentGroup string
	var currentPathItem *model.PathItem
	var currentOperation *model.Operation
	var inDataStructures bool

	// Simplified state machine
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Metadata
		if matches := reMetadata.FindStringSubmatch(line); len(matches) > 0 {
			key := matches[1]
			val := matches[2]
			switch key {
			case "HOST":
				api.Servers = append(api.Servers, model.Server{URL: val})
			case "FORMAT":
				api.Version = val
			}
			continue
		}

		// API Name (First H1)
		if matches := reHeader.FindStringSubmatch(line); len(matches) > 0 && api.Info.Title == "" && !inDataStructures {
			api.Info.Title = matches[1]
			continue
		}

		// Data Structures Section
		if reDataStructures.MatchString(line) {
			inDataStructures = true
			currentGroup = ""
			currentPathItem = nil
			currentOperation = nil
			continue
		}

		// Named Type (Schema Definition) in Data Structures
		if inDataStructures {
			if matches := reNamedType.FindStringSubmatch(line); len(matches) > 0 {
				name := strings.TrimSpace(matches[1])
				// Parse properties
				schema := &model.Schema{Type: model.TypeObject, Properties: make(map[string]*model.Schema)}
				j := i + 1
				for j < len(lines) {
					nextLine := lines[j]
					nextLineTrimmed := strings.TrimSpace(nextLine)
					switch {
					case strings.HasPrefix(nextLineTrimmed, "+"):
						p.parseMSONProperty(nextLine, schema)
						j++
					case nextLineTrimmed == "":
						j++
					default:
						// End of structure properties by breaking inner loop
						// We need to break out of the inner loop (for j < len(lines))
						// Since we are inside a switch, we need a label or set j to break condition
						// But 'break' here only breaks switch.
						// Setting j to a value to break loop? No, that's messy.
						// Use a flag? Or goto?
						// Actually, the original code used 'break' which broke the loop because it wasn't in a switch.
						// So if I use switch, I must use a label for loop.
						goto EndProperties
					}
				}
			EndProperties:
				i = j - 1
				if api.Components.Schemas == nil {
					api.Components.Schemas = make(map[string]*model.Schema)
				}
				api.Components.Schemas[name] = schema
				continue
			}
		}

		// Resource Group (H2, usually)
		if matches := reGroup.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			currentGroup = strings.TrimSpace(matches[1])
			continue
		}

		// Action
		if matches := reAction.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			if currentPathItem == nil {
				continue
			}

			name := strings.TrimSpace(matches[1])
			method := strings.TrimSpace(matches[2])

			op := &model.Operation{
				Summary:   name,
				Responses: make(model.Responses),
			}

			if currentGroup != "" {
				op.Tags = []string{currentGroup}
			}

			currentPathItem.SetOperation(method, op)
			currentOperation = op
			continue
		}

		// Resource
		if matches := reResource.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			// Save previous path item if exists
			if currentPath != "" && currentPathItem != nil {
				api.AddPath(currentPath, currentPathItem)
			}

			// New Resource
			name := strings.TrimSpace(matches[1])
			uri := strings.TrimSpace(matches[2])
			currentPath = uri
			currentPathItem = &model.PathItem{
				Summary: name,
			}
			currentOperation = nil // Reset operation
			continue
		}

		// Response
		if matches := reResponse.FindStringSubmatch(line); len(matches) > 0 && currentOperation != nil {
			statusCode := matches[1]
			contentType := matches[2]
			if contentType == "" {
				contentType = "application/json"
			}

			resp := model.Response{
				Description: "", // Description usually follows
				Content: map[string]model.MediaType{
					contentType: {},
				},
			}

			// Check for body or attributes
			j := i + 1
			var bodyBuilder strings.Builder
			var msonProperties map[string]*model.Schema
			var isMSON bool
			var isArray bool
			var hadBody bool
			var refName string

			// Look ahead for Attributes
			if j < len(lines) {
				trimmedNext := strings.TrimSpace(lines[j])
				if matchesAttr := reAttributes.MatchString(trimmedNext); matchesAttr {
					isMSON = true
					// Check if Attributes has type definition/reference
					// e.g. + Attributes (array[Task])
					attrContent := reAttributes.FindStringSubmatch(trimmedNext)[1]
					if strings.Contains(attrContent, "array[") {
						isArray = true
						// Extract Ref
						start := strings.Index(attrContent, "array[") + 6
						end := strings.Index(attrContent[start:], "]")
						if end > -1 {
							refName = attrContent[start : start+end]
						}
					} else if attrContent != "" && attrContent != "object" {
						refName = attrContent
					}

					j++ // Skip Attributes header
					msonProperties = make(map[string]*model.Schema)
				}
			}

			for j < len(lines) {
				nextLine := lines[j]
				linePrefix := ""
				// Determine indentation level. 4 spaces for Attributes properties, 8 for body examples.
				if isMSON {
					linePrefix = "            " // 12 spaces if from start (4 for Response, 4 for Attributes, 4 for property)
				} else {
					linePrefix = "        " // 8 spaces for body example
				}

				if !strings.HasPrefix(nextLine, linePrefix) && strings.TrimSpace(nextLine) != "" {
					break // End of block due to de-indentation
				}

				if isMSON {
					if strings.TrimSpace(nextLine) != "" {
						p.parseMSONProperty(nextLine, &model.Schema{Properties: msonProperties})
					}
					j++
				} else {
					hadBody = true
					bodyBuilder.WriteString(strings.TrimSpace(nextLine) + "\n")
					j++
				}
			}
			i = j - 1 // Advance outer loop

			if isMSON {
				schema := &model.Schema{}
				if refName != "" {
					if isArray {
						schema.Type = model.TypeArray
						schema.Items = &model.Schema{
							Ref: "#/components/schemas/" + refName,
						}
					} else {
						schema.Ref = "#/components/schemas/" + refName
					}
				} else {
					schema.Type = model.TypeObject
					schema.Properties = msonProperties
				}
				resp.Content[contentType] = model.MediaType{
					Schema: schema,
				}
			} else if hadBody && bodyBuilder.String() != "" {
				resp.Content[contentType] = model.MediaType{
					Example: bodyBuilder.String(),
				}
			}

			currentOperation.AddResponse(statusCode, resp)
			continue
		}
	}

	// Add final path item
	if currentPath != "" && currentPathItem != nil {
		api.AddPath(currentPath, currentPathItem)
	}

	return api, nil
}

// parseMSONProperty parses a single MSON property line and adds it to the schema.
// Format: + name: value (type, attributes) - description
func (p *Parser) parseMSONProperty(line string, parent *model.Schema) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "+") {
		return
	}
	trimmed = strings.TrimPrefix(trimmed, "+")
	trimmed = strings.TrimSpace(trimmed)

	// Split name: value
	parts := strings.SplitN(trimmed, ":", 2)
	namePart := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	} else if strings.Contains(namePart, "(") {
		// handle case where no value is provided but type info might follow
		// e.g. + name (string)
		// Logic below handles rest splitting by '(' so parsing namePart is tricky if ':' is missing.
		// For simplicity, assume ':' exists or namePart contains everything.
		split := strings.SplitN(namePart, "(", 2)
		namePart = split[0]
		rest = "(" + split[1]
	}

	name := strings.TrimSpace(namePart)
	propSchema := &model.Schema{Type: model.TypeString} // Default

	// Parse type/attributes in parens ()
	if start := strings.Index(rest, "("); start > -1 {
		if end := strings.Index(rest, ")"); end > start {
			attrs := rest[start+1 : end]
			attrParts := strings.Split(attrs, ",")
			for _, attr := range attrParts {
				attr = strings.ToLower(strings.TrimSpace(attr))
				switch attr {
				case "required":
					parent.Required = append(parent.Required, name)
				case "optional":
					// default
				case "number":
					propSchema.Type = model.TypeNumber
				case "boolean":
					propSchema.Type = model.TypeBoolean
				case "array":
					propSchema.Type = model.TypeArray
				case "object":
					propSchema.Type = model.TypeObject
				case "string":
					propSchema.Type = model.TypeString
				default:
					// Could be custom type or format
					// propSchema.Type = model.TypeString // Fallback
				}
			}
			// Description might follow after )
			if len(rest) > end+1 {
				desc := strings.TrimSpace(rest[end+1:])
				if strings.HasPrefix(desc, "-") {
					propSchema.Description = strings.TrimSpace(strings.TrimPrefix(desc, "-"))
				}
			}
		}
	} else if idx := strings.Index(rest, "-"); idx > -1 {
		// No type info, check for description separator
		propSchema.Description = strings.TrimSpace(rest[idx+1:])
	}

	if parent.Properties == nil {
		parent.Properties = make(map[string]*model.Schema)
	}
	parent.Properties[name] = propSchema
}
