package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/amer8/apibconv/internal/security"
	"github.com/amer8/apibconv/pkg/converter"
	apierrors "github.com/amer8/apibconv/pkg/errors"
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

// WithIndent sets the YAML indentation level. JSON output uses two spaces.
func WithIndent(spaces int) WriterOption {
	return func(w *Writer) {
		w.indent = spaces
	}
}

// WithYAML selects YAML when enabled and JSON when disabled. A context encoding
// set with converter.WithEncoding takes precedence.
func WithYAML(forceYAML bool) WriterOption {
	return func(w *Writer) {
		w.yaml = forceYAML
	}
}

// WithJSONOutput selects JSON when enabled and YAML when disabled. A context
// encoding set with converter.WithEncoding takes precedence.
func WithJSONOutput(enabled bool) WriterOption {
	return func(w *Writer) {
		w.yaml = !enabled
	}
}

// NewWriter creates a new OpenAPI writer with optional configurations.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		version: "3.1.0",
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
	local := *w
	w = &local
	if err := security.CheckServerUnsupported(api, "OpenAPI"); err != nil {
		return err
	}
	if err := security.CheckChannelServers(api, "OpenAPI"); err != nil {
		return err
	}
	for name := range api.Components.SecuritySchemes {
		scheme := api.Components.SecuritySchemes[name]
		if scheme.Ref != "" {
			continue
		}
		switch scheme.Type {
		case model.SecurityTypeAPIKey, model.SecurityTypeHTTP, model.SecurityTypeOAuth2, model.SecurityTypeOpenIDConnect, "mutualTLS":
		default:
			return fmt.Errorf("%w in OpenAPI output: security scheme %q has unsupported type %q", apierrors.ErrUnsupportedSecurity, name, scheme.Type)
		}
	}
	if v := converter.OpenAPIVersionFromContext(ctx); v != "" {
		w.version = v
	}

	switch w.version {
	case "3.0":
		w.version = "3.0.0"
	case "3.1":
		w.version = "3.1.0"
	}

	encoding := converter.GetEncoding(ctx)
	if encoding == "" {
		if w.yaml {
			encoding = "yaml"
		} else {
			encoding = "json"
		}
	}
	if encoding != "json" && encoding != "yaml" {
		return fmt.Errorf("unsupported OpenAPI output encoding %q", encoding)
	}

	if err := validateSchemaCompatibility(api, w.version); err != nil {
		return err
	}

	var doc interface{}
	var err error

	switch w.version {
	case "2.0", "2.0.0":
		if err := security.CheckCallbacks(api, "OpenAPI 2.0"); err != nil {
			return err
		}
		if len(api.Webhooks) > 0 {
			return fmt.Errorf("OpenAPI 2.0 output cannot represent OpenAPI webhooks without loss")
		}
		if err := security.CheckServerOverrides(api, "OpenAPI 2.0"); err != nil {
			return err
		}
		doc, err = w.convertV2(api)
	case "3.0", "3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1", "3.1.0":
		if strings.HasPrefix(w.version, "3.0") {
			if len(api.Webhooks) > 0 {
				return fmt.Errorf("OpenAPI 3.0 output cannot represent OpenAPI webhooks without loss")
			}
			for name := range api.Components.SecuritySchemes {
				if api.Components.SecuritySchemes[name].Type == "mutualTLS" {
					return fmt.Errorf("security scheme %q: mutualTLS cannot be represented in OpenAPI 3.0", name)
				}
			}
		}
		doc = w.convertFromModel(api)
	default:
		return fmt.Errorf("unsupported OpenAPI output version %q", w.version)
	}

	if err != nil {
		return err
	}

	if encoding == "json" {
		encoder := json.NewEncoder(wr)
		encoder.SetIndent("", "  ")
		return encoder.Encode(doc)
	}

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

func (w *Writer) convertPathItem(item *model.PathItem, api *model.API) PathItem {
	return PathItem{
		Summary:     item.Summary,
		Description: item.Description,
		Parameters:  w.convertParameters(item.Parameters, api),
		Servers:     serversFromModel(item.Servers),
		Get:         w.convertOperation(item.Get, api),
		Put:         w.convertOperation(item.Put, api),
		Post:        w.convertOperation(item.Post, api),
		Delete:      w.convertOperation(item.Delete, api),
		Options:     w.convertOperation(item.Options, api),
		Head:        w.convertOperation(item.Head, api),
		Patch:       w.convertOperation(item.Patch, api),
		Trace:       w.convertOperation(item.Trace, api),
		Extensions:  clonedExtensions(item.Extensions),
	}
}

func (w *Writer) convertCallback(callback model.Callback, extensions map[string]interface{}, api *model.API) Callback {
	paths := make(map[string]PathItem, len(callback))
	for expression := range callback {
		item := callback[expression]
		paths[expression] = w.convertPathItem(&item, api)
	}
	return Callback{Paths: paths, Extensions: clonedExtensions(extensions)}
}

func (w *Writer) convertFromModel(api *model.API) *OpenAPI {
	doc := &OpenAPI{
		OpenAPI: w.version,
		Info: Info{
			Title:          api.Info.Title,
			Description:    api.Info.Description,
			Version:        api.Info.Version,
			TermsOfService: api.Info.TermsOfService,
			Extensions:     clonedExtensions(api.Info.Extensions),
		},
		Paths:           make(map[string]PathItem),
		PathsExtensions: clonedExtensions(api.PathsExtensions),
		Security:        securityRequirementsFromModel(api.Security),
		Tags:            tagsFromModel(api.Tags),
		ExternalDocs:    externalDocsFromModel(api.ExternalDocs),
	}
	if len(api.Extensions) > 0 {
		doc.Extensions = make(map[string]interface{})
		copyExtensions(doc.Extensions, api.Extensions)
	}
	if len(api.Components.SecuritySchemes) > 0 {
		doc.Components.SecuritySchemes = make(map[string]SecurityScheme, len(api.Components.SecuritySchemes))
		for name := range api.Components.SecuritySchemes {
			scheme := api.Components.SecuritySchemes[name]
			doc.Components.SecuritySchemes[name] = securitySchemeFromModel(&scheme)
		}
	}
	if len(api.Components.Callbacks) > 0 {
		doc.Components.Callbacks = make(map[string]Callback, len(api.Components.Callbacks))
		for name, callback := range api.Components.Callbacks {
			doc.Components.Callbacks[name] = w.convertCallback(callback, api.Components.CallbackExtensions[name], api)
		}
	}

	if api.Info.Contact != nil {
		doc.Info.Contact = &Contact{
			Name:       api.Info.Contact.Name,
			URL:        api.Info.Contact.URL,
			Email:      api.Info.Contact.Email,
			Extensions: clonedExtensions(api.Info.Contact.Extensions),
		}
	}
	if api.Info.License != nil {
		doc.Info.License = &License{
			Name:       api.Info.License.Name,
			URL:        api.Info.License.URL,
			Extensions: clonedExtensions(api.Info.License.Extensions),
		}
	}

	for _, s := range api.Servers {
		srv := Server{
			URL:         s.URL,
			Description: s.Description,
			Variables:   make(map[string]ServerVariable),
			Extensions:  clonedExtensions(s.Extensions),
		}
		for k, v := range s.Variables {
			srv.Variables[k] = ServerVariable{
				Default:     v.Default,
				Description: v.Description,
				Enum:        v.Enum,
				Extensions:  clonedExtensions(v.Extensions),
			}
		}
		doc.Servers = append(doc.Servers, srv)
	}

	doc.Components.Extensions = clonedExtensions(api.Components.Extensions)

	// Components
	if len(api.Components.Schemas) > 0 {
		doc.Components.Schemas = make(map[string]*Schema)
		for name, schema := range api.Components.Schemas {
			doc.Components.Schemas[name] = w.convertSchema(schema, api)
		}
	}

	if len(api.Components.Examples) > 0 {
		doc.Components.Examples = examplesFromModel(api.Components.Examples)
	}
	if len(api.Components.Links) > 0 {
		doc.Components.Links = make(map[string]Link, len(api.Components.Links))
		for name, link := range api.Components.Links {
			doc.Components.Links[name] = linkFromModel(&link)
		}
	}
	if len(api.Components.Parameters) > 0 {
		doc.Components.Parameters = make(map[string]Parameter, len(api.Components.Parameters))
		for name := range api.Components.Parameters {
			parameter := api.Components.Parameters[name]
			doc.Components.Parameters[name] = w.convertParameters([]model.Parameter{parameter}, api)[0]
		}
	}
	if len(api.Components.Responses) > 0 {
		doc.Components.Responses = make(map[string]Response, len(api.Components.Responses))
		for name, response := range api.Components.Responses {
			doc.Components.Responses[name] = w.convertResponse(response, api)
		}
	}
	if len(api.Components.RequestBodies) > 0 {
		doc.Components.RequestBodies = make(map[string]RequestBody, len(api.Components.RequestBodies))
		for name, body := range api.Components.RequestBodies {
			doc.Components.RequestBodies[name] = w.convertRequestBody(&body, api)
		}
	}
	if len(api.Components.Headers) > 0 {
		doc.Components.Headers = make(map[string]Header, len(api.Components.Headers))
		for name, header := range api.Components.Headers {
			doc.Components.Headers[name] = w.convertHeader(header, api)
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
			doc.Webhooks[name] = w.convertPathItem(&item, api)
		}
	}

	for path := range api.Paths {
		item := api.Paths[path]
		key := path
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
		doc.Paths[key] = w.convertPathItem(&item, api)
	}

	return doc
}

func (w *Writer) convertOperation(op *model.Operation, api *model.API) *Operation {
	if op == nil {
		return nil
	}
	res := &Operation{
		Tags:                op.Tags,
		Summary:             op.Summary,
		Description:         op.Description,
		OperationID:         op.OperationID,
		ExternalDocs:        externalDocsFromModel(op.ExternalDocs),
		Deprecated:          op.Deprecated,
		Parameters:          w.convertParameters(op.Parameters, api),
		Security:            securityRequirementsFromModel(op.Security),
		Servers:             serversFromModel(op.Servers),
		Extensions:          clonedExtensions(op.Extensions),
		ResponsesExtensions: clonedExtensions(op.ResponsesExtensions),
	}
	if len(op.Callbacks) > 0 {
		res.Callbacks = make(map[string]Callback, len(op.Callbacks))
		for name, callback := range op.Callbacks {
			res.Callbacks[name] = w.convertCallback(callback, op.CallbackExtensions[name], api)
		}
	}

	if op.RequestBody != nil {
		body := w.convertRequestBody(op.RequestBody, api)
		res.RequestBody = &body
	}

	res.Responses = make(map[string]Response)
	found4xx := false
	if len(op.Responses) > 0 {
		for status, resp := range op.Responses {
			if len(status) == 3 && status[0] == '4' {
				found4xx = true
			}
			res.Responses[status] = w.convertResponse(resp, api)
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

func linkFromModel(link *model.Link) Link {
	result := Link{
		Ref: link.Ref, OperationRef: link.OperationRef, OperationID: link.OperationID,
		Parameters: link.Parameters, RequestBody: link.RequestBody,
		Description: link.Description, Extensions: clonedExtensions(link.Extensions),
	}
	if link.Server != nil {
		server := serversFromModel([]model.Server{*link.Server})[0]
		result.Server = &server
	}
	return result
}

func examplesFromModel(examples map[string]model.Example) map[string]interface{} {
	if examples == nil {
		return nil
	}
	result := make(map[string]interface{}, len(examples))
	for name, example := range examples {
		result[name] = Example{
			Ref: example.Ref, Summary: example.Summary, Description: example.Description,
			Value: example.Value, ExternalValue: example.ExternalValue,
			Extensions: clonedExtensions(example.Extensions),
		}
	}
	return result
}

func externalDocsFromModel(docs *model.ExternalDocs) *ExternalDocs {
	if docs == nil {
		return nil
	}
	return &ExternalDocs{Description: docs.Description, URL: docs.URL, Extensions: clonedExtensions(docs.Extensions)}
}

func tagsFromModel(tags []model.Tag) []Tag {
	if tags == nil {
		return nil
	}
	result := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, Tag{Name: tag.Name, Description: tag.Description, ExternalDocs: externalDocsFromModel(tag.ExternalDocs), Extensions: clonedExtensions(tag.Extensions)})
	}
	return result
}

func serversFromModel(servers []model.Server) []Server {
	if len(servers) == 0 {
		return nil
	}
	result := make([]Server, 0, len(servers))
	for _, s := range servers {
		v := Server{URL: s.URL, Description: s.Description, Extensions: clonedExtensions(s.Extensions)}
		if s.Variables != nil {
			v.Variables = make(map[string]ServerVariable, len(s.Variables))
		}
		for k, x := range s.Variables {
			v.Variables[k] = ServerVariable{Default: x.Default, Description: x.Description, Enum: x.Enum, Extensions: clonedExtensions(x.Extensions)}
		}
		result = append(result, v)
	}
	return result
}

func (w *Writer) convertParameters(params []model.Parameter, api *model.API) []Parameter {
	result := make([]Parameter, 0, len(params))
	for i := range params {
		param := &params[i]
		p := Parameter{
			Name:            param.Name,
			In:              string(param.In),
			Description:     param.Description,
			Required:        param.Required,
			Deprecated:      param.Deprecated,
			AllowEmptyValue: param.AllowEmptyValue,
			Style:           param.Style,
			Example:         param.Example,
			Examples:        examplesFromModel(param.Examples),
			Extensions:      clonedExtensions(param.Extensions),
		}
		if param.ExplodeSet || param.Explode {
			value := param.Explode
			p.Explode = &value
		}
		if param.Schema != nil {
			p.Schema = w.convertSchema(param.Schema, api)
		}
		if len(param.Content) > 0 {
			p.Content = make(map[string]MediaType)
			for k, v := range param.Content {
				p.Content[k] = w.convertMediaType(v, api)
			}
		}
		if p.Schema == nil && len(p.Content) == 0 {
			p.Schema = &Schema{Type: "string"}
		}
		result = append(result, p)
	}
	return result
}

func (w *Writer) convertRequestBody(body *model.RequestBody, api *model.API) RequestBody {
	result := RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     make(map[string]MediaType),
		Extensions:  clonedExtensions(body.Extensions),
	}
	for mediaType, content := range body.Content {
		result.Content[mediaType] = w.convertMediaType(content, api)
	}
	return result
}

func (w *Writer) convertHeader(header model.Header, api *model.API) Header {
	result := Header{
		Description: header.Description,
		Required:    header.Required,
		Deprecated:  header.Deprecated,
		Example:     header.Example,
		Examples:    examplesFromModel(header.Examples),
		Extensions:  clonedExtensions(header.Extensions),
	}
	if header.Schema != nil {
		result.Schema = w.convertSchema(header.Schema, api)
	}
	return result
}

func (w *Writer) convertResponse(resp model.Response, api *model.API) Response {
	description := resp.Description
	if description == "" {
		description = "OK"
	}
	res := Response{
		Description: description,
		Content:     make(map[string]MediaType),
		Headers:     make(map[string]Header),
		Extensions:  clonedExtensions(resp.Extensions),
	}

	for ct, mt := range resp.Content {
		res.Content[ct] = w.convertMediaType(mt, api)
	}

	for name, header := range resp.Headers {
		res.Headers[name] = w.convertHeader(header, api)
	}

	if len(resp.Links) > 0 {
		res.Links = make(map[string]Link, len(resp.Links))
		for name, link := range resp.Links {
			res.Links[name] = linkFromModel(&link)
		}
	}
	return res
}

func (w *Writer) convertMediaType(mt model.MediaType, api *model.API) MediaType {
	res := MediaType{
		Example:    mt.Example,
		Examples:   examplesFromModel(mt.Examples),
		Extensions: clonedExtensions(mt.Extensions),
	}
	if mt.Schema != nil {
		res.Schema = w.convertSchema(mt.Schema, api)
	}
	if len(mt.Encoding) > 0 {
		res.Encoding = make(map[string]Encoding, len(mt.Encoding))
		for name, encoding := range mt.Encoding {
			item := Encoding{
				ContentType: encoding.ContentType, Style: encoding.Style,
				Explode: encoding.Explode, AllowReserved: encoding.AllowReserved,
				Extensions: clonedExtensions(encoding.Extensions),
			}
			if len(encoding.Headers) > 0 {
				item.Headers = make(map[string]Header, len(encoding.Headers))
				for headerName, header := range encoding.Headers {
					item.Headers[headerName] = w.convertHeader(header, api)
				}
			}
			res.Encoding[name] = item
		}
	}
	return res
}

func (w *Writer) convertOperationV2(op *model.Operation, api *model.API) *Operation {
	if op == nil {
		return nil
	}

	res := &Operation{
		Tags:                op.Tags,
		Summary:             op.Summary,
		Description:         op.Description,
		OperationID:         op.OperationID,
		ExternalDocs:        externalDocsFromModel(op.ExternalDocs),
		Deprecated:          op.Deprecated,
		Parameters:          w.convertParametersV2(op.Parameters, api),
		Security:            securityRequirementsFromModel(op.Security),
		Extensions:          clonedExtensions(op.Extensions),
		ResponsesExtensions: clonedExtensions(op.ResponsesExtensions),
	}

	if op.RequestBody != nil {
		contentTypes := sortedOpenAPIContentTypes(op.RequestBody.Content)
		if len(contentTypes) > 0 {
			res.Consumes = contentTypes
			for _, contentType := range contentTypes {
				if schema := op.RequestBody.Content[contentType].Schema; schema != nil {
					res.Parameters = append(res.Parameters, Parameter{
						Name:        "body",
						In:          "body",
						Description: op.RequestBody.Description,
						Required:    op.RequestBody.Required,
						Schema:      w.convertSchema(schema, api),
						Extensions:  clonedExtensions(op.RequestBody.Extensions),
					})
					break
				}
			}
		}
	}

	res.Responses = make(map[string]Response)
	found4xx := false
	if len(op.Responses) > 0 {
		for status, resp := range op.Responses {
			if len(status) == 3 && status[0] == '4' {
				found4xx = true
			}
			response := w.convertResponseV2(resp, api)
			for _, contentType := range sortedOpenAPIContentTypes(resp.Content) {
				if !slices.Contains(res.Produces, contentType) {
					res.Produces = append(res.Produces, contentType)
				}
			}
			res.Responses[status] = response
		}
	}

	if !found4xx {
		res.Responses["404"] = Response{
			Description: "Not Found",
			Schema: &Schema{
				Ref: "#/definitions/Error",
			},
		}
	}

	return res
}

func (w *Writer) convertParametersV2(params []model.Parameter, api *model.API) []Parameter {
	result := make([]Parameter, 0, len(params))
	for i := range params {
		param := &params[i]
		p := Parameter{
			Name:            param.Name,
			In:              string(param.In),
			Description:     param.Description,
			Required:        param.Required,
			AllowEmptyValue: param.AllowEmptyValue,
			Extensions:      clonedExtensions(param.Extensions),
		}
		if p.In == "" {
			p.In = "query"
		}

		schema := param.Schema
		if schema == nil {
			schema = &model.Schema{Type: model.TypeString}
		}
		normalized, _ := swaggerParameterSchema(schema, api) // Checked before writing.
		assignV2ParameterSchema(&p, w.convertSchema(normalized, api))
		if p.Type == "array" {
			switch param.Style {
			case "spaceDelimited":
				p.CollectionFormat = "ssv"
			case "pipeDelimited":
				p.CollectionFormat = "pipes"
			case "form":
				if param.Explode || !param.ExplodeSet {
					p.CollectionFormat = "multi"
				}
			case "":
				if p.In == "query" && (param.Explode || !param.ExplodeSet) {
					p.CollectionFormat = "multi"
				}
			}
		}
		result = append(result, p)
	}
	return result
}

func (w *Writer) convertResponseV2(resp model.Response, api *model.API) Response {
	description := resp.Description
	if description == "" {
		description = "OK"
	}
	res := Response{
		Description: description,
		Headers:     make(map[string]Header),
		Extensions:  clonedExtensions(resp.Extensions),
	}

	for _, contentType := range sortedOpenAPIContentTypes(resp.Content) {
		mt := resp.Content[contentType]
		if mt.Schema != nil {
			res.Schema = w.convertSchema(mt.Schema, api)
			break
		}
	}

	for name, h := range resp.Headers {
		header := Header{
			Description: h.Description,
			Extensions:  clonedExtensions(h.Extensions),
		}
		if h.Schema != nil {
			normalized, _ := swaggerHeaderSchema(h.Schema, api) // Checked before writing.
			assignV2HeaderSchema(&header, w.convertSchema(normalized, api))
		} else {
			assignV2HeaderSchema(&header, nil)
		}
		res.Headers[name] = header
	}

	return res
}

func assignV2ParameterSchema(param *Parameter, schema *Schema) {
	if schema == nil {
		param.Type = "string"
		return
	}
	if schema.Ref != "" {
		param.Type = "string"
		return
	}

	param.Type = swaggerType(schema.Type)

	param.Format = schema.Format
	param.Default, param.Enum = schema.Default, schema.Enum
	param.Minimum, param.Maximum, param.MultipleOf = schema.Minimum, schema.Maximum, schema.MultipleOf
	param.ExclusiveMinimum, _ = schema.ExclusiveMinimum.(bool)
	param.ExclusiveMaximum, _ = schema.ExclusiveMaximum.(bool)
	param.MinLength, param.MaxLength, param.Pattern = schema.MinLength, schema.MaxLength, schema.Pattern
	param.MinItems, param.MaxItems, param.UniqueItems = schema.MinItems, schema.MaxItems, schema.UniqueItems
	if param.Type == "array" && schema.Items != nil {
		param.Items = schema.Items
	}
}

func assignV2HeaderSchema(header *Header, schema *Schema) {
	if schema == nil {
		header.Type = "string"
		return
	}
	if schema.Ref != "" {
		header.Type = "string"
		return
	}

	header.Type = swaggerType(schema.Type)

	header.Format = schema.Format
	if header.Type == "array" && schema.Items != nil {
		header.Items = schema.Items
	}
}

// Swagger parameters and headers use a single primitive type.
func swaggerType(value interface{}) string {
	switch value := value.(type) {
	case string:
		if value != "" && value != "object" {
			return value
		}
	case []interface{}:
		for _, item := range value {
			if typ, ok := item.(string); ok && typ != "null" {
				if typ != "" {
					return typ
				}
				break
			}
		}
	}
	return "string"
}

func sortedOpenAPIContentTypes(content map[string]model.MediaType) []string {
	contentTypes := make([]string, 0, len(content))
	for contentType := range content {
		contentTypes = append(contentTypes, contentType)
	}
	sort.Strings(contentTypes)
	return contentTypes
}

func convertServerToV2(rawURL string) (host, basePath string, schemes []string) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return rawURL, "", nil
	}

	if parsed.Scheme != "" {
		schemes = []string{parsed.Scheme}
	}
	basePath = parsed.EscapedPath()
	if basePath == "" {
		basePath = "/"
	}
	return parsed.Host, basePath, schemes
}

func (w *Writer) convertSchema(s *model.Schema, api *model.API) *Schema {
	if s == nil {
		return nil
	}
	ms := &Schema{
		Format:               s.Format,
		ExternalDocs:         externalDocsFromModel(s.ExternalDocs),
		Extensions:           clonedExtensions(s.Extensions),
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
	}
	if s.Discriminator != nil {
		ms.Discriminator = &Discriminator{
			PropertyName: s.Discriminator.PropertyName,
			Mapping:      s.Discriminator.Mapping,
			Extensions:   clonedExtensions(s.Discriminator.Extensions),
		}
	}
	switch nested := s.AdditionalProperties.(type) {
	case *model.Schema:
		ms.AdditionalProperties = w.convertSchema(nested, api)
	case map[string]interface{}:
		// Schemas decoded through JSON can contain an untyped nested object.
		if data, err := json.Marshal(s.AdditionalProperties); err == nil {
			var nested model.Schema
			if json.Unmarshal(data, &nested) == nil {
				ms.AdditionalProperties = w.convertSchema(&nested, api)
			}
		}
	}
	if s.Nullable && strings.HasPrefix(w.version, "3.0") {
		ms.Nullable = true
	}
	if s.ExclusiveMinimum {
		if strings.HasPrefix(w.version, "3.1") && s.Minimum != nil {
			ms.ExclusiveMinimum = *s.Minimum
			ms.Minimum = nil
		} else {
			ms.ExclusiveMinimum = true
		}
	}
	if s.ExclusiveMaximum {
		if strings.HasPrefix(w.version, "3.1") && s.Maximum != nil {
			ms.ExclusiveMaximum = *s.Maximum
			ms.Maximum = nil
		} else {
			ms.ExclusiveMaximum = true
		}
	}

	// Rewrite References
	if s.Ref != "" {
		ref := s.Ref

		// Attempt to resolve aliases (e.g. Message wrapper -> underlying Schema)
		if strings.HasPrefix(ref, "#/components/messages/") && api != nil && api.Components.Schemas != nil {
			name := strings.TrimPrefix(ref, "#/components/messages/")
			if target, ok := api.Components.Schemas[name]; ok && target.Ref != "" {
				// Use the underlying ref if it's just an alias
				ref = target.Ref
			}
		}

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
		if s.Nullable && w.version == "3.1.0" {
			ms.Type = []interface{}{string(s.Type), "null"}
		} else {
			ms.Type = string(s.Type)
		}
	}

	if len(s.Properties) > 0 {
		ms.Properties = make(map[string]*Schema)
		for k, v := range s.Properties {
			ms.Properties[k] = w.convertSchema(v, api)
		}
	}

	if s.Items != nil {
		ms.Items = w.convertSchema(s.Items, api)
	}

	// Composition
	for _, subSchema := range s.AllOf {
		ms.AllOf = append(ms.AllOf, w.convertSchema(subSchema, api))
	}
	for _, subSchema := range s.AnyOf {
		ms.AnyOf = append(ms.AnyOf, w.convertSchema(subSchema, api))
	}
	for _, subSchema := range s.OneOf {
		ms.OneOf = append(ms.OneOf, w.convertSchema(subSchema, api))
	}
	if s.Not != nil {
		ms.Not = w.convertSchema(s.Not, api)
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
			Extensions:     clonedExtensions(api.Info.Extensions),
		},
		Paths:           make(map[string]PathItemV2),
		PathsExtensions: clonedExtensions(api.PathsExtensions),
		Security:        securityRequirementsFromModel(api.Security),
		Tags:            tagsFromModel(api.Tags),
		ExternalDocs:    externalDocsFromModel(api.ExternalDocs),
	}
	if len(api.Extensions) > 0 {
		doc.Extensions = make(map[string]interface{})
		copyExtensions(doc.Extensions, api.Extensions)
	}

	if len(api.Components.SecuritySchemes) > 0 {
		doc.SecurityDefinitions = make(map[string]SecurityScheme, len(api.Components.SecuritySchemes))
		for name := range api.Components.SecuritySchemes {
			scheme := api.Components.SecuritySchemes[name]
			converted, err := securitySchemeToV2(&scheme)
			if err != nil {
				return nil, fmt.Errorf("security scheme %q: %w", name, err)
			}
			doc.SecurityDefinitions[name] = converted
		}
	}

	if api.Info.Contact != nil {
		doc.Info.Contact = &Contact{
			Name:       api.Info.Contact.Name,
			URL:        api.Info.Contact.URL,
			Email:      api.Info.Contact.Email,
			Extensions: clonedExtensions(api.Info.Contact.Extensions),
		}
	}
	if api.Info.License != nil {
		doc.Info.License = &License{
			Name:       api.Info.License.Name,
			URL:        api.Info.License.URL,
			Extensions: clonedExtensions(api.Info.License.Extensions),
		}
	}

	// Swagger 2.0 has one host and base path, so use the first server.
	if len(api.Servers) > 0 {
		doc.Host, doc.BasePath, doc.Schemes = convertServerToV2(api.Servers[0].URL)
	}

	// Components -> Definitions
	if len(api.Components.Schemas) > 0 {
		doc.Definitions = make(map[string]*Schema)
		for name, schema := range api.Components.Schemas {
			doc.Definitions[name] = w.convertSchema(schema, api)
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
			Parameters: w.convertParametersV2(item.Parameters, api),
			Extensions: clonedExtensions(item.Extensions),
		}

		if item.Get != nil {
			pi.Get = w.convertOperationV2(item.Get, api)
		}
		if item.Post != nil {
			pi.Post = w.convertOperationV2(item.Post, api)
		}
		if item.Put != nil {
			pi.Put = w.convertOperationV2(item.Put, api)
		}
		if item.Delete != nil {
			pi.Delete = w.convertOperationV2(item.Delete, api)
		}
		if item.Patch != nil {
			pi.Patch = w.convertOperationV2(item.Patch, api)
		}
		if item.Head != nil {
			pi.Head = w.convertOperationV2(item.Head, api)
		}
		if item.Options != nil {
			pi.Options = w.convertOperationV2(item.Options, api)
		}

		key := path
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
		doc.Paths[key] = pi
	}

	return doc, nil
}
