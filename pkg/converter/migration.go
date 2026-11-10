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
		return f == format.FormatOpenAPI || f == format.FormatAPIBlueprint
	}
	if !known(from) || !known(to) {
		return nil
	}
	targetVersion := writerVersion
	if to == format.FormatOpenAPI {
		if v := OpenAPIVersionFromContext(ctx); v != "" {
			targetVersion = v
		}
	}
	unsupported := func() error {
		return fmt.Errorf("%w: %s %s to %s %s; supported conversions are API Blueprint to OpenAPI 3.1, OpenAPI 2.0/3.0 to 3.1, and same-version OpenAPI", apierrors.ErrUnsupportedConversion, from, sourceVersion, to, targetVersion)
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
