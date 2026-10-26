package main

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func TestCLIRequiredFlags(t *testing.T) {
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			_, _, err := parseOptions([]string{command}, io.Discard)
			requireError(t, err, "--tag is required")
		})
	}
	for _, test := range []struct {
		args    []string
		message string
	}{
		{[]string{"prepare", "--tag", testTag}, "--directory is required"},
		{[]string{"guard", "--tag", testTag}, "--repo is required"},
		{[]string{"verify", "--tag", testTag, "--directory", "assets", "--repo", testRepo}, "--commit is required"},
		{[]string{"export", "--tag", testTag, "--directory", "assets"}, "--bundle is required"},
		{[]string{"guard", "--tag", testTag, "unexpected"}, "unexpected positional arguments"},
		{[]string{"unknown"}, "unknown release subcommand"},
	} {
		_, _, err := parseOptions(test.args, io.Discard)
		requireError(t, err, test.message)
	}
}

func TestCLIHelpAndInvalidTag(t *testing.T) {
	if err := run([]string{"--help"}, io.Discard, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected help, got %v", err)
	}
	requireError(t, run([]string{"guard", "--tag", "../unsafe", "--repo", testRepo}, io.Discard, io.Discard), "semantic version tag")
}

func TestCLIPrepareAndExport(t *testing.T) {
	f := newFixture(t)
	args := []string{"prepare", "--tag", testTag, "--directory", f.directory,
		"--dist", f.dist, "--sbom", f.sbom, "--subjects", f.subjects}
	if err := run(args, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	source := f.root + "/bundle.json"
	writeFile(t, source, f.bundle(t, ""))
	if err := run([]string{"export", "--tag", testTag, "--directory", f.directory, "--bundle", source}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := f.spec.checkInventory(f.directory); err != nil {
		t.Fatal(err)
	}
}
