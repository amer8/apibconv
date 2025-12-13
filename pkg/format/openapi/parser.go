// Package openapi implements the OpenAPI format parser and writer.
package openapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/amer8/apibconv/internal/fs"
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

// WithVersion sets the expected OpenAPI version for parsing (though parser auto-detects).
func WithVersion(version string) ParserOption {
	return func(p *Parser) {
		p.version = version
	}
}

// WithStrict enables strict validation during parsing.
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

	// 1. Detect Version
	var base struct {
		Swagger string `yaml:"swagger"`
		OpenAPI string `yaml:"openapi"`
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to detect openapi version: %w", err)
	}

	// 2. Parse based on version
	if base.Swagger == "2.0" {
		return p.parseV2(ctx, bytes.NewReader(data))
	}

	// Default to V3
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

// SupportsVersion returns true if the parser supports the given version string.
func (p *Parser) SupportsVersion(version string) bool {
	// Support 2.0, 3.0, 3.1
	return true
}

func (p *Parser) convertToModel(doc *OpenAPI) (*model.API, error) {
	api := model.NewAPI()
	api.Version = doc.OpenAPI

	// Info
	api.Info = model.Info{
		Title:          doc.Info.Title,
		Description:    doc.Info.Description,
		Version:        doc.Info.Version,
		TermsOfService: doc.Info.TermsOfService,
	}
	if doc.Info.Contact != nil {
		api.Info.Contact = &model.Contact{
			Name:  doc.Info.Contact.Name,
			URL:   doc.Info.Contact.URL,
			Email: doc.Info.Contact.Email,
		}
	}
	if doc.Info.License != nil {
		api.Info.License = &model.License{
			Name: doc.Info.License.Name,
			URL:  doc.Info.License.URL,
		}
	}

	// Servers
	for _, s := range doc.Servers {
		ms := model.Server{
			URL:         s.URL,
			Description: s.Description,
			Variables:   make(map[string]model.ServerVariable),
		}
		for k, v := range s.Variables {
			ms.Variables[k] = model.ServerVariable{
				Default:     v.Default,
				Description: v.Description,
				Enum:        v.Enum,
			}
		}
		api.Servers = append(api.Servers, ms)
	}

	// Components (Schemas only for now)
	if doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas {
			api.Components.AddSchema(name, p.convertSchema(schema))
		}
	}

	// Webhooks (OpenAPI 3.1)
	if doc.Webhooks != nil {
		api.Webhooks = make(map[string]model.PathItem)
		for name := range doc.Webhooks {
			item := doc.Webhooks[name] // Access by key to avoid copy in range
			pi := model.PathItem{
				Summary:     item.Summary,
				Description: item.Description,
				Parameters:  p.convertParameters(item.Parameters),
			}
			if item.Get != nil {
				pi.Get = p.convertOperation(item.Get)
			}
			if item.Post != nil {
				pi.Post = p.convertOperation(item.Post)
			}
			if item.Put != nil {
				pi.Put = p.convertOperation(item.Put)
			}
			if item.Delete != nil {
				pi.Delete = p.convertOperation(item.Delete)
			}
			if item.Patch != nil {
				pi.Patch = p.convertOperation(item.Patch)
			}
			if item.Head != nil {
				pi.Head = p.convertOperation(item.Head)
			}
			if item.Options != nil {
				pi.Options = p.convertOperation(item.Options)
			}
			if item.Trace != nil {
				pi.Trace = p.convertOperation(item.Trace)
			}

			api.Webhooks[name] = pi
		}
	}

	// Paths
	for path := range doc.Paths {
		item := doc.Paths[path] // Access by key to avoid copy in range
		pi := model.PathItem{
			Summary:     item.Summary,
			Description: item.Description,
			Parameters:  p.convertParameters(item.Parameters),
		}
		// Operations
		if item.Get != nil {
			pi.Get = p.convertOperation(item.Get)
		}
		if item.Post != nil {
			pi.Post = p.convertOperation(item.Post)
		}
		if item.Put != nil {
			pi.Put = p.convertOperation(item.Put)
		}
		if item.Delete != nil {
			pi.Delete = p.convertOperation(item.Delete)
		}
		if item.Patch != nil {
			pi.Patch = p.convertOperation(item.Patch)
		}
		if item.Head != nil {
			pi.Head = p.convertOperation(item.Head)
		}
		if item.Options != nil {
			pi.Options = p.convertOperation(item.Options)
		}
		if item.Trace != nil {
			pi.Trace = p.convertOperation(item.Trace)
		}

		api.AddPath(path, &pi)
	}

	return api, nil
}

func (p *Parser) convertOperation(op *Operation) *model.Operation {
	if op == nil {
		return nil
	}
	res := &model.Operation{
		Tags:        op.Tags,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationID,
		Deprecated:  op.Deprecated,
		Parameters:  p.convertParameters(op.Parameters),
	}

	if op.RequestBody != nil {
		res.RequestBody = &model.RequestBody{
			Description: op.RequestBody.Description,
			Required:    op.RequestBody.Required,
			Content:     make(map[string]model.MediaType),
		}
		for k, v := range op.RequestBody.Content {
			res.RequestBody.Content[k] = p.convertMediaType(v)
		}
	}

	if op.Responses != nil {
		res.Responses = make(model.Responses)
		for status, resp := range op.Responses {
			res.Responses[status] = p.convertResponse(resp)
		}
	}

	return res
}

func (p *Parser) convertParameters(params []Parameter) []model.Parameter {
	result := make([]model.Parameter, 0, len(params)) // Pre-allocate
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
			Explode:         param.Explode,
			Example:         param.Example,
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

func (p *Parser) convertResponse(resp Response) model.Response {
	res := model.Response{
		Description: resp.Description,
		Content:     make(map[string]model.MediaType),
		Headers:     make(map[string]model.Header),
	}

	for ct, mt := range resp.Content {
		res.Content[ct] = p.convertMediaType(mt)
	}

	for name, h := range resp.Headers {
		header := model.Header{
			Description: h.Description,
			Required:    h.Required,
			Deprecated:  h.Deprecated,
		}
		if h.Schema != nil {
			header.Schema = p.convertSchema(h.Schema)
		}
		res.Headers[name] = header
	}

	return res
}

func (p *Parser) convertMediaType(mt MediaType) model.MediaType {
	res := model.MediaType{
		Example: mt.Example,
	}
	if mt.Schema != nil {
		res.Schema = p.convertSchema(mt.Schema)
	}
	return res
}

func (p *Parser) convertSchema(s *Schema) *model.Schema {
	if s == nil {
		return nil
	}
	ms := &model.Schema{
		Ref:         s.Ref,
		Format:      s.Format,
		Title:       s.Title,
		Description: s.Description,
		Default:     s.Default,
		Enum:        s.Enum,
		Required:    s.Required,
	}

	// Rewrite Swagger 2.0 definitions to OpenAPI 3.0 components/schemas
	if ms.Ref != "" && strings.HasPrefix(ms.Ref, "#/definitions/") {
		ms.Ref = strings.Replace(ms.Ref, "#/definitions/", "#/components/schemas/", 1)
	} // Type handling (simple string vs array in 3.1)
	if s.Type != nil {
		switch t := s.Type.(type) {
		case string:
			ms.Type = model.SchemaType(t)
		case []interface{}: // Handle OpenAPI 3.1 type: ["string", "null"]
			for _, item := range t {
				if typeStr, ok := item.(string); ok {
					if typeStr == "null" {
						ms.Nullable = true // Mark as nullable in the model
					} else {
						ms.Type = model.SchemaType(typeStr) // Set the actual type
					}
				}
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
