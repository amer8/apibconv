// Package transform provides utilities for modifying and normalizing API models.
package transform

import (
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
)

// Transformer defines an interface for modifying the API model.
type Transformer interface {
	Transform(api *model.API) error
	Name() string
}

// ChainTransformer allows applying a sequence of transformers.
type ChainTransformer struct {
	transformers []Transformer
}

// NewChainTransformer creates a new ChainTransformer with the given transformers.
func NewChainTransformer(transformers ...Transformer) *ChainTransformer {
	return &ChainTransformer{
		transformers: transformers,
	}
}

// Transform applies all transformers in the chain to the API model.
func (c *ChainTransformer) Transform(api *model.API) error {
	for _, t := range c.transformers {
		if err := t.Transform(api); err != nil {
			return err
		}
	}
	return nil
}

// Add appends a transformer to the chain.
func (c *ChainTransformer) Add(t Transformer) {
	c.transformers = append(c.transformers, t)
}

// RemoveUnusedSchemas removes schemas from Components that are not referenced in the API.
// Note: Implementation is currently a placeholder.
func RemoveUnusedSchemas(api *model.API) error {
	// TODO: Implementation would require traversing all references in paths, operations, parameters, etc.
	// This is a complex operation deferred for future optimization.
	return nil
}

// NormalizeOperationIDs ensures all operations have an OperationID, generating one from the path if missing.
func NormalizeOperationIDs(api *model.API) error {
	for path := range api.Paths {
		item := api.Paths[path]
		pathParts := strings.Split(path, "/")
		var cleanPath []string
		for _, p := range pathParts {
			if p != "" && !strings.HasPrefix(p, "{") {
				cleanPath = append(cleanPath, p)
			}
		}
		baseName := strings.Join(cleanPath, "")
		if baseName == "" {
			baseName = "Root"
		}

		normalize := func(method string, op *model.Operation) {
			if op != nil && op.OperationID == "" {
				op.OperationID = fmt.Sprintf("%s%s", strings.ToLower(method), baseName)
			}
		}

		normalize("Get", item.Get)
		normalize("Post", item.Post)
		normalize("Put", item.Put)
		normalize("Delete", item.Delete)
		normalize("Patch", item.Patch)
		normalize("Options", item.Options)
		normalize("Head", item.Head)
		normalize("Trace", item.Trace)
	}
	return nil
}

// SortPaths is a placeholder as paths are stored in a map.
// Writers should handle deterministic sorting.
func SortPaths(api *model.API) error {
	// Paths are stored in a map, so they cannot be sorted in-place.
	// Writers are responsible for sorting paths during output generation for determinism.
	return nil
}
