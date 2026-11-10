package integration

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/amer8/apibconv/pkg/converter"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format"
)

func TestMigrationPolicy(t *testing.T) {
	openapi2 := "swagger: '2.0'\ninfo: {title: Test, version: '1'}\npaths: {}\n"
	openapi30 := "openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\n"
	openapi31 := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"

	blueprint := "FORMAT: 1A\n# Test\n"
	cases := []struct {
		name, input, target string
		from, to            format.Format
		allowed             bool
	}{
		{"blueprint to 3.1", blueprint, "3.1", format.FormatAPIBlueprint, format.FormatOpenAPI, true},
		{"blueprint to 3.0", blueprint, "3.0", format.FormatAPIBlueprint, format.FormatOpenAPI, false},
		{"swagger to 3.1", openapi2, "3.1", format.FormatOpenAPI, format.FormatOpenAPI, true},
		{"swagger to 3.0", openapi2, "3.0", format.FormatOpenAPI, format.FormatOpenAPI, false},
		{"openapi 3.0 to 3.1", openapi30, "3.1", format.FormatOpenAPI, format.FormatOpenAPI, true},
		{"openapi 3.1 to 3.0", openapi31, "3.0", format.FormatOpenAPI, format.FormatOpenAPI, false},
		{"openapi 3.1 to 2", openapi31, "2.0", format.FormatOpenAPI, format.FormatOpenAPI, false},
		{"swagger same version", openapi2, "2.0", format.FormatOpenAPI, format.FormatOpenAPI, true},
		{"openapi 3.0 same version", openapi30, "3.0", format.FormatOpenAPI, format.FormatOpenAPI, true},
		{"openapi 3.1 same version", openapi31, "3.1", format.FormatOpenAPI, format.FormatOpenAPI, true},

		{"openapi to blueprint", openapi30, "", format.FormatOpenAPI, format.FormatAPIBlueprint, false},
		{"blueprint to blueprint", blueprint, "", format.FormatAPIBlueprint, format.FormatAPIBlueprint, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestConverter(t)
			ctx := converter.WithEncoding(context.Background(), "json")
			if tc.to == format.FormatOpenAPI {
				ctx = converter.WithOpenAPIVersion(ctx, tc.target)
			}
			var out bytes.Buffer
			err := c.Convert(ctx, strings.NewReader(tc.input), &out, tc.from, tc.to)
			if tc.allowed && err != nil {
				t.Fatalf("allowed conversion: %v", err)
			}
			if !tc.allowed && (!errors.Is(err, apierrors.ErrUnsupportedConversion) || out.Len() != 0) {
				t.Fatalf("rejected conversion = %v, output %q", err, out.String())
			}
		})
	}
}
