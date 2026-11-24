// Package fs provides filesystem and I/O utility functions.
package fs

import (
	"context"
	"fmt"
	"io"
	"math"

	apierrors "github.com/amer8/apibconv/pkg/errors"
)

// DefaultMaxInputBytes bounds memory consumed before document parsing begins.
const DefaultMaxInputBytes int64 = 16 << 20

type limitKey struct{}

// WithMaxInputBytes sets the byte budget for an input reader.
func WithMaxInputBytes(ctx context.Context, limit int64) context.Context {
	return context.WithValue(ctx, limitKey{}, limit)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// ReadAll checks cancellation between reads and rejects inputs larger than the
// budget. Reading one extra byte distinguishes an exact fit from truncation.
func ReadAll(ctx context.Context, r io.Reader) ([]byte, error) {
	limit, ok := ctx.Value(limitKey{}).(int64)
	if !ok {
		limit = DefaultMaxInputBytes
	}
	if limit <= 0 {
		return nil, fmt.Errorf("maximum input bytes must be positive")
	}
	if limit < math.MaxInt64 {
		r = io.LimitReader(r, limit+1)
	}
	data, err := io.ReadAll(contextReader{ctx, r})
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: maximum %d bytes", apierrors.ErrInputTooLarge, limit)
	}
	return data, err
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
