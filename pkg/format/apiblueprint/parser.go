// Package apiblueprint implements the API Blueprint format parser and writer.
package apiblueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/amer8/apibconv/internal/fs"
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

	content, err := fs.ReadAll(ctx, r)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.parseDocument(string(content))
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
	reMetadata       = regexp.MustCompile(`^([[:alnum:]_][[:alnum:]_. -]*):\s*(.*)$`)
	reHeader         = regexp.MustCompile(`^#\s+(.*)$`)
	reGroup          = regexp.MustCompile(`^#+\s+Group\s+(.*)$`)
	reResource       = regexp.MustCompile(`^#+\s+(.*)\[(.*)\]$`)
	reAction         = regexp.MustCompile(`^#+\s+(.*?)\s*\[(GET|PUT|POST|DELETE|OPTIONS|HEAD|PATCH|TRACE)(?:\s+([^\]]+))?\]$`)
	reMethodPath     = regexp.MustCompile(`^#{2,}\s+(GET|PUT|POST|DELETE|OPTIONS|HEAD|PATCH|TRACE)\s+(\S+)\s*$`)
	reRequest        = regexp.MustCompile(`^\+\s+Request(?:\s+[^()]*)?\s*\(?([^)]*)\)?$`)
	reResponse       = regexp.MustCompile(`^\+\s+Response\s+(\d+)\s*\(?([^)]*)\)?$`)
	reAttributes     = regexp.MustCompile(`^\+\s+Attributes\s*\(?([^)]*)\)?$`)
	reDataStructures = regexp.MustCompile(`^#+\s+Data Structures$`)
	reNamedType      = regexp.MustCompile(`^#{2,3}\s+(.+?)(?:\s+\(([^()]*)\))?\s*$`)
	reParameterAttrs = regexp.MustCompile(`\(([^)]*)\)`)
)

//nolint:gocyclo // Parsing depends on the current Markdown section and payload.
func (p *Parser) parseDocument(content string) (*model.API, error) {
	api := model.NewAPI()
	assetSchemas := make(map[string]*model.Schema)
	generatedSchemas := make(map[string]bool)
	actionQueries := make(map[*model.Operation][]string)
	structureNames := make(map[string]string)
	api.Version = "1A" // Default version for API Blueprint

	lines := strings.Split(strings.TrimPrefix(content, "\uFEFF"), "\n")
	var currentPath string
	var resourcePath string
	var resourceQuery []string
	var scopedResourceParameters []model.Parameter
	var resourceParametersAreScoped bool
	var currentGroup string
	var currentPathItem *model.PathItem
	var currentOperation *model.Operation
	var inDataStructures bool
	var description *string
	var descriptionBreak bool
	inMetadata := true

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if p.parseMarkdown && description != nil && *description != "" {
				descriptionBreak = true
			}
			continue
		}

		// Metadata
		if matches := reMetadata.FindStringSubmatch(strings.TrimSuffix(lines[i], "\r")); inMetadata && len(matches) > 0 {
			key := matches[1]
			val := strings.TrimSpace(matches[2])
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

		inMetadata = false

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
			resourcePath, resourceQuery = "", nil
			scopedResourceParameters, resourceParametersAreScoped = nil, false
			currentPathItem = nil
			currentOperation = nil
			continue
		}

		if inDataStructures && !reGroup.MatchString(line) && !reResource.MatchString(line) && !reMethodPath.MatchString(line) {
			if matches := reNamedType.FindStringSubmatch(line); len(matches) > 0 {
				name := schemaName(matches[1])
				if previous, ok := structureNames[name]; ok {
					return nil, fmt.Errorf("data structure %q conflicts with %q after component name normalization", matches[1], previous)
				}
				structureNames[name] = matches[1]
				schema, next := parseNamedStructure(lines, i+1, strings.TrimSpace(matches[2]))
				api.Components.AddSchema(name, schema)
				i = next - 1
				continue
			}
		}

		// Resource Group (H2, usually)
		if matches := reGroup.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			if currentPath != "" && currentPathItem != nil {
				api.AddPath(currentPath, currentPathItem)
			}
			currentPath, resourcePath, currentPathItem, currentOperation = "", "", nil, nil
			resourceQuery = nil
			scopedResourceParameters, resourceParametersAreScoped = nil, false
			currentGroup = strings.TrimSpace(matches[1])
			description = nil
			continue
		}

		// Resource and action headings can each introduce a URI.
		action := reAction.FindStringSubmatch(line)
		methodPath := reMethodPath.FindStringSubmatch(line)
		if len(action) > 0 || len(methodPath) > 0 {
			inDataStructures = false
			standalone := len(action) == 0 && len(methodPath) > 0
			name, method, uri := "", "", ""
			if len(action) > 0 {
				name, method, uri = strings.TrimSpace(action[1]), action[2], strings.TrimSpace(action[3])
			} else {
				method, uri = methodPath[1], methodPath[2]
			}
			if standalone {
				resourcePath, resourceQuery = "", nil
				scopedResourceParameters, resourceParametersAreScoped = nil, false
			}
			var queries []string
			targetPath := resourcePath
			if uri != "" {
				var err error
				targetPath, queries, err = normalizeURI(uri)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", i+1, err)
				}
			}
			if standalone {
				resourcePath, resourceQuery = targetPath, append([]string(nil), queries...)
				queries = nil
			}
			if targetPath == "" {
				currentOperation = nil
				continue
			}
			if currentPath != targetPath || currentPathItem == nil {
				if currentPath != "" && currentPathItem != nil {
					api.AddPath(currentPath, currentPathItem)
				}
				currentPath = targetPath
				item := api.Paths[currentPath]
				currentPathItem = &item
			}
			if standalone || uri != "" && targetPath != resourcePath {
				scopePathParameters(currentPathItem)
				if standalone {
					resourceParametersAreScoped = true
				}
			}
			op := &model.Operation{Summary: name, Responses: make(model.Responses)}
			if currentGroup != "" {
				op.Tags = []string{currentGroup}
			}
			if resourcePath != "" && (targetPath != resourcePath || resourceParametersAreScoped) {
				params := scopedResourceParameters
				if !resourceParametersAreScoped {
					params = api.Paths[resourcePath].Parameters
				}
				for i := range params {
					param := &params[i]
					if param.In != model.ParameterInPath || strings.Contains(targetPath, "{"+param.Name+"}") {
						op.Parameters = append(op.Parameters, cloneBlueprintParameter(param))
					}
				}
			}
			if targetPath == resourcePath {
				queries = append(append([]string(nil), resourceQuery...), queries...)
			}
			actionQueries[op] = queries
			currentPathItem.SetOperation(method, op)
			currentOperation = op
			description = &op.Description
			continue
		}
		if matches := reResource.FindStringSubmatch(line); len(matches) > 0 {
			inDataStructures = false
			if currentPath != "" && currentPathItem != nil {
				api.AddPath(currentPath, currentPathItem)
			}
			var queries []string
			var err error
			currentPath, queries, err = normalizeURI(strings.TrimSpace(matches[2]))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			resourcePath, resourceQuery = currentPath, queries
			item, exists := api.Paths[currentPath]
			resourceParametersAreScoped = exists
			scopedResourceParameters = nil
			if exists {
				scopePathParameters(&item)
			}
			item.Summary = strings.TrimSpace(matches[1])
			currentPathItem = &item
			currentOperation = nil
			description = &currentPathItem.Description
			continue
		}

		// Parameters may be declared at either resource or action scope.
		if line == "+ Parameters" && currentPathItem != nil {
			description = nil
			indent := indentation(lines[i])
			parameterIndent := -1
			membersIndent := -1
			parameterIndex := -1
			target := &currentPathItem.Parameters
			if currentOperation != nil {
				target = &currentOperation.Parameters
			} else if resourceParametersAreScoped {
				target = &scopedResourceParameters
			}
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
					if parameterIndex >= 0 {
						param := &(*target)[parameterIndex]
						if membersIndent >= 0 && indentation(raw) > membersIndent {
							param.Schema.Enum = append(param.Schema.Enum, msonValue(strings.TrimSpace(strings.TrimPrefix(trimmed, "+ ")), param.Schema.Type))
						} else {
							membersIndent = -1
							if trimmed == "+ Members" {
								membersIndent = indentation(raw)
							}
							if strings.HasPrefix(trimmed, "+ Default:") {
								param.Schema.Default = msonValue(strings.TrimSpace(strings.TrimPrefix(trimmed, "+ Default:")), param.Schema.Type)
							}
						}
					}
					continue
				}
				membersIndent = -1
				param, err := parseBlueprintParameter(trimmed, currentPath)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", j+1, err)
				}
				if param.Name == "" {
					continue
				}
				*target, parameterIndex = setParameter(*target, &param)
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
			media, headers, next, err := p.parseRequestPayload(lines, i+1, indentation(lines[i]), contentType, assetSchemas, generatedSchemas)
			if err != nil {
				return nil, fmt.Errorf("request JSON schema: %w", err)
			}
			for name, value := range headers {
				switch strings.ToLower(name) {
				case "authorization", "accept", "content-type":
					return nil, fmt.Errorf("request header %q cannot be represented as an OpenAPI header parameter; define its security or media types explicitly", name)
				}
				currentOperation.Parameters = mergeHeader(currentOperation.Parameters, name, value)
			}
			if media.Schema != nil || media.Example != nil {
				if currentOperation.RequestBody == nil {
					currentOperation.RequestBody = &model.RequestBody{Content: make(map[string]model.MediaType)}
				}
				currentOperation.RequestBody.Content[contentType] = media
			}
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
	for path := range api.Paths {
		item := api.Paths[path]
		for _, op := range []*model.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op == nil {
				continue
			}
			for _, names := range [][]string{actionQueries[op]} {
				for _, name := range names {
					if !hasParameter(item.Parameters, name, model.ParameterInQuery) && !hasParameter(op.Parameters, name, model.ParameterInQuery) {
						op.Parameters = append(op.Parameters, inferredParameter(name, model.ParameterInQuery))
					}
				}
			}
			for _, match := range uriExpression.FindAllStringSubmatch(path, -1) {
				name := match[1]
				if !hasParameter(item.Parameters, name, model.ParameterInPath) && !hasParameter(op.Parameters, name, model.ParameterInPath) {
					op.Parameters = append(op.Parameters, inferredParameter(name, model.ParameterInPath))
				}
			}
		}
		api.Paths[path] = item
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
	switch {
	case strings.HasPrefix(typ, "array[") && strings.HasSuffix(typ, "]"):
		schema.Type, schema.Properties = model.TypeArray, nil
		schema.Items = msonArrayItem(typ[6 : len(typ)-1])
	case isPrimitiveType(typ):
		schema.Type = model.SchemaType(typ)
	case typ != "":
		schema.AllOf = []*model.Schema{{Ref: "#/components/schemas/" + schemaName(typ)}}
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

func (p *Parser) parseResponse(lines []string, i int, matches []string, op *model.Operation, assetSchemas map[string]*model.Schema, generatedSchemas map[string]bool) (int, error) {
	status, contentType := matches[1], strings.TrimSpace(matches[2])
	if contentType == "" {
		contentType = "application/json"
	}
	media, headers, next, err := p.parseRequestPayload(lines, i+1, indentation(lines[i]), contentType, assetSchemas, generatedSchemas)
	if err != nil {
		return 0, fmt.Errorf("response %s: %w", status, err)
	}
	resp := op.Responses[status]
	if resp.Description == "" {
		if code, err := strconv.Atoi(status); err == nil {
			resp.Description = http.StatusText(code)
		}
	}
	if len(headers) > 0 {
		if resp.Headers == nil {
			resp.Headers = make(map[string]model.Header)
		}
		for name, value := range headers {
			resp.Headers[name] = model.Header{Schema: &model.Schema{Type: model.TypeString}, Example: value}
		}
	}
	if media.Schema != nil || media.Example != nil {
		if resp.Content == nil {
			resp.Content = make(map[string]model.MediaType)
		}
		resp.Content[contentType] = media
	}
	op.AddResponse(status, resp)
	return next - 1, nil
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
func (p *Parser) parseRequestPayload(lines []string, start, parentIndent int, contentType string, assetSchemas map[string]*model.Schema, generatedSchemas map[string]bool) (media model.MediaType, headers map[string]string, next int, err error) {
	var bodyLines []string
	var schemaJSON strings.Builder
	var schema *model.Schema
	var properties *msonParser
	mode, sectionIndent := "body", parentIndent
	i := start
	for ; i < len(lines); i++ {
		raw, line := lines[i], strings.TrimSpace(lines[i])
		indent := indentation(raw)
		if line != "" && indent <= parentIndent {
			break
		}
		if line == "" {
			if mode == "body" && len(bodyLines) > 0 {
				bodyLines = append(bodyLines, "")
			}
			continue
		}
		if indent <= sectionIndent || mode == "body" {
			if matches := reAttributes.FindStringSubmatch(line); len(matches) > 0 {
				mode, sectionIndent = "attributes", indent
				typ := strings.TrimSpace(matches[1])
				schema = &model.Schema{Type: model.TypeObject}
				switch {
				case strings.HasPrefix(typ, "array[") && strings.HasSuffix(typ, "]"):
					schema.Type = model.TypeArray
					schema.Items = msonArrayItem(typ[6 : len(typ)-1])
				case isPrimitiveType(typ):
					schema.Type = model.SchemaType(typ)
				case typ != "":
					schema = &model.Schema{Ref: "#/components/schemas/" + schemaName(typ)}
				}
				properties = newMSONParser(schema)
				continue
			}
			switch line {
			case "+ Body":
				mode, sectionIndent = "body", indent
				continue
			case "+ Schema":
				mode, sectionIndent = "schema", indent
				continue
			case "+ Headers":
				mode, sectionIndent = "headers", indent
				continue
			}
		}
		if mode != "body" && indent <= sectionIndent {
			mode = "other"
		}
		switch mode {
		case "attributes":
			if strings.HasPrefix(line, "+ ") {
				properties.add(raw)
			}
		case "headers":
			if name, value, ok := strings.Cut(strings.TrimPrefix(line, "+ "), ":"); ok {
				if headers == nil {
					headers = make(map[string]string)
				}
				headers[strings.TrimSpace(name)] = strings.TrimSpace(value)
			}
		case "schema":
			schemaJSON.WriteString(line)
		case "body":
			bodyLines = append(bodyLines, raw)
		}
	}
	if schemaJSON.Len() > 0 {
		schema, err = parseSchemaAsset([]byte(schemaJSON.String()), assetSchemas, generatedSchemas)
		if err != nil {
			return media, headers, i, err
		}
	}
	media.Schema = schema
	if len(bodyLines) > 0 {
		minIndent := -1
		for _, raw := range bodyLines {
			if strings.TrimSpace(raw) != "" && (minIndent < 0 || indentation(raw) < minIndent) {
				minIndent = indentation(raw)
			}
		}
		for i, raw := range bodyLines {
			count, index := 0, 0
			for index < len(raw) && count < minIndent {
				switch raw[index] {
				case ' ':
					count++
				case '\t':
					count += 4
				default:
					count = minIndent
					continue
				}
				index++
			}
			bodyLines[i] = strings.TrimRight(raw[index:], "\r")
		}
		value := strings.Trim(strings.Join(bodyLines, "\n"), "\n")
		media.Example = value
		if strings.Contains(strings.ToLower(contentType), "json") {
			var decoded interface{}
			decoder := json.NewDecoder(strings.NewReader(value))
			decoder.UseNumber()
			if decoder.Decode(&decoded) == nil && json.Valid([]byte(value)) {
				media.Example = preserveJSONValue(decoded)
			}
		}
	}
	return media, headers, i, nil
}

func preserveJSONValue(value interface{}) interface{} {
	switch value := value.(type) {
	case nil:
		return model.NullExample{}
	case json.Number:
		return model.JSONNumber(value)
	case []interface{}:
		for i := range value {
			value[i] = preserveJSONValue(value[i])
		}
		return value
	case map[string]interface{}:
		for key, item := range value {
			value[key] = preserveJSONValue(item)
		}
		return value
	default:
		return value
	}
}

// normalizeURI maps supported URI templates to OpenAPI paths and query parameters.
var uriExpression = regexp.MustCompile(`\{([^{}]+)\}`)
var uriVariable = regexp.MustCompile(`^[A-Za-z0-9_.%-]+$`)
var invalidComponentName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func normalizeURI(uri string) (path string, queries []string, parseErr error) {
	path = uriExpression.ReplaceAllStringFunc(uri, func(expr string) string {
		inner := expr[1 : len(expr)-1]
		operator := byte(0)
		if strings.ContainsRune("+?#&./;", rune(inner[0])) {
			operator, inner = inner[0], inner[1:]
		}
		if strings.ContainsRune("+./;", rune(operator)) {
			parseErr = fmt.Errorf("URI template expression %q cannot be represented in an OpenAPI path", expr)
			return ""
		}
		var variables []string
		for _, name := range strings.Split(inner, ",") {
			if !uriVariable.MatchString(name) {
				parseErr = fmt.Errorf("unsupported URI template variable %q", name)
				return ""
			}
			variables = append(variables, name)
		}
		if operator == '?' || operator == '&' {
			queries = append(queries, variables...)
			return ""
		}
		if operator == '#' {
			return ""
		}
		for i, name := range variables {
			variables[i] = "{" + name + "}"
		}
		return strings.Join(variables, ",")
	})
	if parseErr == nil && strings.ContainsAny(uriExpression.ReplaceAllString(uri, ""), "{}") {
		parseErr = fmt.Errorf("malformed URI template %q", uri)
	}
	return path, queries, parseErr
}
func schemaName(name string) string {
	return invalidComponentName.ReplaceAllString(strings.TrimSpace(name), "_")
}
func hasParameter(params []model.Parameter, name string, location model.ParameterLocation) bool {
	for i := range params {
		if params[i].Name == name && params[i].In == location {
			return true
		}
	}
	return false
}
func inferredParameter(name string, location model.ParameterLocation) model.Parameter {
	return model.Parameter{Name: name, In: location, Required: location == model.ParameterInPath, Schema: &model.Schema{Type: model.TypeString}}
}

func inheritParameters(inherited, explicit []model.Parameter) []model.Parameter {
	result := make([]model.Parameter, 0, len(inherited)+len(explicit))
	for i := range inherited {
		p := &inherited[i]
		if !hasParameter(explicit, p.Name, p.In) {
			result = append(result, cloneBlueprintParameter(p))
		}
	}
	return append(result, explicit...)
}

// Copy the schema and JSON values when a resource parameter becomes operation-specific.
func cloneBlueprintParameter(parameter *model.Parameter) model.Parameter {
	result := *parameter
	result.Example = cloneBlueprintValue(parameter.Example)
	if parameter.Schema != nil {
		schema := *parameter.Schema
		schema.Default = cloneBlueprintValue(schema.Default)
		if schema.Enum != nil {
			schema.Enum = make([]interface{}, len(parameter.Schema.Enum))
			for i, value := range parameter.Schema.Enum {
				schema.Enum[i] = cloneBlueprintValue(value)
			}
		}
		result.Schema = &schema
	}
	return result
}

func cloneBlueprintValue(value interface{}) interface{} {
	switch value := value.(type) {
	case []interface{}:
		cloned := make([]interface{}, len(value))
		for i, item := range value {
			cloned[i] = cloneBlueprintValue(item)
		}
		return cloned
	case map[string]interface{}:
		cloned := make(map[string]interface{}, len(value))
		for key, item := range value {
			cloned[key] = cloneBlueprintValue(item)
		}
		return cloned
	default:
		return value
	}
}

func scopePathParameters(item *model.PathItem) {
	if len(item.Parameters) == 0 {
		return
	}
	for _, op := range []*model.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
		if op != nil {
			op.Parameters = inheritParameters(item.Parameters, op.Parameters)
		}
	}
	item.Parameters = nil
}

func setParameter(params []model.Parameter, param *model.Parameter) (updated []model.Parameter, index int) {
	for i := range params {
		if params[i].In == param.In && (params[i].Name == param.Name || param.In == model.ParameterInHeader && strings.EqualFold(params[i].Name, param.Name)) {
			params[i] = *param
			return params, i
		}
	}
	return append(params, *param), len(params)
}
func mergeHeader(params []model.Parameter, name, value string) []model.Parameter {
	for i := range params {
		param := &params[i]
		if param.In != model.ParameterInHeader || !strings.EqualFold(param.Name, name) {
			continue
		}
		if param.Examples == nil && param.Example == nil {
			param.Example = value
			return params
		}
		if param.Examples == nil && param.Example == value {
			return params
		}
		if param.Examples == nil {
			param.Examples = map[string]model.Example{"example1": {Value: param.Example}}
			param.Example = nil
		}
		for _, example := range param.Examples {
			if example.Value == value {
				return params
			}
		}
		param.Examples[fmt.Sprintf("example%d", len(param.Examples)+1)] = model.Example{Value: value}
		return params
	}
	return append(params, model.Parameter{Name: name, In: model.ParameterInHeader, Schema: &model.Schema{Type: model.TypeString}, Example: value})
}

// parseMSONProperty parses a single MSON property line and adds it to the schema.
// Format: + name: value (type, attributes) - description
func parseMSONProperty(line string, parent *model.Schema) *model.Schema {
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "+"))
	head, desc, _ := strings.Cut(text, " - ")
	nameValue := reParameterAttrs.ReplaceAllString(head, "")
	name, sample, hasSample := strings.Cut(nameValue, ":")
	name = strings.Trim(strings.TrimSpace(name), "`")
	if name == "" {
		return nil
	}
	prop := &model.Schema{Type: model.TypeString, Description: strings.TrimSpace(desc)}
	required, fixed, defaultValue := false, false, false
	for _, group := range reParameterAttrs.FindAllStringSubmatch(head, -1) {
		for _, raw := range strings.Split(group[1], ",") {
			attr := strings.TrimSpace(raw)
			lower := strings.ToLower(attr)
			switch {
			case lower == "required":
				required = true
			case lower == "optional":
			case lower == "nullable":
				prop.Nullable = true
			case lower == "fixed":
				fixed = true
			case lower == "default":
				defaultValue = true
			case strings.HasPrefix(lower, "array[") && strings.HasSuffix(lower, "]"):
				prop.Type = model.TypeArray
				prop.Items = msonArrayItem(attr[6 : len(attr)-1])
			case strings.HasPrefix(lower, "enum[") && strings.HasSuffix(lower, "]"):
				typ := strings.TrimSpace(lower[5 : len(lower)-1])
				if isPrimitiveType(typ) {
					prop.Type = model.SchemaType(typ)
				}
			case isPrimitiveType(lower):
				prop.Type = model.SchemaType(lower)
			case attr != "":
				prop.Type = ""
				prop.Ref = "#/components/schemas/" + schemaName(attr)
			}
		}
	}
	if hasSample && strings.TrimSpace(sample) != "" {
		value := msonValue(strings.TrimSpace(sample), prop.Type)
		prop.Example = value
		if fixed {
			prop.Enum = []interface{}{value}
		}
		if defaultValue {
			prop.Default = value
		}
	}
	if required {
		for _, existing := range parent.Required {
			if existing == name {
				required = false
				break
			}
		}
		if required {
			parent.Required = append(parent.Required, name)
		}
	}
	if parent.Properties == nil {
		parent.Properties = make(map[string]*model.Schema)
	}
	parent.Properties[name] = prop
	return prop
}
func msonValue(value string, typ model.SchemaType) interface{} {
	value = strings.Trim(strings.TrimSpace(value), "`")
	if typ == model.TypeInteger || typ == model.TypeNumber || typ == model.TypeBoolean {
		return parseBlueprintJSONValue(value)
	}
	return strings.Trim(value, "\"")
}

func parseBlueprintJSONValue(value string) interface{} {
	var decoded interface{}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	if decoder.Decode(&decoded) == nil && json.Valid([]byte(value)) {
		return preserveJSONValue(decoded)
	}
	return strings.Trim(value, "\"")
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
	root          *model.Schema
	levels        []msonLevel
	members       *model.Schema
	membersIndent int
}

func newMSONParser(root *model.Schema) *msonParser { return &msonParser{root: root} }
func (p *msonParser) add(line string) {
	indent := indentation(line)
	text := strings.TrimSpace(line)
	if p.members != nil {
		if indent > p.membersIndent {
			p.members.Enum = append(p.members.Enum, msonValue(strings.TrimSpace(strings.TrimPrefix(text, "+ ")), p.members.Type))
			return
		}
		p.members = nil
	}
	if len(p.levels) > 0 && indent > p.levels[len(p.levels)-1].indent {
		parent := p.levels[len(p.levels)-1].schema
		if text == "+ Members" {
			p.members, p.membersIndent = parent, indent
			return
		}
		if strings.HasPrefix(text, "+ Default:") {
			parent.Default = msonValue(strings.TrimSpace(strings.TrimPrefix(text, "+ Default:")), parent.Type)
			return
		}
	}
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
func parseBlueprintParameter(line, path string) (model.Parameter, error) {
	text := strings.TrimSpace(strings.TrimPrefix(line, "+"))
	name := text
	if i := strings.IndexAny(name, ": (\t"); i >= 0 {
		name = name[:i]
	}
	sample, tail := splitParameterExample(text[len(name):])
	attrs, desc, _ := strings.Cut(tail, " - ")
	name = strings.Trim(name, "`")
	param := model.Parameter{Name: name, In: model.ParameterInQuery, Description: strings.TrimSpace(desc), Schema: &model.Schema{Type: model.TypeString}}
	if name != "" && strings.Contains(path, "{"+name+"}") {
		param.In = model.ParameterInPath
		param.Required = true
	}
	for _, group := range reParameterAttrs.FindAllStringSubmatch(attrs, -1) {
		for _, raw := range strings.Split(group[1], ",") {
			attr := strings.ToLower(strings.TrimSpace(raw))
			switch {
			case attr == "required":
				param.Required = true
			case attr == "nullable" || attr == "fixed" || attr == "default":
				return model.Parameter{}, fmt.Errorf("URI parameter %q does not support inline modifier %q", name, attr)
			case isPrimitiveType(attr):
				param.Schema.Type = model.SchemaType(attr)
			case strings.HasPrefix(attr, "enum[") && strings.HasSuffix(attr, "]"):
				typ := strings.TrimSuffix(strings.TrimPrefix(attr, "enum["), "]")
				if isPrimitiveType(typ) {
					param.Schema.Type = model.SchemaType(typ)
				}
			case attr != "" && attr != "optional":
				param.Schema.Format = attr
			}
		}
	}
	if sample != "" {
		literal := strings.Trim(sample, "`")
		param.Example = literal
		if param.Schema.Type != model.TypeString {
			param.Example = parseBlueprintJSONValue(literal)
		}
	}
	return param, nil
}

// splitParameterExample separates the optional value after the name from the
// attributes and description. A Markdown code span can contain either delimiter.
func splitParameterExample(tail string) (sample, rest string) {
	value, ok := strings.CutPrefix(strings.TrimLeft(tail, " \t"), ":")
	if !ok {
		return "", tail
	}
	trimmed := strings.TrimLeft(value, " \t")
	content := strings.TrimLeft(trimmed, "`")
	fenceLen := len(trimmed) - len(content)
	if fenceLen > 0 {
		for {
			start := strings.IndexByte(content, '`')
			if start < 0 {
				break
			}
			after := strings.TrimLeft(content[start:], "`")
			if len(content[start:])-len(after) == fenceLen {
				return trimmed[:len(trimmed)-len(after)], after
			}
			content = after
		}
	}
	beforeDesc, _, _ := strings.Cut(value, " - ")
	beforeAttrs, _, _ := strings.Cut(beforeDesc, "(")
	return strings.TrimSpace(beforeAttrs), value[len(beforeAttrs):]
}

func msonArrayItem(itemType string) *model.Schema {
	itemType = strings.TrimSpace(itemType)
	if isPrimitiveType(itemType) {
		return &model.Schema{Type: model.SchemaType(itemType)}
	}
	return &model.Schema{Ref: "#/components/schemas/" + schemaName(itemType)}
}
