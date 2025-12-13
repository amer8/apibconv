// Package asyncapi implements the AsyncAPI format parser and writer.
package asyncapi

import (
	"context"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/amer8/apibconv/internal/fs"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// Parser implements the format.Parser interface for AsyncAPI.
type Parser struct {
	version string
}

// ParserOption configures the Parser.
type ParserOption func(*Parser)

// WithAsyncVersion sets the AsyncAPI version for parsing (unused in current detection logic but kept for interface consistency).
func WithAsyncVersion(version string) ParserOption {
	return func(p *Parser) {
		p.version = version
	}
}

// NewParser creates a new AsyncAPI parser with optional configurations.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Parse parses an AsyncAPI document from the given reader into a unified model.API.
func (p *Parser) Parse(ctx context.Context, r io.Reader) (*model.API, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read everything to detect version
	data, err := fs.ReadAll(ctx, r)
	if err != nil {
		return nil, err
	}

	// Simple check for version 3
	var base struct {
		AsyncAPI string `yaml:"asyncapi"`
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to parse asyncapi version: %w", err)
	}

	if len(base.AsyncAPI) >= 1 && base.AsyncAPI[0] == '3' {
		return p.parseV3(data)
	}

	return p.parseV2(data)
}

// Format returns the format type for the parser.
func (p *Parser) Format() format.Format {
	return format.FormatAsyncAPI
}

// SupportsVersion returns true for any version supported by the parser logic.
func (p *Parser) SupportsVersion(version string) bool {
	return true
}
