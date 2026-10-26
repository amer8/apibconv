package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

const predicateType = "https://slsa.dev/provenance/v1"

var versionTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

type releaseSpec struct {
	tag          string
	archives     []string
	bundleName   string
	envelopeName string
}

func newReleaseSpec(tag string) (releaseSpec, error) {
	if !versionTag.MatchString(tag) {
		return releaseSpec{}, fmt.Errorf("expected a semantic version tag, got %q", tag)
	}
	prefix := "apibconv_" + strings.TrimPrefix(tag, "v")
	spec := releaseSpec{
		tag:          tag,
		archives:     make([]string, 0, 6),
		bundleName:   prefix + "_provenance.sigstore.json",
		envelopeName: prefix + "_provenance.intoto.jsonl",
	}
	for _, platform := range []struct{ name, extension string }{
		{"Darwin", "tar.gz"}, {"Linux", "tar.gz"}, {"Windows", "zip"},
	} {
		for _, arch := range []string{"x86_64", "arm64"} {
			spec.archives = append(spec.archives, prefix+"_"+platform.name+"_"+arch+"."+platform.extension)
		}
	}
	return spec, nil
}

func (s releaseSpec) assetNames(provenance bool) []string {
	names := make([]string, 0, 21)
	names = append(names, "bom.json")
	for _, archive := range s.archives {
		names = append(names, archive, archive+"_checksums.txt", archive+"_checksums.txt.sigstore.json")
	}
	if provenance {
		names = append(names, s.bundleName, s.envelopeName)
	}
	slices.Sort(names)
	return names
}

func requireFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("missing regular release asset %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("missing regular release asset: %s", path)
	}
	return nil
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	return errors.Join(copyErr, output.Close())
}

func (s releaseSpec) assetDigests(directory string) (map[string]string, error) {
	digests := make(map[string]string, 19)
	for _, name := range s.assetNames(false) {
		digest, err := digestFile(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}
		digests[name] = digest
	}
	return digests, nil
}

func (s releaseSpec) checkChecksums(directory string) error {
	for _, archive := range s.archives {
		data, err := os.ReadFile(filepath.Join(directory, archive+"_checksums.txt"))
		if err != nil {
			return err
		}
		fields := strings.Fields(string(data))
		if len(fields) != 1 && len(fields) != 2 {
			return fmt.Errorf("invalid checksum file for %s", archive)
		}
		if decoded, err := hex.DecodeString(fields[0]); err != nil || len(decoded) != sha256.Size {
			return fmt.Errorf("invalid checksum file for %s", archive)
		}
		if len(fields) == 2 && fields[1] != archive && fields[1] != "*"+archive {
			return fmt.Errorf("checksum names a different archive: %s", archive)
		}
		digest, err := digestFile(filepath.Join(directory, archive))
		if err != nil {
			return err
		}
		if !strings.EqualFold(fields[0], digest) {
			return fmt.Errorf("checksum mismatch for %s", archive)
		}
	}
	var sbom struct {
		Format string `json:"bomFormat"`
	}
	if err := readJSON(filepath.Join(directory, "bom.json"), &sbom); err != nil {
		return err
	}
	if sbom.Format != "CycloneDX" {
		return errors.New("bom.json must be a CycloneDX SBOM")
	}
	return nil
}

func (s releaseSpec) prepare(directory, dist, sbom, subjects string) error {
	if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(directory, 0o755); err != nil {
		return err
	}
	for _, name := range s.assetNames(false) {
		source := filepath.Join(dist, name)
		if name == "bom.json" {
			source = sbom
		}
		if err := requireFile(source); err != nil {
			return err
		}
		if err := copyFile(source, filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	if err := s.checkChecksums(directory); err != nil {
		return err
	}
	digests, err := s.assetDigests(directory)
	if err != nil {
		return err
	}
	var checksums strings.Builder
	for _, name := range s.assetNames(false) {
		fmt.Fprintf(&checksums, "%s  %s\n", digests[name], name)
	}
	if err := os.MkdirAll(filepath.Dir(subjects), 0o755); err != nil {
		return err
	}
	return os.WriteFile(subjects, []byte(checksums.String()), 0o644)
}

type envelope struct {
	PayloadType string            `json:"payloadType"`
	Payload     string            `json:"payload"`
	Signatures  []json.RawMessage `json:"signatures"`
}

type subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type statement struct {
	Type          string    `json:"_type"`
	PredicateType string    `json:"predicateType"`
	Subjects      []subject `json:"subject"`
}

func (s releaseSpec) checkBundle(data []byte, directory string) (json.RawMessage, error) {
	var bundle struct {
		Envelope json.RawMessage `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("decode provenance bundle: %w", err)
	}
	var signed envelope
	if err := json.Unmarshal(bundle.Envelope, &signed); err != nil {
		return nil, fmt.Errorf("decode in-toto envelope: %w", err)
	}
	if signed.PayloadType != "application/vnd.in-toto+json" || len(signed.Signatures) == 0 {
		return nil, errors.New("provenance bundle must contain a signed in-toto envelope")
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(signed.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode provenance payload: %w", err)
	}
	var provenance statement
	if err := json.Unmarshal(payload, &provenance); err != nil {
		return nil, fmt.Errorf("decode provenance statement: %w", err)
	}
	if provenance.Type != "https://in-toto.io/Statement/v1" || provenance.PredicateType != predicateType {
		return nil, errors.New("expected an in-toto statement with SLSA v1 build provenance")
	}
	expected, err := s.assetDigests(directory)
	if err != nil {
		return nil, err
	}
	actual := make(map[string]string, len(provenance.Subjects))
	for _, artifact := range provenance.Subjects {
		if _, exists := actual[artifact.Name]; exists {
			return nil, fmt.Errorf("duplicate provenance subject: %s", artifact.Name)
		}
		actual[artifact.Name] = artifact.Digest["sha256"]
	}
	if !maps.Equal(actual, expected) {
		return nil, errors.New("provenance subjects must match every release asset and its SHA-256 digest")
	}
	return bundle.Envelope, nil
}

func (s releaseSpec) exportBundle(directory, source string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	signed, err := s.checkBundle(data, directory)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, s.bundleName), data, 0o644); err != nil {
		return err
	}
	// Compact the original envelope without changing its signed payload or
	// signatures. The native bundle retains certificates and transparency evidence.
	var line bytes.Buffer
	if err := json.Compact(&line, signed); err != nil {
		return err
	}
	line.WriteByte('\n')
	return os.WriteFile(filepath.Join(directory, s.envelopeName), line.Bytes(), 0o644)
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (s releaseSpec) checkInventory(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	actual := make([]string, 0, len(entries))
	for _, entry := range entries {
		actual = append(actual, entry.Name())
	}
	expected := s.assetNames(true)
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("release inventory mismatch: expected %v; found %v", expected, actual)
	}
	for _, name := range expected {
		if err := requireFile(filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	if err := s.checkChecksums(directory); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(directory, s.bundleName))
	if err != nil {
		return err
	}
	signed, err := s.checkBundle(data, directory)
	if err != nil {
		return err
	}
	exported, err := os.ReadFile(filepath.Join(directory, s.envelopeName))
	if err != nil {
		return err
	}
	var originalJSON, exportedJSON any
	if err := json.Unmarshal(signed, &originalJSON); err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(exported), "\n"), "\n")
	if len(lines) != 1 || json.Unmarshal(exported, &exportedJSON) != nil || !reflect.DeepEqual(originalJSON, exportedJSON) {
		return errors.New("exported in-toto envelope differs from the verification bundle")
	}
	return nil
}
