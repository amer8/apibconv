package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestCLILicenses(t *testing.T) {
	if (runtime.GOOS != "linux" && runtime.GOOS != "darwin") || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("target does not have a generated release report")
	}
	stdout, stderr, code := invokeCLI(t, "--licenses")
	if code != 0 || stderr != "" {
		t.Fatalf("--licenses: exit=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"Third-party licenses for " + runtime.GOOS + "/" + runtime.GOARCH,
		"go.yaml.in/yaml/v3",
		"Copyright 2011-2016 Canonical Ltd.",
		"Copyright (c) 2006-2010 Kirill Simonov",
		"OpenAPI Initiative",
		"Copyright 2009 The Go Authors.",
		"Additional IP Rights Grant (Patents)",
		"===== golang.org/x/text/PATENTS =====",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--licenses output is missing %q", want)
		}
	}
}
