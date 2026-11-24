// Package yamlguard bounds the work done by typed YAML decoders.
package yamlguard

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ErrUnsafe distinguishes a rejected YAML graph from text in another format.
var ErrUnsafe = errors.New("unsafe YAML document")

// Parse builds a syntax tree before the decoder performs mapping-key comparisons
// or invokes custom unmarshallers (which start independent alias trackers).
func Parse(data []byte) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if err := Check(&root); err != nil {
		return nil, err
	}
	return &root, nil
}

// StringKey returns the string named by a checked YAML mapping key.
// Alias targets must already have passed the graph check.
func StringKey(n *yaml.Node) (string, bool) {
	for n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n.Value, n.Kind == yaml.ScalarNode && n.ShortTag() == "!!str"
}

// Check visits alias targets on every use, just as decoding will. Charge the
// entire graph for both expanded nodes and the pinned decoder's pairwise work.
func Check(root *yaml.Node) error {
	remainingNodes := 250_000
	remainingComparisons := int64(16_000_000)
	remainingKeyBytes := int64(256 << 20)
	expandedBytes := int64(16 << 20)
	active := make(map[*yaml.Node]bool)
	var visit func(*yaml.Node, int, bool, int) error
	visit = func(n *yaml.Node, depth int, aliased bool, copies int) error {
		remainingNodes -= copies
		if remainingNodes < 0 || depth > 128 {
			return fmt.Errorf("%w: expansion or nesting limit exceeded", ErrUnsafe)
		}
		if n.Kind == yaml.AliasNode {
			if n.Alias == nil || active[n.Alias] {
				return fmt.Errorf("%w: cyclic or missing alias target", ErrUnsafe)
			}
			active[n.Alias] = true
			err := visit(n.Alias, depth+1, true, copies)
			delete(active, n.Alias)
			return err
		}
		if aliased || copies > 1 {
			expandedBytes -= int64(len(n.Value)) * int64(copies)
			if expandedBytes < 0 {
				return fmt.Errorf("%w: expanded values exceed byte limit", ErrUnsafe)
			}
		}
		if n.Kind == yaml.MappingNode {
			pairs := len(n.Content) / 2
			if pairs > 2048 {
				return fmt.Errorf("%w: mapping exceeds 2048 entries", ErrUnsafe)
			}
			// Generic additionalProperties maps can be decoded at multiple
			// enclosing schema levels. Reserve work for those passes as well.
			remainingComparisons -= int64(pairs*(pairs-1)/2) * int64(depth+1)
			if remainingComparisons < 0 {
				return fmt.Errorf("%w: mapping comparison limit exceeded", ErrUnsafe)
			}
			// Reject on the first duplicate before yaml collects an error for
			// every matching pair. Match its comparison of key kind and value.
			type key struct {
				kind  yaml.Kind
				value string
			}
			seen := make(map[key]bool, pairs)
			lengths := make(map[int]int, pairs)
			for i := 0; i < len(n.Content); i += 2 {
				k := key{n.Content[i].Kind, n.Content[i].Value}
				// Equal-length keys can require a full byte comparison.
				remainingKeyBytes -= int64(lengths[len(k.value)]) * int64(len(k.value)) * int64(depth+1)
				if remainingKeyBytes < 0 {
					return fmt.Errorf("%w: mapping key comparison limit exceeded", ErrUnsafe)
				}
				lengths[len(k.value)]++
				if seen[k] {
					if strings.HasPrefix(k.value, "x-") {
						return fmt.Errorf("%w: duplicate extension %q", ErrUnsafe, k.value)
					}
					return fmt.Errorf("%w: duplicate mapping key on line %d", ErrUnsafe, n.Content[i].Line)
				}
				seen[k] = true
			}
		}
		for i, child := range n.Content {
			childCopies := copies
			// The public schema retains the raw value as well as its typed child.
			// Enclosing additionalProperties maps each decode this subtree again.
			if n.Kind == yaml.MappingNode && i%2 == 1 {
				if key, ok := StringKey(n.Content[i-1]); ok && key == "additionalProperties" {
					childCopies++
				}
			}
			if err := visit(child, depth+1, aliased, childCopies); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(root, 0, false, 1)
}
