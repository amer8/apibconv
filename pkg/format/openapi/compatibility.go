package openapi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
)

// validateSchemaCompatibility rejects schema constructs the target dialect
// cannot express, before the writer emits any bytes.
//
//nolint:gocyclo // The traversal checks the complete model before output.
func validateSchemaCompatibility(api *model.API, version string) error {
	swagger := strings.HasPrefix(version, "2")
	oas30 := strings.HasPrefix(version, "3.0")
	seen := map[*model.Schema]bool{}
	var schema func(*model.Schema) error
	schema = func(s *model.Schema) error {
		if s == nil || seen[s] {
			return nil
		}
		seen[s] = true
		if (swagger || oas30) && s.Ref != "" {
			withoutRef := *s
			withoutRef.Ref = ""
			if !reflect.DeepEqual(withoutRef, model.Schema{}) {
				return fmt.Errorf("schema reference siblings cannot be represented in OpenAPI %s", version)
			}
		}
		if swagger && (s.Nullable || len(s.AnyOf) != 0 || len(s.OneOf) != 0 || s.Not != nil || s.WriteOnly || s.Deprecated) {
			return fmt.Errorf("schema contains constraints not representable in OpenAPI 2.0")
		}
		if (swagger || oas30) && s.Type == "null" {
			return fmt.Errorf("null schema type is not representable in OpenAPI %s", version)
		}
		if swagger || (oas30 && !s.Nullable) {
			for _, v := range s.Enum {
				if v == nil {
					return fmt.Errorf("null enum value is not representable in OpenAPI %s", version)
				}
			}
		}
		children := []*model.Schema{s.Items, s.Not}
		for _, p := range s.Properties {
			children = append(children, p)
		}
		children = append(children, s.AllOf...)
		children = append(children, s.AnyOf...)
		children = append(children, s.OneOf...)
		if nested, ok := s.AdditionalProperties.(*model.Schema); ok {
			children = append(children, nested)
		}
		for _, child := range children {
			if err := schema(child); err != nil {
				return err
			}
		}
		return nil
	}
	checkContent := func(content map[string]model.MediaType) error {
		var first *model.Schema
		firstSet := false
		for _, media := range content {
			if err := schema(media.Schema); err != nil {
				return err
			}
			if swagger && firstSet && !reflect.DeepEqual(first, media.Schema) {
				return fmt.Errorf("different content type schemas cannot be represented in OpenAPI 2.0")
			}
			first, firstSet = media.Schema, true
		}
		return nil
	}
	var path func(model.PathItem) error
	seenOps := map[*model.Operation]bool{}
	activeOps := map[*model.Operation]bool{}
	path = func(item model.PathItem) error {
		for _, p := range item.Parameters {
			if swagger {
				if _, err := swaggerParameterSchema(p.Schema, api); err != nil {
					return err
				}
			}
			if err := schema(p.Schema); err != nil {
				return err
			}
			if err := checkContent(p.Content); err != nil {
				return err
			}
		}
		for _, op := range []*model.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op == nil {
				continue
			}
			if activeOps[op] {
				return fmt.Errorf("cyclic callback operation cannot be written")
			}
			if seenOps[op] {
				continue
			}
			seenOps[op] = true
			activeOps[op] = true
			for _, p := range op.Parameters {
				if swagger {
					if _, err := swaggerParameterSchema(p.Schema, api); err != nil {
						return err
					}
				}
				if err := schema(p.Schema); err != nil {
					return err
				}
				if err := checkContent(p.Content); err != nil {
					return err
				}
			}
			if op.RequestBody != nil {
				if err := checkContent(op.RequestBody.Content); err != nil {
					return err
				}
			}
			for _, resp := range op.Responses {
				if err := checkContent(resp.Content); err != nil {
					return err
				}
				for _, h := range resp.Headers {
					if swagger {
						if _, err := swaggerHeaderSchema(h.Schema, api); err != nil {
							return err
						}
					}
					if err := schema(h.Schema); err != nil {
						return err
					}
				}
			}
			for _, callback := range op.Callbacks {
				for expression := range callback {
					item := callback[expression]
					if err := path(item); err != nil {
						return err
					}
				}
			}
			delete(activeOps, op)
		}
		return nil
	}
	for _, s := range api.Components.Schemas {
		if err := schema(s); err != nil {
			return err
		}
	}
	for _, paths := range []map[string]model.PathItem{api.Paths, api.Webhooks} {
		for name := range paths {
			item := paths[name]
			if err := path(item); err != nil {
				return err
			}
		}
	}
	for _, callback := range api.Components.Callbacks {
		for name := range callback {
			item := callback[name]
			if err := path(item); err != nil {
				return err
			}
		}
	}
	return nil
}
