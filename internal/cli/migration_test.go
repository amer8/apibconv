package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/format/openapi"
)

func TestVersionDefaultsAndExplicitFlagsOverrideEnvironment(t *testing.T) {
	t.Setenv("APIBCONV_OPENAPI_VERSION", "3.0")
	t.Setenv("APIBCONV_ASYNCAPI_VERSION", "2.6")
	flags, args, err := ParseFlags([]string{"--openapi-version", "3.1", "--asyncapi-version", "3.0"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAPIVersion != "3.1" || cfg.AsyncAPIVersion != "3.0" {
		t.Fatalf("explicit versions = %#v", cfg)
	}
	flags, args, err = ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAPIVersion != "3.0" || cfg.AsyncAPIVersion != "2.6" {
		t.Fatalf("environment versions = %#v", cfg)
	}
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	t.Setenv("APIBCONV_ASYNCAPI_VERSION", "")
	cfg, err = ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAPIVersion != "3.1" || cfg.AsyncAPIVersion != "3.0" {
		t.Fatalf("default versions = %#v", cfg)
	}
}

func TestAsyncOutputFormatInferredFromInput(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input.yaml"), filepath.Join(dir, "output.json")
	doc := "asyncapi: 2.6.0\ninfo: {title: Test, version: '1'}\nchannels: {}\n"
	if err := os.WriteFile(input, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := converter.New()
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(asyncapi.NewParser())
	c.RegisterWriter(asyncapi.NewWriter())
	c.RegisterWriter(openapi.NewWriter())
	if err := NewRunner(c).Convert(context.Background(), &Config{InputPath: input, OutputPath: output}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"asyncapi": "3.0.0"`) {
		t.Fatalf("output = %s", data)
	}
}
