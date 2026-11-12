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
	var original, clean map[string]interface{}
	if err := json.Unmarshal(data, &original); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &clean); err != nil {
		return nil, err
	}
	stripAssetExtensions(clean)
	data, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
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
	visitAssetSchema(original, &schema, func(object map[string]interface{}, schema *model.Schema) {
		for key, value := range object {
			if strings.HasPrefix(key, "x-") {
				if schema.Extensions == nil {
					schema.Extensions = make(map[string]interface{})
				}
				schema.Extensions[key] = value
			}
		}
		if discriminator, ok := object["discriminator"].(map[string]interface{}); ok && schema.Discriminator != nil {
			for key, value := range discriminator {
				if strings.HasPrefix(key, "x-") {
					if schema.Discriminator.Extensions == nil {
						schema.Discriminator.Extensions = make(map[string]interface{})
					}
					schema.Discriminator.Extensions[key] = value
				}
			}
		}
	})
	return &schema, nil
}

// stripAssetExtensions removes extensions only from schema objects, leaving
// examples, defaults and extension values untouched.
func stripAssetExtensions(object map[string]interface{}) {
	for key := range object {
		if strings.HasPrefix(key, "x-") {
			delete(object, key)
		}
	}
	if discriminator, ok := object["discriminator"].(map[string]interface{}); ok {
		for key := range discriminator {
			if strings.HasPrefix(key, "x-") {
				delete(discriminator, key)
			}
		}
	}
	for _, key := range []string{"items", "not", "additionalProperties"} {
		if child, ok := object[key].(map[string]interface{}); ok {
			stripAssetExtensions(child)
		}
	}
	if properties, ok := object["properties"].(map[string]interface{}); ok {
		for _, value := range properties {
			if child, ok := value.(map[string]interface{}); ok {
				stripAssetExtensions(child)
			}
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if children, ok := object[key].([]interface{}); ok {
			for _, value := range children {
				if child, ok := value.(map[string]interface{}); ok {
					stripAssetExtensions(child)
				}
			}
		}
	}
}

func visitAssetSchema(object map[string]interface{}, schema *model.Schema, visit func(map[string]interface{}, *model.Schema)) {
	if object == nil || schema == nil {
		return
	}
	visit(object, schema)
	if properties, ok := object["properties"].(map[string]interface{}); ok {
		for name, value := range properties {
			child, _ := value.(map[string]interface{})
			visitAssetSchema(child, schema.Properties[name], visit)
		}
	}
	for key, childSchema := range map[string]*model.Schema{"items": schema.Items, "not": schema.Not} {
		child, _ := object[key].(map[string]interface{})
		visitAssetSchema(child, childSchema, visit)
	}
	if childSchema, ok := schema.AdditionalProperties.(*model.Schema); ok {
		child, _ := object["additionalProperties"].(map[string]interface{})
		visitAssetSchema(child, childSchema, visit)
	}
	for key, children := range map[string][]*model.Schema{"allOf": schema.AllOf, "anyOf": schema.AnyOf, "oneOf": schema.OneOf} {
		values, _ := object[key].([]interface{})
		for i, childSchema := range children {
			if i < len(values) {
				child, _ := values[i].(map[string]interface{})
				visitAssetSchema(child, childSchema, visit)
			}
		}
	}
}

func encodeAssetSchema(schema *model.Schema) (interface{}, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	object, _ := value.(map[string]interface{})
	visitAssetSchema(object, schema, func(object map[string]interface{}, schema *model.Schema) {
		for key, value := range schema.Extensions {
			if strings.HasPrefix(key, "x-") {
				object[key] = value
			}
		}
		if discriminator, ok := object["discriminator"].(map[string]interface{}); ok && schema.Discriminator != nil {
			for key, value := range schema.Discriminator.Extensions {
				if strings.HasPrefix(key, "x-") {
					discriminator[key] = value
				}
			}
		}
	})
	return value, nil
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
		if ref != "#" && !strings.HasPrefix(ref, "#/") {
			return "", fmt.Errorf("external schema reference %q cannot be represented", ref)
		}
		target, err := assetPointerTarget(original, ref)
		if err != nil {
			return "", err
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
	value, err := encodeAssetSchema(schema)
	if err != nil {
		return nil, err
	}
	root, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema asset must be an object")
	}
	if len(api.Components.Schemas) > 0 {
		schemas := make(map[string]interface{}, len(api.Components.Schemas))
		for name, component := range api.Components.Schemas {
			schemas[name], err = encodeAssetSchema(component)
			if err != nil {
				return nil, fmt.Errorf("component %q: %w", name, err)
			}
		}
		root["components"] = map[string]interface{}{"schemas": schemas}
	}
	data, err := json.Marshal(root)
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
				if _, err := assetPointerTarget(root, ref); err != nil {
					return err
				}
			}
			for key, child := range v {
				if strings.HasPrefix(key, "x-") {
					continue
				}
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

// assetPointerTarget resolves a local JSON Pointer against a decoded schema asset.
func assetPointerTarget(root interface{}, ref string) (interface{}, error) {
	if ref == "#" {
		return root, nil
	}
	fragment, err := url.PathUnescape(ref[2:])
	if err != nil {
		return nil, fmt.Errorf("invalid schema reference %q", ref)
	}
	current := root
	for _, token := range strings.Split(fragment, "/") {
		token, err = unescapeAssetPointerToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid schema reference %q: %w", ref, err)
		}
		switch value := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = value[token]
			if !ok {
				return nil, fmt.Errorf("unresolved schema reference %q", ref)
			}
		case []interface{}:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(value) {
				return nil, fmt.Errorf("unresolved schema reference %q", ref)
			}
			current = value[index]
		default:
			return nil, fmt.Errorf("unresolved schema reference %q", ref)
		}
	}
	return current, nil
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
