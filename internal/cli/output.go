package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// atomicOutput keeps failed conversions from modifying an existing destination.
type atomicOutput struct {
	file     *os.File
	path     string
	original os.FileInfo
	digest   [sha256.Size]byte
}

// outputDigest reads a stable snapshot and verifies that the path still names
// the same regular file. This catches in-place changes as well as replacement.
func outputDigest(path string, expected os.FileInfo) ([sha256.Size]byte, error) {
	var sum [sha256.Size]byte
	f, err := os.Open(path) // #nosec G304 -- The CLI intentionally reads the user-selected output file to detect concurrent changes.
	if err != nil {
		return sum, fmt.Errorf("cannot read existing output for change detection: %w", err)
	}
	defer func() { _ = f.Close() }()
	before, err := f.Stat()
	if err != nil || !sameOutputState(expected, before) {
		return sum, fmt.Errorf("output file changed during conversion")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return sum, fmt.Errorf("cannot read existing output for change detection: %w", err)
	}
	after, err := f.Stat()
	if err != nil || !sameOutputState(before, after) {
		return sum, fmt.Errorf("output file changed during conversion")
	}
	current, err := os.Lstat(path)
	if err != nil || !sameOutputState(after, current) {
		return sum, fmt.Errorf("output file changed during conversion")
	}
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}

func sameOutputState(a, b os.FileInfo) bool {
	return a != nil && b != nil && b.Mode().IsRegular() && os.SameFile(a, b) &&
		a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) && a.Mode() == b.Mode()
}

func newAtomicOutput(inputPath, outputPath string) (*atomicOutput, error) {
	info, err := os.Lstat(outputPath)
	var digest [sha256.Size]byte
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect output file: %w", err)
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("output must be a regular file, not a directory or symbolic link")
		}
		if inputPath != "" && inputPath != "-" {
			inputInfo, statErr := os.Stat(inputPath)
			if statErr != nil {
				return nil, fmt.Errorf("failed to inspect input file: %w", statErr)
			}
			if os.SameFile(inputInfo, info) {
				return nil, fmt.Errorf("input and output refer to the same file")
			}
		}
		if err := checkOutputACL(outputPath); err != nil {
			return nil, err
		}
		digest, err = outputDigest(outputPath, info)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.CreateTemp(filepath.Dir(outputPath), ".apibconv-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary output file: %w", err)
	}
	destination := &atomicOutput{file: file, path: outputPath, original: info, digest: digest}
	if err := checkTemporaryOutputACL(file.Name()); err != nil {
		destination.cleanup()
		return nil, fmt.Errorf("temporary output permissions are unsafe: %w", err)
	}
	return destination, nil
}

func (o *atomicOutput) cleanup() {
	_ = o.file.Close()
	_ = os.Remove(o.file.Name())
}

func (o *atomicOutput) commit() error {
	if info, err := os.Lstat(o.path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output must be a regular file")
		}
		if !sameOutputState(o.original, info) {
			return fmt.Errorf("output file changed during conversion")
		}
		if err := checkOutputACL(o.path); err != nil {
			return err
		}
		tempInfo, err := o.file.Stat()
		if err != nil {
			return fmt.Errorf("failed to inspect temporary output: %w", err)
		}
		if err := checkOutputOwner(info, tempInfo); err != nil {
			return err
		}
		if err := o.file.Chmod(info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to preserve output permissions: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect output file: %w", err)
	} else if o.original != nil {
		return fmt.Errorf("output file changed during conversion")
	}
	if err := o.file.Sync(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	if err := o.file.Close(); err != nil {
		return fmt.Errorf("failed to close output: %w", err)
	}
	if o.original != nil {
		digest, err := outputDigest(o.path, o.original)
		if err != nil {
			return err
		}
		if digest != o.digest {
			return fmt.Errorf("output file changed during conversion")
		}
	} else if _, err := os.Lstat(o.path); err == nil {
		return fmt.Errorf("output file changed during conversion")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect output file: %w", err)
	}
	if err := os.Rename(o.file.Name(), o.path); err != nil {
		return fmt.Errorf("failed to replace output: %w", err)
	}
	return nil
}
