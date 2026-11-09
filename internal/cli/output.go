package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicOutput keeps failed conversions from modifying an existing destination.
type atomicOutput struct {
	file     *os.File
	path     string
	original os.FileInfo
}

func newAtomicOutput(inputPath, outputPath string) (*atomicOutput, error) {
	info, err := os.Lstat(outputPath)
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
	}
	file, err := os.CreateTemp(filepath.Dir(outputPath), ".apibconv-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary output file: %w", err)
	}
	destination := &atomicOutput{file: file, path: outputPath, original: info}
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
		if o.original == nil || !os.SameFile(info, o.original) {
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
	if err := os.Rename(o.file.Name(), o.path); err != nil {
		return fmt.Errorf("failed to replace output: %w", err)
	}
	return nil
}
