package specschema

import (
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
)

func TestValidateOpenAPI(t *testing.T) {
	valid := []byte(`
openapi: 3.0.0
info:
  title: Example
  version: 1.0.0
paths: {}
`)
	if errs := Validate(format.FormatOpenAPI, valid); len(errs) != 0 {
		t.Fatalf("Validate() returned unexpected errors: %#v", errs)
	}

	invalid := []byte(`
openapi: 3.0.0
info:
  title: Example
paths: {}
`)
	errs := Validate(format.FormatOpenAPI, invalid)
	if len(errs) == 0 {
		t.Fatal("Validate() returned no errors for an invalid OpenAPI document")
	}
}

func TestValidateSkipsUnsupportedSchemas(t *testing.T) {
	data := []byte(`
FORMAT: 1A

# Example
`)
	if errs := Validate(format.FormatAPIBlueprint, data); len(errs) != 0 {
		t.Fatalf("Validate() returned unexpected errors for API Blueprint: %#v", errs)
	}
}

func TestValidateRejectsMissingOrUnsupportedSpecVersion(t *testing.T) {
	tests := []struct {
		name       string
		formatType format.Format
		data       []byte
		want       string
	}{
		{
			name:       "OpenAPI missing version marker",
			formatType: format.FormatOpenAPI,
			data: []byte(`
info:
  title: Example
  version: 1.0.0
paths: {}
`),
			want: "missing OpenAPI version",
		},
		{
			name:       "OpenAPI unsupported version marker",
			formatType: format.FormatOpenAPI,
			data: []byte(`
openapi: 4.0.0
info:
  title: Example
  version: 1.0.0
paths: {}
`),
			want: `unsupported OpenAPI version "4.0.0"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.formatType, tt.data)
			if len(errs) == 0 {
				t.Fatal("Validate() returned no errors")
			}
			if !strings.Contains(errs[0].Message, tt.want) {
				t.Fatalf("Validate() error message = %q, want it to contain %q", errs[0].Message, tt.want)
			}
		})
	}
}
