// Package main provides a command-line interface for converting between OpenAPI 3.0/3.1, AsyncAPI 2.6/3.0, and API Blueprint specifications.
package main

import (
	"os"

	"github.com/amer8/apibconv/internal/cli"
)

// version is set by GoReleaser at build time via -ldflags
var version = "dev"

func main() {
	os.Exit(cli.Run(version))
}
