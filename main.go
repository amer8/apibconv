// Package main provides a command-line interface for converting between OpenAPI 3.0/3.1, AsyncAPI 2.6/3.0, and API Blueprint specifications.
package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/amer8/apibconv/internal/cli"
)

// version is set by GoReleaser at build time via -ldflags
var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = strings.TrimPrefix(info.Main.Version, "v")
			}
		}
	}
	os.Exit(cli.Run(version))
}
