//go:build linux

package cli

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func checkOutputACL(path string) error {
	_, err := syscall.Getxattr(path, "system.posix_acl_access", nil)
	if err == nil {
		return fmt.Errorf("cannot replace an output file with an ACL without preserving its permissions")
	}
	if errors.Is(err, syscall.ENODATA) {
		return nil
	}
	return fmt.Errorf("cannot inspect output ACL: %w", err)
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
