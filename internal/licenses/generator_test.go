package licenses

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorIncludesWorkspacePatentsAndFallbackMaterials(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, content string, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"scripts/generate-licenses.sh", "scripts/licenses-report.tmpl", "THIRD_PARTY_NOTICES.md", "licenses/APACHE-2.0", "licenses/GO-BSD-3-CLAUSE", "licenses/GO-PATENTS"} {
		data, err := os.ReadFile(filepath.Join("..", "..", name))
		if err != nil {
			t.Fatal(err)
		}
		write("app/"+name, string(data), 0o644)
	}
	write("go.work", "go 1.26.0\nuse (\n ./app\n ./dep\n)\n", 0o644)
	write("app/go.mod", "module github.com/amer8/apibconv\n\ngo 1.26.0\n\nrequire example.com/dep v0.0.0\n", 0o644)
	write("app/cmd/apibconv/main.go", "package main\nimport _ \"example.com/dep\"\nfunc main() {}\n", 0o644)
	write("dep/go.mod", "module example.com/dep\n\ngo 1.26.0\n", 0o644)
	write("dep/dep.go", "package dep\n", 0o644)
	write("dep/PATENTS", "workspace patent grant\n", 0o644)
	for _, target := range []string{"linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64"} {
		write("app/internal/licenses/reports/"+target+".txt", "other target material: "+target+"\n", 0o644)
	}
	write("fake-go-licenses", `#!/bin/sh
set -eu
case "$1" in
  report) printf 'fixture dependency inventory\n' ;;
  save)
    while [ "$#" -gt 0 ]; do
      if [ "$1" = --save_path ]; then shift; mkdir -p "$1/fixture"; printf 'fixture license\n' > "$1/fixture/LICENSE"; exit 0; fi
      shift
    done
    exit 1 ;;
  *) exit 1 ;;
esac
`, 0o755)
	run := func(mode string) (string, error) {
		t.Helper()
		cmd := exec.Command("sh", "scripts/generate-licenses.sh", mode, "linux", "amd64")
		cmd.Dir = filepath.Join(root, "app")
		cmd.Env = append(os.Environ(), "GOWORK="+filepath.Join(root, "go.work"), "GOPROXY=off", "GO_LICENSES="+filepath.Join(root, "fake-go-licenses"))
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run("--write"); err != nil {
		t.Fatalf("generate reports: %v\n%s", err, out)
	}
	for _, name := range []string{"linux_amd64.txt", "fallback.txt"} {
		data, err := os.ReadFile(filepath.Join(root, "app", "internal", "licenses", "reports", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"fixture license", "example.com/dep/PATENTS", "workspace patent grant"} {
			if !strings.Contains(string(data), want) {
				t.Errorf("%s does not contain %q", name, want)
			}
		}
		if name == "fallback.txt" && !strings.Contains(string(data), "other target material: darwin_arm64") {
			t.Error("fallback does not contain materials from the other targets")
		}
	}
	if out, err := run("--check"); err != nil {
		t.Fatalf("check regenerated reports: %v\n%s", err, out)
	}
	write("app/internal/licenses/reports/darwin_arm64.txt", "updated other target material\n", 0o644)
	if out, err := run("--check"); err == nil || !strings.Contains(out, "Fallback report is out of date") {
		t.Fatalf("stale fallback was not detected: error=%v output=%s", err, out)
	}
	write("app/internal/licenses/reports/darwin_arm64.txt", "other target material: darwin_arm64\n", 0o644)
	write("dep/PATENTS", "updated workspace patent grant\n", 0o644)
	if out, err := run("--check"); err == nil || !strings.Contains(out, "out of date") {
		t.Fatalf("changed patent was not detected: error=%v output=%s", err, out)
	}
}
