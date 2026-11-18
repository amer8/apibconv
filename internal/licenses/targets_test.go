package licenses

import (
	"go/build"
	"testing"
)

func TestReportTargetSelection(t *testing.T) {
	for _, tc := range []struct {
		goos, goarch, report string
	}{
		{"linux", "amd64", "report_linux_amd64.go"},
		{"linux", "arm64", "report_linux_arm64.go"},
		{"darwin", "amd64", "report_darwin_amd64.go"},
		{"darwin", "arm64", "report_darwin_arm64.go"},
		{"android", "amd64", "report_other.go"},
		{"android", "arm64", "report_other.go"},
		{"ios", "amd64", "report_other.go"},
		{"ios", "arm64", "report_other.go"},
		{"freebsd", "amd64", "report_other.go"},
	} {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			ctx := build.Default
			ctx.GOOS, ctx.GOARCH = tc.goos, tc.goarch
			pkg, err := ctx.ImportDir(".", 0)
			if err != nil {
				t.Fatal(err)
			}
			var reports []string
			for _, file := range pkg.GoFiles {
				if len(file) >= len("report_") && file[:len("report_")] == "report_" {
					reports = append(reports, file)
				}
			}
			if len(reports) != 1 || reports[0] != tc.report {
				t.Fatalf("selected reports %v, want [%s]", reports, tc.report)
			}
		})
	}
}
