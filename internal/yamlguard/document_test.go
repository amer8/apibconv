package yamlguard

import (
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestAliasGraphRejectsBackEdge(t *testing.T) {
	// Test graph validation directly, before any typed decoder can run.
	root := &yaml.Node{Kind: yaml.SequenceNode}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: root}
	root.Content = []*yaml.Node{alias}
	if err := Check(root); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("cyclic graph: %v", err)
	}

	leaf := &yaml.Node{Kind: yaml.ScalarNode, Value: "okay"}
	alias.Alias = leaf
	root.Content = append(root.Content, alias)
	if err := Check(root); err != nil {
		t.Fatalf("reused leaf: %v", err)
	}
}

func TestFirstDuplicateFailsBeforeTypedDecoding(t *testing.T) {
	_, err := Parse([]byte("status: one\nstatus: two\n"))
	if !errors.Is(err, ErrUnsafe) || !strings.Contains(err.Error(), "duplicate mapping key") {
		t.Fatalf("duplicate: %v", err)
	}
}
