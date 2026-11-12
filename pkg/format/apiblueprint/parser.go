// Package apiblueprint implements the API Blueprint format parser and writer.
package apiblueprint

import (
	"bufio"
	"context"
	"fmt"
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

// WithMarkdownParsing preserves Markdown descriptions on the API, resources, and actions.
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

// SupportsVersion reports whether the parser handles the API Blueprint version.
func (p *Parser) SupportsVersion(version string) bool {
	return version == "1A"
}

// Regex patterns for API Blueprint parsing
var (
	reMetadata       = regexp.MustCompile(`^(\w+):\s*(.*)$`)
	reHeader         = regexp.MustCompile(`^#\s+(.*)$`)
	reGroup          = regexp.MustCompile(`^#+\s+Group\s+(.*)$`)
	reResource       = regexp.MustCompile(`^#+\s+(.*)\[(.*)\]$`)
	reAction         = regexp.MustCompile(`^#+\s+(.*)\[([A-Z]+)\]$`)
	reRequest        = regexp.MustCompile(`^\+\s+Request(?:\s+[^()]*)?\s*\(?([^)]*)\)?$`)
	reResponse       = regexp.MustCompile(`^\+\s+Response\s+(\d+)\s*\(?([^)]*)\)?$`)
	reAttributes     = regexp.MustCompile(`^\+\s+Attributes\s*\(?([^)]*)\)?$`)
	reDataStructures = regexp.MustCompile(`^#+\s+Data Structures$`)
	reNamedType      = regexp.MustCompile(`^#{2,3}\s+(.*)\s*\((.*)\)$`)
	reParameterAttrs = regexp.MustCompile(`\(([^)]*)\)`)
)

//nolint:gocyclo // Parsing depends on the current Markdown section and payload.
func (p *Parser) parseDocument(content string) (*model.API, error) {
	api := model.NewAPI()
	assetSchemas := make(map[string]*model.Schema)
	generatedSchemas := make(map[string]bool)
	api.Version = "1A" // Default version for API Blueprint

	lines := strings.Split(content, "\n")
	var currentPath string
	var currentGroup string
	var currentPathItem *model.PathItem
	var currentOperation *model.Operation
	var inDataStructures bool
	var description *string
	var descriptionBreak bool

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if p.parseMarkdown && description != nil && *description != "" {
				descriptionBreak = true
			}
			continue
		}

		// Metadata
		if matches := reMetadata.FindStringSubmatch(line); len(matches) > 0 {
			key := matches[1]
			val := matches[2]
			switch key {
			case "HOST":
				api.Servers = append(api.Servers, model.Server{URL: val})
				continue
			case "FORMAT":
				api.Version = val
				continue
			}
			if !p.parseMarkdown || description == nil {
				continue
			}
		}

		// API Name (First H1)
		if matches := reHeader.FindStringSubmatch(line); len(matches) > 0 && api.Info.Title == "" && !inDataStructures {
			api.Info.Title = matches[1]
			description = &api.Info.Description
			continue
		}

		// Data Structures Section
		if reDataStructures.MatchString(line) {
			if currentPath != "" && currentPathItem != nil {
				api.AddPath(currentPath, currentPathItem)
			}
			inDataStructures = true
			description = nil
			currentGroup = ""
			currentPathItem = nil
			currentOperation = nil
			continue
		}

		if inDataStructures {
			if matches := reNamedType.FindStringSubmatch(line); len(matches) > 0 {
				name := strings.TrimSpace(matches[1])
				schema, next := parseNamedStructure(lines, i+1, strings.TrimSpace(matches[2]))
				api.Components.AddSchema(name, schema)
				i = next - 1
				continue
			}
		}

		// Resource Group (H2, usually)
		if matches := reGroup.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			currentGroup = strings.TrimSpace(matches[1])
			description = nil
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
			description = &op.Description
			continue
		}

		// Resource
		if matches := reResource.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			if currentPath != "" && currentPathItem != nil {
				api.AddPath(currentPath, currentPathItem)
			}

			name := strings.TrimSpace(matches[1])
			uri := strings.TrimSpace(matches[2])
			currentPath = uri
			currentPathItem = &model.PathItem{
				Summary: name,
			}
			currentOperation = nil
			description = &currentPathItem.Description
			continue
		}

		// Parameters may be declared at either resource or action scope.
		if line == "+ Parameters" && currentPathItem != nil {
			description = nil
			indent := indentation(lines[i])
			parameterIndent := -1
			j := i + 1
			for ; j < len(lines); j++ {
				raw := lines[j]
				trimmed := strings.TrimSpace(raw)
				if trimmed == "" {
					continue
				}
				if indentation(raw) <= indent {
					break
				}
				if !strings.HasPrefix(trimmed, "+ ") {
					continue
				}
				if parameterIndent < 0 {
					parameterIndent = indentation(raw)
				}
				if indentation(raw) != parameterIndent {
					continue
				}
				param := parseBlueprintParameter(trimmed, currentPath)
				if param.Name == "" {
					continue
				}
				if currentOperation != nil {
					currentOperation.Parameters = append(currentOperation.Parameters, param)
				} else {
					currentPathItem.Parameters = append(currentPathItem.Parameters, param)
				}
			}
			i = j - 1
			continue
		}

		// Request
		if matches := reRequest.FindStringSubmatch(line); len(matches) > 0 && currentOperation != nil {
			description = nil
			contentType := strings.TrimSpace(matches[1])
			if contentType == "" {
				contentType = "application/json"
			}
			media, next, err := p.parseRequestPayload(lines, i+1, assetSchemas, generatedSchemas)
			if err != nil {
				return nil, fmt.Errorf("request JSON schema: %w", err)
			}
			if currentOperation.RequestBody == nil {
				currentOperation.RequestBody = &model.RequestBody{Content: make(map[string]model.MediaType)}
			}
			currentOperation.RequestBody.Content[contentType] = media
			i = next - 1
			continue
		}

		// Response
		if matches := reResponse.FindStringSubmatch(line); len(matches) > 0 && currentOperation != nil {
			description = nil
			var err error
			i, err = p.parseResponse(lines, i, matches, currentOperation, assetSchemas, generatedSchemas)
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.parseMarkdown && description != nil {
			if *description != "" {
				*description += "\n"
				if descriptionBreak {
					*description += "\n"
				}
			}
			*description += strings.TrimRight(lines[i], "\r")
			descriptionBreak = false
		}
	}

	if currentPath != "" && currentPathItem != nil {
		api.AddPath(currentPath, currentPathItem)
	}
	for name, schema := range assetSchemas {
		if generatedSchemas[name] && api.Components.Schemas[name] != nil {
			return nil, fmt.Errorf("generated schema name %q conflicts with a data structure", name)
		}
		api.Components.Schemas[name] = schema
	}

	return api, nil
}

// parseNamedStructure reads a Data Structures entry up to the next section.
func parseNamedStructure(lines []string, start int, typ string) (schema *model.Schema, next int) {
	schema = &model.Schema{Type: model.TypeObject, Properties: make(map[string]*model.Schema)}
	if strings.HasPrefix(typ, "array[") && strings.HasSuffix(typ, "]") {
		schema.Type, schema.Properties = model.TypeArray, nil
		schema.Items = msonArrayItem(typ[6 : len(typ)-1])
	} else if isPrimitiveType(typ) {
		schema.Type = model.SchemaType(typ)
	}

	properties := newMSONParser(schema)
	for i := start; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "+"):
			properties.add(line)
		case trimmed == "":
		case strings.HasPrefix(line, "    ") && !strings.HasPrefix(trimmed, "#"):
			if schema.Description == "" {
				schema.Description = trimmed
			}
		default:
			return schema, i
		}
	}
	return schema, len(lines)
}

func (p *Parser) parseResponse(lines []string, i int, matches []string, currentOperation *model.Operation, assetSchemas map[string]*model.Schema, generatedSchemas map[string]bool) (int, error) {
	statusCode := matches[1]
	contentType := matches[2]
	if contentType == "" {
		contentType = "application/json"
	}

	resp := model.Response{
		Content: map[string]model.MediaType{
			contentType: {},
		},
	}

	j := i + 1
	var bodyBuilder strings.Builder
	var msonSchema *model.Schema
	var isMSON bool
	var isArray bool
	var refName string
	var attributeType string
	var properties *msonParser

	if j < len(lines) {
		trimmedNext := strings.TrimSpace(lines[j])
		if matches := reAttributes.FindStringSubmatch(trimmedNext); len(matches) > 0 {
			isMSON = true
			attrContent := matches[1]
			attributeType = attrContent
			if strings.Contains(attrContent, "array[") {
				isArray = true
				start := strings.Index(attrContent, "array[") + 6
				end := strings.Index(attrContent[start:], "]")
				if end > -1 {
					refName = attrContent[start : start+end]
				}
			} else if attrContent != "" && attrContent != "object" && !isPrimitiveType(attrContent) {
				refName = attrContent
			}

			j++ // Skip Attributes header
			msonSchema = &model.Schema{Properties: make(map[string]*model.Schema)}
			properties = newMSONParser(msonSchema)
		}
	}

	// Attributes are nested one level deeper than body examples.
	linePrefix := "        "
	if isMSON {
		linePrefix = "            "
	}
	for j < len(lines) {
		line := lines[j]
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(line, linePrefix) && trimmed != "" {
			break
		}
		if isMSON {
			if trimmed != "" {
				properties.add(line)
			}
		} else {
			bodyBuilder.WriteString(trimmed + "\n")
		}
		j++
	}
	i = j - 1 // Advance outer loop

	if isMSON {
		schema := &model.Schema{}
		if refName != "" {
			if isArray {
				schema.Type = model.TypeArray
				schema.Items = msonArrayItem(refName)
				if schema.Items.Type == model.TypeObject {
					schema.Items.Properties = msonSchema.Properties
					schema.Items.Required = msonSchema.Required
				}
			} else {
				schema.Ref = "#/components/schemas/" + refName
			}
		} else {
			schema.Type = model.TypeObject
			if isPrimitiveType(attributeType) {
				schema.Type = model.SchemaType(attributeType)
			}
			schema.Properties = msonSchema.Properties
			schema.Required = msonSchema.Required
		}
		resp.Content[contentType] = model.MediaType{
			Schema: schema,
		}
	} else if bodyBuilder.Len() > 0 {
		example := bodyBuilder.String()
		if strings.TrimSpace(example) == "" {
			example = ""
		}
		resp.Content[contentType] = model.MediaType{
			Example: example,
		}
	}

	// A Blueprint Schema asset preserves JSON Schema constraints that MSON
	// cannot express. Prefer it to the accompanying Attributes section.
	if j < len(lines) && strings.TrimSpace(lines[j]) == "+ Schema" {
		j++
		var raw strings.Builder
		for j < len(lines) {
			line := lines[j]
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "        ") {
				break
			}
			raw.WriteString(strings.TrimSpace(line))
			j++
		}
		schema, err := parseSchemaAsset([]byte(raw.String()), assetSchemas, generatedSchemas)
		if err != nil {
			return 0, fmt.Errorf("response %s JSON schema: %w", statusCode, err)
		}
		media := resp.Content[contentType]
		media.Schema = schema
		resp.Content[contentType] = media
		i = j - 1
	}
	if j < len(lines) && strings.TrimSpace(lines[j]) == "+ Body" {
		media, next, err := p.parseRequestPayload(lines, j, assetSchemas, generatedSchemas)
		if err != nil {
			return 0, fmt.Errorf("response %s body: %w", statusCode, err)
		}
		current := resp.Content[contentType]
		current.Example = media.Example
		resp.Content[contentType] = current
		i = next - 1
	}
	currentOperation.AddResponse(statusCode, resp)
	return i, nil
}

func isPrimitiveType(value string) bool {
	switch value {
	case "string", "number", "integer", "boolean", "array", "object":
		return true
	}
	return false
}

// parseRequestPayload handles both directly indented examples and Blueprint
// Body/Attributes sections, stopping before the next action or payload.
func (p *Parser) parseRequestPayload(lines []string, start int, assetSchemas map[string]*model.Schema, generatedSchemas map[string]bool) (media model.MediaType, next int, err error) {
	var body strings.Builder
	var schemaJSON strings.Builder
	var schema *model.Schema
	var properties *msonParser
	mode := "body"
	i := start
	for ; i < len(lines); i++ {
		raw := lines[i]
		line := strings.TrimSpace(raw)
		if line != "" && !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			break
		}
		if matches := reAttributes.FindStringSubmatch(line); len(matches) > 0 {
			mode = "attributes"
			typ := strings.TrimSpace(matches[1])
			schema = &model.Schema{Type: model.TypeObject}
			switch {
			case strings.HasPrefix(typ, "array[") && strings.HasSuffix(typ, "]"):
				schema.Type = model.TypeArray
				schema.Items = msonArrayItem(typ[6 : len(typ)-1])
			case isPrimitiveType(typ):
				schema.Type = model.SchemaType(typ)
			case typ != "":
				schema = &model.Schema{Ref: "#/components/schemas/" + typ}
			}
			properties = newMSONParser(schema)
			continue
		}
		if strings.HasPrefix(line, "+ Body") {
			mode = "body"
			continue
		}
		if line == "+ Schema" {
			mode = "schema"
			continue
		}
		if strings.HasPrefix(line, "+ Headers") {
			mode = "headers"
			continue
		}
		switch {
		case mode == "attributes" && strings.HasPrefix(line, "+ "):
			properties.add(raw)
		case mode == "schema" && line != "":
			schemaJSON.WriteString(line)
		case mode == "body" && line != "":
			body.WriteString(line + "\n")
		}
	}
	if schemaJSON.Len() > 0 {
		var err error
		schema, err = parseSchemaAsset([]byte(schemaJSON.String()), assetSchemas, generatedSchemas)
		if err != nil {
			return media, i, err
		}
	}
	media.Schema = schema
	if body.Len() > 0 {
		media.Example = body.String()
	}
	return media, i, nil
}

// parseMSONProperty parses a single MSON property line and adds it to the schema.
// Format: + name: value (type, attributes) - description
func parseMSONProperty(line string, parent *model.Schema) *model.Schema {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "+") {
		return nil
	}
	trimmed = strings.TrimPrefix(trimmed, "+")
	trimmed = strings.TrimSpace(trimmed)

	parts := strings.SplitN(trimmed, ":", 2)
	namePart := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	} else if strings.Contains(namePart, "(") {
		// A property can specify a type without a value, as in "+ name (string)".
		split := strings.SplitN(namePart, "(", 2)
		namePart = split[0]
		rest = "(" + split[1]
	}

	name := strings.TrimSpace(namePart)
	if strings.Contains(rest, ") (required)") {
		parent.Required = append(parent.Required, name)
	}
	propSchema := &model.Schema{Type: model.TypeString} // Default

	if start := strings.Index(rest, "("); start > -1 {
		if end := strings.Index(rest, ")"); end > start {
			attrs := rest[start+1 : end]
			attrParts := strings.Split(attrs, ",")
			for _, attr := range attrParts {
				rawAttr := strings.TrimSpace(attr)
				attr = strings.ToLower(rawAttr)
				if strings.HasPrefix(attr, "array[") && strings.HasSuffix(attr, "]") {
					itemType := strings.TrimSpace(rawAttr[len("array[") : len(rawAttr)-1])
					propSchema.Type = model.TypeArray
					propSchema.Items = msonArrayItem(itemType)
					continue
				}
				switch attr {
				case "required":
					parent.Required = append(parent.Required, name)
				case "optional":
					// default
				case "number":
					propSchema.Type = model.TypeNumber
				case "integer":
					propSchema.Type = model.TypeInteger
				case "boolean":
					propSchema.Type = model.TypeBoolean
				case "array":
					propSchema.Type = model.TypeArray
				case "object":
					propSchema.Type = model.TypeObject
				case "string":
					propSchema.Type = model.TypeString
				}
			}
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
	return propSchema
}

// indentation counts the nesting level while allowing Markdown tabs.
func indentation(line string) int {
	n := 0
	for _, r := range line {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

type msonLevel struct {
	indent int
	schema *model.Schema
}
type msonParser struct {
	root   *model.Schema
	levels []msonLevel
}

func newMSONParser(root *model.Schema) *msonParser { return &msonParser{root: root} }
func (p *msonParser) add(line string) {
	indent := indentation(line)
	for len(p.levels) > 0 && p.levels[len(p.levels)-1].indent >= indent {
		p.levels = p.levels[:len(p.levels)-1]
	}
	parent := p.root
	if len(p.levels) > 0 {
		parent = p.levels[len(p.levels)-1].schema
	}
	if parent.Type == model.TypeArray {
		if parent.Items == nil {
			parent.Items = &model.Schema{Type: model.TypeObject}
		}
		parent = parent.Items
	}
	if child := parseMSONProperty(line, parent); child != nil {
		p.levels = append(p.levels, msonLevel{indent: indent, schema: child})
	}
}

// parseBlueprintParameter infers path versus query from the URI template.
func parseBlueprintParameter(line, path string) model.Parameter {
	text := strings.TrimSpace(strings.TrimPrefix(line, "+"))
	head, desc, _ := strings.Cut(text, " - ")
	name := strings.TrimSpace(head)
	if i := strings.IndexAny(name, ": ("); i >= 0 {
		name = name[:i]
	}
	param := model.Parameter{Name: name, In: model.ParameterInQuery, Description: strings.TrimSpace(desc), Schema: &model.Schema{Type: model.TypeString}}
	if name != "" && strings.Contains(path, "{"+name+"}") {
		param.In = model.ParameterInPath
		param.Required = true
	}
	for _, group := range reParameterAttrs.FindAllStringSubmatch(head, -1) {
		for _, raw := range strings.Split(group[1], ",") {
			attr := strings.ToLower(strings.TrimSpace(raw))
			switch {
			case attr == "required":
				param.Required = true
			case isPrimitiveType(attr):
				param.Schema.Type = model.SchemaType(attr)
			case attr != "" && attr != "optional":
				param.Schema.Format = attr
			}
		}
	}
	return param
}

func msonArrayItem(itemType string) *model.Schema {
	itemType = strings.TrimSpace(itemType)
	if isPrimitiveType(itemType) {
		return &model.Schema{Type: model.SchemaType(itemType)}
	}
	return &model.Schema{Ref: "#/components/schemas/" + itemType}
}
