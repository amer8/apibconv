package model

import "testing"

func TestWithoutExtensionsPreservesCyclesAndOriginal(t *testing.T) {
	api := NewAPI()
	api.Extensions["x-root"] = true
	schema := &Schema{Type: TypeObject, Extensions: map[string]interface{}{"x-schema": true}}
	schema.Properties = map[string]*Schema{"child": schema}
	api.Components.Schemas["Node"] = schema
	op := &Operation{Extensions: map[string]interface{}{"x-op": true}}
	op.Callbacks = map[string]Callback{"loop": {"event": {Get: op}}}
	opaqueExample := &Schema{Extensions: map[string]interface{}{"x-data": "keep"}}
	api.Components.Examples["sample"] = Example{Value: opaqueExample, Extensions: map[string]interface{}{"x-example": true}}
	op.RequestBody = &RequestBody{Content: map[string]MediaType{"application/json": {
		Schema: schema, Example: map[string]interface{}{"x-data": "keep"}, Extensions: map[string]interface{}{"x-media": true},
	}}}
	api.Paths["/items"] = PathItem{Post: op}

	cloned := api.WithoutExtensions()
	if len(cloned.Extensions) != 0 || len(cloned.Components.Schemas["Node"].Extensions) != 0 || len(cloned.Paths["/items"].Post.Extensions) != 0 {
		t.Fatal("extensions remain in the copy")
	}
	clonedSchema := cloned.Components.Schemas["Node"]
	clonedOp := cloned.Paths["/items"].Post
	if clonedSchema.Properties["child"] != clonedSchema || clonedOp.Callbacks["loop"]["event"].Get != clonedOp {
		t.Fatal("model cycles were not preserved")
	}
	media := clonedOp.RequestBody.Content["application/json"]
	if media.Schema != clonedSchema || len(media.Extensions) != 0 || media.Example.(map[string]interface{})["x-data"] != "keep" {
		t.Fatal("schema sharing or example data was lost")
	}
	if cloned.Components.Examples["sample"].Value != opaqueExample || opaqueExample.Extensions["x-data"] != "keep" || len(cloned.Components.Examples["sample"].Extensions) != 0 {
		t.Fatal("example value was traversed or example extensions were retained")
	}
	if len(api.Extensions) != 1 || len(schema.Extensions) != 1 || len(op.Extensions) != 1 {
		t.Fatal("original model was modified")
	}
}
