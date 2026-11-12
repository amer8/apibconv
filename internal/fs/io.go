// Package fs provides filesystem and I/O utility functions.
package fs

import (
	"context"
	"io"
)

// ReadAll reads from r until an error or EOF and returns the data it read.
// It checks the context before each read; it cannot interrupt a blocked read.
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

// NewProgressReader creates a reader that reports progress. onProgress must not be nil.
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
