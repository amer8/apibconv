// Package main demonstrates how to use custom plugins with apibconv.
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

// MyCustomTransformer implements a custom transformation logic
type MyCustomTransformer struct{}

func (t *MyCustomTransformer) Name() string {
	return "my-custom-plugin"
}

func (t *MyCustomTransformer) Transform(api *model.API) error {
	fmt.Println("  [Plugin] Running custom transformation...")
	api.Info.Title = fmt.Sprintf("[Plugin] %s", api.Info.Title)
	api.Info.Description += "\n\nProcessed by MyCustomTransformer plugin."
	return nil
}

func main() {
	// Define the transformation function wrapper
	transformFn := func(api *model.API) error {
		transformer := &MyCustomTransformer{}
		return transformer.Transform(api)
	}

	// Create a converter with the custom transform
	conv, err := converter.New(
		converter.WithTransform(transformFn),
	)
	if err != nil {
		log.Fatalf("failed to create converter: %v", err)
	}

	// Register parsers and writers
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter(openapi.WithIndent(2)))

	// Create a dummy input file for demonstration
	inputFile, err := os.CreateTemp("", "input-*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if cerr := os.Remove(inputFile.Name()); cerr != nil {
			log.Printf("Error removing temp file: %v", cerr)
		}
	}()
	
	if _, err := inputFile.WriteString(`openapi: 3.0.0
info:
  title: Original API
  version: 1.0.0
paths: {}`); err != nil {
		log.Printf("failed to write to input file: %v", err)
		return
	}
	if cerr := inputFile.Close(); cerr != nil {
		log.Printf("Error closing input file: %v", cerr)
	}

	// Re-open for reading
	input, err := os.Open(inputFile.Name())
	if err != nil {
		log.Printf("failed to re-open input file: %v", err)
		return
	}
	defer func() {
		if cerr := input.Close(); cerr != nil {
			log.Printf("Error closing input reader: %v", cerr)
		}
	}()

	// Convert
	fmt.Println("Running conversion with plugin...")
	err = conv.Convert(context.Background(), input, os.Stdout, format.FormatOpenAPI, format.FormatOpenAPI)
	if err != nil {
		log.Printf("conversion failed: %v", err)
		return
	}
}
