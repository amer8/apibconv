package asyncapi

import (
	"context"
	"io"

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
		version: "2.0.0",
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
		return w.writeV3(api, wr, targetProtocol, jsonOutput)
	}
	return w.writeV2(api, wr, targetProtocol, jsonOutput)
}

// Format returns the format type for the writer.
func (w *Writer) Format() format.Format {
	return format.FormatAsyncAPI
}

// Version returns the AsyncAPI version being written.
func (w *Writer) Version() string {
	return w.version
}

