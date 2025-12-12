package converter

import (
	"strings"
	"testing"
)

func TestMSONGeneration(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "MSON Gen Test",
			Version: "1.0",
		},
		Paths: map[string]PathItem{
			"/user": {
				Get: &Operation{
					Summary: "Get User",
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"id":   {Type: "integer"},
											"name": {Type: "string"},
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

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Check for MSON syntax
	if !strings.Contains(output, "+ Attributes") {
		t.Error("Output should contain '+ Attributes'")
	}
	if !strings.Contains(output, "+ id (number") && !strings.Contains(output, "+ id (integer") {
		t.Errorf("Output should contain '+ id (number/integer)'. Got:\n%s", output)
	}
	if !strings.Contains(output, "+ name (string") {
		t.Error("Output should contain '+ name (string)'")
	}

	// Should NOT contain raw JSON schema
	if strings.Contains(output, "\"type\": \"object\"") {
		t.Error("Output should not contain raw JSON schema")
	}
}

func TestMSON_Arrays(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Array Test", Version: "1.0"},
		Paths: map[string]PathItem{
			"/list": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"tags": {
												Type:  "array",
												Items: &Schema{Type: "string"},
											},
											"scores": {
												Type:  "array",
												Items: &Schema{Type: "integer"},
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

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(output, "+ tags (array[string], optional)") {
		t.Errorf("Output should contain typed array definition. Got:\n%s", output)
	}
	if !strings.Contains(output, "+ scores (array[integer], optional)") {
		t.Errorf("Output should contain typed array definition. Got:\n%s", output)
	}
}

func TestMSON_References(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Ref Test", Version: "1.0"},
		Paths: map[string]PathItem{
			"/ref": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"user": {Ref: "#/components/schemas/User"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"User": {Type: "object"},
			},
		},
	}

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(output, "+ user (User, optional)") {
		t.Errorf("Output should contain reference type. Got:\n%s", output)
	}
}

func TestMSON_DescriptionAndExample(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Desc Test", Version: "1.0"},
		Paths: map[string]PathItem{
			"/desc": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"field": {
												Type:        "string",
												Description: "A test field",
												Example:     "test-value",
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

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(output, "+ field: `test-value` (string, optional) - A test field") {
		t.Errorf("Output should contain description and example. Got:\n%s", output)
	}
}

func TestMSON_RequiredOptional(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Req Test", Version: "1.0"},
		Paths: map[string]PathItem{
			"/req": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"reqField": {Type: "string"},
											"optField": {Type: "string"},
										},
										Required: []string{"reqField"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(output, "+ reqField (string, required)") {
		t.Error("Output should contain required field")
	}
	if !strings.Contains(output, "+ optField (string, optional)") {
		t.Error("Output should contain optional field")
	}
}

func TestMSON_NestedObject(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Nested Test", Version: "1.0"},
		Paths: map[string]PathItem{
			"/nested": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {
							Description: "OK",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "object",
										Properties: map[string]*Schema{
											"parent": {
												Type: "object",
												Properties: map[string]*Schema{
													"child": {Type: "string"},
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

	output, err := spec.ToBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !strings.Contains(output, "+ parent (object, optional)") {
		t.Error("Output should contain parent object")
	}
	// Indentation check roughly
	if !strings.Contains(output, "    + child (string, optional)") {
		t.Error("Output should contain nested child")
	}
}
