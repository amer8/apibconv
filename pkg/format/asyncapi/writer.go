package asyncapi

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/amer8/apibconv/internal/security"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Writer implements the format.Writer interface for AsyncAPI.
type Writer struct {
	version string
	json    bool
}

// WriterOption configures the Writer.
type WriterOption func(*Writer)

// WithAsyncWriterVersion sets the target AsyncAPI version for writing.
func WithAsyncWriterVersion(version string) WriterOption {
	return func(w *Writer) {
		w.version = version
	}
}

// WithJSONOutput sets the writer to output JSON.
func WithJSONOutput(enabled bool) WriterOption {
	return func(w *Writer) {
		w.json = enabled
	}
}

// NewWriter creates a new AsyncAPI writer with optional configurations.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		version: "3.0.0",
		json:    false,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Write writes the unified API model to an AsyncAPI document.
func (w *Writer) Write(ctx context.Context, api *model.API, wr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	local := *w
	w = &local
	if err := security.CheckServerOverrides(api, "AsyncAPI"); err != nil {
		return err
	}

	// Check context for overrides
	if v := converter.AsyncAPIVersionFromContext(ctx); v != "" {
		w.version = v
	}

	// Normalize version string
	switch w.version {
	case "2.0", "2.1", "2.2", "2.3", "2.4", "2.5":
		w.version += ".0"
	case "2.6":
		w.version = "2.6.0"
	case "3.0":
		w.version = "3.0.0"
	}
	switch w.version {
	case "2.0.0", "2.1.0", "2.2.0", "2.3.0", "2.4.0", "2.5.0", "2.6.0", "3.0.0":
	default:
		return fmt.Errorf("unsupported AsyncAPI output version %q", w.version)
	}
	if encoding := converter.GetEncoding(ctx); encoding != "" && encoding != "json" && encoding != "yaml" {
		return fmt.Errorf("unsupported AsyncAPI output encoding %q", encoding)
	}
	plan, err := prepareSecurity(api, w.version >= "3.0.0", converter.GetProtocol(ctx))
	if err != nil {
		return err
	}
	if err := security.CheckCallbacks(api, "AsyncAPI"); err != nil {
		return err
	}
	if len(api.Webhooks) > 0 {
		return fmt.Errorf("AsyncAPI output cannot represent OpenAPI webhooks without loss")
	}
	api, err = promoteOperationParameters(api)
	if err != nil {
		return err
	}
	for path := range api.Paths {
		item := api.Paths[path]
		if w.version < "3.0.0" && len(item.ChannelServers) != 0 && item.Get == nil && item.Post == nil {
			return fmt.Errorf("channel %q has server restrictions without a representable operation", path)
		}
		names := make(map[string]bool, len(api.Servers))
		for i, server := range api.Servers {
			name := server.Name
			if name == "" {
				name = fmt.Sprintf("server%d", i)
			}
			names[name] = true
		}
		for _, name := range item.ChannelServers {
			if !names[name] {
				return fmt.Errorf("channel %q references unknown server %q", path, name)
			}
		}
		if item.Put != nil || item.Delete != nil || item.Patch != nil || item.Head != nil || item.Options != nil || item.Trace != nil {
			return fmt.Errorf("path %q has operations that cannot be represented in AsyncAPI", path)
		}
		for _, param := range item.Parameters {
			if param.In != model.ParameterInPath {
				return fmt.Errorf("path %q has a non-channel parameter %q", path, param.Name)
			}
			if w.version >= "3.0.0" && param.Schema != nil {
				return fmt.Errorf("channel %q parameter %q has a schema that AsyncAPI 3 cannot represent", path, param.Name)
			}
		}
		if item.Post != nil && item.Post.RequestBody != nil && len(item.Post.RequestBody.Content) > 1 {
			return fmt.Errorf("channel %q has multiple message payloads that cannot be represented without loss", path)
		}
		if item.Get != nil {
			count := 0
			for _, response := range item.Get.Responses {
				count += len(response.Content)
			}
			if count > 1 {
				return fmt.Errorf("channel %q has multiple message payloads that cannot be represented without loss", path)
			}
		}
	}

	// Check if protocol is overridden in context
	targetProtocol := converter.GetProtocol(ctx)

	// Check encoding in context
	encoding := converter.GetEncoding(ctx)
	jsonOutput := w.json
	switch encoding {
	case "json":
		jsonOutput = true
	case "yaml":
		jsonOutput = false
	}

	if w.version >= "3.0.0" {
		return w.writeV3(api, wr, targetProtocol, jsonOutput, plan)
	}
	return w.writeV2(api, wr, targetProtocol, jsonOutput, plan)
}

func promoteOperationParameters(api *model.API) (*model.API, error) {
	// Promote representable operation-level path parameters to the channel.
	// Copy the path map and parameter slices so writing never changes the caller's model.
	copyAPI := *api
	copyAPI.Paths = make(map[string]model.PathItem, len(api.Paths))
	for path := range api.Paths {
		item := api.Paths[path]
		item.Parameters = append([]model.Parameter(nil), item.Parameters...)
		for _, op := range []*model.Operation{item.Get, item.Post} {
			if op == nil {
				continue
			}
			for _, param := range op.Parameters {
				if param.In != model.ParameterInPath {
					return nil, fmt.Errorf("path %q has a non-channel operation parameter %q", path, param.Name)
				}
				if !param.Required || param.Deprecated || param.AllowEmptyValue || param.Style != "" || param.ExplodeSet || param.Example != nil || len(param.Examples) != 0 || len(param.Content) != 0 {
					return nil, fmt.Errorf("path %q operation parameter %q has attributes that AsyncAPI cannot represent", path, param.Name)
				}
				found := false
				for _, existing := range item.Parameters {
					if existing.Name == param.Name && existing.In == param.In {
						if !reflect.DeepEqual(existing, param) {
							return nil, fmt.Errorf("path %q has conflicting channel parameter %q", path, param.Name)
						}
						found = true
						break
					}
				}
				if !found {
					item.Parameters = append(item.Parameters, param)
				}
			}
		}
		copyAPI.Paths[path] = item
	}
	return &copyAPI, nil
}

func writeChannelParameters(params []model.Parameter) (map[string]Parameter, error) {
	if len(params) == 0 {
		return nil, nil
	}
	result := make(map[string]Parameter, len(params))
	for _, param := range params {
		p := Parameter{Description: param.Description}
		if param.Schema != nil {
			schema, err := asyncSchemaObject(param.Schema)
			if err != nil {
				return nil, fmt.Errorf("parameter %q schema: %w", param.Name, err)
			}
			p.Schema = schema
		}
		result[param.Name] = p
	}
	return result, nil
}

// Format returns the format type for the writer.
func (w *Writer) Format() format.Format {
	return format.FormatAsyncAPI
}

// Version returns the AsyncAPI version being written.
func (w *Writer) Version() string {
	return w.version
}
