package apiblueprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/amer8/apibconv/pkg/model"
)

// parseSchemaAsset retains the definitions embedded by schemaAsset across a
// Blueprint roundtrip and rejects fields the model would silently discard.
func parseSchemaAsset(data []byte, components map[string]*model.Schema, generated map[string]bool) (*model.Schema, error) {
	var err error
	data, err = localizeAssetReferences(data, components, generated)
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil || root == nil {
		return nil, fmt.Errorf("schema asset must be an object")
	}
	if raw, ok := root["components"]; ok {
		var bundle struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&bundle); err != nil {
			return nil, fmt.Errorf("invalid schema components: %w", err)
		}
		for name, rawSchema := range bundle.Schemas {
			schema, err := decodeAssetSchema(rawSchema)
			if err != nil {
				return nil, fmt.Errorf("component %q: %w", name, err)
			}
			if old, exists := components[name]; exists && !reflect.DeepEqual(old, schema) {
				return nil, fmt.Errorf("conflicting embedded component %q", name)
			}
			components[name] = schema
		}
		delete(root, "components")
	}
	clean, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	return decodeAssetSchema(clean)
}

func decodeAssetSchema(data []byte) (*model.Schema, error) {
	var schema model.Schema
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}
	var visit func(*model.Schema) error
	visit = func(s *model.Schema) error {
		if s == nil {
			return nil
		}
		if raw, ok := s.AdditionalProperties.(map[string]interface{}); ok {
			data, err := json.Marshal(raw)
			if err != nil {
				return err
			}
			child, err := decodeAssetSchema(data)
			if err != nil {
				return err
			}
			s.AdditionalProperties = child
		}
		children := []*model.Schema{s.Items, s.Not}
		for _, child := range s.Properties {
			children = append(children, child)
		}
		children = append(children, s.AllOf...)
		children = append(children, s.AnyOf...)
		children = append(children, s.OneOf...)
		for _, child := range children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(&schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// localizeAssetReferences gives each asset-local JSON Pointer its own reusable
// schema, so the reference retains its original resource scope in OpenAPI.
func localizeAssetReferences(data []byte, components map[string]*model.Schema, generated map[string]bool) ([]byte, error) {
	var root, original map[string]interface{}
	if json.Unmarshal(data, &root) != nil || root == nil || json.Unmarshal(data, &original) != nil {
		return nil, fmt.Errorf("schema asset must be an object")
	}
	visited := map[string]bool{}
	assetHash := sha256.Sum256(data)
	var walk func(map[string]interface{}) error
	localize := func(ref string) (string, error) {
		sum := sha256.Sum256(append(append([]byte(nil), assetHash[:]...), []byte("\x00"+ref)...))
		name := fmt.Sprintf("BlueprintAsset_%x", sum)
		if visited[name] {
			return name, nil
		}
		visited[name] = true
		var target interface{} = original
		if ref != "#" {
			if !strings.HasPrefix(ref, "#/") {
				return "", fmt.Errorf("external schema reference %q cannot be represented", ref)
			}
			fragment, err := url.PathUnescape(ref[2:])
			if err != nil {
				return "", fmt.Errorf("invalid schema reference %q", ref)
			}
			for _, token := range strings.Split(fragment, "/") {
				var err error
				token, err = unescapeAssetPointerToken(token)
				if err != nil {
					return "", fmt.Errorf("invalid schema reference %q: %w", ref, err)
				}
				switch v := target.(type) {
				case map[string]interface{}:
					var ok bool
					target, ok = v[token]
					if !ok {
						return "", fmt.Errorf("unresolved schema reference %q", ref)
					}
				case []interface{}:
					i, err := strconv.Atoi(token)
					if err != nil || i < 0 || i >= len(v) {
						return "", fmt.Errorf("unresolved schema reference %q", ref)
					}
					target = v[i]
				default:
					return "", fmt.Errorf("unresolved schema reference %q", ref)
				}
			}
		}
		raw, err := json.Marshal(target)
		if err != nil {
			return "", err
		}
		var child map[string]interface{}
		if json.Unmarshal(raw, &child) != nil || child == nil {
			return "", fmt.Errorf("schema reference %q does not address an object schema", ref)
		}
		delete(child, "components")
		if err := walk(child); err != nil {
			return "", err
		}
		raw, err = json.Marshal(child)
		if err != nil {
			return "", err
		}
		schema, err := decodeAssetSchema(raw)
		if err != nil {
			return "", fmt.Errorf("schema reference %q: %w", ref, err)
		}
		if old, ok := components[name]; ok && !reflect.DeepEqual(old, schema) {
			return "", fmt.Errorf("generated schema %q conflicts with a component", name)
		}
		components[name] = schema
		generated[name] = true
		return name, nil
	}
	walk = func(schema map[string]interface{}) error {
		if ref, ok := schema["$ref"].(string); ok {
			const prefix = "#/components/schemas/"
			token := strings.TrimPrefix(ref, prefix)
			if !strings.HasPrefix(ref, prefix) || strings.Contains(token, "/") {
				name, err := localize(ref)
				if err != nil {
					return err
				}
				schema["$ref"] = "#/components/schemas/" + name
			}
		}
		for _, key := range []string{"items", "additionalProperties", "not"} {
			if child, ok := schema[key].(map[string]interface{}); ok {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			for _, v := range props {
				if child, ok := v.(map[string]interface{}); ok {
					if err := walk(child); err != nil {
						return err
					}
				}
			}
		}
		for _, key := range []string{"allOf", "anyOf", "oneOf"} {
			if list, ok := schema[key].([]interface{}); ok {
				for _, v := range list {
					if child, ok := v.(map[string]interface{}); ok {
						if err := walk(child); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	if bundle, ok := root["components"].(map[string]interface{}); ok {
		if schemas, ok := bundle["schemas"].(map[string]interface{}); ok {
			for _, v := range schemas {
				if child, ok := v.(map[string]interface{}); ok {
					if err := walk(child); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return json.Marshal(root)
}

// schemaAsset makes the JSON Schema self-contained while retaining the
// component pointers used by the model, including recursive references.
func schemaAsset(schema *model.Schema, api *model.API) ([]byte, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(api.Components.Schemas) > 0 {
		root["components"] = map[string]interface{}{"schemas": api.Components.Schemas}
	}
	data, err = json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var walk func(interface{}) error
	walk = func(value interface{}) error {
		switch v := value.(type) {
		case map[string]interface{}:
			if ref, ok := v["$ref"].(string); ok {
				if ref != "#" && !strings.HasPrefix(ref, "#/") {
					return fmt.Errorf("external schema reference %q cannot be embedded", ref)
				}
				current := interface{}(root)
				if ref != "#" {
					fragment, err := url.PathUnescape(ref[2:])
					if err != nil {
						return fmt.Errorf("invalid schema reference %q", ref)
					}
					for _, token := range strings.Split(fragment, "/") {
						var err error
						token, err = unescapeAssetPointerToken(token)
						if err != nil {
							return fmt.Errorf("invalid schema reference %q: %w", ref, err)
						}
						switch c := current.(type) {
						case map[string]interface{}:
							var exists bool
							current, exists = c[token]
							if !exists {
								return fmt.Errorf("unresolved schema reference %q", ref)
							}
						case []interface{}:
							i, err := strconv.Atoi(token)
							if err != nil || i < 0 || i >= len(c) {
								return fmt.Errorf("unresolved schema reference %q", ref)
							}
							current = c[i]
						default:
							return fmt.Errorf("unresolved schema reference %q", ref)
						}
					}
				}
			}
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		case []interface{}:
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	return data, nil
}

func unescapeAssetPointerToken(token string) (string, error) {
	for i := 0; i < len(token); i++ {
		if token[i] == '~' {
			if i+1 >= len(token) || (token[i+1] != '0' && token[i+1] != '1') {
				return "", fmt.Errorf("invalid JSON Pointer escape")
			}
			i++
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~"), nil
}
