package openapi

import (
	"fmt"
	"reflect"

	"github.com/amer8/apibconv/pkg/model"
)

// swaggerParameterSchema resolves representable reusable schemas and folds the
// single-value enum constraints used for OpenAPI 3.1 const. Swagger non-body
// parameters cannot contain arbitrary schema compositions or object schemas.
func swaggerParameterSchema(input *model.Schema, api *model.API) (*model.Schema, error) {
	seen := make(map[string]bool)
	var normalize func(*model.Schema) (*model.Schema, error)
	normalize = func(s *model.Schema) (*model.Schema, error) {
		if s == nil {
			return nil, nil
		}
		if s.Ref != "" {
			copy := *s
			copy.Ref = ""
			if !reflect.DeepEqual(copy, model.Schema{}) {
				return nil, fmt.Errorf("swagger parameter schema reference has siblings")
			}
			if seen[s.Ref] {
				return nil, fmt.Errorf("cyclic Swagger parameter schema reference %q", s.Ref)
			}
			seen[s.Ref] = true
			name, err := referenceName(s.Ref, "#/components/schemas/")
			if err != nil {
				return nil, err
			}
			target, ok := api.Components.Schemas[name]
			if !ok {
				return nil, fmt.Errorf("parameter schema reference %q not found", s.Ref)
			}
			resolved, err := normalize(target)
			delete(seen, s.Ref)
			return resolved, err
		}
		result := *s
		result.AllOf = nil
		if len(s.AnyOf) > 0 || len(s.OneOf) > 0 || s.Not != nil || s.Nullable {
			return nil, fmt.Errorf("swagger parameter schema composition cannot be represented")
		}
		for _, sub := range s.AllOf {
			if sub == nil {
				return nil, fmt.Errorf("invalid Swagger parameter schema composition")
			}
			bare := *sub
			bare.Enum = nil
			if len(sub.Enum) == 0 || !reflect.DeepEqual(bare, model.Schema{}) {
				return nil, fmt.Errorf("swagger parameter schema composition cannot be represented")
			}
			if result.Enum == nil {
				result.Enum = append([]interface{}(nil), sub.Enum...)
				continue
			}
			var intersection []interface{}
			for _, value := range result.Enum {
				for _, allowed := range sub.Enum {
					if reflect.DeepEqual(value, allowed) {
						intersection = append(intersection, value)
						break
					}
				}
			}
			if len(intersection) == 0 {
				return nil, fmt.Errorf("empty Swagger parameter enum cannot be represented")
			}
			result.Enum = intersection
		}
		switch result.Type {
		case model.TypeString, model.TypeInteger, model.TypeNumber, model.TypeBoolean:
		case model.TypeArray:
			item, err := normalize(result.Items)
			if err != nil {
				return nil, err
			}
			if item == nil {
				return nil, fmt.Errorf("swagger array parameter requires items")
			}
			result.Items = item
		default:
			return nil, fmt.Errorf("swagger parameter schema type %q cannot be represented", result.Type)
		}
		return &result, nil
	}
	return normalize(input)
}

func swaggerHeaderSchema(input *model.Schema, api *model.API) (*model.Schema, error) {
	schema, err := swaggerParameterSchema(input, api)
	if err != nil || schema == nil {
		return schema, err
	}
	bare := *schema
	bare.Type = ""
	bare.Format = ""
	bare.Items = nil
	if !reflect.DeepEqual(bare, model.Schema{}) {
		return nil, fmt.Errorf("swagger response header schema constraints cannot be represented")
	}
	return schema, nil
}
