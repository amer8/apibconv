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

	bp, err := spec.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	output := bp.String()
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

func TestDataStructures_Empty(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Empty Components", Version: "1.0"},
		Paths:   map[string]PathItem{},
	}
	// No components

	bp, err := spec.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	output := bp.String()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if strings.Contains(output, "## Data Structures") {
		t.Error("Should not contain Data Structures section for empty components")
	}

	// Empty schemas
	spec.Components = &Components{Schemas: map[string]*Schema{}}
	bp, err = spec.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	output = bp.String()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	if strings.Contains(output, "## Data Structures") {
		t.Error("Should not contain Data Structures section for empty schemas map")
	}
}

func TestDataStructures_Sorting(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Sort Test", Version: "1.0"},
		Components: &Components{
			Schemas: map[string]*Schema{
				"Zebra": {Type: "object"},
				"Alpha": {Type: "object"},
				"Beta":  {Type: "object"},
			},
		},
		Paths: map[string]PathItem{},
	}

	bp, err := spec.ToAPIBlueprint()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	output := bp.String()
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	idxA := strings.Index(output, "### Alpha")
	idxB := strings.Index(output, "### Beta")
	idxZ := strings.Index(output, "### Zebra")

	if idxA == -1 || idxB == -1 || idxZ == -1 {
		t.Fatal("Missing expected schemas")
	}

	if idxA > idxB || idxB > idxZ {
		t.Error("Schemas should be sorted alphabetically")
	}
}
