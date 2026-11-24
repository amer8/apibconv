package converter

import (
	"context"

	"github.com/amer8/apibconv/internal/fs"
)

type contextKey string

const (
	contextKeyOpenAPIVersion = contextKey("openAPIVersion")
	contextKeyEncoding       = contextKey("encoding")
	contextKeyAPIInfoVersion = contextKey("apiInfoVersion")
)

// WithEncoding adds the target encoding (json/yaml) to the context.
func WithEncoding(ctx context.Context, encoding string) context.Context {
	return context.WithValue(ctx, contextKeyEncoding, encoding)
}

// GetEncoding retrieves the target encoding from the context.
func GetEncoding(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyEncoding).(string); ok {
		return v
	}
	return ""
}

// WithOpenAPIVersion adds the target OpenAPI version to the context.
func WithOpenAPIVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, contextKeyOpenAPIVersion, version)
}

// OpenAPIVersionFromContext retrieves the target OpenAPI version from the context.
func OpenAPIVersionFromContext(ctx context.Context) string {
	if version, ok := ctx.Value(contextKeyOpenAPIVersion).(string); ok {
		return version
	}
	return ""
}

// WithAPIInfoVersion overrides the API's own info.version in generated OpenAPI.
func WithAPIInfoVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, contextKeyAPIInfoVersion, version)
}

// APIInfoVersionFromContext retrieves the optional API version override.
func APIInfoVersionFromContext(ctx context.Context) string {
	version, _ := ctx.Value(contextKeyAPIInfoVersion).(string)
	return version
}

// WithMaxInputBytes sets the positive input byte limit for built-in parsing,
// format detection and validation. The default is 16 MiB.
func WithMaxInputBytes(ctx context.Context, limit int64) context.Context {
	return fs.WithMaxInputBytes(ctx, limit)
}
