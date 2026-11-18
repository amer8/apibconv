package model

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"
)

type cyclicSnapshotExample map[string]interface{}

func (e cyclicSnapshotExample) MarshalJSON() ([]byte, error) {
	return json.Marshal(e["value"])
}

func TestReferenceSnapshotAcceptsCyclicOpaqueExample(t *testing.T) {
	const childEnv = "APIBCONV_SNAPSHOT_CYCLE_CHILD"
	if os.Getenv(childEnv) == "1" {
		debug.SetMaxStack(1 << 20)
		example := cyclicSnapshotExample{"value": "before"}
		example["self"] = example
		parameter := Parameter{Ref: "#/components/parameters/P", Example: example}
		parameter.CaptureReference()
		if !parameter.ReferenceUnchanged() {
			t.Fatal("unchanged custom example differs from its snapshot")
		}
		example["value"] = "after"
		if parameter.ReferenceUnchanged() {
			t.Fatal("custom example change was missed")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestReferenceSnapshotAcceptsCyclicOpaqueExample$")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("snapshot subprocess failed: %v\n%s", err, output)
	}
}

func TestReferenceSnapshotTracksNestedModelExtensions(t *testing.T) {
	parameter := Parameter{Ref: "#/components/parameters/Example", Schema: &Schema{Type: TypeObject, Properties: map[string]*Schema{
		"child": {Type: TypeString, Extensions: map[string]interface{}{"x-property": "old"}, ExternalDocs: &ExternalDocs{URL: "https://example.com", Extensions: map[string]interface{}{"x-docs": "old"}}, Discriminator: &Discriminator{PropertyName: "type", Extensions: map[string]interface{}{"x-disc": "old"}}},
	}}}
	child := parameter.Schema.Properties["child"]
	changes := []struct {
		name   string
		fields map[string]interface{}
		key    string
	}{
		{"property", child.Extensions, "x-property"},
		{"external docs", child.ExternalDocs.Extensions, "x-docs"},
		{"discriminator", child.Discriminator.Extensions, "x-disc"},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			parameter.CaptureReference()
			if !parameter.ReferenceUnchanged() {
				t.Fatal("new snapshot differs")
			}
			change.fields[change.key] = "new"
			if parameter.ReferenceUnchanged() {
				t.Fatalf("%s extension change was missed", change.name)
			}
			change.fields[change.key] = "old"
			if !parameter.ReferenceUnchanged() {
				t.Fatal("restored value differs")
			}
		})
	}
}

func TestReferenceSnapshotTracksAdditionalPropertiesExtensions(t *testing.T) {
	child := &Schema{Type: TypeString, Extensions: map[string]interface{}{"x-note": "before"}}
	p := Parameter{Ref: "#/components/parameters/P", Schema: &Schema{Type: TypeObject, AdditionalProperties: child}}
	p.CaptureReference()
	child.Extensions["x-note"] = "after"
	if p.ReferenceUnchanged() {
		t.Fatal("additionalProperties extension change was missed")
	}
}
