package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
)

const safeInput = `{"openapi":"3.0.3","info":{"title":"Test","version":"1"},"paths":{}}`

type failedOutputWriter struct{}

func (failedOutputWriter) Format() format.Format { return format.FormatOpenAPI }
func (failedOutputWriter) Version() string       { return "3.0.3" }
func (failedOutputWriter) Write(_ context.Context, _ *model.API, w io.Writer) error {
	if _, err := io.WriteString(w, "partial output"); err != nil {
		return err
	}
	return fmt.Errorf("injected write failure")
}

func TestConvertRejectsDowngradeAndPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.yaml")
	output := filepath.Join(dir, "output.yaml")
	doc := `openapi: 3.0.0
info: {title: Test, version: '1'}
paths: {}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
`
	if err := os.WriteFile(input, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := converter.New()
	if err != nil {
		t.Fatal(err)
	}
	c.RegisterParser(openapi.NewParser())
	c.RegisterWriter(openapi.NewWriter())
	err = NewRunner(c).Convert(context.Background(), &Config{
		InputPath: input, OutputPath: output, InputFormat: format.FormatOpenAPI,
		OutputFormat: format.FormatOpenAPI, OpenAPIVersion: "2.0",
	})
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if err == nil || !strings.Contains(err.Error(), "ACL") {
			t.Fatalf("expected ACL preservation error, got %v", err)
		}
	} else if !errors.Is(err, errors.ErrConversionFailed) || !strings.Contains(err.Error(), "unsupported conversion") {
		t.Fatalf("conversion error = %v, want unsupported conversion", err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("destination changed: %q", got)
	}
}

func TestUnsupportedCrossFormatPreservesDestination(t *testing.T) {
	for _, writer := range []format.Writer{apiblueprint.NewWriter(), asyncapi.NewWriter()} {
		t.Run(string(writer.Format()), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.yaml")
			output := filepath.Join(dir, "output.txt")
			doc := `openapi: 3.0.0
info: {title: Test, version: '1'}
security: [{key: []}]
paths:
  /items:
    get:
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
`
			if err := os.WriteFile(input, []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(output, []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			c, err := converter.New()
			if err != nil {
				t.Fatal(err)
			}
			c.RegisterParser(openapi.NewParser())
			c.RegisterWriter(writer)
			err = NewRunner(c).Convert(context.Background(), &Config{
				InputPath: input, OutputPath: output, InputFormat: format.FormatOpenAPI,
				OutputFormat: writer.Format(), Protocol: "kafka",
			})
			if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
				if err == nil || !strings.Contains(err.Error(), "ACL") {
					t.Fatalf("expected ACL preservation error, got %v", err)
				}
			} else if !errors.Is(err, errors.ErrConversionFailed) || !strings.Contains(err.Error(), "unsupported conversion") {
				t.Fatalf("conversion error = %v, want unsupported conversion", err)
			}
			got, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "original" {
				t.Fatalf("destination changed: %q", got)
			}
		})
	}
}

func TestConvertPreservesDestinationOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		failWriter  bool
	}{
		{name: "detection", input: "invalid document"},
		{name: "parsing", input: `{"openapi":"3.0.3","paths":[]}`},
		{name: "writing", input: safeInput, failWriter: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.json")
			output := filepath.Join(dir, "output.json")
			if err := os.WriteFile(input, []byte(tc.input), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(output, []byte("original"), 0o640); err != nil {
				t.Fatal(err)
			}
			c, err := converter.New()
			if err != nil {
				t.Fatal(err)
			}
			c.RegisterParser(openapi.NewParser())
			if tc.failWriter {
				c.RegisterWriter(failedOutputWriter{})
			} else {
				c.RegisterWriter(openapi.NewWriter())
			}
			err = NewRunner(c).Convert(context.Background(), &Config{InputPath: input, OutputPath: output})
			if err == nil {
				t.Fatal("expected conversion failure")
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "original" {
				t.Fatalf("destination changed: %q", data)
			}
			leftovers, err := filepath.Glob(filepath.Join(dir, ".apibconv-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(leftovers) != 0 {
				t.Fatalf("temporary files remain: %v", leftovers)
			}
		})
	}
}

func TestAtomicOutputRejectsInputAliases(t *testing.T) {
	for _, kind := range []string{"same-path", "hard-link", "symbolic-link"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.json")
			output := filepath.Join(dir, "output.json")
			if err := os.WriteFile(input, []byte(safeInput), 0o600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "same-path":
				output = input
			case "hard-link":
				if err := os.Link(input, output); err != nil {
					t.Skip(err)
				}
			case "symbolic-link":
				if err := os.Symlink(input, output); err != nil {
					t.Skip(err)
				}
			}
			if _, err := newAtomicOutput(input, output); err == nil {
				t.Fatal("accepted alias")
			}
			data, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != safeInput {
				t.Fatal("input changed")
			}
		})
	}
}

func TestAtomicOutputCommitsAndPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.json")
	if err := os.WriteFile(output, []byte("original"), 0o640); err != nil {
		t.Fatal(err)
	}
	destination, err := newAtomicOutput("", output)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if err == nil {
			t.Fatal("expected unsupported ACL preservation error")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer destination.cleanup()
	if _, err := destination.file.WriteString("converted"); err != nil {
		t.Fatal(err)
	}
	if err := destination.commit(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "converted" {
		t.Fatalf("got %q", data)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions: %o", info.Mode().Perm())
	}
}

func TestAtomicOutputRejectsConcurrentInPlaceEdit(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("replacing existing output is unsupported on this platform")
	}
	dir := t.TempDir()
	destination := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(destination, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	output, err := newAtomicOutput("", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer output.cleanup()
	if _, err := output.file.WriteString("converted"); err != nil {
		t.Fatal(err)
	}
	// Preserve both size and mtime: inode/metadata checks alone are insufficient.
	if err := os.WriteFile(destination, []byte("modified"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(destination, original.ModTime(), original.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := output.commit(); err == nil {
		t.Fatal("in-place edit was overwritten")
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "modified" {
		t.Fatalf("concurrent edit lost: %q, %v", got, err)
	}
}
