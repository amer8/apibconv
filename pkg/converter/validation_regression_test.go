package converter_test

import (
	"context"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"strings"
	"testing"
)

func TestValidationReportsNullSchema(t *testing.T) {
	c, err := converter.New()
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(openapi.NewParser())
	issues, err := c.Validate(context.Background(), strings.NewReader("openapi: 3.0.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents: {schemas: {Bad: null}}\n"), format.FormatOpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("null schema was accepted")
	}
}
