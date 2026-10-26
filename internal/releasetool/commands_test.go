package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	testRepo   = "amer8/apibconv"
	testCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type fakeRunner struct {
	calls     [][]string
	metadata  releaseMetadata
	commit    string
	respond   commandRunner
	verifyErr int
}

func metadataFor(spec releaseSpec, draft, immutable bool) releaseMetadata {
	metadata := releaseMetadata{Draft: &draft, Immutable: immutable}
	for _, name := range spec.assetNames(true) {
		metadata.Assets = append(metadata.Assets, releaseAsset{Name: name})
	}
	return metadata
}

func newTestHelper(t *testing.T, f *fixture, metadata releaseMetadata) (*helper, *fakeRunner) {
	t.Helper()
	fake := &fakeRunner{metadata: metadata, commit: testCommit}
	h := &helper{spec: f.spec, repo: testRepo, commit: testCommit, execute: fake.execute,
		stdout: io.Discard, stderr: io.Discard, sleep: func(time.Duration) { t.Fatal("unexpected sleep") }}
	return h, fake
}

func (f *fakeRunner) execute(name string, args ...string) (commandResult, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.respond != nil {
		return f.respond(name, args...)
	}
	if name == "gh" && len(args) > 1 && args[0] == "api" {
		var value any
		switch {
		case strings.Contains(args[1], "/commits/"):
			value = map[string]string{"sha": f.commit}
		case strings.Contains(args[1], "/releases/tags/"):
			if f.metadata.Draft != nil && *f.metadata.Draft {
				return commandResult{exitCode: 1, stderr: "Not Found (HTTP 404)"}, nil
			}
			value = f.metadata
		case strings.Contains(args[1], "/releases?"):
			value = []map[string]any{{"tag_name": testTag, "draft": f.metadata.Draft,
				"immutable": f.metadata.Immutable, "assets": f.metadata.Assets}}
		default:
			return commandResult{}, fmt.Errorf("unexpected API endpoint: %s", args[1])
		}
		data, err := json.Marshal(value)
		return commandResult{stdout: string(data)}, err
	}
	if name == "gh" && len(args) > 1 && args[0] == "release" && args[1] == "verify" && f.verifyErr > 0 {
		f.verifyErr--
		return commandResult{exitCode: 1}, nil
	}
	return commandResult{}, nil
}

func callsWithPrefix(calls [][]string, prefix ...string) [][]string {
	var matched [][]string
	for _, call := range calls {
		if len(call) >= len(prefix) && slices.Equal(call[:len(prefix)], prefix) {
			matched = append(matched, call)
		}
	}
	return matched
}

func TestVerificationPinsWorkflowCommitAndRef(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	h, fake := newTestHelper(t, f, releaseMetadata{})
	fake.respond = func(name string, args ...string) (commandResult, error) {
		if name == "gh" && args[0] == "attestation" && filepath.Dir(args[2]) != f.directory {
			if mustDigest(t, args[2]) == mustDigest(t, filepath.Join(f.directory, f.spec.archives[0])) {
				t.Fatal("negative verification did not modify the archive")
			}
			return commandResult{exitCode: 1}, nil
		}
		return commandResult{}, nil
	}
	if err := h.verify(f.directory); err != nil {
		t.Fatal(err)
	}
	calls := callsWithPrefix(fake.calls, "gh", "attestation", "verify")
	if len(calls) != 8 { // Seven subjects plus the tampered archive.
		t.Fatalf("expected 8 provenance checks, got %d", len(calls))
	}
	for _, call := range calls {
		for _, argument := range []string{h.identity(), testCommit, "refs/tags/" + testTag, "--deny-self-hosted-runners"} {
			if !slices.Contains(call, argument) {
				t.Errorf("verification did not require %q: %v", argument, call)
			}
		}
	}
	if count := len(callsWithPrefix(fake.calls, "cosign", "verify-blob")); count != 6 {
		t.Fatalf("expected six checksum signature checks, got %d", count)
	}
}

func TestVerifierAcceptingTamperingFailsTheRelease(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	h, _ := newTestHelper(t, f, releaseMetadata{})
	requireError(t, h.verify(f.directory), "accepted a tampered archive")
}

func TestPublishedReleaseCannotBeRebuilt(t *testing.T) {
	f := newFixture(t)
	h, _ := newTestHelper(t, f, metadataFor(f.spec, false, true))
	requireError(t, h.guard(), "already published")
}

func TestPublicationRetriesDoNotModifyPublishedRelease(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, false, true))
	if err := h.publish(); err != nil {
		t.Fatal(err)
	}
	if calls := callsWithPrefix(fake.calls, "gh", "release"); len(calls) != 0 {
		t.Fatalf("retry modified an existing release: %v", calls)
	}
	if calls := callsWithPrefix(fake.calls, "gh", "api"); len(calls) != 2 {
		t.Fatalf("published release lookup unexpectedly needed a fallback: %v", calls)
	}
}

func TestMovedTagPreventsPublication(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, true, false))
	fake.commit = strings.Repeat("b", 40)
	requireError(t, h.publish(), "no longer points")
	if len(callsWithPrefix(fake.calls, "gh", "release")) != 0 {
		t.Fatal("published assets for a moved tag")
	}
}

func TestIncompleteDraftCannotBePublished(t *testing.T) {
	f := newFixture(t)
	metadata := metadataFor(f.spec, true, false)
	metadata.Assets = metadata.Assets[:1]
	h, fake := newTestHelper(t, f, metadata)
	requireError(t, h.publish(), "complete expected asset inventory")
	if len(callsWithPrefix(fake.calls, "gh", "release")) != 0 {
		t.Fatal("published an incomplete draft")
	}
}

func TestCompleteDraftIsPublishedWithoutRebuilding(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, true, false))
	if err := h.publish(); err != nil {
		t.Fatal(err)
	}
	calls := callsWithPrefix(fake.calls, "gh", "release")
	expected := []string{"gh", "release", "edit", testTag, "--repo", testRepo, "--draft=false", "--verify-tag"}
	if len(calls) != 1 || !slices.Equal(calls[0], expected) {
		t.Fatalf("expected publication without rebuilding, got %v", calls)
	}
}

func TestDraftLookupPaginatesPastOtherReleases(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, true, false))
	fake.respond = func(name string, args ...string) (commandResult, error) {
		if name != "gh" || args[0] != "api" {
			return commandResult{}, nil
		}
		var value any
		switch args[1] {
		case "repos/" + testRepo + "/commits/" + testTag:
			value = map[string]string{"sha": testCommit}
		case "repos/" + testRepo + "/releases/tags/" + testTag:
			return commandResult{exitCode: 1, stderr: "Not Found (HTTP 404)"}, nil
		case "repos/" + testRepo + "/releases?per_page=100&page=1":
			releases := make([]map[string]any, 100)
			for i := range releases {
				releases[i] = map[string]any{"tag_name": "v0.0." + strconv.Itoa(i), "draft": false}
			}
			value = releases
		case "repos/" + testRepo + "/releases?per_page=100&page=2":
			value = []map[string]any{{"tag_name": testTag, "draft": true, "assets": fake.metadata.Assets}}
		default:
			t.Fatalf("unexpected API endpoint: %s", args[1])
		}
		return commandResult{stdout: string(encodeJSON(t, value))}, nil
	}
	if err := h.publish(); err != nil {
		t.Fatal(err)
	}
	if len(callsWithPrefix(fake.calls, "gh", "release", "edit")) != 1 {
		t.Fatal("did not publish the draft from the second page")
	}
}

func TestMutableReleaseIsRejected(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, false, false))
	requireError(t, h.verifyPublished(f.directory), "published immutable release")
	if len(callsWithPrefix(fake.calls, "gh", "release")) != 0 {
		t.Fatal("accepted a mutable release")
	}
}

func TestReleaseAttestationRetriesThenVerifiesEveryAsset(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, false, true))
	fake.verifyErr = 1
	var sleeps []time.Duration
	h.sleep = func(duration time.Duration) { sleeps = append(sleeps, duration) }
	if err := h.verifyPublished(f.directory); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sleeps, []time.Duration{5 * time.Second}) {
		t.Fatalf("unexpected retries: %v", sleeps)
	}
	calls := callsWithPrefix(fake.calls, "gh", "release", "verify-asset")
	names := f.spec.assetNames(true)
	if len(calls) != len(names) {
		t.Fatalf("expected 21 asset checks, got %d", len(calls))
	}
	for i, call := range calls {
		if call[3] != testTag || call[4] != filepath.Join(f.directory, names[i]) {
			t.Fatalf("verified the wrong release asset: %v", call)
		}
	}
}

func TestUnavailableReleaseAttestationExhaustsBoundedRetries(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, metadataFor(f.spec, false, true))
	fake.verifyErr = 6
	var sleeps []time.Duration
	h.sleep = func(duration time.Duration) { sleeps = append(sleeps, duration) }
	requireError(t, h.verifyPublished(f.directory), "after six attempts")
	if len(callsWithPrefix(fake.calls, "gh", "release", "verify")) != 6 || len(sleeps) != 5 {
		t.Fatalf("retry limit was not enforced: calls=%v sleeps=%v", fake.calls, sleeps)
	}
	if len(callsWithPrefix(fake.calls, "gh", "release", "verify-asset")) != 0 {
		t.Fatal("verified assets without a valid release attestation")
	}
}

func TestMissingReleaseAndAPIFailureAreDistinct(t *testing.T) {
	for _, test := range []struct {
		name, stderr string
		startErr     error
		wantErr      bool
		listFailure  bool
	}{
		{name: "missing release", stderr: "Not Found (HTTP 404)"},
		{name: "connection failure", stderr: "connection error", wantErr: true},
		{name: "command unavailable", startErr: errors.New("gh not found"), wantErr: true},
		{name: "listing permission failure", stderr: "Forbidden (HTTP 403)", listFailure: true, wantErr: true},
		{name: "listing not found", stderr: "Not Found (HTTP 404)", listFailure: true, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			h, fake := newTestHelper(t, f, releaseMetadata{})
			fake.respond = func(_ string, args ...string) (commandResult, error) {
				if strings.Contains(args[1], "/releases?") && !test.listFailure {
					return commandResult{stdout: "[]"}, nil
				}
				if strings.Contains(args[1], "/releases/tags/") && test.listFailure {
					return commandResult{exitCode: 1, stderr: "Not Found (HTTP 404)"}, nil
				}
				return commandResult{exitCode: 1, stderr: test.stderr}, test.startErr
			}
			metadata, err := h.getRelease(true)
			if test.wantErr {
				requireError(t, err, "API request failed")
			} else if err != nil || metadata != nil {
				t.Fatalf("expected a missing release, got metadata=%v error=%v", metadata, err)
			}
		})
	}
}

func TestPublicationRequiresAnExistingRelease(t *testing.T) {
	f := newFixture(t)
	h, fake := newTestHelper(t, f, releaseMetadata{})
	fake.respond = func(_ string, args ...string) (commandResult, error) {
		if strings.Contains(args[1], "/releases/tags/") {
			return commandResult{exitCode: 1, stderr: "Not Found (HTTP 404)"}, nil
		}
		return commandResult{stdout: "[]"}, nil
	}
	_, err := h.getRelease(false)
	requireError(t, err, "release "+testTag+" not found")
}

func TestReleaseMetadataRequiresDraftStatus(t *testing.T) {
	f := newFixture(t)
	h, _ := newTestHelper(t, f, releaseMetadata{})
	_, err := h.getRelease(false)
	requireError(t, err, "missing draft status")
}

func TestUnsupportedReleaseExecutableIsRejected(t *testing.T) {
	_, err := executeCommand("sh", "-c", "exit 0")
	requireError(t, err, "unsupported release executable")
}
