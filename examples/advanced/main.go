// Package main demonstrates a custom transformation with apibconv.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	transform := func(api *model.API) error {
		api.Info.Title = "Transformed API Title"
		api.Info.Description = "This description was added by a transformer."
		return nil
	}
	conv, err := converter.New(
		converter.WithValidation(true, false),
		converter.WithTransform(transform),
	)
	if err != nil {
		return fmt.Errorf("create converter: %w", err)
	}
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter(openapi.WithIndent(2), openapi.WithYAML(true)))

	input, err := os.Open("test/integration/testdata/openapi_v2.json")
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer func() { _ = input.Close() }()

	output, err := os.Create("transformed.yaml")
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	convertErr := conv.Convert(context.Background(), input, output, format.FormatOpenAPI, format.FormatOpenAPI)
	closeErr := output.Close()
	if convertErr != nil {
		return fmt.Errorf("convert: %w", convertErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close output: %w", closeErr)
	}

	fmt.Println("Created transformed.yaml")
	return nil
}
