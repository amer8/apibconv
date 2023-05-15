package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type OpenAPI struct {
	Info struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Version     string `json:"version"`
	} `json:"info"`
	Paths      map[string]map[string]Method `json:"paths"`
	Components struct {
		Schemas map[string]Schema `json:"schemas"`
	} `json:"components"`
}

type Method struct {
	OperationID string  `json:"operationId"`
	Summary     string  `json:"summary"`
	Description *string `json:"description"`
	Parameters  []struct {
		Name     string `json:"name"`
		Required bool   `json:"required"`
		In       string `json:"in"`
		Schema   struct {
			Type string `json:"type"`
		} `json:"schema"`
	} `json:"parameters"`
	RequestBody struct {
		Required bool `json:"required"`
		Content  struct {
			ApplicationJSON struct {
				Schema struct {
					Ref   string `json:"$ref"`
					Type  string `json:"type"`
					Items struct {
						Ref string `json:"$ref"`
					} `json:"items"`
				} `json:"schema"`
			} `json:"application/json"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Description string `json:"description"`
		Content     struct {
			ApplicationJSON struct {
				Schema struct {
					Ref   string `json:"$ref"`
					Type  string `json:"type"`
					Items struct {
						Ref string `json:"$ref"`
					} `json:"items"`
				} `json:"schema"`
			} `json:"application/json"`
		} `json:"content"`
	} `json:"responses"`
	Tags []string `json:"tags"`
}

type Schema struct {
	Type       string `json:"type"`
	Properties map[string]struct {
		Format   string  `json:"format"`
		Type     string  `json:"type"`
		Example  *string `json:"example"`
		Nullable bool    `json:"nullable"`
	} `json:"properties"`
	Required []string `json:"required"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: apibconv -f input.json -o output.apib")
		return
	}

	inputFlag := flag.String("f", "", "Path to the input OpenAPI JSON file")
	outputFlag := flag.String("o", "", "Path to the output API Blueprint file")

	flag.Parse()

	if *inputFlag == "" {
		fmt.Println("Error: Input file is required")
		os.Exit(1)
	}

	if *outputFlag == "" {
		fmt.Println("Error: Output file is required")
		os.Exit(1)
	}

	inputData, err := ioutil.ReadFile(*inputFlag)
	if err != nil {
		fmt.Printf("Error: Cannot read input file '%s': %v\n", *inputFlag, err)
		os.Exit(1)
	}

	api, err := parseOpenAPI(inputData)
	if err != nil {
		fmt.Printf("Error: Cannot parse input file '%s': %v\n", *inputFlag, err)
		os.Exit(1)
	}

	apiBlueprint, err := createAPIBlueprint(api)
	if err != nil {
		fmt.Printf("Error: Cannot convert to API Blueprint: %v\n", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*outputFlag, []byte(apiBlueprint), 0o644)
	if err != nil {
		fmt.Printf("Error: Cannot write output file '%s': %v\n", *outputFlag, err)
		os.Exit(1)
	}
}

func parseOpenAPI(data []byte) (OpenAPI, error) {
	var api OpenAPI
	err := json.Unmarshal(data, &api)
	if err != nil {
		return OpenAPI{}, errors.New("unable to parse OpenAPI JSON")
	}

	return api, nil
}

func formatAttributes(schema Schema) string {
	var sb strings.Builder

	sb.WriteString("+ Attributes \n")
	for propName, prop := range schema.Properties {
		var example string
		if prop.Example != nil {
			example = ": " + *prop.Example
		}
		if prop.Nullable {
			sb.WriteString("    + " + propName + example + " (" + prop.Type + ", optional, nullable)\n")
		} else {
			sb.WriteString("    + " + propName + example + " (" + prop.Type + ", optional)\n")
		}
	}

	return sb.String()
}

func formatBody(schemaType string, schema Schema) string {
	if schemaType == "object" {
		example := make(map[string]interface{})

		for propName, prop := range schema.Properties {
			propValue := propValue(prop.Type)
			if prop.Example != nil {
				propValue = *prop.Example
			}

			if prop.Nullable {
				example[propName] = nil
			} else {
				example[propName] = propValue
			}
		}

		jsonBytes, _ := json.MarshalIndent(example, "        ", "    ")
		return string(jsonBytes)
	} else if schemaType == "array" {
		example := make([]interface{}, 1)

		for propName, prop := range schema.Properties {
			propValue := propValue(prop.Type)
			if prop.Example != nil {
				propValue = *prop.Example
			}
			if prop.Nullable {
				example[0] = map[string]interface{}{propName: nil}
			} else {
				example[0] = map[string]interface{}{propName: propValue}
			}
		}

		jsonBytes, _ := json.MarshalIndent(example, "        ", "    ")
		return string(jsonBytes)
	}
	return ""
}

func isRequired(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}

func propValue(valueType string) interface{} {
	switch valueType {
	case "string":
		return "string"
	case "number":
		return 0
	case "boolean":
		return true
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}

func formatOperation(method, path string, operation Method, componentSchemas map[string]Schema) string {
	var sb strings.Builder

	sb.WriteString("## " + operation.Summary + " [" + strings.ToUpper(method) + " " + path + "]\n")
	if operation.Description != nil {
		sb.WriteString(*operation.Description + "\n")
	}
	sb.WriteString("\n")

	var hasQuery bool
	for _, param := range operation.Parameters {
		if param.In == "query" {
			hasQuery = true
		}
	}

	if hasQuery {
		sb.WriteString("+ Parameters\n")
	}
	for _, param := range operation.Parameters {
		if param.In == "query" {
			sb.WriteString("    + " + param.Name + " (" + param.Schema.Type + ", " + isRequired(param.Required) + ") \n")
		}
	}
	if hasQuery {
		sb.WriteString("\n")
	}

	if strings.ToUpper(method) == "PATCH" || strings.ToUpper(method) == "POST" {
		var refPath, schemaType string
		requestBodySchema := operation.RequestBody.Content.ApplicationJSON.Schema

		if requestBodySchema.Ref != "" {
			schemaType = "object"
			refPath = requestBodySchema.Ref
		}

		if requestBodySchema.Items.Ref != "" {
			schemaType = "array"
			refPath = requestBodySchema.Items.Ref
		}

		ref := strings.TrimPrefix(refPath, "#/components/schemas/")
		attributesSchema := componentSchemas[ref]

		sb.WriteString(formatAttributes(attributesSchema))
		sb.WriteString("\n")

		sb.WriteString("+ Request (application/json)\n\n")
		sb.WriteString("  + Headers\n\n")
		sb.WriteString("    Authorization: Bearer <JWT>\n\n")
		sb.WriteString("  + Body\n\n")
		sb.WriteString("        " + formatBody(schemaType, attributesSchema) + "\n\n")
	}

	for code, response := range operation.Responses {
		var refPath, schemaType string
		responseBodySchema := response.Content.ApplicationJSON.Schema
		if responseBodySchema.Ref != "" {
			schemaType = "object"
			refPath = responseBodySchema.Ref
		}

		if responseBodySchema.Items.Ref != "" {
			schemaType = "array"
			refPath = responseBodySchema.Items.Ref
		}

		ref := strings.TrimPrefix(refPath, "#/components/schemas/")
		sb.WriteString("+ Response " + code + " (application/json)\n")
		responseSchema := componentSchemas[ref]

		sb.WriteString("  + Body\n\n")
		sb.WriteString("        " + formatBody(schemaType, responseSchema) + "\n\n")
	}

	sb.WriteString("\n")

	return sb.String()
}

func createAPIBlueprint(api OpenAPI) (string, error) {
	var sb strings.Builder

	sb.WriteString("FORMAT: 1A\n")
	sb.WriteString("HOST: http://api.example.com\n\n")

	sb.WriteString("# " + api.Info.Title + "\n\n")

	sortedPaths := make([]string, 0, len(api.Paths))
	for path := range api.Paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	var currentGroup string
	for _, path := range sortedPaths {

		methods := api.Paths[path]
		for method, operation := range methods {
			if len(operation.Tags) > 0 {
				if operation.Tags[0] != currentGroup {
					sb.WriteString("# Group " + operation.Tags[0] + "\n")
					sb.WriteString("\nResources related to " + operation.Tags[0] + "\n\n")
				}

				currentGroup = operation.Tags[0]
			}
			sb.WriteString(formatOperation(method, path, operation, api.Components.Schemas))
		}
	}

	return sb.String(), nil
}
