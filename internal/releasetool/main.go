// Command releasetool stages and verifies apibconv release assets using gh and cosign.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

type options struct {
	tag, directory, repo, commit string
	dist, sbom, subjects, bundle string
}

var commands = []string{"prepare", "export", "verify", "guard", "publish", "verify-published"}

func parseOptions(args []string, stderr io.Writer) (string, options, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		if _, err := fmt.Fprintf(stderr, "Usage: releasetool <%s> [flags]\n", strings.Join(commands, "|")); err != nil {
			return "", options{}, err
		}
		if len(args) > 0 {
			return "", options{}, flag.ErrHelp
		}
		return "", options{}, errors.New("a release subcommand is required")
	}
	command := args[0]
	if !slices.Contains(commands, command) {
		return "", options{}, fmt.Errorf("unknown release subcommand %q", command)
	}
	var opts options
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.tag, "tag", "", "release version tag (required)")
	needsDirectory := slices.Contains([]string{"prepare", "export", "verify", "verify-published"}, command)
	needsRepo := slices.Contains([]string{"verify", "guard", "publish", "verify-published"}, command)
	needsCommit := slices.Contains([]string{"verify", "publish", "verify-published"}, command)
	if needsDirectory {
		flags.StringVar(&opts.directory, "directory", "", "release asset directory (required)")
	}
	if needsRepo {
		flags.StringVar(&opts.repo, "repo", "", "GitHub owner/repository (required)")
	}
	if needsCommit {
		flags.StringVar(&opts.commit, "commit", "", "expected source commit SHA (required)")
	}
	if command == "prepare" {
		flags.StringVar(&opts.dist, "dist", "dist", "GoReleaser output directory")
		flags.StringVar(&opts.sbom, "sbom", ".release-artifacts/bom.json", "CycloneDX SBOM path")
		flags.StringVar(&opts.subjects, "subjects", ".release-artifacts/subjects.sha256", "output checksums for attestation subjects")
	}
	if command == "export" {
		flags.StringVar(&opts.bundle, "bundle", "", "native Sigstore provenance bundle (required)")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return "", opts, err
	}
	if flags.NArg() != 0 {
		return "", opts, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	for _, required := range []struct {
		name, value string
		needed      bool
	}{
		{"tag", opts.tag, true}, {"directory", opts.directory, needsDirectory},
		{"repo", opts.repo, needsRepo}, {"commit", opts.commit, needsCommit},
		{"bundle", opts.bundle, command == "export"},
	} {
		if required.needed && required.value == "" {
			return "", opts, fmt.Errorf("--%s is required", required.name)
		}
	}
	return command, opts, nil
}

func run(args []string, stdout, stderr io.Writer) error {
	command, opts, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	spec, err := newReleaseSpec(opts.tag)
	if err != nil {
		return err
	}
	h := helper{spec: spec, repo: opts.repo, commit: opts.commit,
		execute: executeCommand, sleep: time.Sleep, stdout: stdout, stderr: stderr}
	switch command {
	case "prepare":
		return spec.prepare(opts.directory, opts.dist, opts.sbom, opts.subjects)
	case "export":
		return spec.exportBundle(opts.directory, opts.bundle)
	case "verify":
		return h.verify(opts.directory)
	case "guard":
		return h.guard()
	case "publish":
		return h.publish()
	case "verify-published":
		return h.verifyPublished(opts.directory)
	default:
		return fmt.Errorf("unknown release subcommand %q", command)
	}
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "Release verification failed: %v\n", err)
		os.Exit(1)
	}
}
