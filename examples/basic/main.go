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
	// Create a new converter instance
	conv, err := converter.New(
		converter.WithStrict(true),
		converter.WithValidation(true, true),
	)
	if err != nil {
		log.Fatalf("failed to create converter: %v", err)
	}

	// Register format parsers and writers
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter())
	conv.RegisterParser(apiblueprint.NewParser())
	conv.RegisterWriter(apiblueprint.NewWriter())

	// Open input file from testdata
	inputPath := "test/integration/testdata/openapi_v3_0.yaml"
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
	outputFile, err := os.Create("petstore.apib")
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

	// Convert OpenAPI to API Blueprint
	fmt.Println("Converting OpenAPI to API Blueprint...")
	err = conv.Convert(ctx, inputFile, outputFile,
		format.FormatOpenAPI,
		format.FormatAPIBlueprint,
	)
	if err != nil {
		log.Printf("conversion failed: %v", err)
		return
	}

	fmt.Println("Conversion successful: petstore.apib created")
}
