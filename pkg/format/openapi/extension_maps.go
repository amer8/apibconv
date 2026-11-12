package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// splitMapExtensions separates x- fields from maps whose remaining keys name
// paths or responses. The original node is left unchanged.
func splitMapExtensions(node *yaml.Node, field string) (*yaml.Node, map[string]interface{}, error) {
	if node.Kind != yaml.MappingNode {
		return node, nil, nil
	}
	root := *node
	root.Content = append([]*yaml.Node(nil), node.Content...)
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != field || root.Content[i+1].Kind != yaml.MappingNode {
			continue
		}
		original := root.Content[i+1]
		child := *original
		child.Content = nil
		var extensions map[string]interface{}
		for j := 0; j+1 < len(original.Content); j += 2 {
			key, value := original.Content[j], original.Content[j+1]
			if !strings.HasPrefix(key.Value, "x-") {
				child.Content = append(child.Content, key, value)
				continue
			}
			if extensions == nil {
				extensions = make(map[string]interface{})
			}
			if _, exists := extensions[key.Value]; exists {
				return nil, nil, fmt.Errorf("duplicate extension %q", key.Value)
			}
			var decoded interface{}
			if err := value.Decode(&decoded); err != nil {
				return nil, nil, fmt.Errorf("extension %q: %w", key.Value, err)
			}
			extensions[key.Value] = decoded
		}
		root.Content[i+1] = &child
		return &root, extensions, nil
	}
	return &root, nil, nil
}

func addMapExtensionsJSON(data []byte, field string, extensions map[string]interface{}) ([]byte, error) {
	if len(extensions) == 0 {
		return data, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if raw := root[field]; len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
	}
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	for key, value := range extensions {
		if !strings.HasPrefix(key, "x-") {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("extension %q: %w", key, err)
		}
		fields[key] = encoded
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	root[field] = encoded
	return json.Marshal(root)
}

func addMapExtensionsYAML(value interface{}, field string, extensions map[string]interface{}) (interface{}, error) {
	if len(extensions) == 0 {
		return value, nil
	}
	node := value.(*yaml.Node)
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != field {
			continue
		}
		child := node.Content[i+1]
		if child.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s must be an object", field)
		}
		keys := make([]string, 0, len(extensions))
		for key := range extensions {
			if strings.HasPrefix(key, "x-") {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			extension := extensions[key]
			var encoded yaml.Node
			if err := encoded.Encode(extension); err != nil {
				return nil, fmt.Errorf("extension %q: %w", key, err)
			}
			child.Content = append(child.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &encoded)
		}
		return node, nil
	}
	return nil, fmt.Errorf("%s is missing", field)
}

// UnmarshalYAML preserves extensions on the Paths Object.
func (doc *OpenAPI) UnmarshalYAML(node *yaml.Node) error {
	clean, extensions, err := splitMapExtensions(node, "paths")
	if err != nil {
		return err
	}
	type plain OpenAPI
	var decoded plain
	if err := clean.Decode(&decoded); err != nil {
		return err
	}
	*doc = OpenAPI(decoded)
	doc.PathsExtensions = extensions
	return nil
}

// UnmarshalYAML preserves extensions on Swagger's Paths Object.
func (doc *Swagger20) UnmarshalYAML(node *yaml.Node) error {
	clean, extensions, err := splitMapExtensions(node, "paths")
	if err != nil {
		return err
	}
	type plain Swagger20
	var decoded plain
	if err := clean.Decode(&decoded); err != nil {
		return err
	}
	*doc = Swagger20(decoded)
	doc.PathsExtensions = extensions
	return nil
}

// UnmarshalYAML preserves extensions on the Responses Object.
func (op *Operation) UnmarshalYAML(node *yaml.Node) error {
	clean, extensions, err := splitMapExtensions(node, "responses")
	if err != nil {
		return err
	}
	type plain Operation
	var decoded plain
	if err := clean.Decode(&decoded); err != nil {
		return err
	}
	*op = Operation(decoded)
	op.ResponsesExtensions = extensions
	return nil
}
