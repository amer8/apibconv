package asyncapi

import (
	"encoding/json"
	"fmt"

	"github.com/amer8/apibconv/pkg/model"
)

// asyncSchemaObject translates the model's OpenAPI-style nullable and numeric
// exclusivity to the JSON Schema draft-07 dialect used by AsyncAPI schemas.
func asyncSchemaObject(schema *model.Schema) (map[string]interface{}, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err = json.Unmarshal(data, &raw); err != nil || raw == nil {
		return nil, fmt.Errorf("invalid schema")
	}
	if err := walkAsyncSchema(raw, true); err != nil {
		return nil, err
	}
	return raw, nil
}

// walkAsyncSchema traverses schema-valued keywords only; examples and enum
// values may themselves contain ordinary objects with keys such as "type".
func walkAsyncSchema(raw map[string]interface{}, toAsync bool) error {
	if toAsync {
		if nullable, _ := raw["nullable"].(bool); nullable {
			if typ, ok := raw["type"].(string); ok && typ != "null" {
				raw["type"] = []interface{}{typ, "null"}
			}
		}
		delete(raw, "nullable")
		for _, pair := range [][2]string{{"exclusiveMinimum", "minimum"}, {"exclusiveMaximum", "maximum"}} {
			if exclusive, ok := raw[pair[0]].(bool); ok {
				delete(raw, pair[0])
				if exclusive {
					value, exists := raw[pair[1]]
					if !exists {
						return fmt.Errorf("%s requires a bound", pair[0])
					}
					delete(raw, pair[1])
					raw[pair[0]] = value
				}
			}
		}
		// Draft-07 ignores siblings of $ref; preserve their intersection.
		if ref, ok := raw["$ref"]; ok && len(raw) > 1 {
			delete(raw, "$ref")
			all, _ := raw["allOf"].([]interface{})
			raw["allOf"] = append(all, map[string]interface{}{"$ref": ref})
		}
	} else {
		// nullable has no validation meaning in the AsyncAPI schema dialect.
		delete(raw, "nullable")
		for _, pair := range [][2]string{{"exclusiveMinimum", "minimum"}, {"exclusiveMaximum", "maximum"}} {
			value, ok := raw[pair[0]]
			if !ok {
				continue
			}
			bound, valid := value.(float64)
			if !valid {
				return fmt.Errorf("%s must be numeric", pair[0])
			}
			old, exists := raw[pair[1]].(float64)
			stronger := exists && ((pair[1] == "minimum" && old > bound) || (pair[1] == "maximum" && old < bound))
			delete(raw, pair[0])
			if !stronger {
				raw[pair[1]] = bound
				raw[pair[0]] = true
			}
		}
		if types, ok := raw["type"].([]interface{}); ok {
			if len(types) == 0 {
				return fmt.Errorf("empty schema type union")
			}
			var nonnull string
			nulls := 0
			for _, v := range types {
				typ, ok := v.(string)
				if !ok {
					return fmt.Errorf("invalid schema type")
				}
				switch {
				case typ == "null":
					nulls++
				case nonnull == "":
					nonnull = typ
				default:
					nonnull = "*"
				}
			}
			switch {
			case len(types) == 2 && nulls == 1 && nonnull != "" && nonnull != "*":
				raw["type"] = nonnull
				raw["nullable"] = true
			case len(types) == 1:
				raw["type"] = types[0]
			default:
				delete(raw, "type")
				var alternatives []interface{}
				for _, v := range types {
					alternatives = append(alternatives, map[string]interface{}{"type": v})
				}
				all, _ := raw["allOf"].([]interface{})
				raw["allOf"] = append(all, map[string]interface{}{"anyOf": alternatives})
			}
		}
		if _, ok := raw["$ref"]; ok && len(raw) > 1 {
			return fmt.Errorf("schema reference with siblings cannot be represented safely")
		}
	}
	for _, key := range []string{"items", "additionalProperties", "not"} {
		if child, ok := raw[key].(map[string]interface{}); ok {
			if err := walkAsyncSchema(child, toAsync); err != nil {
				return err
			}
		}
	}
	if props, ok := raw["properties"].(map[string]interface{}); ok {
		for _, v := range props {
			if child, ok := v.(map[string]interface{}); ok {
				if err := walkAsyncSchema(child, toAsync); err != nil {
					return err
				}
			}
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if list, ok := raw[key].([]interface{}); ok {
			for _, v := range list {
				if child, ok := v.(map[string]interface{}); ok {
					if err := walkAsyncSchema(child, toAsync); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
