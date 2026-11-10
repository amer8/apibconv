package cli

import (
	"testing"
)

func TestVersionDefaultsAndExplicitFlagsOverrideEnvironment(t *testing.T) {
	t.Setenv("APIBCONV_OPENAPI_VERSION", "3.0")
	flags, args, err := ParseFlags([]string{"--openapi-version", "3.1"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.OpenAPIVersion != "3.1" {
		t.Fatalf("explicit version = %q", cfg.OpenAPIVersion)
	}
	flags, args, err = ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.OpenAPIVersion != "3.0" {
		t.Fatalf("environment version = %q", cfg.OpenAPIVersion)
	}
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	cfg, err = ConfigFromFlags(flags, args)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAPIVersion != "3.1" {
		t.Fatalf("default version = %q", cfg.OpenAPIVersion)
	}
}
