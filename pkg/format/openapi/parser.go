// Package openapi implements the OpenAPI format parser and writer.
package openapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/internal/specschema"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Parser implements the format.Parser interface for OpenAPI.
type Parser struct {
	version string
	strict  bool
}

// ParserOption configures the Parser.
type ParserOption func(*Parser)

// WithVersion requires the document to declare the given version. A major.minor
// value accepts any patch version; a full version must match exactly.
func WithVersion(version string) ParserOption {
	return func(p *Parser) {
		p.version = version
	}
}

// WithStrict validates the input against the bundled OpenAPI schema before parsing.
func WithStrict(strict bool) ParserOption {
	return func(p *Parser) {
		p.strict = strict
	}
}

// NewParser creates a new OpenAPI parser with optional configurations.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Parse parses an OpenAPI document from the given reader into a unified model.API.
func (p *Parser) Parse(ctx context.Context, r io.Reader) (*model.API, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read all data to allow multiple decoding attempts
	data, err := fs.ReadAll(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	var base struct {
		Swagger string `yaml:"swagger"`
		OpenAPI string `yaml:"openapi"`
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to detect openapi version: %w", err)
	}

	actualVersion := base.OpenAPI
	if base.Swagger != "" {
		actualVersion = base.Swagger
	}
	if p.version != "" {
		if !p.SupportsVersion(p.version) {
			return nil, fmt.Errorf("%w: unsupported expected OpenAPI version %q", apierrors.ErrIncompatibleVersion, p.version)
		}
		if !p.SupportsVersion(actualVersion) || !matchesVersion(actualVersion, p.version) {
			return nil, fmt.Errorf("%w: expected OpenAPI version %q, got %q", apierrors.ErrIncompatibleVersion, p.version, actualVersion)
		}
	}
	if p.strict {
		issues := specschema.Validate(format.FormatOpenAPI, data)
		if len(issues) > 0 {
			failures := make([]error, 0, len(issues))
			for _, issue := range issues {
				if issue.Level == format.LevelError {
					failures = append(failures, issue)
				}
			}
			if len(failures) > 0 {
				return nil, fmt.Errorf("%w: OpenAPI schema validation: %w", apierrors.ErrValidationFailed, errors.Join(failures...))
			}
		}
	}
	if base.Swagger == "2.0" {
		return p.parseV2(ctx, bytes.NewReader(data))
	}

	var doc OpenAPI
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("openapi v3 parse failed: %w", err)
	}

	return p.convertToModel(&doc)
}

// Format returns the format type for the parser.
func (p *Parser) Format() format.Format {
	return format.FormatOpenAPI
}

var supportedVersion = regexp.MustCompile(`^(2\.0|3\.[01](?:\.\d+)?)$`)

// SupportsVersion reports whether the parser handles the given OpenAPI version.
func (p *Parser) SupportsVersion(version string) bool {
	return supportedVersion.MatchString(version)
}

func matchesVersion(actual, expected string) bool {
	if strings.Count(expected, ".") == 1 && strings.HasPrefix(expected, "3.") {
		return actual == expected || strings.HasPrefix(actual, expected+".")
	}
	return actual == expected
}

func (p *Parser) convertToModel(doc *OpenAPI) (*model.API, error) {
	api := model.NewAPI()
	copyExtensions(api.Extensions, doc.Extensions)
	for name, callback := range doc.Components.Callbacks {
		resolved, err := resolveCallback(callback, doc, 0)
		if err != nil {
			return nil, fmt.Errorf("callback component %q: %w", name, err)
		}
		api.Components.Callbacks[name] = p.convertCallback(resolved)
		if len(resolved.Extensions) > 0 {
			if api.Components.CallbackExtensions == nil {
				api.Components.CallbackExtensions = make(map[string]map[string]interface{})
			}
			api.Components.CallbackExtensions[name] = clonedExtensions(resolved.Extensions)
		}
	}
	api.Components.Extensions = clonedExtensions(doc.Components.Extensions)
	api.PathsExtensions = clonedExtensions(doc.PathsExtensions)
	api.Tags = tagsToModel(doc.Tags)
	api.ExternalDocs = externalDocsToModel(doc.ExternalDocs)
	api.Version = doc.OpenAPI
	api.Security = securityRequirementsToModel(doc.Security)
	for name := range doc.Components.SecuritySchemes {
		scheme := doc.Components.SecuritySchemes[name]
		api.Components.SecuritySchemes[name] = securitySchemeToModel(&scheme)
	}

	api.Info = model.Info{
		Title:          doc.Info.Title,
		Description:    doc.Info.Description,
		Version:        doc.Info.Version,
		TermsOfService: doc.Info.TermsOfService,
		Extensions:     clonedExtensions(doc.Info.Extensions),
	}
	if doc.Info.Contact != nil {
		api.Info.Contact = &model.Contact{
			Name:       doc.Info.Contact.Name,
			URL:        doc.Info.Contact.URL,
			Email:      doc.Info.Contact.Email,
			Extensions: clonedExtensions(doc.Info.Contact.Extensions),
		}
	}
	if doc.Info.License != nil {
		api.Info.License = &model.License{
			Name:       doc.Info.License.Name,
			URL:        doc.Info.License.URL,
			Extensions: clonedExtensions(doc.Info.License.Extensions),
		}
	}

	for _, s := range doc.Servers {
		ms := model.Server{
			URL:         s.URL,
			Description: s.Description,
			Variables:   make(map[string]model.ServerVariable),
			Extensions:  clonedExtensions(s.Extensions),
		}
		for k, v := range s.Variables {
			ms.Variables[k] = model.ServerVariable{
				Default:     v.Default,
				Description: v.Description,
				Enum:        v.Enum,
				Extensions:  clonedExtensions(v.Extensions),
			}
		}
		api.Servers = append(api.Servers, ms)
	}

	if doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas {
			api.Components.AddSchema(name, p.convertSchema(schema))
		}
	}

	for name := range doc.Components.Parameters {
		parameter := doc.Components.Parameters[name]
		resolved, err := resolveParameter(&parameter, doc.Components.Parameters, "#/components/parameters/")
		if err != nil {
			return nil, fmt.Errorf("parameter component %q: %w", name, err)
		}
		api.Components.Parameters[name] = p.convertParameters([]Parameter{resolved})[0]
	}
	for name, response := range doc.Components.Responses {
		resolved, err := resolveResponse(response, doc.Components.Responses, "#/components/responses/")
		if err != nil {
			return nil, fmt.Errorf("response component %q: %w", name, err)
		}
		api.Components.Responses[name] = p.convertResponse(resolved)
	}
	for name, body := range doc.Components.RequestBodies {
		resolved, err := resolveRequestBody(body, doc.Components.RequestBodies)
		if err != nil {
			return nil, fmt.Errorf("request body component %q: %w", name, err)
		}
		api.Components.RequestBodies[name] = p.convertRequestBody(&resolved)
	}
	for name, header := range doc.Components.Headers {
		api.Components.Headers[name] = p.convertHeader(&header)
	}

	for name, example := range examplesToModel(doc.Components.Examples) {
		api.Components.Examples[name] = example
	}

	for name, link := range doc.Components.Links {
		api.Components.Links[name] = linkToModel(&link)
	}

	for name := range doc.Webhooks {
		item := doc.Webhooks[name]
		resolved, err := resolvePathV3(&item, doc)
		if err != nil {
			return nil, fmt.Errorf("webhook %q: %w", name, err)
		}
		api.Webhooks[name] = p.convertPathItem(&resolved)
	}

	for path := range doc.Paths {
		item := doc.Paths[path]
		resolved, err := resolvePathV3(&item, doc)
		if err != nil {
			return nil, fmt.Errorf("path %q: %w", path, err)
		}
		converted := p.convertPathItem(&resolved)
		api.AddPath(path, &converted)
	}

	return api, nil
}

func copyExtensions(dst, src map[string]interface{}) {
	for key, value := range src {
		if strings.HasPrefix(key, "x-") {
			dst[key] = value
		}
	}
}

func clonedExtensions(src map[string]interface{}) map[string]interface{} {
	var dst map[string]interface{}
	for key, value := range src {
		if strings.HasPrefix(key, "x-") {
			if dst == nil {
				dst = make(map[string]interface{})
			}
			dst[key] = value
		}
	}
	return dst
}

func linkToModel(link *Link) model.Link {
	result := model.Link{
		Ref: link.Ref, OperationRef: link.OperationRef, OperationID: link.OperationID,
		Parameters: link.Parameters, RequestBody: link.RequestBody,
		Description: link.Description, Extensions: clonedExtensions(link.Extensions),
	}
	if link.Server != nil {
		server := serversToModel([]Server{*link.Server})[0]
		result.Server = &server
	}
	return result
}

func examplesToModel(examples map[string]interface{}) map[string]model.Example {
	if examples == nil {
		return nil
	}
	result := make(map[string]model.Example, len(examples))
	for name, value := range examples {
		fields, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		item := model.Example{Value: fields["value"], Extensions: clonedExtensions(fields)}
		item.Ref, _ = fields["$ref"].(string)
		item.Summary, _ = fields["summary"].(string)
		item.Description, _ = fields["description"].(string)
		item.ExternalValue, _ = fields["externalValue"].(string)
		result[name] = item
	}
	return result
}

func externalDocsToModel(docs *ExternalDocs) *model.ExternalDocs {
	if docs == nil {
		return nil
	}
	return &model.ExternalDocs{Description: docs.Description, URL: docs.URL, Extensions: clonedExtensions(docs.Extensions)}
}

func tagsToModel(tags []Tag) []model.Tag {
	if tags == nil {
		return nil
	}
	result := make([]model.Tag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, model.Tag{Name: tag.Name, Description: tag.Description, ExternalDocs: externalDocsToModel(tag.ExternalDocs), Extensions: clonedExtensions(tag.Extensions)})
	}
	return result
}

func (p *Parser) convertOperation(op *Operation) *model.Operation {
	if op == nil {
		return nil
	}
	res := &model.Operation{
		Tags:                op.Tags,
		Summary:             op.Summary,
		Description:         op.Description,
		OperationID:         op.OperationID,
		Deprecated:          op.Deprecated,
		Parameters:          p.convertParameters(op.Parameters),
		Security:            securityRequirementsToModel(op.Security),
		Servers:             serversToModel(op.Servers),
		ExternalDocs:        externalDocsToModel(op.ExternalDocs),
		Extensions:          clonedExtensions(op.Extensions),
		ResponsesExtensions: clonedExtensions(op.ResponsesExtensions),
	}
	if len(op.Callbacks) != 0 {
		res.Callbacks = make(map[string]model.Callback, len(op.Callbacks))
		for name, callback := range op.Callbacks {
			res.Callbacks[name] = p.convertCallback(callback)
			if len(callback.Extensions) > 0 {
				if res.CallbackExtensions == nil {
					res.CallbackExtensions = make(map[string]map[string]interface{})
				}
				res.CallbackExtensions[name] = clonedExtensions(callback.Extensions)
			}
		}
	}

	if op.RequestBody != nil {
		body := p.convertRequestBody(op.RequestBody)
		res.RequestBody = &body
	}

	if op.Responses != nil {
		res.Responses = make(model.Responses)
		for status, resp := range op.Responses {
			res.Responses[status] = p.convertResponse(resp)
		}
	}

	return res
}

func (p *Parser) convertCallback(callback Callback) model.Callback {
	paths := make(model.Callback, len(callback.Paths))
	for expression := range callback.Paths {
		item := callback.Paths[expression]
		paths[expression] = p.convertPathItem(&item)
	}
	return paths
}

func (p *Parser) convertPathItem(item *PathItem) model.PathItem {
	return model.PathItem{
		Summary:     item.Summary,
		Description: item.Description,
		Parameters:  p.convertParameters(item.Parameters),
		Servers:     serversToModel(item.Servers),
		Get:         p.convertOperation(item.Get),
		Put:         p.convertOperation(item.Put),
		Post:        p.convertOperation(item.Post),
		Delete:      p.convertOperation(item.Delete),
		Options:     p.convertOperation(item.Options),
		Head:        p.convertOperation(item.Head),
		Patch:       p.convertOperation(item.Patch),
		Trace:       p.convertOperation(item.Trace),
		Extensions:  clonedExtensions(item.Extensions),
	}
}

func serversToModel(servers []Server) []model.Server {
	if len(servers) == 0 {
		return nil
	}
	result := make([]model.Server, 0, len(servers))
	for _, s := range servers {
		m := model.Server{URL: s.URL, Description: s.Description, Extensions: clonedExtensions(s.Extensions)}
		if s.Variables != nil {
			m.Variables = make(map[string]model.ServerVariable, len(s.Variables))
		}
		for k, v := range s.Variables {
			m.Variables[k] = model.ServerVariable{Default: v.Default, Description: v.Description, Enum: v.Enum, Extensions: clonedExtensions(v.Extensions)}
		}
		result = append(result, m)
	}
	return result
}

func (p *Parser) convertParameters(params []Parameter) []model.Parameter {
	result := make([]model.Parameter, 0, len(params))
	for i := range params {
		param := &params[i]
		mp := model.Parameter{
			Name:            param.Name,
			In:              model.ParameterLocation(param.In),
			Description:     param.Description,
			Required:        param.Required,
			Deprecated:      param.Deprecated,
			AllowEmptyValue: param.AllowEmptyValue,
			Style:           param.Style,
			Example:         param.Example,
			Examples:        examplesToModel(param.Examples),
			Extensions:      clonedExtensions(param.Extensions),
		}
		if param.Explode != nil {
			mp.Explode, mp.ExplodeSet = *param.Explode, true
		}
		if param.Schema != nil {
			mp.Schema = p.convertSchema(param.Schema)
		}
		if len(param.Content) > 0 {
			mp.Content = make(map[string]model.MediaType)
			for k, v := range param.Content {
				mp.Content[k] = p.convertMediaType(v)
			}
		}
		result = append(result, mp)
	}
	return result
}

func (p *Parser) convertRequestBody(body *RequestBody) model.RequestBody {
	result := model.RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     make(map[string]model.MediaType),
		Extensions:  clonedExtensions(body.Extensions),
	}
	for mediaType, content := range body.Content {
		result.Content[mediaType] = p.convertMediaType(content)
	}
	return result
}

func (p *Parser) convertHeader(header *Header) model.Header {
	result := model.Header{
		Description: header.Description,
		Required:    header.Required,
		Deprecated:  header.Deprecated,
		Example:     header.Example,
		Examples:    examplesToModel(header.Examples),
		Extensions:  clonedExtensions(header.Extensions),
	}
	if header.Schema != nil {
		result.Schema = p.convertSchema(header.Schema)
	}
	return result
}

func (p *Parser) convertResponse(resp Response) model.Response {
	res := model.Response{
		Description: resp.Description,
		Content:     make(map[string]model.MediaType),
		Headers:     make(map[string]model.Header),
		Extensions:  clonedExtensions(resp.Extensions),
	}

	for ct, mt := range resp.Content {
		res.Content[ct] = p.convertMediaType(mt)
	}

	for name, header := range resp.Headers {
		res.Headers[name] = p.convertHeader(&header)
	}

	if len(resp.Links) > 0 {
		res.Links = make(map[string]model.Link, len(resp.Links))
		for name, link := range resp.Links {
			res.Links[name] = linkToModel(&link)
		}
	}
	return res
}

func (p *Parser) convertMediaType(mt MediaType) model.MediaType {
	res := model.MediaType{
		Example:    mt.Example,
		Examples:   examplesToModel(mt.Examples),
		Extensions: clonedExtensions(mt.Extensions),
	}
	if mt.Schema != nil {
		res.Schema = p.convertSchema(mt.Schema)
	}
	if len(mt.Encoding) > 0 {
		res.Encoding = make(map[string]model.Encoding, len(mt.Encoding))
		for name, encoding := range mt.Encoding {
			item := model.Encoding{
				ContentType: encoding.ContentType, Style: encoding.Style,
				Explode: encoding.Explode, AllowReserved: encoding.AllowReserved,
				Extensions: clonedExtensions(encoding.Extensions),
			}
			if len(encoding.Headers) > 0 {
				item.Headers = make(map[string]model.Header, len(encoding.Headers))
				for headerName, header := range encoding.Headers {
					item.Headers[headerName] = p.convertHeader(&header)
				}
			}
			res.Encoding[name] = item
		}
	}
	return res
}

func (p *Parser) convertSchema(s *Schema) *model.Schema {
	if s == nil {
		return nil
	}
	ms := &model.Schema{
		Ref:                  s.Ref,
		ExternalDocs:         externalDocsToModel(s.ExternalDocs),
		Extensions:           clonedExtensions(s.Extensions),
		Format:               s.Format,
		Title:                s.Title,
		Description:          s.Description,
		Default:              s.Default,
		Enum:                 s.Enum,
		Required:             s.Required,
		MinLength:            s.MinLength,
		MaxLength:            s.MaxLength,
		Pattern:              s.Pattern,
		Minimum:              s.Minimum,
		Maximum:              s.Maximum,
		MultipleOf:           s.MultipleOf,
		MinItems:             s.MinItems,
		MaxItems:             s.MaxItems,
		UniqueItems:          s.UniqueItems,
		MinProperties:        s.MinProperties,
		MaxProperties:        s.MaxProperties,
		AdditionalProperties: s.AdditionalProperties,
		Example:              s.Example,
		ReadOnly:             s.ReadOnly,
		WriteOnly:            s.WriteOnly,
		Deprecated:           s.Deprecated,
		Nullable:             s.Nullable,
	}
	if s.Discriminator != nil {
		ms.Discriminator = &model.Discriminator{
			PropertyName: s.Discriminator.PropertyName,
			Mapping:      s.Discriminator.Mapping,
			Extensions:   clonedExtensions(s.Discriminator.Extensions),
		}
	}
	ms.ExclusiveMinimum, ms.Minimum = schemaExclusive(s.ExclusiveMinimum, ms.Minimum, false)
	ms.ExclusiveMaximum, ms.Maximum = schemaExclusive(s.ExclusiveMaximum, ms.Maximum, true)
	if s.AdditionalProperties != nil {
		if _, ok := s.AdditionalProperties.(bool); !ok {
			if data, err := yaml.Marshal(s.AdditionalProperties); err == nil {
				var nested Schema
				if yaml.Unmarshal(data, &nested) == nil {
					ms.AdditionalProperties = p.convertSchema(&nested)
				}
			}
		}
	}

	// Rewrite Swagger 2.0 definitions to OpenAPI 3.0 components/schemas
	if ms.Ref != "" && strings.HasPrefix(ms.Ref, "#/definitions/") {
		ms.Ref = strings.Replace(ms.Ref, "#/definitions/", "#/components/schemas/", 1)
	}
	if s.Type != nil {
		switch t := s.Type.(type) {
		case string:
			ms.Type = model.SchemaType(t)
		case []interface{}: // OpenAPI 3.1 permits a union of JSON Schema types.
			var alternatives []*model.Schema
			for _, item := range t {
				if item == nil {
					alternatives = append(alternatives, &model.Schema{Type: "null"})
					continue
				}
				if typeStr, ok := item.(string); ok {
					alternatives = append(alternatives, &model.Schema{Type: model.SchemaType(typeStr)})
				}
			}
			switch {
			case len(alternatives) == 2 && (alternatives[0].Type == "null" || alternatives[1].Type == "null"):
				ms.Nullable = true
				if alternatives[0].Type == "null" {
					ms.Type = alternatives[1].Type
				} else {
					ms.Type = alternatives[0].Type
				}
			case len(alternatives) == 1:
				ms.Type = alternatives[0].Type
			case len(alternatives) > 1:
				// allOf preserves the intersection with an existing anyOf on this schema.
				ms.AllOf = append(ms.AllOf, &model.Schema{AnyOf: alternatives})
			}
		}
	}

	// Properties
	if s.Properties != nil {
		ms.Properties = make(map[string]*model.Schema)
		for k, v := range s.Properties {
			ms.Properties[k] = p.convertSchema(v)
		}
	}

	if s.Items != nil {
		ms.Items = p.convertSchema(s.Items)
	}

	// Composition
	for _, subSchema := range s.AllOf {
		ms.AllOf = append(ms.AllOf, p.convertSchema(subSchema))
	}
	for _, subSchema := range s.AnyOf {
		ms.AnyOf = append(ms.AnyOf, p.convertSchema(subSchema))
	}
	for _, subSchema := range s.OneOf {
		ms.OneOf = append(ms.OneOf, p.convertSchema(subSchema))
	}
	if s.Not != nil {
		ms.Not = p.convertSchema(s.Not)
	}

	return ms
}

func schemaExclusive(value interface{}, bound *float64, upper bool) (exclusive bool, limit *float64) {
	var f float64
	switch v := value.(type) {
	case bool:
		return v, bound
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	case uint64:
		f = float64(v)
	case float64:
		f = v
	default:
		return false, bound
	}
	if bound != nil && ((!upper && *bound > f) || (upper && *bound < f)) {
		return false, bound
	}
	return true, &f
}
