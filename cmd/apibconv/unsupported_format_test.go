package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amer8/apibconv/internal/cli"
)

func TestRejectUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "async.yaml")
	output := filepath.Join(dir, "out.yaml")
	if err := os.WriteFile(input, []byte("eventapi: 2.6.0\ninfo: {title: Test, version: '1'}\nchannels: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"detect on conversion", []string{"-o", output, input}, "unsupported specification format"},
		{"detect on validation", []string{"--validate", input}, "unsupported specification format"},
		{"input format", []string{"--from", "eventapi", "-o", output, input}, "invalid input format"},
		{"output format", []string{"--to", "eventapi", input}, "invalid output format"},
		{"version flag", []string{"--eventapi-version", "3.0", input}, "flag provided but not defined"},
		{"protocol flag", []string{"--protocol", "kafka", input}, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conv, err := setupConverter()
			if err != nil {
				t.Fatal(err)
			}
			err = cli.NewApp(conv, "test").Run(context.Background(), tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Run(%v) = %v, want %q", tc.args, err, tc.want)
			}
			got, err := os.ReadFile(output)
			if err != nil || string(got) != "keep" {
				t.Fatalf("output = %q, %v; want unchanged", got, err)
			}
		})
	}
}
