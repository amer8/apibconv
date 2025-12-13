package converter

import (
	"fmt"

	"github.com/amer8/apibconv/pkg/format"
)

// Registry maintains a collection of supported parsers and writers.
type Registry struct {
	parsers map[format.Format]format.Parser
	writers map[format.Format]format.Writer
}

// NewRegistry creates a new Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[format.Format]format.Parser),
		writers: make(map[format.Format]format.Writer),
	}
}

// RegisterParser adds a parser to the registry.
func (r *Registry) RegisterParser(p format.Parser) error {
	if p == nil {
		return fmt.Errorf("parser cannot be nil")
	}
	r.parsers[p.Format()] = p
	return nil
}

// RegisterWriter adds a writer to the registry.
func (r *Registry) RegisterWriter(w format.Writer) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	r.writers[w.Format()] = w
	return nil
}

// GetParser retrieves a parser for the specified format.
func (r *Registry) GetParser(f format.Format) (format.Parser, bool) {
	p, ok := r.parsers[f]
	return p, ok
}

// GetWriter retrieves a writer for the specified format.
func (r *Registry) GetWriter(f format.Format) (format.Writer, bool) {
	w, ok := r.writers[f]
	return w, ok
}

// ListFormats returns a list of all registered formats.
func (r *Registry) ListFormats() []format.Format {
	formats := make([]format.Format, 0, len(r.parsers))
	for f := range r.parsers {
		formats = append(formats, f)
	}
	return formats
}
