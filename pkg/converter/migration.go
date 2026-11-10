package converter

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
)

// checkMigration limits built-in format conversions while leaving custom plugin
// formats to their own parsers and writers.
func checkMigration(ctx context.Context, from, to format.Format, sourceVersion, writerVersion string) error {
	known := func(f format.Format) bool {
		return f == format.FormatOpenAPI || f == format.FormatAsyncAPI || f == format.FormatAPIBlueprint
	}
	if !known(from) || !known(to) {
		return nil
	}
	targetVersion := writerVersion
	switch to {
	case format.FormatOpenAPI:
		if v := OpenAPIVersionFromContext(ctx); v != "" {
			targetVersion = v
		}
	case format.FormatAsyncAPI:
		if v := AsyncAPIVersionFromContext(ctx); v != "" {
			targetVersion = v
		}
	}
	unsupported := func() error {
		return fmt.Errorf("%w: %s %s to %s %s; supported conversions are API Blueprint to OpenAPI 3.1, OpenAPI 2.0/3.0 to 3.1, AsyncAPI 2.x to 3.0, and same-version OpenAPI or AsyncAPI", apierrors.ErrUnsupportedConversion, from, sourceVersion, to, targetVersion)
	}
	target := majorMinor(targetVersion)
	source := majorMinor(sourceVersion)
	switch {
	case from == format.FormatAPIBlueprint && to == format.FormatOpenAPI:
		if target == "3.1" {
			return nil
		}
	case from == format.FormatOpenAPI && to == format.FormatOpenAPI:
		if (source == "2.0" || source == "3.0" || source == "3.1") && (source == target || target == "3.1") {
			return nil
		}
	case from == format.FormatAsyncAPI && to == format.FormatAsyncAPI:
		if (source == "3.0" || async2(source)) && (source == target || target == "3.0") {
			return nil
		}
	}
	return unsupported()
}

func majorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return ""
	}
	for _, part := range parts {
		if part == "" {
			return ""
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return ""
		}
	}
	return parts[0] + "." + parts[1]
}

func async2(version string) bool {
	switch version {
	case "2.0", "2.1", "2.2", "2.3", "2.4", "2.5", "2.6":
		return true
	default:
		return false
	}
}
