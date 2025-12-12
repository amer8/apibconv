package converter

// Package converter provides bidirectional conversion between OpenAPI 3.0/3.1, AsyncAPI 2.x/3.x, and API Blueprint specifications.
//
// # Overview
//
// This package enables high-performance, zero-allocation conversion between three popular API specification formats:
//   - OpenAPI 3.0/3.1 (JSON/YAML)
//   - AsyncAPI 2.x/3.x (JSON/YAML)
//   - API Blueprint (Markdown-based format)
//
// The converter supports multiple directions:
//   - OpenAPI ↔ API Blueprint
//   - AsyncAPI ↔ API Blueprint
//   - OpenAPI 3.0 ↔ OpenAPI 3.1
//
// It makes it easy to work with any of these formats based on your needs.
//
// # Key Features
//
//   - Bidirectional conversion (OpenAPI ↔ API Blueprint)
//   - Bidirectional conversion (AsyncAPI ↔ API Blueprint)
//   - **JSON** and **YAML** input/output support
//   - OpenAPI version support (3.0 and 3.1) with automatic conversion
//   - AsyncAPI version support (2.0-2.6 and 3.0)
//   - Zero-allocation buffer operations using sync.Pool
//   - Streaming API for large files
//   - Support for paths, operations, parameters, request bodies, and responses
//   - MSON Data Structures support (Attributes, Groups, Named Types)
//   - Automatic content type handling (application/json)
//   - Handles OpenAPI 3.1 features: type arrays, webhooks, JSON Schema 2020-12
//
// # Quick Start
//
// Convert OpenAPI to API Blueprint:
//
//	openapiJSON := `{"openapi": "3.0.0", "info": {"title": "My API", "version": "1.0.0"}, "paths": {}}`
//	apiBlueprint, err := converter.FromJSONString(openapiJSON)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(apiBlueprint)
//
// Convert API Blueprint to OpenAPI:
//
//	apibContent := `FORMAT: 1A
//	# My API
//	## Group Users
//	## /users [/users]
//	### List Users [GET]
//	+ Response 200 (application/json)`
//
//	openapiJSON, err := converter.ToOpenAPIString(apibContent)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(openapiJSON)
//
// Convert AsyncAPI to API Blueprint:
//
//	asyncapiJSON := `{"asyncapi": "2.6.0", "info": {"title": "My Event API", "version": "1.0.0"}, "channels": {}}`
//	// Auto-detects version (2.x or 3.x)
//	spec, err := converter.ParseAsync([]byte(asyncapiJSON))
//	blueprint := spec.ToBlueprint()
//	fmt.Println(blueprint)
//
// # Version Conversion
//
// Convert API Blueprint to OpenAPI 3.1:
//
//	opts := &converter.ConversionOptions{
//	    OutputVersion: converter.Version31,
//	}
//	spec, err := converter.ParseBlueprintWithOptions(apibData, opts)
//	// spec.OpenAPI is "3.1.0"
//
// Convert between OpenAPI versions:
//
//	s, _ := converter.Parse([]byte(`{"openapi": "3.0.0", ...}`))
//	spec30 := s.(*converter.OpenAPI)
//	spec31, err := spec30.ConvertTo(converter.Version31, nil)
//	// Nullable fields become type arrays: ["string", "null"]
//
//	spec31to30, err := spec31.ConvertTo(converter.Version30, nil)
//	// Type arrays become nullable: true
//
// # Conversion Workflows
//
// There are four main workflows for using this package:
//
// 1. Direct Conversion (simplest):
//
//	result, err := converter.FromJSONString(openapiJSON)
//	result, err := converter.ToOpenAPIString(apibContent)
//
// 2. Parse, Modify, Format (for programmatic manipulation):
//
//	s, err := converter.Parse(data)
//	if spec, ok := s.(*converter.OpenAPI); ok {
//	    spec.Info.Title = "Modified API"
//	    // Serialize back to JSON
//	    jsonResult := spec.String()
//	    // Or convert to Blueprint
//	    blueprint, err := spec.ToBlueprint()
//	}
//
// 3. Streaming (for large files):
//
//	// OpenAPI → API Blueprint
//	input, err := os.Open("examples/openapi/petstore/petstore.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input.Close()
//
//	output, err := os.Create("petstore.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output.Close()
//
//	err = converter.Convert(input, output)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// API Blueprint → OpenAPI
//	input2, err := os.Open("examples/apib/mson-example/mson-example.apib")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer input2.Close()
//
//	output2, err := os.Create("mson-example.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer output2.Close()
//
//	err = converter.ConvertToOpenAPI(input2, output2)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 4. Version Conversion:
//
//	spec31, err := spec30.ConvertTo(converter.Version31, nil)
//	spec30, err := spec31.ConvertTo(converter.Version30, nil)
//
// # Performance
//
// The package is optimized for performance with zero allocations for buffer operations:
//
//   - Uses sync.Pool for buffer reuse
//   - Streaming API avoids loading entire files into memory
//   - Zero external dependencies (uses standard library only)
//
// Benchmark results:
//
//	BenchmarkWriteAPIBlueprint-16     34.5M      73.19 ns/op      0 B/op    0 allocs/op
//	BenchmarkBufferPool-16            1B+         1.75 ns/op      0 B/op    0 allocs/op
//
// # Function Categories
//
// Version Conversion:
//
//   - OpenAPI.ConvertTo: Convert between OpenAPI 3.0 and 3.1
//   - DetectVersion: Detect OpenAPI version from spec string
//   - Schema.TypeName: Get primary type from schema (handles both 3.0 and 3.1)
//   - Schema.IsNullable: Check if schema allows null values
//
// OpenAPI → API Blueprint Conversion:
//
//   - FromJSON, FromJSONString: Convert JSON bytes/string to API Blueprint
//   - ToBytes: Convert JSON bytes to API Blueprint bytes
//   - ConvertString: Alias for FromJSONString
//   - Convert: Streaming I/O conversion
//
// API Blueprint → OpenAPI Conversion:
//
//   - ParseBlueprint: Parse API Blueprint to OpenAPI 3.0
//   - ParseBlueprintWithOptions: Parse with version options
//   - ParseBlueprintReader: Parse API Blueprint from reader
//
// AsyncAPI Conversion:
//
//   - ParseAsync: Parse AsyncAPI 2.x JSON/YAML
//   - ParseAsyncV3: Parse AsyncAPI 3.x JSON/YAML
//   - AsyncAPI.ToBlueprint: Convert AsyncAPI 2.x to API Blueprint
//   - AsyncAPIV3.ToBlueprint: Convert AsyncAPI 3.x to API Blueprint
//   - OpenAPI.ToAsyncAPI: Convert API Blueprint (via OpenAPI struct) to AsyncAPI 2.x
//   - OpenAPI.ToAsyncAPIV3: Convert API Blueprint (via OpenAPI struct) to AsyncAPI 3.x
//   - ConvertAPIBlueprintToAsyncAPI: Streaming conversion to AsyncAPI 2.x
//   - ConvertAPIBlueprintToAsyncAPIV3: Streaming conversion to AsyncAPI 3.x
//
// Parsing:
//
//   - Parse, ParseReader: Parse OpenAPI JSON (3.0 or 3.1)
//   - ParseWithConversion: Parse and convert to target version
//   - ParseBlueprint: Parse API Blueprint to OpenAPI 3.0
//   - ParseBlueprintWithOptions: Parse with version options
//   - ParseBlueprintReader: Parse API Blueprint from reader
//
// Formatting:
//
//   - OpenAPI.ToAPIBlueprint: Convert OpenAPI to API Blueprint string
//   - OpenAPI.WriteTo: Write OpenAPI as API Blueprint to writer
//   - MustFormat, MustFromJSON: Panic on error (useful for testing)
//
// # OpenAPI Structure
//
// The OpenAPI type represents a minimal but complete OpenAPI 3.0/3.1 specification:
//
//	type OpenAPI struct {
//	    OpenAPI            string                 // Version (e.g., "3.0.0", "3.1.0")
//	    Info               Info                   // API metadata
//	    Servers            []Server               // Server URLs
//	    Paths              map[string]PathItem    // API endpoints
//	    Webhooks           map[string]PathItem    // Webhooks (3.1+)
//	    Components         *Components            // Reusable schemas
//	    JSONSchemaDialect  string                 // JSON Schema dialect (3.1+)
//	}
//
// Each PathItem contains HTTP operations (GET, POST, PUT, DELETE, PATCH), and each
// Operation includes parameters, request bodies, and responses.
//
// The Schema type supports both OpenAPI 3.0 and 3.1:
//   - In 3.0: Type is a string, Nullable is a boolean
//   - In 3.1: Type can be string or []string, nullable types use type arrays
//
// # Error Handling
//
// Functions return descriptive errors for:
//
//   - Invalid JSON in Parse functions
//   - Malformed API Blueprint in ParseAPIBlueprint functions
//   - Nil spec pointers in Format functions
//   - I/O errors in streaming functions
//
// Use the Must* variants (MustFormat, MustFromJSON) when you're certain the input
// is valid and want to panic on errors (typically in tests).
//
// # Thread Safety
//
// All exported functions are safe for concurrent use. The internal buffer pool
// (sync.Pool) is thread-safe and optimized for concurrent access.
