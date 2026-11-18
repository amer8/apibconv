package apiblueprint

import (
	"context"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/model"
)

func TestReviewHeadingAndPayloadRegressions(t *testing.T) {
	input := "FORMAT: 1A\n# Example\n## Resource [/things{?filter}]\n### Read [GET]\n+ Response 200 (application/json)\n\n    + Attributes (object)\n        + name: Jane (string, required)\n\n### Remove [DELETE /things/{id}]\n+ Request\n    + Headers\n\n            X-Client: client\n+ Response 201 (application/json)\n\t+ Headers\n\t\tLocation: /things/1\n\t+ Body\n\t\t{\"ok\":true}\n+ Response 201 (application/xml)\n    + Body\n\n            <ok/>\n## GET /ping\n+ Response 204\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	get := api.Paths["/things"].Get
	if get == nil || get.Responses["200"].Content["application/json"].Schema.Properties["name"].Example != "Jane" {
		t.Fatalf("attributes lost: %#v", get)
	}
	del := api.Paths["/things/{id}"].Delete
	if del == nil || del.RequestBody != nil || len(del.Parameters) != 2 || del.Parameters[0].Name != "X-Client" || del.Parameters[1].Name != "id" || !del.Parameters[1].Required {
		t.Fatalf("request headers or body: %#v", del)
	}
	response := del.Responses["201"]
	if response.Headers["Location"].Example != "/things/1" || len(response.Content) != 2 || response.Content["application/xml"].Example != "<ok/>" {
		t.Fatalf("response: %#v", response)
	}
	if example, ok := response.Content["application/json"].Example.(map[string]interface{}); !ok || example["ok"] != true {
		t.Fatalf("JSON example: %#v", response.Content)
	}
	ping := api.Paths["/ping"].Get
	if ping == nil || len(ping.Responses["204"].Content) != 0 || ping.Responses["204"].Description != "No Content" {
		t.Fatalf("ping: %#v", ping)
	}
	if _, found := del.Responses["204"]; found {
		t.Fatal("response attached to previous action")
	}
}

func TestReviewStructuresAndParameters(t *testing.T) {
	input := "FORMAT: 1A\n# Example\n## Item [/items/{id}{?status}]\n+ Parameters\n    + id: 1 (number)\n    + status (enum[number], optional)\n        + Default: 1\n        + Members\n            + 1\n            + 2\n### Get [GET]\n+ Response 200\n    + Attributes (array[User Name])\n# Data Structures\n## Choice\n+ `created-at`: 2020-01-01 (string, required) (required)\n## User Name (Choice)\n+ address (Choice)\n+ active: true (boolean, nullable, fixed)\n+ role (enum[string])\n    + Default: member\n    + Members\n        + member\n        + admin\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/items/{id}"]
	if len(item.Parameters) != 2 || item.Parameters[0].In != model.ParameterInPath {
		t.Fatalf("params: %#v", item.Parameters)
	}
	param := item.Parameters[1]
	if param.Schema.Type != model.TypeNumber || param.Schema.Format != "" || param.Schema.Default != model.JSONNumber("1") || len(param.Schema.Enum) != 2 {
		t.Fatalf("enum: %#v", param.Schema)
	}
	choice := api.Components.Schemas["Choice"]
	if choice == nil || choice.Properties["created-at"].Example != "2020-01-01" || choice.Properties["created-at"].Description != "" || len(choice.Required) != 1 {
		t.Fatalf("choice: %#v", choice)
	}
	user := api.Components.Schemas["User_Name"]
	if user == nil || len(user.AllOf) != 1 || user.AllOf[0].Ref != "#/components/schemas/Choice" || user.Properties["address"].Ref != "#/components/schemas/Choice" || !user.Properties["active"].Nullable || len(user.Properties["active"].Enum) != 1 {
		t.Fatalf("user: %#v", user)
	}
	if role := user.Properties["role"]; role.Default != "member" || len(role.Enum) != 2 {
		t.Fatalf("role: %#v", role)
	}
	if schema := item.Get.Responses["200"].Content["application/json"].Schema; schema.Items.Ref != "#/components/schemas/User_Name" {
		t.Fatalf("array: %#v", schema)
	}
}

func TestLongBlueprintLine(t *testing.T) {
	input := "FORMAT: 1A\n# API\n## Ping [/ping]\n### Read [GET]\n+ Response 200 (text/plain)\n\n        " + strings.Repeat("x", 70000) + "\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if example := api.Paths["/ping"].Get.Responses["200"].Content["text/plain"].Example.(string); len(example) != 70000 {
		t.Fatalf("example length %d", len(example))
	}
}

func TestURITemplatesInferParametersAndRejectUnsupportedModifiers(t *testing.T) {
	input := "FORMAT: 1A\n# API\n## Widgets [/widgets/{id}{?filter,limit}]\n### Read [GET]\n+ Response 200\n### Update [PATCH /widgets/{id}{?verbose}]\n+ Response 204\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/widgets/{id}"]
	if item.Get == nil || !hasParameter(item.Get.Parameters, "id", model.ParameterInPath) || !hasParameter(item.Get.Parameters, "filter", model.ParameterInQuery) || !hasParameter(item.Get.Parameters, "limit", model.ParameterInQuery) {
		t.Fatalf("GET parameters: %#v", item.Get)
	}
	if item.Patch == nil || !hasParameter(item.Patch.Parameters, "id", model.ParameterInPath) || !hasParameter(item.Patch.Parameters, "verbose", model.ParameterInQuery) {
		t.Fatalf("PATCH parameters: %#v", item.Patch)
	}
	for _, uri := range []string{"/widgets{?filter*}", "/widgets/{+id}", "/widgets{/id}", "/widgets/{id:3}", "/widgets/{broken"} {
		doc := "FORMAT: 1A\n# API\n## Widgets [" + uri + "]\n### Read [GET]\n+ Response 200\n"
		if _, err := NewParser().Parse(context.Background(), strings.NewReader(doc)); err == nil {
			t.Fatalf("unsupported URI %q accepted", uri)
		}
	}
}

func TestDataStructureNamesAreValidAndCollisionChecked(t *testing.T) {
	input := "FORMAT: 1A\n# API\n# Data Structures\n## User / Name (object)\n+ id (string)\n"
	api, err := NewParser().Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if api.Components.Schemas["User_Name"] == nil {
		t.Fatalf("schemas: %#v", api.Components.Schemas)
	}
	input += "## User Name (object)\n+ name (string)\n"
	if _, err := NewParser().Parse(context.Background(), strings.NewReader(input)); err == nil {
		t.Fatal("normalized names collided silently")
	}
}

func TestActionURIKeepsResourceScopeAndParameters(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Items [/items/{id}]
+ Parameters
    + id (integer, required) - Identifier
### Lock [POST /items/{id}/lock]
+ Response 204
### Read [GET]
+ Response 200
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	resource, lock := api.Paths["/items/{id}"], api.Paths["/items/{id}/lock"]
	if resource.Get == nil || lock.Post == nil || lock.Get != nil {
		t.Fatalf("misplaced actions: %#v %#v", resource, lock)
	}
	if len(lock.Post.Parameters) != 1 || lock.Post.Parameters[0].Name != "id" || lock.Post.Parameters[0].Schema.Type != model.TypeInteger || lock.Post.Parameters[0].Description != "Identifier" {
		t.Fatalf("inherited parameters lost: %#v", lock.Post.Parameters)
	}
}

func TestQueriesStayWithinDeclaringResource(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Search [/items{?name}]
### List [GET]
+ Response 200
## Create [/items{?notify}]
### Create [POST]
+ Response 201
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/items"]
	if len(item.Get.Parameters) != 1 || item.Get.Parameters[0].Name != "name" || len(item.Post.Parameters) != 1 || item.Post.Parameters[0].Name != "notify" {
		t.Fatalf("queries leaked: %#v %#v", item.Get.Parameters, item.Post.Parameters)
	}
}

func TestRepeatedRequestHeadersAreMerged(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Items [/items]
### Create [POST]
+ Request (application/json)
    + Headers

            X-Client: first
+ Request (application/xml)
    + Headers

            x-client: second
+ Response 204
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	params := api.Paths["/items"].Post.Parameters
	if len(params) != 1 || len(params[0].Examples) != 2 || params[0].Example != nil {
		t.Fatalf("headers: %#v", params)
	}
}

func TestReservedRequestHeadersFailExplicitly(t *testing.T) {
	for _, name := range []string{"Authorization", "accept", "Content-Type"} {
		doc := "FORMAT: 1A\n# API\n## X [/x]\n### X [POST]\n+ Request\n    + Headers\n\n            " + name + ": sample\n+ Response 204\n"
		_, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("header %s: %v", name, err)
		}
	}
}

func TestExplicitResourceParametersRemainScopedToActions(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Search [/items{?name}]
+ Parameters
    + name: bob (string, required) - search name
        + Default: bob
### Search [GET]
+ Response 200
## Create [/items{?notify}]
+ Parameters
    + notify: true (boolean, required) - notify user
### Create [POST]
+ Response 201
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/items"]
	if len(item.Parameters) != 0 || item.Get == nil || item.Post == nil || len(item.Get.Parameters) != 1 || len(item.Post.Parameters) != 1 {
		t.Fatalf("parameters leaked: %#v", item)
	}
	name, notify := item.Get.Parameters[0], item.Post.Parameters[0]
	if name.Name != "name" || !name.Required || name.Schema.Default != "bob" || name.Description != "search name" {
		t.Fatalf("search: %#v", name)
	}
	if notify.Name != "notify" || !notify.Required || notify.Schema.Type != model.TypeBoolean || notify.Description != "notify user" {
		t.Fatalf("create: %#v", notify)
	}
}

func TestSharedPathResourceParametersAndActionOverrides(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Read [/items/{id}{?name}]
+ Parameters
    + id (integer, required) - first id
    + name (string, required)
### Read [GET]
+ Response 200
### Remove [DELETE /items/{id}/history]
+ Response 204
## Create [/items/{id}{?notify}]
+ Parameters
    + id (string, required) - second id
    + notify (boolean, required)
### Create [POST]
+ Parameters
    + id (string, required) - overridden id
        + Default: xyz
    + notify (boolean, optional) - overridden
+ Response 201
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	item, history := api.Paths["/items/{id}"], api.Paths["/items/{id}/history"]
	if len(item.Parameters) != 0 || item.Get == nil || item.Post == nil || history.Delete == nil {
		t.Fatalf("paths: %#v %#v", item, history)
	}
	if len(item.Get.Parameters) != 2 || item.Get.Parameters[0].Schema.Type != model.TypeInteger || item.Get.Parameters[1].Name != "name" {
		t.Fatalf("GET: %#v", item.Get.Parameters)
	}
	if len(item.Post.Parameters) != 2 || item.Post.Parameters[0].Schema.Type != model.TypeString || item.Post.Parameters[0].Schema.Default != "xyz" || item.Post.Parameters[1].Schema.Default != nil || item.Post.Parameters[1].Name != "notify" || item.Post.Parameters[1].Required || item.Post.Parameters[1].Description != "overridden" {
		t.Fatalf("POST: %#v", item.Post.Parameters)
	}
	if len(history.Delete.Parameters) != 2 || history.Delete.Parameters[0].Schema.Type != model.TypeInteger || history.Delete.Parameters[1].Name != "name" {
		t.Fatalf("DELETE: %#v", history.Delete.Parameters)
	}
}

func TestActionTargetingExistingPathDoesNotInheritOtherResourceParameters(t *testing.T) {
	for _, tc := range []struct{ name, intervening, heading string }{
		{"action", "## Other [/different]\n", "### Create [POST /items{?notify}]"},
		{"standalone after another resource", "## Other [/different]\n", "## POST /items{?notify}"},
		{"standalone after same resource", "", "## POST /items{?notify}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := `FORMAT: 1A
# API
## Search [/items{?name}]
+ Parameters
    + name (string, required)
### Search [GET]
+ Response 200
` + tc.intervening + tc.heading + `
+ Response 201
`
			api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
			if err != nil {
				t.Fatal(err)
			}
			item := api.Paths["/items"]
			if len(item.Parameters) != 0 || item.Get == nil || item.Post == nil || len(item.Get.Parameters) != 1 || item.Get.Parameters[0].Name != "name" || !item.Get.Parameters[0].Required || len(item.Post.Parameters) != 1 || item.Post.Parameters[0].Name != "notify" {
				t.Fatalf("parameters leaked: %#v, GET %#v, POST %#v", item.Parameters, item.Get, item.Post)
			}
		})
	}
}

func TestInheritedParametersAreIndependent(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Items [/items{?filter}]
+ Parameters
    + filter (enum[string])
        + Members
            + old
### Read [GET]
+ Response 200
### Create [POST]
+ Response 201
### History [GET /history]
+ Response 200
## Update [/items{?other}]
### Change [PATCH]
+ Response 200
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/items"]
	get, post, history := item.Get.Parameters[0].Schema, item.Post.Parameters[0].Schema, api.Paths["/history"].Get.Parameters[0].Schema
	get.Description = "GET only"
	get.Enum[0] = "changed"
	if post.Description != "" || history.Description != "" || post.Enum[0] != "old" || history.Enum[0] != "old" {
		t.Fatalf("inherited schema mutation leaked: POST %#v; history %#v", post, history)
	}
}

func TestInheritedParameterJSONValuesAreIndependent(t *testing.T) {
	doc := `FORMAT: 1A
# API
## Items [/items{?value}]
+ Parameters
    + value (number)
        + Default: {"nested":[1]}
        + Members
            + {"nested":[2]}
### Read [GET]
+ Response 200
### Create [POST]
+ Response 201
## Update [/items]
### Change [PATCH]
+ Response 200
`
	api, err := NewParser().Parse(context.Background(), strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	item := api.Paths["/items"]
	get, post := item.Get.Parameters[0].Schema, item.Post.Parameters[0].Schema
	get.Default.(map[string]interface{})["nested"].([]interface{})[0] = "changed"
	get.Enum[0].(map[string]interface{})["nested"].([]interface{})[0] = "changed"
	if post.Default.(map[string]interface{})["nested"].([]interface{})[0] != model.JSONNumber("1") || post.Enum[0].(map[string]interface{})["nested"].([]interface{})[0] != model.JSONNumber("2") {
		t.Fatal("nested default or enum mutation leaked to POST")
	}
}
