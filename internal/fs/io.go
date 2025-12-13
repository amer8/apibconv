// Package fs provides filesystem and I/O utility functions.
package fs

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// ReadAll reads from r until an error or EOF and returns the data it read.
// It checks ctx.Done() before each read operation to allow cancellation.
func ReadAll(ctx context.Context, r io.Reader) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b := make([]byte, 0, 512)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}

		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

// ReadFile reads the entire file content into a byte slice.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file, creating any necessary parent directories.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// CopyWithProgress copies from src to dst, invoking onProgress with total bytes written.
func CopyWithProgress(dst io.Writer, src io.Reader, onProgress func(int64)) (int64, error) {
	if onProgress == nil {
		return io.Copy(dst, src)
	}
	pr := NewProgressReader(src, onProgress)
	return io.Copy(dst, pr)
}

// NewProgressReader creates a new reader that reports progress.
func NewProgressReader(r io.Reader, onProgress func(int64)) io.Reader {
	return &ProgressReader{r: r, onProgress: onProgress}
}

// ProgressReader wraps an io.Reader to report progress.
type ProgressReader struct {
	r          io.Reader
	onProgress func(int64)
	total      int64
}

// Read reads data and reports the total bytes read so far.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.total += int64(n)
		pr.onProgress(pr.total)
	}
	return n, err
}
