//go:build darwin

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestP1RefusesToReplaceACLProtectedOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "protected")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("/bin/chmod", "+a", "everyone deny read", path).CombinedOutput(); err != nil {
		t.Skipf("ACL unavailable: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("/bin/chmod", "-N", path).Run() })
	if _, err := newAtomicOutput("", path); err == nil || !strings.Contains(err.Error(), "ACL") {
		t.Fatalf("got %v, want ACL error", err)
	}
}
