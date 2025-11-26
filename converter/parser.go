package converter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Pre-compiled regular expressions for parser performance
var (
	// Matches resource definition: ## Resource Name [/path]
	reResource = regexp.MustCompile(`^## (.+?) \[(.+?)\]`)

	// Matches action definition: ### Action Name [METHOD]
	reAction = regexp.MustCompile(`^### (.+?) \[(GET|POST|PUT|DELETE|PATCH)\]`)

	// Matches parameter definition: + name (type, required) - description
	reParameter = regexp.MustCompile(`^(\w+)\s*\((\w+),\s*(\w+)\)(?:\s*-\s*(.+))?`)

	// Matches response definition: + Response 200 (application/json)
	reResponse = regexp.MustCompile(`\+ Response (\d+)(?:\s*\(([^)]+)\))?`)
)

// ParseAPIBlueprint parses an API Blueprint format document and returns an OpenAPI specification.
//
// This function converts API Blueprint markdown format to a structured OpenAPI 3.0 specification.
// It supports the core API Blueprint features including groups, resources, actions, parameters,
// requests, and responses.
//
// By default, this outputs OpenAPI 3.0.0. Use ParseAPIBlueprintWithOptions to specify a different version.
//
// Parameters:
//   - data: API Blueprint content as bytes
//
// Returns:
//   - *OpenAPI: Parsed OpenAPI 3.0 specification
//   - error: Error if parsing fails
//
// Example:
//
//	apibContent := []byte(`FORMAT: 1A
//	# My API
//
//	A simple API description
//
//	HOST: https://api.example.com
//
//	## Group Users
//
//	## /users [/users]
//
//	### List Users [GET]
//
//	+ Response 200 (application/json)
//
//	        {
//	            "users": []
//	        }
//	`)
//
//	spec, err := converter.ParseAPIBlueprint(apibContent)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("API Title: %s\n", spec.Info.Title)
func ParseAPIBlueprint(data []byte) (*OpenAPI, error) {
	return parseAPIBlueprintReader(strings.NewReader(string(data)))
}

// ParseAPIBlueprintWithOptions parses an API Blueprint with custom conversion options.
//
// This allows you to specify the output OpenAPI version (3.0 or 3.1) and other
// conversion behaviors.
//
// Parameters:
//   - data: API Blueprint content as bytes
//   - opts: Conversion options (nil for defaults)
//
// Returns:
//   - *OpenAPI: Parsed OpenAPI specification
//   - error: Error if parsing fails
//
// Example:
//
//	opts := &converter.ConversionOptions{
//	    OutputVersion: converter.Version31,
//	}
//	spec, err := converter.ParseAPIBlueprintWithOptions(apibContent, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("OpenAPI version: %s\n", spec.OpenAPI) // "3.1.0"
func ParseAPIBlueprintWithOptions(data []byte, opts *ConversionOptions) (*OpenAPI, error) {
	return parseAPIBlueprintReaderWithOptions(strings.NewReader(string(data)), opts)
}

// ParseAPIBlueprintReader parses an API Blueprint format from an io.Reader.
//
// This is the streaming version of ParseAPIBlueprint, useful for reading from files,
// network connections, or other io.Reader sources without loading the entire content
// into memory first.
//
// Parameters:
//   - r: An io.Reader containing API Blueprint content
//
// Returns:
//   - *OpenAPI: Parsed OpenAPI 3.0 specification
//   - error: Error if reading or parsing fails
//
// Example:
//
//	file, err := os.Open("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	spec, err := converter.ParseAPIBlueprintReader(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Parsed %d paths\n", len(spec.Paths))
func ParseAPIBlueprintReader(r io.Reader) (*OpenAPI, error) {
	return parseAPIBlueprintReader(r)
}

type parserState struct {
	spec          *OpenAPI
	currentPath   string
	currentMethod string
	currentOp     *Operation
	inRequest     bool
	inResponse    bool
	currentResp   *Response
	currentStatus string
	jsonBuffer    []string
	inJSON        bool
}

func parseAPIBlueprintReader(r io.Reader) (*OpenAPI, error) {
	return parseAPIBlueprintReaderWithOptions(r, nil)
}

func parseAPIBlueprintReaderWithOptions(r io.Reader, opts *ConversionOptions) (*OpenAPI, error) {
	if opts == nil {
		opts = DefaultConversionOptions()
	}

	scanner := bufio.NewScanner(r)
	state := &parserState{
		spec: &OpenAPI{
			OpenAPI: opts.OutputVersion.ToFullVersion(),
			Info:    Info{},
			Paths:   make(map[string]PathItem),
		},
	}

	for scanner.Scan() {
		line := scanner.Text()
		if err := parseLine(state, line); err != nil {
			return nil, err
		}
	}

	// Finalize any remaining response
	if state.inResponse && state.currentOp != nil {
		finalizeResponse(state)
	}

	// Finalize any remaining operation
	if state.currentOp != nil && state.currentPath != "" {
		finalizeOperation(state)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading API Blueprint: %w", err)
	}

	return state.spec, nil
}

func parseLine(state *parserState, line string) error {
	trimmed := strings.TrimSpace(line)

	// Handle JSON body parsing
	if state.inJSON {
		return handleJSONLine(state, line, trimmed)
	}

	// Skip empty lines
	if trimmed == "" {
		return nil
	}

	// Parse different line types
	if err := parseHeaderLine(state, trimmed); err != nil || isHeaderLine(trimmed) {
		return err
	}

	if err := parseStructureLine(state, trimmed); err != nil || isStructureLine(trimmed) {
		return err
	}

	if err := parseContentLine(state, line, trimmed); err != nil || isContentLine(line, trimmed) {
		return err
	}

	// Default: handle as description
	return handleDescription(state, line, trimmed)
}

func handleJSONLine(state *parserState, line, trimmed string) error {
	if trimmed == "" && len(state.jsonBuffer) > 0 {
		// End of JSON block
		if err := finalizeJSON(state); err != nil {
			return err
		}
		state.inJSON = false
		state.jsonBuffer = nil
	} else if trimmed != "" {
		// Continue collecting JSON
		state.jsonBuffer = append(state.jsonBuffer, strings.TrimLeft(line, " \t"))
	}
	return nil
}

func isHeaderLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "FORMAT:") ||
		(strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ")) ||
		strings.HasPrefix(trimmed, "HOST:")
}

func parseHeaderLine(state *parserState, trimmed string) error {
	if strings.HasPrefix(trimmed, "FORMAT:") {
		return nil
	}

	if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
		state.spec.Info.Title = strings.TrimPrefix(trimmed, "# ")
		return nil
	}

	if strings.HasPrefix(trimmed, "HOST:") {
		host := strings.TrimSpace(strings.TrimPrefix(trimmed, "HOST:"))
		state.spec.Servers = []Server{{URL: host}}
		return nil
	}

	return nil
}

func isStructureLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "## Group ") ||
		(strings.HasPrefix(trimmed, "## ") && strings.Contains(trimmed, "[")) ||
		(strings.HasPrefix(trimmed, "### ") && strings.Contains(trimmed, "["))
}

func parseStructureLine(state *parserState, trimmed string) error {
	if strings.HasPrefix(trimmed, "## Group ") {
		return nil
	}

	if strings.HasPrefix(trimmed, "## ") && strings.Contains(trimmed, "[") {
		if state.currentOp != nil && state.currentPath != "" {
			finalizeOperation(state)
		}
		return parseResource(state, trimmed)
	}

	if strings.HasPrefix(trimmed, "### ") && strings.Contains(trimmed, "[") {
		if state.currentOp != nil && state.currentPath != "" {
			finalizeOperation(state)
		}
		return parseAction(state, trimmed)
	}

	return nil
}

func isContentLine(line, trimmed string) bool {
	return strings.HasPrefix(trimmed, "+ Parameters") ||
		(strings.HasPrefix(line, "    + ") && !strings.Contains(line, "Response") && !strings.Contains(line, "Request")) ||
		strings.HasPrefix(trimmed, "+ Request") ||
		strings.HasPrefix(trimmed, "+ Response")
}

func parseContentLine(state *parserState, line, trimmed string) error {
	if strings.HasPrefix(trimmed, "+ Parameters") {
		if state.currentOp != nil {
			state.currentOp.Parameters = []Parameter{}
		}
		return nil
	}

	if strings.HasPrefix(line, "    + ") && !strings.Contains(line, "Response") && !strings.Contains(line, "Request") {
		return parseParameter(state, line)
	}

	if strings.HasPrefix(trimmed, "+ Request") {
		if state.currentOp != nil {
			state.inRequest = true
			state.inResponse = false
			state.currentOp.RequestBody = &RequestBody{
				Content: make(map[string]MediaType),
			}
			state.inJSON = true
			state.jsonBuffer = nil
		}
		return nil
	}

	if strings.HasPrefix(trimmed, "+ Response") {
		if state.inResponse && state.currentOp != nil {
			finalizeResponse(state)
		}
		state.inResponse = true
		state.inRequest = false
		return parseResponse(state, trimmed)
	}

	return nil
}

func handleDescription(state *parserState, line, trimmed string) error {
	// API description (first non-special line after title)
	if state.spec.Info.Title != "" && len(state.spec.Servers) == 0 && len(state.spec.Paths) == 0 && state.spec.Info.Description == "" {
		state.spec.Info.Description = trimmed
		return nil
	}

	// Operation description
	if state.currentOp != nil && state.currentOp.Description == "" && !state.inRequest && !state.inResponse {
		state.currentOp.Description = trimmed
		return nil
	}

	// Response description (indented under response)
	if state.inResponse && state.currentResp != nil && strings.HasPrefix(line, "    ") {
		state.currentResp.Description = trimmed
		return nil
	}

	return nil
}

func parseResource(state *parserState, line string) error {
	// ## /path [/path]
	matches := reResource.FindStringSubmatch(line)
	if len(matches) >= 3 {
		state.currentPath = matches[2]
		if _, exists := state.spec.Paths[state.currentPath]; !exists {
			state.spec.Paths[state.currentPath] = PathItem{}
		}
	}
	return nil
}

func parseAction(state *parserState, line string) error {
	// ### Action Name [METHOD]
	matches := reAction.FindStringSubmatch(line)
	if len(matches) >= 3 {
		summary := matches[1]
		method := matches[2]
		state.currentMethod = method
		state.currentOp = &Operation{
			Summary:   summary,
			Responses: make(map[string]Response),
		}
		state.inRequest = false
		state.inResponse = false
	}
	return nil
}

func parseParameter(state *parserState, line string) error {
	if state.currentOp == nil {
		return nil
	}

	// + name (required, type) - description
	// Remove leading spaces and +
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "+"))

	matches := reParameter.FindStringSubmatch(line)
	if len(matches) >= 4 {
		param := Parameter{
			Name:     matches[1],
			Required: matches[2] == "required",
			Schema:   &Schema{Type: matches[3]},
		}
		if len(matches) >= 5 && matches[4] != "" {
			param.Description = matches[4]
		}

		// Determine parameter location
		if strings.Contains(state.currentPath, "{"+param.Name+"}") {
			param.In = "path"
		} else {
			param.In = "query"
		}

		state.currentOp.Parameters = append(state.currentOp.Parameters, param)
	}
	return nil
}

func parseResponse(state *parserState, line string) error {
	// + Response 200 (application/json)
	matches := reResponse.FindStringSubmatch(line)
	if len(matches) >= 2 {
		state.currentStatus = matches[1]
		state.currentResp = &Response{
			Description: "Response " + matches[1],
			Content:     make(map[string]MediaType),
		}

		// Start collecting JSON on next line if content type is specified
		if len(matches) >= 3 && matches[2] != "" {
			state.inJSON = true
			state.jsonBuffer = nil
		}
	}
	return nil
}

func finalizeJSON(state *parserState) error {
	if len(state.jsonBuffer) == 0 {
		return nil
	}

	jsonStr := strings.Join(state.jsonBuffer, "\n")
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		// If JSON parsing fails, just skip it
		return nil
	}

	mediaType := MediaType{Example: jsonData}

	if state.inRequest && state.currentOp != nil && state.currentOp.RequestBody != nil {
		state.currentOp.RequestBody.Content["application/json"] = mediaType
	} else if state.inResponse && state.currentResp != nil {
		state.currentResp.Content["application/json"] = mediaType
	}

	return nil
}

func finalizeResponse(state *parserState) {
	if state.currentOp != nil && state.currentResp != nil && state.currentStatus != "" {
		state.currentOp.Responses[state.currentStatus] = *state.currentResp
		state.currentResp = nil
		state.currentStatus = ""
	}
}

func finalizeOperation(state *parserState) {
	if state.currentOp == nil || state.currentPath == "" || state.currentMethod == "" {
		return
	}

	// Finalize any remaining response
	if state.currentResp != nil {
		finalizeResponse(state)
	}

	pathItem := state.spec.Paths[state.currentPath]
	switch state.currentMethod {
	case "GET":
		pathItem.Get = state.currentOp
	case "POST":
		pathItem.Post = state.currentOp
	case "PUT":
		pathItem.Put = state.currentOp
	case "DELETE":
		pathItem.Delete = state.currentOp
	case "PATCH":
		pathItem.Patch = state.currentOp
	}
	state.spec.Paths[state.currentPath] = pathItem

	state.currentOp = nil
	state.currentMethod = ""
}

// ToOpenAPI converts API Blueprint bytes to OpenAPI JSON bytes.
//
// This function provides a one-step conversion from API Blueprint format to
// OpenAPI 3.0 JSON. The output is formatted with indentation for readability.
//
// By default, this outputs OpenAPI 3.0.0. Use ToOpenAPIWithOptions to specify a different version.
//
// Parameters:
//   - data: API Blueprint content as bytes
//
// Returns:
//   - []byte: OpenAPI 3.0 specification as formatted JSON bytes
//   - error: Error if parsing or marshaling fails
//
// Example:
//
//	apibContent := []byte(`FORMAT: 1A
//	# Pet Store API
//
//	## /pets [/pets]
//
//	### List Pets [GET]
//
//	+ Response 200 (application/json)
//
//	        [{"id": 1, "name": "Fluffy"}]
//	`)
//
//	openapiJSON, err := converter.ToOpenAPI(apibContent)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Write to file
//	os.WriteFile("openapi.json", openapiJSON, 0644)
func ToOpenAPI(data []byte) ([]byte, error) {
	spec, err := ParseAPIBlueprint(data)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(spec, "", "  ")
}

// ToOpenAPIWithOptions converts API Blueprint bytes to OpenAPI JSON with custom options.
//
// This allows you to specify the output OpenAPI version (3.0 or 3.1).
//
// Parameters:
//   - data: API Blueprint content as bytes
//   - opts: Conversion options (nil for defaults)
//
// Returns:
//   - []byte: OpenAPI specification as formatted JSON bytes
//   - error: Error if parsing or marshaling fails
//
// Example:
//
//	opts := &converter.ConversionOptions{
//	    OutputVersion: converter.Version31,
//	}
//	openapiJSON, err := converter.ToOpenAPIWithOptions(apibContent, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ToOpenAPIWithOptions(data []byte, opts *ConversionOptions) ([]byte, error) {
	spec, err := ParseAPIBlueprintWithOptions(data, opts)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(spec, "", "  ")
}

// ToOpenAPIString converts API Blueprint string to OpenAPI JSON string.
//
// This is a convenience wrapper around ToOpenAPI for string inputs. The output
// is formatted JSON with 2-space indentation.
//
// Parameters:
//   - apibStr: API Blueprint content as a string
//
// Returns:
//   - string: OpenAPI 3.0 specification as a formatted JSON string
//   - error: Error if parsing or marshaling fails
//
// Example:
//
//	apibContent := `FORMAT: 1A
//	# My API
//
//	## /users [/users]
//
//	### Get Users [GET]
//
//	+ Response 200`
//
//	openapiJSON, err := converter.ToOpenAPIString(apibContent)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(openapiJSON)
func ToOpenAPIString(apibStr string) (string, error) {
	jsonBytes, err := ToOpenAPI([]byte(apibStr))
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// ConvertToOpenAPI converts API Blueprint from a reader to OpenAPI JSON written to a writer.
//
// This is the streaming version for converting API Blueprint to OpenAPI, useful
// for working with files or network streams. The output JSON is formatted with
// 2-space indentation for readability.
//
// Parameters:
//   - r: An io.Reader containing API Blueprint content
//   - w: An io.Writer where the OpenAPI JSON output will be written
//
// Returns an error if:
//   - Reading from r fails
//   - Parsing the API Blueprint fails
//   - Writing to w fails
//
// Example:
//
//	input, err := os.Open("api.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input.Close()
//
//	output, err := os.Create("openapi.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	if err := converter.ConvertToOpenAPI(input, output); err != nil {
//	    log.Fatal(err)
//	}
func ConvertToOpenAPI(r io.Reader, w io.Writer) error {
	spec, err := ParseAPIBlueprintReader(r)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(spec)
}
