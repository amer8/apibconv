package openapi

import (
	"context"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/amer8/apibconv/pkg/model"
)

// Swagger20 represents the root of a Swagger 2.0 document.
type Swagger20 struct {
	Swagger             string                    `yaml:"swagger" json:"swagger"`
	Info                Info                      `yaml:"info" json:"info"`
	Host                string                    `yaml:"host,omitempty" json:"host,omitempty"`
	BasePath            string                    `yaml:"basePath,omitempty" json:"basePath,omitempty"`
	Schemes             []string                  `yaml:"schemes,omitempty" json:"schemes,omitempty"`
	Consumes            []string                  `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Produces            []string                  `yaml:"produces,omitempty" json:"produces,omitempty"`
	Paths               map[string]PathItemV2     `yaml:"paths" json:"paths"`
	Definitions         map[string]*Schema        `yaml:"definitions,omitempty" json:"definitions,omitempty"`
	Parameters          map[string]Parameter      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Responses           map[string]Response       `yaml:"responses,omitempty" json:"responses,omitempty"`
	SecurityDefinitions map[string]SecurityScheme `yaml:"securityDefinitions,omitempty" json:"securityDefinitions,omitempty"`
	Security            []map[string][]string     `yaml:"security,omitempty" json:"security,omitempty"`
	Tags                []Tag                     `yaml:"tags,omitempty" json:"tags,omitempty"`
	ExternalDocs        *ExternalDocs             `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
}

// PathItemV2 describes the operations available on a single path in Swagger 2.0.
type PathItemV2 struct {
	Ref        string      `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Get        *Operation  `yaml:"get,omitempty" json:"get,omitempty"`
	Put        *Operation  `yaml:"put,omitempty" json:"put,omitempty"`
	Post       *Operation  `yaml:"post,omitempty" json:"post,omitempty"`
	Delete     *Operation  `yaml:"delete,omitempty" json:"delete,omitempty"`
	Options    *Operation  `yaml:"options,omitempty" json:"options,omitempty"`
	Head       *Operation  `yaml:"head,omitempty" json:"head,omitempty"`
	Patch      *Operation  `yaml:"patch,omitempty" json:"patch,omitempty"`
	Parameters []Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

func (p *Parser) parseV2(ctx context.Context, r io.Reader) (*model.API, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var doc Swagger20
	if err := yaml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("swagger 2.0 parse failed: %w", err)
	}

	api := model.NewAPI()
	api.Version = doc.Swagger

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

	// Servers (Host + BasePath + Schemes -> Servers)
	if doc.Host != "" {
		schemes := doc.Schemes
		if len(schemes) == 0 {
			schemes = []string{"https"} // Default
		}
		for _, scheme := range schemes {
			url := fmt.Sprintf("%s://%s%s", scheme, doc.Host, doc.BasePath)
			api.Servers = append(api.Servers, model.Server{
				URL: url,
			})
		}
	}

	// Definitions -> Components/Schemas
	if doc.Definitions != nil {
		for name, schema := range doc.Definitions {
			api.Components.AddSchema(name, p.convertSchema(schema))
		}
	}

	// Paths
	for path, item := range doc.Paths {
		pi := model.PathItem{
			Parameters: p.convertV2Parameters(item.Parameters),
		}

		convertOp := func(op *Operation) *model.Operation {
			return p.convertV2Operation(op, doc.Consumes, doc.Produces)
		}

		pi.Get = convertOp(item.Get)
		pi.Post = convertOp(item.Post)
		pi.Put = convertOp(item.Put)
		pi.Delete = convertOp(item.Delete)
		pi.Patch = convertOp(item.Patch)
		pi.Head = convertOp(item.Head)
		pi.Options = convertOp(item.Options)

		api.AddPath(path, &pi)
	}

	return api, nil
}

func (p *Parser) convertV2Operation(op *Operation, globalConsumes, globalProduces []string) *model.Operation {
	if op == nil {
		return nil
	}

	// Determine effective media types
	consumes := globalConsumes
	if len(op.Consumes) > 0 {
		consumes = op.Consumes
	}
	if len(consumes) == 0 {
		consumes = []string{"application/json"} // Default
	}

	produces := globalProduces
	if len(op.Produces) > 0 {
		produces = op.Produces
	}
	if len(produces) == 0 {
		produces = []string{"application/json"} // Default
	}

	res := &model.Operation{
		Tags:        op.Tags,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationID,
		Deprecated:  op.Deprecated,
		Parameters:  p.convertV2Parameters(op.Parameters),
	}

	// In V2, body parameters are in the parameters list, handled by convertV2Parameters.
	// But in Model/V3, RequestBody is separate.
	// We need to extract the 'body' parameter if it exists and move it to RequestBody.
	var parameters []model.Parameter
	for _, param := range res.Parameters {
		if param.In == "body" {
			// Found body parameter, move to RequestBody
			content := make(map[string]model.MediaType)
			for _, ct := range consumes {
				content[ct] = model.MediaType{
					Schema: param.Schema,
				}
			}

			res.RequestBody = &model.RequestBody{
				Description: param.Description,
				Required:    param.Required,
				Content:     content,
			}
		} else {
			parameters = append(parameters, param)
		}
	}
	res.Parameters = parameters

	if op.Responses != nil {
		res.Responses = make(model.Responses)
		for status, resp := range op.Responses {
			res.Responses[status] = p.convertV2Response(resp, produces)
		}
	}

	return res
}

func (p *Parser) convertV2Parameters(params []Parameter) []model.Parameter {
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
			Explode:         param.Explode,
			Example:         param.Example,
		}

		// Handle V2 body parameter specifically
		if param.In == "body" {
			if param.Schema != nil {
				mp.Schema = p.convertSchema(param.Schema)
			}
		} else {
			// For non-body parameters in V2, type/format/items are direct fields.
			// In V3/Model, they must be wrapped in a Schema.
			if param.Type != "" {
				schema := &model.Schema{
					Type:   model.SchemaType(param.Type),
					Format: param.Format,
				}
				if param.Items != nil {
					schema.Items = p.convertSchema(param.Items)
				}
				mp.Schema = schema
			} else if param.Schema != nil {
				// Fallback if schema is somehow used
				mp.Schema = p.convertSchema(param.Schema)
			}
		}
		
		result = append(result, mp)
	}
	return result
}

func (p *Parser) convertV2Response(resp Response, produces []string) model.Response {
	res := model.Response{
		Description: resp.Description,
		Content:     make(map[string]model.MediaType),
		Headers:     make(map[string]model.Header),
	}

	// V2 Response has a direct Schema field.
	// In V3/Model this maps to Content -> application/json (or others) -> Schema
	if resp.Schema != nil {
		for _, ct := range produces {
			res.Content[ct] = model.MediaType{
				Schema: p.convertSchema(resp.Schema),
			}
		}
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
