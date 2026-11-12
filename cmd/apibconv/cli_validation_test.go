package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIValidationHelper(t *testing.T) {
	if os.Getenv("APIBCONV_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"apibconv"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func invokeCLI(t *testing.T, args ...string) (stdoutText, stderrText string, exitCode int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIValidationHelper$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "APIBCONV_TEST_HELPER=1", "APIBCONV_DEFAULT_ENCODING=", "APIBCONV_OPENAPI_VERSION=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatal(err)
	return "", "", -1
}

func writeSpec(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLIJSONValidationReports(t *testing.T) {
	valid := writeSpec(t, "openapi: 3.1.0\ninfo: {title: API, version: '1'}\npaths: {}\n")
	invalid := writeSpec(t, "openapi: 3.1.0\ninfo: {title: API}\npaths: {}\n")
	malformed := writeSpec(t, "openapi: [\n")
	for _, tc := range []struct {
		name, path  string
		valid       bool
		exit        int
		wantIssue   bool
		wantFailure bool
	}{
		{"valid", valid, true, 0, false, false},
		{"invalid schema", invalid, false, 2, true, false},
		{"parse failure retains diagnostics", malformed, false, 1, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := invokeCLI(t, "--validate", "--from", "openapi", "--report", "json", tc.path)
			var report struct {
				Valid  bool `json:"valid"`
				Issues []struct {
					Level, Path, Message string
				} `json:"issues"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatalf("invalid JSON stdout %q: %v; stderr=%q", stdout, err, stderr)
			}
			if code != tc.exit || report.Valid != tc.valid || (len(report.Issues) > 0) != tc.wantIssue || (report.Error != "") != tc.wantFailure {
				t.Fatalf("exit=%d report=%+v stderr=%q", code, report, stderr)
			}
			if tc.wantIssue && (report.Issues[0].Level != "error" || report.Issues[0].Path == "" || report.Issues[0].Message == "") {
				t.Fatalf("incomplete issue: %+v", report.Issues[0])
			}
			if report.Issues == nil {
				t.Fatal("issues must be an array, including on success")
			}
		})
	}
	_, stderr, code := invokeCLI(t, "--validate", "--from", "openapi", invalid)
	if code != 2 || !strings.Contains(stderr, "error: /info:") {
		t.Fatalf("expected path in text diagnostics, exit=%d stderr=%q", code, stderr)
	}
}

func TestCLIValidateOutputBeforeWriting(t *testing.T) {
	good := writeSpec(t, "openapi: 3.1.0\ninfo: {title: API, version: '1'}\npaths: {}\n")
	blueprint := writeSpec(t, "FORMAT: 1A\n# API\n## Items [/items]\n### List [GET]\n+ Response 200\n")
	bad := writeSpec(t, "FORMAT: 1A\n# API\n## Items [/items]\n### List [GET]\n+ Response 200 (application/json)\n    + Attributes (array[Missing])\n")
	stdout, stderr, code := invokeCLI(t, "--from", "openapi", "--to", "openapi", "-e", "json", "--validate-output", good)
	if code != 0 || !json.Valid([]byte(stdout)) {
		t.Fatalf("valid output: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	stdout, stderr, code = invokeCLI(t, "--from", "apiblueprint", "--to", "openapi", "-e", "json", "--validate-output", blueprint)
	if code != 0 || !strings.Contains(stdout, `"version": ""`) {
		t.Fatalf("empty API version behavior: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	stdout, stderr, code = invokeCLI(t, "--from", "apiblueprint", "--to", "openapi", "--validate-output", bad)
	if code != 2 || stdout != "" || !strings.Contains(stderr, "Missing") {
		t.Fatalf("invalid output: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return // Existing outputs cannot be replaced on these platforms.
	}
	dir := t.TempDir()
	destination := filepath.Join(dir, "out.yaml")
	_, stderr, code = invokeCLI(t, "--from", "openapi", "--validate-output", "-o", destination, good)
	if data, err := os.ReadFile(destination); code != 0 || err != nil || !strings.Contains(string(data), "openapi: 3.1.0") {
		t.Fatalf("valid destination: exit=%d output=%q read error=%v stderr=%q", code, data, err, stderr)
	}
	if err := os.WriteFile(destination, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code = invokeCLI(t, "--from", "apiblueprint", "--validate-output", "-o", destination, bad)
	data, err := os.ReadFile(destination)
	if code != 2 || err != nil || string(data) != "original" {
		t.Fatalf("existing destination: exit=%d output=%q read error=%v stderr=%q", code, data, err, stderr)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".apibconv-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary output remains: %v, %v", leftovers, err)
	}
}
