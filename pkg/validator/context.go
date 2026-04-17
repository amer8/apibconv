package validator

import (
	"context"

	"github.com/amer8/apibconv/pkg/format"
)

type contextKey string

const validationFormatKey contextKey = "validation-format"

// WithFormat annotates a validation context with the specification format.
func WithFormat(ctx context.Context, specFormat format.Format) context.Context {
	return context.WithValue(ctx, validationFormatKey, specFormat)
}

func formatFromContext(ctx context.Context) format.Format {
	if ctx == nil {
		return format.FormatUnknown
	}

	specFormat, ok := ctx.Value(validationFormatKey).(format.Format)
	if !ok {
		return format.FormatUnknown
	}

	return specFormat
}
