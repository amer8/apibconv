package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicOutput keeps failed conversions from modifying an existing destination.
type atomicOutput struct {
	file *os.File
	path string
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
	}
	file, err := os.CreateTemp(filepath.Dir(outputPath), ".apibconv-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary output file: %w", err)
	}
	destination := &atomicOutput{file: file, path: outputPath}
	if info != nil {
		if err := file.Chmod(info.Mode().Perm()); err != nil {
			destination.cleanup()
			return nil, fmt.Errorf("failed to preserve output permissions: %w", err)
		}
	}
	return destination, nil
}

func (o *atomicOutput) cleanup() {
	_ = o.file.Close()
	_ = os.Remove(o.file.Name())
}

func (o *atomicOutput) commit() error {
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
