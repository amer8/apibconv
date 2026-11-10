//go:build darwin

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// macOS does not expose ACLs through Go's file mode. Refuse to replace files
// with an ACL rather than publish a new inode without its access restrictions.
func checkOutputACL(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	cmd := exec.Command("/bin/ls", "-lde", absolute) // #nosec G204 -- The executable is fixed and the absolute path is passed as one argument without a shell.
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot inspect output ACL: %w", err)
	}
	if len(out) < 11 {
		return fmt.Errorf("cannot inspect output ACL: unexpected ls output")
	}
	if bytes.Count(out, []byte{'\n'}) != 1 {
		return fmt.Errorf("cannot replace an output file with an ACL or ambiguous filename without preserving its permissions")
	}
	switch out[10] {
	case '+':
		return fmt.Errorf("cannot replace an output file with an ACL without preserving its permissions")
	case ' ', '@':
		return nil
	default:
		return fmt.Errorf("cannot inspect output ACL: unexpected file mode")
	}
}

func checkOutputOwner(original, temp os.FileInfo) error {
	a, okA := original.Sys().(*syscall.Stat_t)
	b, okB := temp.Sys().(*syscall.Stat_t)
	if !okA || !okB || a.Uid != b.Uid || a.Gid != b.Gid {
		return fmt.Errorf("cannot preserve ownership of existing output file")
	}
	return nil
}

func checkTemporaryOutputACL(path string) error { return checkOutputACL(path) }
