package converter

import (
	"bytes"
	"strings"
	"testing"
)

func TestAPIBlueprint_ToOpenAPI_Basic(t *testing.T) {
	bp := createBasicAPIBlueprint()

	openapi, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openapi == nil {
		t.Fatal("openapi spec is nil")
	}

	checkOpenAPIBasicInfo(t, openapi)
	checkOpenAPIPath(t, openapi)
}

func createBasicAPIBlueprint() *APIBlueprint {
	return &APIBlueprint{
		TitleField:  "My API",
		Description: "A simple API",
		Metadata:    map[string]string{"HOST": "https://api.example.com", "VERSION": "1.0.0"},
		Groups: []*ResourceGroup{
			{
				Name: "Users",
				Resources: []*Resource{
					{
						URITemplate: "/users",
						Actions: []*Action{
							{
								Method: "GET",
								Name:   "List Users",
								Examples: []*Transaction{
									{
										Response: &BlueprintResponse{
											Name: "200",
											Response: Response{
												Content: map[string]MediaType{
													MediaTypeJSON: {Schema: &Schema{Type: "array"}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func checkOpenAPIBasicInfo(t *testing.T, openapi *OpenAPI) {
	if openapi.OpenAPI != "3.0.0" {
		t.Errorf("expected OpenAPI version %s, got %s", "3.0.0", openapi.OpenAPI)
	}
	if openapi.Info.Title != "My API" {
		t.Errorf("expected title %s, got %s", "My API", openapi.Info.Title)
	}
	if openapi.Info.Description != "A simple API" {
		t.Errorf("expected description %s, got %s", "A simple API", openapi.Info.Description)
	}
	if openapi.Info.Version != "1.0.0" {
		t.Errorf("expected info version %s, got %s", "1.0.0", openapi.Info.Version)
	}
	if len(openapi.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(openapi.Servers))
	} else if openapi.Servers[0].URL != "https://api.example.com" {
		t.Errorf("expected server URL %s, got %s", "https://api.example.com", openapi.Servers[0].URL)
	}
}

func checkOpenAPIPath(t *testing.T, openapi *OpenAPI) {
	if len(openapi.Paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(openapi.Paths))
	}

	pathItem, ok := openapi.Paths["/users"]
	if !ok {
		t.Fatal("expected path /users not found")
	}
	if pathItem.Get == nil {
		t.Fatal("expected GET operation for /users")
	}
	if pathItem.Get.Summary != "List Users" {
		t.Errorf("expected summary %s, got %s", "List Users", pathItem.Get.Summary)
	}
	if _, ok := pathItem.Get.Responses["200"]; !ok {
		t.Error("expected 200 response")
	} else if _, ok := pathItem.Get.Responses["200"].Content[MediaTypeJSON]; !ok {
		t.Errorf("expected content type %s for 200 response", MediaTypeJSON)
	} else if pathItem.Get.Responses["200"].Content[MediaTypeJSON].Schema.Type != "array" {
		t.Errorf("expected schema type %s, got %s", "array", pathItem.Get.Responses["200"].Content[MediaTypeJSON].Schema.Type)
	}
}

func TestAPIBlueprint_ToOpenAPI_MultipleMethodsAndParameters(t *testing.T) {
	bp := &APIBlueprint{
		TitleField: "Test API",
		Groups: []*ResourceGroup{
			{
				Resources: []*Resource{
					{
						URITemplate: "/posts/{id}",
						Parameters: []Parameter{
							{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
						},
						Actions: []*Action{
							{
								Method: "GET",
								Name:   "Get Post",
								Parameters: []Parameter{
									{Name: "token", In: "query", Required: false, Schema: &Schema{Type: "string"}},
								},
								Examples: []*Transaction{
									{Response: &BlueprintResponse{Name: "200"}},
								},
							},
							{
								Method: "PUT",
								Name:   "Update Post",
								Examples: []*Transaction{
									{Request: &Request{Content: map[string]MediaType{MediaTypeJSON: {Schema: &Schema{Type: "object"}}}}}, // This line has an extra comma
									{Response: &BlueprintResponse{Name: "200"}},
								},
							},
						},
					},
				},
			},
		},
	}

	openapi, err := bp.ToOpenAPI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openapi == nil {
		t.Fatal("openapi spec is nil")
	}

	pathItem, ok := openapi.Paths["/posts/{id}"]
	if !ok {
		t.Fatal("expected path /posts/{id} not found")
	}

	// GET operation
	if pathItem.Get == nil {
		t.Fatal("expected GET operation")
	}
	if pathItem.Get.Summary != "Get Post" {
		t.Errorf("expected summary %s, got %s", "Get Post", pathItem.Get.Summary)
	}
	if len(pathItem.Get.Parameters) != 2 { // path 'id' + query 'token'
		t.Errorf("expected 2 parameters, got %d", len(pathItem.Get.Parameters))
	}
	if _, ok := pathItem.Get.Responses["200"]; !ok {
		t.Error("expected 200 response for GET")
	}

	// PUT operation
	if pathItem.Put == nil {
		t.Fatal("expected PUT operation")
	}
	if pathItem.Put.Summary != "Update Post" {
		t.Errorf("expected summary %s, got %s", "Update Post", pathItem.Put.Summary)
	}
	if len(pathItem.Put.Parameters) != 1 { // path 'id'
		t.Errorf("expected 1 parameter for PUT, got %d", len(pathItem.Put.Parameters))
	}
	if pathItem.Put.RequestBody == nil {
		t.Fatal("expected request body for PUT")
	}
	if _, ok := pathItem.Put.RequestBody.Content[MediaTypeJSON]; !ok {
		t.Errorf("expected content type %s for PUT request body", MediaTypeJSON)
	}
}

func TestAPIBlueprint_ToBlueprint_RoundTrip(t *testing.T) {
	// This test relies on the correctness of OpenAPI -> APIBlueprint conversion
	// which is handled by openapi.ToBlueprint() internally.
	// We'll use a simple API Blueprint structure, convert it to OpenAPI,
	// then convert the OpenAPI back to API Blueprint string.
	// The original ToBlueprint just calls the OpenAPI one, so we are testing that path.
	apibContent := "FORMAT: 1A\nHOST: https://example.com\n# My Test API\n## Group Users\n### User [/users/{id}]\n#### Get User [GET]\n+ Response 200 (application/json)\n\n\t{ \"id\": 1, \"name\": \"test\" }"

	// Parse the API Blueprint content
	bp, err := ParseBlueprint([]byte(apibContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bp == nil {
		t.Fatal("blueprint spec is nil")
	}

	// Convert to Blueprint string (which internally goes via OpenAPI)
	convertedApib := bp.String()

	// Basic assertion for now, as exact string matching can be brittle due to formatting
	if !strings.Contains(convertedApib, "# My Test API") {
		t.Error("converted API Blueprint missing title")
	}
	if !strings.Contains(convertedApib, "HOST: https://example.com") {
		t.Error("converted API Blueprint missing host")
	}
	if !strings.Contains(convertedApib, "## Group Users") {
		t.Errorf("converted API Blueprint missing resource group. Got:\n%s", convertedApib)
	}
	// Note: Resource name "User" is lost during OpenAPI conversion and becomes the path
	if !strings.Contains(convertedApib, "### /users/{id} [/users/{id}]") {
		t.Errorf("converted API Blueprint missing user resource (expected path as name). Got:\n%s", convertedApib)
	}
	if !strings.Contains(convertedApib, "#### Get User [GET]") {
		t.Errorf("converted API Blueprint missing GET operation. Got:\n%s", convertedApib)
	}
	if !strings.Contains(convertedApib, "+ Response 200 (application/json)") {
		t.Errorf("converted API Blueprint missing 200 response. Got:\n%s", convertedApib)
	}
	// Check for body content (indented JSON)
	if !strings.Contains(convertedApib, `"id": 1`) || !strings.Contains(convertedApib, `"name": "test"`) {
		t.Errorf("converted API Blueprint missing response body example. Got:\n%s", convertedApib)
	}
}

func TestAPIBlueprint_WriteTo(t *testing.T) {
	apibContent := "FORMAT: 1A\n# Simple API\n## Resource [/status]\n### GET\n+ Response 200"
	bp, err := ParseBlueprint([]byte(apibContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, err = bp.WriteTo(&buf)
	if err != nil {
		t.Fatalf("unexpected error writing blueprint: %v", err)
	}

	if !strings.Contains(buf.String(), "# Simple API") {
		t.Error("written API Blueprint missing title")
	}
	// Resource name "Resource" is lost, path is used. Resources are nested in groups.
	if !strings.Contains(buf.String(), "### /status [/status]") {
		t.Errorf("written API Blueprint missing resource. Got:\n%s", buf.String())
	}
}

func TestAPIBlueprint_Version(t *testing.T) {
	bp1 := &APIBlueprint{Metadata: map[string]string{"VERSION": "2.0.0"}}
	if bp1.Version() != "2.0.0" {
		t.Errorf("expected version %s, got %s", "2.0.0", bp1.Version())
	}

	bp2 := &APIBlueprint{VersionField: "1A"}
	if bp2.Version() != "1A" {
		t.Errorf("expected version %s, got %s", "1A", bp2.Version())
	}

	bp3 := &APIBlueprint{Metadata: map[string]string{"VERSION": "v3"}, VersionField: "1A"}
	if bp3.Version() != "v3" { // Metadata takes precedence
		t.Errorf("expected version %s, got %s", "v3", bp3.Version())
	}

	bp4 := &APIBlueprint{}
	if bp4.Version() != "" {
		t.Errorf("expected empty version, got %s", bp4.Version())
	}
}

func TestAPIBlueprint_Title(t *testing.T) {
	bp1 := &APIBlueprint{TitleField: "My Title"}
	if bp1.Title() != "My Title" {
		t.Errorf("expected title %s, got %s", "My Title", bp1.Title())
	}

	bp2 := &APIBlueprint{}
	if bp2.Title() != "" {
		t.Errorf("expected empty title, got %s", bp2.Title())
	}
}

func TestAPIBlueprint_AsTypeMethods(t *testing.T) {
	bp := &APIBlueprint{}

	o, ok := bp.AsOpenAPI()
	if ok {
		t.Error("expected AsOpenAPI to return false")
	}
	if o != nil {
		t.Errorf("expected nil OpenAPI, got %v", o)
	}

	a, ok := bp.AsAsyncAPI()
	if ok {
		t.Error("expected AsAsyncAPI to return false")
	}
	if a != nil {
		t.Errorf("expected nil AsyncAPI, got %v", a)
	}

	av3, ok := bp.AsAsyncAPIV3()
	if ok {
		t.Error("expected AsAsyncAPIV3 to return false")
	}
	if av3 != nil {
		t.Errorf("expected nil AsyncAPIV3, got %v", av3)
	}

	ap, ok := bp.AsAPIBlueprint()
	if !ok {
		t.Error("expected AsAPIBlueprint to return true")
	}
	if ap != bp {
		t.Errorf("expected APIBlueprint spec %p, got %p", bp, ap)
	}
}
