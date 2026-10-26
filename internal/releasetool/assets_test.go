package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testTag = "v1.2.3-rc.1"

type fixture struct {
	spec                                  releaseSpec
	root, dist, directory, sbom, subjects string
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func encodeJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustDigest(t *testing.T, path string) string {
	t.Helper()
	digest, err := digestFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func requireError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("expected error containing %q, got %v", message, err)
	}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	spec, err := newReleaseSpec(testTag)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	f := &fixture{spec: spec, root: root, dist: filepath.Join(root, "dist"),
		directory: filepath.Join(root, "assets"), sbom: filepath.Join(root, "bom.json"),
		subjects: filepath.Join(root, "subjects.sha256")}
	if err := os.Mkdir(f.dist, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, f.sbom, []byte(`{"bomFormat":"CycloneDX","specVersion":"1.6"}`))
	for _, archive := range f.spec.archives {
		path := filepath.Join(f.dist, archive)
		writeFile(t, path, []byte("test archive: "+archive))
		writeFile(t, path+"_checksums.txt", []byte(mustDigest(t, path)+"  "+archive+"\n"))
		writeFile(t, path+"_checksums.txt.sigstore.json", []byte("{}"))
	}
	return f
}

func (f *fixture) prepare(t *testing.T) {
	t.Helper()
	if err := f.spec.prepare(f.directory, f.dist, f.sbom, f.subjects); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) bundle(t *testing.T, omit string) []byte {
	t.Helper()
	// These fixtures test coverage and serialization, not cryptographic validity.
	provenance := statement{Type: "https://in-toto.io/Statement/v1", PredicateType: predicateType}
	for _, name := range f.spec.assetNames(false) {
		if name != omit {
			provenance.Subjects = append(provenance.Subjects, subject{
				Name: name, Digest: map[string]string{"sha256": mustDigest(t, filepath.Join(f.directory, name))},
			})
		}
	}
	return encodeJSON(t, map[string]any{"dsseEnvelope": envelope{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     base64.StdEncoding.EncodeToString(encodeJSON(t, provenance)),
		Signatures:  []json.RawMessage{json.RawMessage(`{"sig":"test-signature"}`)},
	}})
}

func (f *fixture) complete(t *testing.T) {
	t.Helper()
	f.prepare(t)
	source := filepath.Join(f.root, "bundle.json")
	writeFile(t, source, f.bundle(t, ""))
	if err := f.spec.exportBundle(f.directory, source); err != nil {
		t.Fatal(err)
	}
}

func mutateBundle(t *testing.T, data []byte, change func(*envelope, *statement)) []byte {
	t.Helper()
	var bundle struct {
		Envelope envelope `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	payload, err := base64.StdEncoding.DecodeString(bundle.Envelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	var provenance statement
	if err := json.Unmarshal(payload, &provenance); err != nil {
		t.Fatal(err)
	}
	change(&bundle.Envelope, &provenance)
	bundle.Envelope.Payload = base64.StdEncoding.EncodeToString(encodeJSON(t, provenance))
	return encodeJSON(t, bundle)
}

func TestCompleteInventoryAndSignedEnvelopeExport(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	if err := f.spec.checkInventory(f.directory); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(f.subjects)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(data), "\n"); lines != 19 {
		t.Fatalf("expected 19 provenance subjects, got %d", lines)
	}
	entries, err := os.ReadDir(f.directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 21 {
		t.Fatalf("expected 21 assets, got %d", len(entries))
	}
}

func TestSingleHashChecksumsAreSupported(t *testing.T) {
	f := newFixture(t)
	for _, archive := range f.spec.archives {
		path := filepath.Join(f.dist, archive)
		writeFile(t, path+"_checksums.txt", []byte(mustDigest(t, path)+"\n"))
	}
	f.complete(t)
	if err := f.spec.checkInventory(f.directory); err != nil {
		t.Fatal(err)
	}
}

func TestMissingPlatformPreventsStaging(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.dist, f.spec.archives[5])); err != nil {
		t.Fatal(err)
	}
	requireError(t, f.spec.prepare(f.directory, f.dist, f.sbom, f.subjects), "missing regular release asset")
}

func TestModifiedArchiveFailsChecksumValidation(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	writeFile(t, filepath.Join(f.directory, f.spec.archives[0]), []byte("tampered"))
	requireError(t, f.spec.checkInventory(f.directory), "checksum mismatch")
}

func TestChangedArchiveAndChecksumStillFailProvenance(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	path := filepath.Join(f.directory, f.spec.archives[0])
	writeFile(t, path, []byte("tampered"))
	writeFile(t, path+"_checksums.txt", []byte(mustDigest(t, path)))
	requireError(t, f.spec.checkInventory(f.directory), "provenance subjects must match")
}

func TestUncoveredArchiveRejectsBundle(t *testing.T) {
	f := newFixture(t)
	f.prepare(t)
	_, err := f.spec.checkBundle(f.bundle(t, f.spec.archives[5]), f.directory)
	requireError(t, err, "provenance subjects must match")
}

func TestDuplicateSubjectRejectsBundle(t *testing.T) {
	f := newFixture(t)
	f.prepare(t)
	bundle := mutateBundle(t, f.bundle(t, ""), func(_ *envelope, provenance *statement) {
		provenance.Subjects = append(provenance.Subjects, provenance.Subjects[0])
	})
	_, err := f.spec.checkBundle(bundle, f.directory)
	requireError(t, err, "duplicate provenance subject")
}

func TestUnsignedOrWrongPredicateBundleIsRejected(t *testing.T) {
	for _, test := range []struct {
		name, message string
		change        func(*envelope, *statement)
	}{
		{"unsigned", "signed in-toto envelope", func(signed *envelope, _ *statement) { signed.Signatures = nil }},
		{"wrong predicate", "SLSA v1", func(_ *envelope, provenance *statement) {
			provenance.PredicateType = "https://example.com/other-predicate"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			f.prepare(t)
			_, err := f.spec.checkBundle(mutateBundle(t, f.bundle(t, ""), test.change), f.directory)
			requireError(t, err, test.message)
		})
	}
}

func TestExportedEnvelopeCannotBeReplaced(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	writeFile(t, filepath.Join(f.directory, f.spec.envelopeName), []byte("{}\n"))
	requireError(t, f.spec.checkInventory(f.directory), "envelope differs")
}

func TestExtraAssetIsRejected(t *testing.T) {
	f := newFixture(t)
	f.complete(t)
	writeFile(t, filepath.Join(f.directory, "unexpected.txt"), []byte("unexpected"))
	requireError(t, f.spec.checkInventory(f.directory), "inventory mismatch")
}

func TestSignedEnvelopeFieldsArePreserved(t *testing.T) {
	f := newFixture(t)
	f.prepare(t)
	var bundle map[string]any
	if err := json.Unmarshal(f.bundle(t, ""), &bundle); err != nil {
		t.Fatal(err)
	}
	signed := bundle["dsseEnvelope"].(map[string]any)
	signed["futureField"] = map[string]any{"retained": true}
	source := filepath.Join(f.root, "bundle.json")
	writeFile(t, source, encodeJSON(t, bundle))
	if err := f.spec.exportBundle(f.directory, source); err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := readJSON(filepath.Join(f.directory, f.spec.envelopeName), &exported); err != nil {
		t.Fatal(err)
	}
	if _, exists := exported["futureField"]; !exists {
		t.Fatal("export dropped an envelope field")
	}
	if err := f.spec.checkInventory(f.directory); err != nil {
		t.Fatal(err)
	}
}
