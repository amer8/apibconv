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

	output, err := Format(spec)
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
