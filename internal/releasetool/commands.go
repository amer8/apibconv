package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const workflowPath = ".github/workflows/release.yml"

type commandResult struct {
	stdout, stderr string
	exitCode       int
}

type commandRunner func(name string, args ...string) (commandResult, error)

func executeCommand(name string, args ...string) (commandResult, error) {
	var stdout, stderr bytes.Buffer
	// Executable names are fixed by the callers; arguments are passed directly,
	// without shell expansion.
	command := exec.Command(name, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := commandResult{stdout: stdout.String(), stderr: stderr.String()}
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return result, err
		}
		result.exitCode = exitError.ExitCode()
	}
	return result, nil
}

type helper struct {
	spec         releaseSpec
	repo, commit string
	execute      commandRunner
	sleep        func(time.Duration)
	stdout       io.Writer
	stderr       io.Writer
}

func (h *helper) run(name string, args ...string) error {
	result, err := h.execute(name, args...)
	if err != nil {
		return fmt.Errorf("execute %s: %w", name, err)
	}
	_, _ = io.WriteString(h.stdout, result.stdout)
	_, _ = io.WriteString(h.stderr, result.stderr)
	if result.exitCode != 0 {
		return fmt.Errorf("%s exited with status %d", name, result.exitCode)
	}
	return nil
}

func (h *helper) api(endpoint string, target any, missingOK bool) (bool, error) {
	result, err := h.execute("gh", "api", endpoint)
	if err != nil {
		return false, fmt.Errorf("GitHub API request failed: %w", err)
	}
	if result.exitCode != 0 {
		if missingOK && strings.Contains(result.stderr, "(HTTP 404)") {
			return false, nil
		}
		return false, fmt.Errorf("GitHub API request failed: %s", strings.TrimSpace(result.stderr))
	}
	if err := json.Unmarshal([]byte(result.stdout), target); err != nil {
		return false, fmt.Errorf("decode GitHub API response: %w", err)
	}
	return true, nil
}

type releaseMetadata struct {
	Draft     *bool          `json:"draft"`
	Immutable bool           `json:"immutable"`
	Assets    []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name string `json:"name"`
}

func (h *helper) getRelease(missingOK bool) (*releaseMetadata, error) {
	var metadata releaseMetadata
	found, err := h.api("repos/"+h.repo+"/releases/tags/"+url.PathEscape(h.spec.tag), &metadata, missingOK)
	if err != nil || !found {
		return nil, err
	}
	if metadata.Draft == nil {
		return nil, errors.New("GitHub release metadata is missing draft status")
	}
	return &metadata, nil
}

func (h *helper) guard() error {
	metadata, err := h.getRelease(true)
	if err != nil {
		return err
	}
	if metadata != nil && !*metadata.Draft {
		return fmt.Errorf("%s is already published. Rerun only failed verification or Docker jobs; use a new version for changed assets", h.spec.tag)
	}
	return nil
}

func (h *helper) checkCommit() error {
	var commit struct {
		SHA string `json:"sha"`
	}
	if _, err := h.api("repos/"+h.repo+"/commits/"+url.PathEscape(h.spec.tag), &commit, false); err != nil {
		return err
	}
	if commit.SHA != h.commit {
		return fmt.Errorf("release tag %s no longer points to workflow commit %s", h.spec.tag, h.commit)
	}
	return nil
}

func (h *helper) checkRemoteInventory(metadata *releaseMetadata) error {
	names := make([]string, 0, len(metadata.Assets))
	for _, asset := range metadata.Assets {
		names = append(names, asset.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, h.spec.assetNames(true)) {
		return errors.New("release does not contain the complete expected asset inventory")
	}
	return nil
}

func (h *helper) publish() error {
	if err := h.checkCommit(); err != nil {
		return err
	}
	metadata, err := h.getRelease(false)
	if err != nil {
		return err
	}
	if err := h.checkRemoteInventory(metadata); err != nil {
		return err
	}
	if *metadata.Draft {
		return h.run("gh", "release", "edit", h.spec.tag, "--repo", h.repo, "--draft=false", "--verify-tag")
	}
	// Publication may have succeeded before a connection was lost.
	_, err = fmt.Fprintf(h.stdout, "%s is already published; continuing to release verification.\n", h.spec.tag)
	return err
}

func (h *helper) identity() string {
	return "https://github.com/" + h.repo + "/" + workflowPath + "@refs/tags/" + h.spec.tag
}

func (h *helper) verificationArgs() []string {
	return []string{
		"--repo", h.repo, "--cert-identity", h.identity(),
		"--source-ref", "refs/tags/" + h.spec.tag, "--source-digest", h.commit,
		"--deny-self-hosted-runners",
	}
}

func (h *helper) verify(directory string) error {
	if err := h.spec.checkInventory(directory); err != nil {
		return err
	}
	for _, archive := range h.spec.archives {
		checksum := filepath.Join(directory, archive+"_checksums.txt")
		if err := h.run("cosign", "verify-blob", "--bundle", checksum+".sigstore.json",
			"--certificate-identity", h.identity(),
			"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com", checksum); err != nil {
			return err
		}
	}
	bundle := filepath.Join(directory, h.spec.bundleName)
	policy := h.verificationArgs()
	for _, name := range append(slices.Clone(h.spec.archives), "bom.json") {
		args := append([]string{"attestation", "verify", filepath.Join(directory, name), "--bundle", bundle}, policy...)
		if err := h.run("gh", args...); err != nil {
			return err
		}
	}
	if err := h.verifyTamperRejection(directory, bundle, policy); err != nil {
		return err
	}
	_, err := fmt.Fprintln(h.stdout, "Verified complete release inventory, signatures, provenance, and tamper rejection.")
	return err
}

func (h *helper) verifyTamperRejection(directory, bundle string, policy []string) error {
	temporary, err := os.MkdirTemp("", "apibconv-tampered-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	changed := filepath.Join(temporary, h.spec.archives[0])
	if err := copyFile(filepath.Join(directory, h.spec.archives[0]), changed); err != nil {
		return err
	}
	file, err := os.OpenFile(changed, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString("\ntampered\n")
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	args := append([]string{"attestation", "verify", changed, "--bundle", bundle}, policy...)
	result, err := h.execute("gh", args...)
	if err != nil {
		return fmt.Errorf("execute tamper verification: %w", err)
	}
	if result.exitCode == 0 {
		return errors.New("provenance verification incorrectly accepted a tampered archive")
	}
	return nil
}

func (h *helper) verifyPublished(directory string) error {
	if err := h.checkCommit(); err != nil {
		return err
	}
	metadata, err := h.getRelease(false)
	if err != nil {
		return err
	}
	if err := h.checkRemoteInventory(metadata); err != nil {
		return err
	}
	if *metadata.Draft || !metadata.Immutable {
		return errors.New("expected a published immutable release; enable repository release immutability before publishing")
	}
	// GitHub may need a short time to expose the release attestation.
	for attempt := range 6 {
		if err := h.run("gh", "release", "verify", h.spec.tag, "--repo", h.repo); err == nil {
			break
		}
		if attempt == 5 {
			return errors.New("release attestation could not be verified after six attempts")
		}
		h.sleep(5 * time.Second)
	}
	for _, name := range h.spec.assetNames(true) {
		if err := h.run("gh", "release", "verify-asset", h.spec.tag, filepath.Join(directory, name), "--repo", h.repo); err != nil {
			return err
		}
	}
	return nil
}
