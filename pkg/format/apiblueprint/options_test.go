package apiblueprint

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestMarkdownDescriptionsOption(t *testing.T) {
	input := `FORMAT: 1A
# Example
API **description**.
Note: an inline **note**.

Second paragraph.
## Items [/items]
Resource *description*.
### List [GET]
Action description.
+ Response 200
`
	for _, enabled := range []bool{false, true} {
		api, err := NewParser(WithMarkdownParsing(enabled)).Parse(context.Background(), strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		item := api.Paths["/items"]
		if enabled {
			if api.Info.Description != "API **description**.\nNote: an inline **note**.\n\nSecond paragraph." || item.Description != "Resource *description*." || item.Get.Description != "Action description." {
				t.Fatalf("descriptions: %q, %q, %q", api.Info.Description, item.Description, item.Get.Description)
			}
		} else if api.Info.Description != "" || item.Description != "" || item.Get.Description != "" {
			t.Fatal("descriptions were captured with Markdown parsing disabled")
		}
	}
}

func TestBlueprintVersions(t *testing.T) {
	parser := NewParser()
	for version, want := range map[string]bool{"1A": true, "1B": false, "": false, "3.0": false} {
		if got := parser.SupportsVersion(version); got != want {
			t.Errorf("SupportsVersion(%q) = %v", version, got)
		}
	}
	api := model.NewAPI()
	for _, version := range []string{"", "1A"} {
		var out bytes.Buffer
		writer := NewWriter(WithFormat(version))
		if err := writer.Write(context.Background(), api, &out); err != nil || !strings.HasPrefix(out.String(), "FORMAT: 1A\n") || writer.Version() != "1A" {
			t.Fatalf("version %q: output=%q error=%v", version, out.String(), err)
		}
	}
	var out bytes.Buffer
	writer := NewWriter(WithFormat("1B"))
	if err := writer.Write(context.Background(), api, &out); err == nil || out.Len() != 0 || writer.Version() != "1B" {
		t.Fatalf("unsupported version: output=%q error=%v", out.String(), err)
	}
}
