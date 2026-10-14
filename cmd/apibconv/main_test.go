package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	for _, tc := range []struct {
		name, buildVersion, moduleVersion, want string
		noBuildInfo                             bool
	}{
		{name: "release", buildVersion: "0.9.0", moduleVersion: "v0.8.0", want: "v0.9.0"},
		{name: "prefixed release", buildVersion: "v0.9.0", want: "v0.9.0"},
		{name: "prerelease", buildVersion: "0.9.0-rc.1", want: "v0.9.0-rc.1"},
		{name: "go install", buildVersion: "dev", moduleVersion: "v0.9.0", want: "v0.9.0"},
		{name: "checkout revision", buildVersion: "dev", moduleVersion: "v0.9.1-0.20261014152222-dd9bd267586e+dirty", want: "v0.9.1-0.20261014152222-dd9bd267586e+dirty"},
		{name: "empty linker version", moduleVersion: "v0.9.0", want: "v0.9.0"},
		{name: "checkout without VCS metadata", buildVersion: "dev", moduleVersion: "(devel)", want: "dev"},
		{name: "empty metadata", want: "dev"},
		{name: "missing metadata", noBuildInfo: true, want: "dev"},
		{name: "release without metadata", buildVersion: "0.9.0", noBuildInfo: true, want: "v0.9.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var info *debug.BuildInfo
			if !tc.noBuildInfo {
				info = &debug.BuildInfo{Main: debug.Module{Version: tc.moduleVersion}}
			}
			if got := resolveVersion(tc.buildVersion, info); got != tc.want {
				t.Fatalf("version = %q, want %q", got, tc.want)
			}
		})
	}
}
