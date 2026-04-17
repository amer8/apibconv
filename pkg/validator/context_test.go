package validator

import (
	"context"
	"testing"

	"github.com/amer8/apibconv/pkg/format"
)

func TestFormatFromContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		if got := formatFromContext(nil); got != format.FormatUnknown {
			t.Fatalf("formatFromContext(nil) = %q, want %q", got, format.FormatUnknown)
		}
	})

	t.Run("missing format", func(t *testing.T) {
		if got := formatFromContext(context.Background()); got != format.FormatUnknown {
			t.Fatalf("formatFromContext(background) = %q, want %q", got, format.FormatUnknown)
		}
	})

	t.Run("stored format", func(t *testing.T) {
		ctx := WithFormat(context.Background(), format.FormatAsyncAPI)
		if got := formatFromContext(ctx); got != format.FormatAsyncAPI {
			t.Fatalf("formatFromContext(with format) = %q, want %q", got, format.FormatAsyncAPI)
		}
	})
}
