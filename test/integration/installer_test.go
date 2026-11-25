package integration

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerRequiresAuthenticatedArchive(t *testing.T) {
	var platform, runnerOS string
	switch runtime.GOOS {
	case "darwin":
		platform, runnerOS = "Darwin", "macOS"
	case "linux":
		platform, runnerOS = "Linux", "Linux"
	default:
		t.Skip("installer supports macOS and Linux")
	}
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	assets := t.TempDir()
	const binary = "installer test fixture"
	write(filepath.Join(assets, "apibconv"), binary)
	archiveName := "apibconv_1.2.3_" + platform + "_x86_64.tar.gz"
	archive := filepath.Join(assets, archiveName)
	if out, err := exec.Command("tar", "-czf", archive, "-C", assets, "apibconv").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, out)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	checksums := archive + "_checksums.txt"
	write(checksums, digest+"  "+archiveName+"\n")
	// Stub only the external verifier. The real script downloads, hashes,
	// extracts and installs the local archive through its normal entry point.
	toolsDir := t.TempDir()
	write(filepath.Join(toolsDir, "cosign"), `#!/bin/sh
set -eu
test "$1" = verify-blob
test "$2" = --certificate-identity
test "$3" = https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/v1.2.3
test "$4" = --certificate-oidc-issuer
test "$5" = https://token.actions.githubusercontent.com
test "$6" = --bundle
test "$(cat "$7")" = "test-verified"
test -f "$8"
`)
	for _, tc := range []struct {
		name, pin, bundle string
		success           bool
	}{
		{"missing bundle", "", "", false},
		{"wrong legacy pin", strings.Repeat("0", 64), "", false},
		{"pinned legacy", strings.ToUpper(digest), "", true},
		{"verified bundle", "", "test-verified", true},
		{"invalid bundle with valid pin", digest, "invalid", false},
		{"signed archive with wrong pin", strings.Repeat("0", 64), "test-verified", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := checksums + ".sigstore.json"
			if tc.bundle != "" {
				if err := os.WriteFile(bundle, []byte(tc.bundle), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Remove(bundle); err != nil {
						t.Error(err)
					}
				})
			}
			tmp := t.TempDir()
			pathFile, outputs := filepath.Join(tmp, "path"), filepath.Join(tmp, "outputs")
			cmd := exec.Command("sh", "../../scripts/install-apibconv-release.sh")
			cmd.Env = append(os.Environ(), "PATH="+toolsDir+":"+os.Getenv("PATH"),
				"RUNNER_TEMP="+tmp, "RUNNER_OS="+runnerOS, "RUNNER_ARCH=X64", "APIBCONV_VERSION=v1.2.3",
				"APIBCONV_ASSET_BASE_URL=file://"+assets, "APIBCONV_ARCHIVE_SHA256="+tc.pin,
				"GITHUB_PATH="+pathFile, "GITHUB_OUTPUT="+outputs)
			log, err := cmd.CombinedOutput()
			if (err == nil) != tc.success {
				t.Fatalf("installer: %v\n%s", err, log)
			}
			if !tc.success {
				for _, path := range []string{pathFile, outputs, filepath.Join(tmp, "apibconv-1.2.3-"+platform+"-x86_64", "extract", "apibconv")} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("%s after failure: %v", path, err)
					}
				}
				return
			}
			path, err := os.ReadFile(pathFile)
			if err != nil {
				t.Fatal(err)
			}
			installed, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(path)), "apibconv"))
			if err != nil || string(installed) != binary {
				t.Fatalf("installed bytes: %q, %v", installed, err)
			}
			result, err := os.ReadFile(outputs)
			if err != nil || !strings.Contains(string(result), fmt.Sprintf("signature-verified=%t\n", tc.bundle != "")) {
				t.Fatalf("outputs: %s, %v", result, err)
			}
		})
	}
}
