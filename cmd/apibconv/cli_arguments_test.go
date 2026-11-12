package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIArgumentsAroundInput(t *testing.T) {
	t.Setenv("APIBCONV_DEFAULT_ENCODING", "")
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	dir := t.TempDir()
	input := filepath.Join(dir, "input.apib")
	output := filepath.Join(dir, "out.json")
	if err := os.WriteFile(input, []byte("FORMAT: 1A\n# Example\n## Items [/items]\n### List [GET]\n+ Response 200\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--from", "apiblueprint", input, "--encoding=json", "-o", output}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || !strings.Contains(string(data), `"openapi": "3.1.0"`) {
		t.Fatalf("output = %q, %v", data, err)
	}
}

func TestCLIRejectsArgumentsBeforeOpeningFiles(t *testing.T) {
	t.Setenv("APIBCONV_DEFAULT_ENCODING", "")
	t.Setenv("APIBCONV_OPENAPI_VERSION", "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"missing.apib", "--unknown"}, "flag provided but not defined"},
		{[]string{"missing.apib", "another.apib"}, "expected at most one input file"},
		{[]string{"--validate", "missing.apib", "-o", "out.json"}, "--output cannot be used with --validate"},
		{[]string{"missing.apib", "--to", "apiblueprint"}, "CLI conversion supports only openapi"},
		{[]string{"missing.apib", "-o", "out.md"}, "implies unsupported API Blueprint output"},
		{[]string{"missing.apib", "-e", "xml"}, "invalid output encoding"},
	} {
		err := run(tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("run(%q) = %v, want %q", tc.args, err, tc.want)
		}
	}
}
