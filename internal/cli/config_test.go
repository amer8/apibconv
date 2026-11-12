package cli

import (
	"strings"
	"testing"
)

func TestConfigRejectsInvalidArguments(t *testing.T) {
	t.Setenv("APIBCONV_DEFAULT_ENCODING", "")
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"multiple files", []string{"one.apib", "two.apib"}, "expected at most one input file"},
		{"after terminator", []string{"--", "-one", "-two"}, "expected at most one input file"},
		{"validate output", []string{"--validate", "input.apib", "-o", "out.json"}, "--output cannot be used with --validate"},
		{"validate empty output", []string{"--validate", "--output="}, "--output cannot be used with --validate"},
		{"validate format", []string{"--validate", "--to", "openapi"}, "--to cannot be used with --validate"},
		{"validate encoding", []string{"--validate", "-e", "json"}, "--encoding cannot be used with --validate"},
		{"validate version", []string{"--validate", "--openapi-version", "3.1"}, "--openapi-version cannot be used with --validate"},
		{"invalid encoding", []string{"-e", "xml"}, "invalid output encoding"},
		{"unsupported writer version", []string{"--openapi-version", "3.0.4"}, "invalid OpenAPI output version"},
		{"invalid version", []string{"--openapi-version", "banana"}, "invalid OpenAPI output version"},
		{"blueprint output", []string{"--to", "apiblueprint"}, "CLI conversion supports only openapi"},
		{"blueprint extension", []string{"-o", "result.APIB"}, "implies unsupported API Blueprint output"},
		{"markdown extension", []string{"-o", "result.md"}, "implies unsupported API Blueprint output"},
		{"report without validation", []string{"--report", "json"}, "--report requires --validate"},
		{"report with output validation", []string{"--validate-output", "--report", "json"}, "--report requires --validate"},
		{"invalid report", []string{"--validate", "--report", "xml"}, "invalid report format"},
		{"both validation modes", []string{"--validate", "--validate-output"}, "--validate-output cannot be used with --validate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags, args, err := ParseFlags(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ConfigFromFlags(flags, args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ConfigFromFlags(%q) = %v, want %q", tc.args, err, tc.want)
			}
		})
	}
}

func TestConfigEnvironmentAndOverrides(t *testing.T) {
	t.Setenv("APIBCONV_DEFAULT_ENCODING", "xml")
	t.Setenv("APIBCONV_OPENAPI_VERSION", "banana")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"invalid environment encoding", nil, "invalid output encoding"},
		{"invalid environment version", []string{"-e", "json"}, "invalid OpenAPI output version"},
		{"explicit flags override environment", []string{"-e", "yaml", "--openapi-version", "3.0.3"}, ""},
		{"validation ignores output defaults", []string{"--validate", "input.apib"}, ""},
		{"explicit output overrides extension", []string{"--to", "openapi", "-o", "result.apib", "-e", "json", "--openapi-version", "3.1"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags, args, err := ParseFlags(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ConfigFromFlags(flags, args)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("ConfigFromFlags(%q) = %v, want %q", tc.args, err, tc.want)
			}
		})
	}
}

func TestConfigAcceptsWriterVersions(t *testing.T) {
	t.Setenv("APIBCONV_DEFAULT_ENCODING", "")
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	for _, version := range []string{"2.0", "2.0.0", "3.0", "3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1", "3.1.0"} {
		t.Run(version, func(t *testing.T) {
			flags, args, err := ParseFlags([]string{"--openapi-version", version})
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := ConfigFromFlags(flags, args)
			if err != nil || cfg.OpenAPIVersion != version {
				t.Fatalf("version %q: config = %+v, error = %v", version, cfg, err)
			}
		})
	}
}
