// Package main is the entry point for the apibconv command-line tool.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/amer8/apibconv/internal/cli"
	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/asyncapi"
	"github.com/amer8/apibconv/pkg/format/openapi"
)

// version is set by GoReleaser and Docker using -X main.version.
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		exitCode := 1 // Default error code
		if errors.Is(err, errors.ErrValidationFailed) {
			exitCode = 2
		} else if errors.Is(err, errors.ErrConversionFailed) {
			exitCode = 3
		} else if _, ok := err.(*errors.ConversionError); ok {
			exitCode = 3 // If it's a wrapped conversion error
		}

		os.Exit(exitCode)
	}
}

func run(args []string) error {
	conv, err := setupConverter()
	if err != nil {
		return err
	}

	info, _ := debug.ReadBuildInfo()
	app := cli.NewApp(conv, resolveVersion(version, info))
	return app.Run(context.Background(), args)
}

func setupConverter() (*converter.Converter, error) {
	conv, err := converter.New()
	if err != nil {
		return nil, err
	}

	// Register parsers
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterParser(apiblueprint.NewParser())
	conv.RegisterParser(asyncapi.NewParser())

	jsonOutput := false
	conv.RegisterWriter(openapi.NewWriter(openapi.WithJSONOutput(jsonOutput)))
	conv.RegisterWriter(apiblueprint.NewWriter())
	conv.RegisterWriter(asyncapi.NewWriter(asyncapi.WithJSONOutput(jsonOutput)))

	return conv, nil
}

// resolveVersion prefers an explicit release version, then embedded module metadata.
func resolveVersion(buildVersion string, info *debug.BuildInfo) string {
	if buildVersion == "" || buildVersion == "dev" {
		buildVersion = "dev"
		if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
			buildVersion = info.Main.Version
		}
	}
	if buildVersion == "dev" {
		return buildVersion
	}
	return "v" + strings.TrimPrefix(buildVersion, "v")
}
