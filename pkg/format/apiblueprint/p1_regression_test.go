package apiblueprint

import (
	"context"
	"strings"
	"testing"
)

func TestP1ResourceBeforeDataStructures(t *testing.T) {
	input := `FORMAT: 1A
# Example
## Resource [/resource]
### Read [GET]
+ Response 200
# Data Structures
## Thing (object)
+ id (string)
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if item, ok := api.Paths["/resource"]; !ok || item.Get == nil {
		t.Fatalf("lost resource: %#v", api.Paths)
	}
	if api.Components.Schemas["Thing"] == nil {
		t.Fatal("lost schema")
	}
}
