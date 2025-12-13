// Package main demonstrates advanced usage of the apibconv library, including custom transformations.
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
	// Define a custom transformation function
	transformFn := func(api *model.API) error {
		api.Info.Title = "Transformed API Title"
		api.Info.Description = "This description was added by a transformer."
		return nil
	}

	// Create a new converter with custom options and a transformer
	conv, err := converter.New(
		converter.WithStrict(true),
		converter.WithValidation(true, false), // Validate input only
		converter.WithTransform(transformFn),
		converter.WithWarningHandler(func(warning string) {
			fmt.Fprintf(os.Stderr, "WARNING: %s\n", warning)
		}),
	)
	if err != nil {
		log.Fatalf("failed to create converter: %v", err)
	}

	// Register format parsers and writers
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter(openapi.WithIndent(2), openapi.WithYAML(true)))

	// Open input file from testdata
	inputPath := "test/integration/testdata/openapi_v2.json"
	fmt.Printf("Opening input from %s...\n", inputPath)
	inputFile, err := os.Open(inputPath)
	if err != nil {
		log.Fatalf("failed to open input file: %v", err)
	}
	defer func() {
		if err := inputFile.Close(); err != nil {
			log.Printf("failed to close input file: %v", err)
		}
	}()

	// Create output file
	outputFile, err := os.Create("transformed.yaml")
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return
	}
	defer func() {
		if cerr := outputFile.Close(); cerr != nil {
			log.Printf("Error closing output file: %v", cerr)
		}
	}()

	ctx := context.Background()

	// Convert OpenAPI to OpenAPI (with transformation)
	fmt.Println("Converting OpenAPI with transformation...")
	err = conv.Convert(ctx, inputFile, outputFile,
		format.FormatOpenAPI,
		format.FormatOpenAPI,
	)
	if err != nil {
		log.Printf("conversion failed: %v", err)
		return
	}

	fmt.Println("Conversion successful: transformed.yaml created")
}
