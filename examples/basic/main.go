// Package main demonstrates basic usage of the apibconv library.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/apiblueprint"
	"github.com/amer8/apibconv/pkg/format/openapi"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	conv, err := converter.New(converter.WithValidation(true, true))
	if err != nil {
		return fmt.Errorf("create converter: %w", err)
	}
	conv.RegisterParser(apiblueprint.NewParser())
	conv.RegisterWriter(openapi.NewWriter())

	input, err := os.Open("docs/demo/api.apib")
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer func() { _ = input.Close() }()

	output, err := os.Create("petstore.json")
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	convertErr := conv.Convert(context.Background(), input, output, format.FormatAPIBlueprint, format.FormatOpenAPI)
	closeErr := output.Close()
	if convertErr != nil {
		return fmt.Errorf("convert: %w", convertErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close output: %w", closeErr)
	}

	fmt.Println("Created petstore.json")
	return nil
}
