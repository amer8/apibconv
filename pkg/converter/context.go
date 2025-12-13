package converter

import "context"

type contextKey string

const (
	contextKeyOpenAPIVersion  = contextKey("openAPIVersion")
	contextKeyAsyncAPIVersion = contextKey("asyncAPIVersion")
	contextKeyProtocol        = contextKey("protocol")
	contextKeyEncoding        = contextKey("encoding")
)

// WithEncoding adds the target encoding (json/yaml) to the context
func WithEncoding(ctx context.Context, encoding string) context.Context {
	return context.WithValue(ctx, contextKeyEncoding, encoding)
}

// GetEncoding retrieves the target encoding from the context
func GetEncoding(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyEncoding).(string); ok {
		return v
	}
	return ""
}

// WithProtocol adds the target protocol to the context
func WithProtocol(ctx context.Context, protocol string) context.Context {
	return context.WithValue(ctx, contextKeyProtocol, protocol)
}

// GetProtocol retrieves the target protocol from the context
func GetProtocol(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyProtocol).(string); ok {
		return v
	}
	return ""
}

// WithOpenAPIVersion adds the target OpenAPI version to the context
func WithOpenAPIVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, contextKeyOpenAPIVersion, version)
}

// OpenAPIVersionFromContext retrieves the target OpenAPI version from the context
func OpenAPIVersionFromContext(ctx context.Context) string {
	if version, ok := ctx.Value(contextKeyOpenAPIVersion).(string); ok {
		return version
	}
	return ""
}

// WithAsyncAPIVersion adds the target AsyncAPI version to the context
func WithAsyncAPIVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, contextKeyAsyncAPIVersion, version)
}

// AsyncAPIVersionFromContext retrieves the target AsyncAPI version from the context
func AsyncAPIVersionFromContext(ctx context.Context) string {
	if version, ok := ctx.Value(contextKeyAsyncAPIVersion).(string); ok {
		return version
	}
	return ""
}
