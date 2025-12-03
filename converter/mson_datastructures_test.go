package converter

import (
	"strings"
	"testing"
)

func TestDataStructuresGeneration(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Data Structures Test",
			Version: "1.0",
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"User": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
						"tags": {
							Type:  "array",
							Items: &Schema{Type: "string"},
						},
					},
					Required: []string{"id"},
				},
				"Error": {
					Type: "object",
					Properties: map[string]*Schema{
						"code":    {Type: "integer"},
						"message": {Type: "string"},
					},
				},
			},
		},
		Paths: map[string]PathItem{},
	}

	output, err := Format(spec)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Check header
	if !strings.Contains(output, "## Data Structures") {
		t.Error("Output should contain '## Data Structures'")
	}

	// Check User type
	if !strings.Contains(output, "### User (object)") {
		t.Error("Output should contain '### User (object)'")
	}
	if !strings.Contains(output, "+ id (integer, required)") {
		t.Errorf("Output should contain '+ id (integer, required)'. Got:\n%s", output)
	}
	if !strings.Contains(output, "+ tags (array[string], optional)") {
		t.Errorf("Output should contain '+ tags (array[string], optional)'. Got:\n%s", output)
	}

	// Check Error type
	if !strings.Contains(output, "### Error (object)") {
		t.Error("Output should contain '### Error (object)'")
	}
}
