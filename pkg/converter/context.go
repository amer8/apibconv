package converter

import "context"

type contextKey string

const (
	contextKeyOpenAPIVersion = contextKey("openAPIVersion")
	contextKeyEncoding       = contextKey("encoding")
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
