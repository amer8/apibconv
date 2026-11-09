//go:build !darwin && !linux

package cli

import (
	"fmt"
	"os"
)

func checkOutputACL(path string) error {
	return fmt.Errorf("cannot safely replace an existing output file: ACL preservation is unsupported on this platform")
}

func checkOutputOwner(original, temp os.FileInfo) error {
	return fmt.Errorf("cannot preserve ownership of existing output file")
}

func checkTemporaryOutputACL(path string) error { return nil }
