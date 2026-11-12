// Package main demonstrates a custom transformation with apibconv.
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/amer8/apibconv/pkg/converter"
	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/format/openapi"
	"github.com/amer8/apibconv/pkg/model"
)

func main() {
	conv, err := converter.New(converter.WithTransform(addPrefix))
	if err != nil {
		log.Fatalf("create converter: %v", err)
	}
	conv.RegisterParser(openapi.NewParser())
	conv.RegisterWriter(openapi.NewWriter())

	input := strings.NewReader(`openapi: 3.0.0
info:
  title: Original API
  version: 1.0.0
paths: {}`)
	if err := conv.Convert(context.Background(), input, os.Stdout, format.FormatOpenAPI, format.FormatOpenAPI); err != nil {
		log.Fatalf("convert: %v", err)
	}
}

// addPrefix changes the model between parsing and writing.
func addPrefix(api *model.API) error {
	api.Info.Title = "[Plugin] " + api.Info.Title
	api.Info.Description += "\n\nProcessed by a custom transformation."
	return nil
}
