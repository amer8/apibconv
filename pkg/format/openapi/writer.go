package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Writer implements the format.Writer interface for OpenAPI.
type Writer struct {
	version string
	indent  int
	yaml    bool
}

// WriterOption configures the Writer.
type WriterOption func(*Writer)

// WithWriterVersion sets the target OpenAPI version for writing.
func WithWriterVersion(version string) WriterOption {
	return func(w *Writer) {
		w.version = version
	}
}

// WithIndent sets the indentation level for output.
func WithIndent(spaces int) WriterOption {
	return func(w *Writer) {
		w.indent = spaces
	}
}

// WithYAML forces YAML output.
func WithYAML(forceYAML bool) WriterOption {
	return func(w *Writer) {
		w.yaml = forceYAML
	}
}

// WithJSONOutput sets the writer to output JSON.
func WithJSONOutput(enabled bool) WriterOption {
	return func(w *Writer) {
		w.yaml = !enabled
	}
}

// NewWriter creates a new OpenAPI writer with optional configurations.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		version: "3.0.0",
		indent:  2,
		yaml:    false,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Write writes the unified API model to an OpenAPI document.
func (w *Writer) Write(ctx context.Context, api *model.API, wr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Check context for overrides
	if v := converter.OpenAPIVersionFromContext(ctx); v != "" {
		w.version = v
	}
	encoding := converter.GetEncoding(ctx)
	if encoding == "" {
		if w.yaml {
			encoding = "yaml"
		} else {
			encoding = "json"
		}
	}

	var doc interface{}
	var err error

	switch w.version {
	case "2.0", "2.0.0":
		doc, err = w.convertV2(api)
	case "3.0", "3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1", "3.1.0": // All OpenAPI 3.x
		doc = w.convertFromModel(api)
	default:
		// Default to V3 if unknown
		doc = w.convertFromModel(api)
	}

	if err != nil {
		return err
	}

	if encoding == "json" {
		encoder := json.NewEncoder(wr)
		encoder.SetIndent("", "  ")
		return encoder.Encode(doc)
	}

	// Default to YAML
	encoder := yaml.NewEncoder(wr)
	encoder.SetIndent(w.indent)
	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("openapi write failed: %w", err)
	}
	return encoder.Close()
}

// Format returns the format type for the writer.
func (w *Writer) Format() format.Format {
	return format.FormatOpenAPI
}

// Version returns the OpenAPI version being written.
func (w *Writer) Version() string {
	return w.version
}

func (w *Writer) convertPathItemOperations(item *model.PathItem) PathItem {
	pi := PathItem{
		Summary:     item.Summary,
		Description: item.Description,
		Parameters:  w.convertParameters(item.Parameters),
	}

	if item.Get != nil {
		pi.Get = w.convertOperation(item.Get)
	}
	if item.Post != nil {
		pi.Post = w.convertOperation(item.Post)
	}
	if item.Put != nil {
		pi.Put = w.convertOperation(item.Put)
	}
	if item.Delete != nil {
		pi.Delete = w.convertOperation(item.Delete)
	}
	if item.Patch != nil {
		pi.Patch = w.convertOperation(item.Patch)
	}
	if item.Head != nil {
		pi.Head = w.convertOperation(item.Head)
	}
	if item.Options != nil {
		pi.Options = w.convertOperation(item.Options)
	}
	if item.Trace != nil {
		pi.Trace = w.convertOperation(item.Trace)
	}

	return pi
}

func (w *Writer) convertFromModel(api *model.API) *OpenAPI {
	doc := &OpenAPI{
		OpenAPI: w.version, // Use writer's version
		Info: Info{
			Title:          api.Info.Title,
			Description:    api.Info.Description,
			Version:        api.Info.Version,
			TermsOfService: api.Info.TermsOfService,
		},
		Paths: make(map[string]PathItem),
	}

	if api.Info.Contact != nil {
		doc.Info.Contact = &Contact{
			Name:  api.Info.Contact.Name,
			URL:   api.Info.Contact.URL,
			Email: api.Info.Contact.Email,
		}
	}
	if api.Info.License != nil {
		doc.Info.License = &License{
			Name: api.Info.License.Name,
			URL:  api.Info.License.URL,
		}
	}

	for _, s := range api.Servers {
		srv := Server{
			URL:         s.URL,
			Description: s.Description,
			Variables:   make(map[string]ServerVariable),
		}
		for k, v := range s.Variables {
			srv.Variables[k] = ServerVariable{
				Default:     v.Default,
				Description: v.Description,
				Enum:        v.Enum,
			}
		}
		doc.Servers = append(doc.Servers, srv)
	}

	// Components
	if len(api.Components.Schemas) > 0 {
		doc.Components.Schemas = make(map[string]*Schema)
		for name, schema := range api.Components.Schemas {
			doc.Components.Schemas[name] = w.convertSchema(schema)
		}
	}

	// Ensure Error schema is present, as it's used by the default 404 response
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = make(map[string]*Schema)
	}
	if _, ok := doc.Components.Schemas["Error"]; !ok {
		doc.Components.Schemas["Error"] = &Schema{
			Type:     "object",
			Required: []string{"code", "message"},
			Properties: map[string]*Schema{
				"code":    {Type: "integer"},
				"message": {Type: "string"},
			},
		}
	}

	// Webhooks (OpenAPI 3.1)
	if w.version >= "3.1" && len(api.Webhooks) > 0 {
		doc.Webhooks = make(map[string]PathItem)
		for name := range api.Webhooks {
			item := api.Webhooks[name]
			doc.Webhooks[name] = w.convertPathItemOperations(&item)
		}
	}

	for path := range api.Paths {
		item := api.Paths[path]
		key := path
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
		doc.Paths[key] = w.convertPathItemOperations(&item)
	}

	return doc
}

func (w *Writer) convertOperation(op *model.Operation) *Operation {
	if op == nil {
		return nil
	}
	res := &Operation{
		Tags:        op.Tags,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationID,
		Deprecated:  op.Deprecated,
		Parameters:  w.convertParameters(op.Parameters),
	}

	if op.RequestBody != nil {
		res.RequestBody = &RequestBody{
			Description: op.RequestBody.Description,
			Required:    op.RequestBody.Required,
			Content:     make(map[string]MediaType),
		}
		for k, v := range op.RequestBody.Content {
			res.RequestBody.Content[k] = w.convertMediaType(v)
		}
	}

	res.Responses = make(map[string]Response)
	found4xx := false
	if len(op.Responses) > 0 {
		for status, resp := range op.Responses {
			if len(status) == 3 && status[0] == '4' {
				found4xx = true
			}
			res.Responses[status] = w.convertResponse(resp)
		}
	}

	// Add a default 404 response if no 4XX response is present.
	if !found4xx {
		res.Responses["404"] = Response{
			Description: "Not Found",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: "#/components/schemas/Error",
					},
				},
			},
		}
	}

	return res
}

func (w *Writer) convertParameters(params []model.Parameter) []Parameter {
	result := make([]Parameter, 0, len(params))
	for _, param := range params {
		p := Parameter{
			Name:            param.Name,
			In:              string(param.In),
			Description:     param.Description,
			Required:        param.Required,
			Deprecated:      param.Deprecated,
			AllowEmptyValue: param.AllowEmptyValue,
			Style:           param.Style,
			Explode:         param.Explode,
			Example:         param.Example,
		}
		if param.Schema != nil {
			p.Schema = w.convertSchema(param.Schema)
		}
		if len(param.Content) > 0 {
			p.Content = make(map[string]MediaType)
			for k, v := range param.Content {
				p.Content[k] = w.convertMediaType(v)
			}
		}
		result = append(result, p)
	}
	return result
}

func (w *Writer) convertResponse(resp model.Response) Response {
	res := Response{
		Description: resp.Description,
		Content:     make(map[string]MediaType),
		Headers:     make(map[string]Header),
	}

	for ct, mt := range resp.Content {
		res.Content[ct] = w.convertMediaType(mt)
	}

	for name, h := range resp.Headers {
		header := Header{
			Description: h.Description,
			Required:    h.Required,
			Deprecated:  h.Deprecated,
		}
		if h.Schema != nil {
			header.Schema = w.convertSchema(h.Schema)
		}
		res.Headers[name] = header
	}

	return res
}

func (w *Writer) convertMediaType(mt model.MediaType) MediaType {
	res := MediaType{
		Example: mt.Example,
	}
	if mt.Schema != nil {
		res.Schema = w.convertSchema(mt.Schema)
	}
	return res
}

func (w *Writer) convertSchema(s *model.Schema) *Schema {
	if s == nil {
		return nil
	}
	ms := &Schema{
		Format:      s.Format,
		Title:       s.Title,
		Description: s.Description,
		Default:     s.Default,
		Enum:        s.Enum,
		Required:    s.Required,
	}

	// Rewrite References
	if s.Ref != "" {
		ref := s.Ref
		if strings.HasPrefix(w.version, "2") {
			// OpenAPI 2.0 uses #/definitions/
			if strings.HasPrefix(ref, "#/components/schemas/") {
				ref = strings.Replace(ref, "#/components/schemas/", "#/definitions/", 1)
			} else if strings.HasPrefix(ref, "#/components/messages/") {
				ref = strings.Replace(ref, "#/components/messages/", "#/definitions/", 1)
			}
		} else {
			// OpenAPI 3.x uses #/components/schemas/
			if strings.HasPrefix(ref, "#/components/messages/") {
				ref = strings.Replace(ref, "#/components/messages/", "#/components/schemas/", 1)
			}
		}
		ms.Ref = ref
	}

	if s.Type != "" {
		if s.Nullable && w.version == "3.1.0" { // Assuming we target 3.1.0 for this logic
			ms.Type = []interface{}{string(s.Type), "null"}
		} else {
			ms.Type = string(s.Type)
		}
	}

	if len(s.Properties) > 0 {
		ms.Properties = make(map[string]*Schema)
		for k, v := range s.Properties {
			ms.Properties[k] = w.convertSchema(v)
		}
	}

	if s.Items != nil {
		ms.Items = w.convertSchema(s.Items)
	}

	// Composition
	for _, subSchema := range s.AllOf {
		ms.AllOf = append(ms.AllOf, w.convertSchema(subSchema))
	}
	for _, subSchema := range s.AnyOf {
		ms.AnyOf = append(ms.AnyOf, w.convertSchema(subSchema))
	}
	for _, subSchema := range s.OneOf {
		ms.OneOf = append(ms.OneOf, w.convertSchema(subSchema))
	}
	if s.Not != nil {
		ms.Not = w.convertSchema(s.Not)
	}

	return ms
}

func (w *Writer) convertV2(api *model.API) (interface{}, error) {
	doc := &Swagger20{
		Swagger: "2.0",
		Info: Info{
			Title:          api.Info.Title,
			Description:    api.Info.Description,
			Version:        api.Info.Version,
			TermsOfService: api.Info.TermsOfService,
		},
		Paths: make(map[string]PathItemV2),
	}

	if api.Info.Contact != nil {
		doc.Info.Contact = &Contact{
			Name:  api.Info.Contact.Name,
			URL:   api.Info.Contact.URL,
			Email: api.Info.Contact.Email,
		}
	}
	if api.Info.License != nil {
		doc.Info.License = &License{
			Name: api.Info.License.Name,
			URL:  api.Info.License.URL,
		}
	}

	// Basic Server mapping (V2 supports one Host+BasePath or Schemes)
	if len(api.Servers) > 0 {
		// Just take the first one for now as V2 is more restrictive
		// TODO: Parse URL to split into host/basePath/scheme
		doc.Host = api.Servers[0].URL
	}

	// Components -> Definitions
	if len(api.Components.Schemas) > 0 {
		doc.Definitions = make(map[string]*Schema)
		for name, schema := range api.Components.Schemas {
			doc.Definitions[name] = w.convertSchema(schema)
		}
	}

	// Ensure Error definition is present, as it's used by the default 404 response
	if doc.Definitions == nil {
		doc.Definitions = make(map[string]*Schema)
	}
	if _, ok := doc.Definitions["Error"]; !ok {
		doc.Definitions["Error"] = &Schema{
			Type:     "object",
			Required: []string{"code", "message"},
			Properties: map[string]*Schema{
				"code":    {Type: "integer"},
				"message": {Type: "string"},
			},
		}
	}

	for path := range api.Paths {
		item := api.Paths[path] // Access by key to avoid rangeValCopy
		pi := PathItemV2{
			Parameters: w.convertParameters(item.Parameters),
		}

		convertOp := func(op *model.Operation) *Operation {
			if op == nil {
				return nil
			}
			res := w.convertOperation(op)
			return res
		}

		if item.Get != nil {
			pi.Get = convertOp(item.Get)
		}
		if item.Post != nil {
			pi.Post = convertOp(item.Post)
		}
		if item.Put != nil {
			pi.Put = convertOp(item.Put)
		}
		if item.Delete != nil {
			pi.Delete = convertOp(item.Delete)
		}
		if item.Patch != nil {
			pi.Patch = convertOp(item.Patch)
		}
		if item.Head != nil {
			pi.Head = convertOp(item.Head)
		}
		if item.Options != nil {
			pi.Options = convertOp(item.Options)
		}

		key := path
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
		doc.Paths[key] = pi
	}

	return doc, nil
}
