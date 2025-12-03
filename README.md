# apibconv

[![CI](https://github.com/amer8/apibconv/actions/workflows/ci.yml/badge.svg)](https://github.com/amer8/apibconv/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/amer8/apibconv/branch/main/graph/badge.svg)](https://codecov.io/gh/amer8/apibconv)
[![Go Reference](https://pkg.go.dev/badge/github.com/amer8/apibconv.svg)](https://pkg.go.dev/github.com/amer8/apibconv)
[![Go Report Card](https://goreportcard.com/badge/github.com/amer8/apibconv)](https://goreportcard.com/report/github.com/amer8/apibconv)
[![Go Version](https://img.shields.io/github/go-mod/go-version/amer8/apibconv)](go.mod)
[![Docker Image](https://ghcr-badge.egpl.dev/amer8/apibconv/latest_tag?label=latest&ignore=latest,0,0.1)](https://github.com/amer8/apibconv/pkgs/container/apibconv)
[![License](https://img.shields.io/github/license/amer8/apibconv)](LICENSE)

Convert between API Blueprint (*.apib), OpenAPI 3.0/3.1, and AsyncAPI 2.6/3.0 specifications.

- [API Blueprint](https://apiblueprint.org/documentation/specification.html) and [MSON](https://apiblueprint.org/documentation/mson/specification.html)
- [AsyncAPI v3.x](https://www.asyncapi.com/docs/reference/specification/v3.0.0)
- [AsyncAPI v2.x](https://v2.asyncapi.com/docs/reference/specification/v2.6.0)
- [OpenAPI v3.1.x](https://swagger.io/specification/)
- [OpenAPI v3.0.x](https://swagger.io/specification/v3/)

## Installation

### Using Go

```sh
# Latest version
go install github.com/amer8/apibconv@latest

# Specific version
go install github.com/amer8/apibconv@v1.0.0
```

### Using Docker

```sh
# Pull the image
docker pull ghcr.io/amer8/apibconv:latest

# Run directly
docker run --rm -v $(pwd):/data -w /data ghcr.io/amer8/apibconv:latest -f openapi.json -o output.apib

# Or create an alias (add to ~/.bashrc or ~/.zshrc)
alias apibconv='docker run --rm -v $(pwd):/data -w /data ghcr.io/amer8/apibconv:latest'
```

### Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/amer8/apibconv/releases):

```sh
# Linux (amd64)
wget https://github.com/amer8/apibconv/releases/download/v1.0.0/apibconv_1.0.0_Linux_x86_64.tar.gz
tar -xzf apibconv_1.0.0_Linux_x86_64.tar.gz
sudo mv apibconv /usr/local/bin/

# macOS (Apple Silicon)
wget https://github.com/amer8/apibconv/releases/download/v1.0.0/apibconv_1.0.0_Darwin_arm64.tar.gz
tar -xzf apibconv_1.0.0_Darwin_arm64.tar.gz
sudo mv apibconv /usr/local/bin/
```

## Usage

### Basic Usage

The tool automatically detects the input format and converts accordingly:

```sh
# Convert OpenAPI to API Blueprint
apibconv -f examples/openapi/petstore/petstore.json -o petstore.apib

# Convert API Blueprint to OpenAPI 3.0 (default)
apibconv -f petstore.apib -o openapi.json

# Convert API Blueprint to OpenAPI 3.1
apibconv -f petstore.apib -o openapi.json --openapi-version 3.1

# Convert AsyncAPI to API Blueprint (auto-detects v2.6 or v3.0)
apibconv -f asyncapi.json -o api.apib

# Convert API Blueprint to AsyncAPI 2.6 (default, WebSocket protocol)
apibconv -f api.apib -o asyncapi.json --output-format asyncapi --protocol ws

# Convert API Blueprint to AsyncAPI 3.0 (Kafka protocol)
apibconv -f api.apib -o asyncapi-v3.json --output-format asyncapi --asyncapi-version 3.0 --protocol kafka

# Convert API Blueprint to AsyncAPI 2.6 (MQTT protocol)
apibconv -f api.apib -o asyncapi.json --output-format asyncapi --asyncapi-version 2.6 --protocol mqtt
```

### OpenAPI Version Support

The tool supports both OpenAPI 3.0 and 3.1:

- **Default**: Outputs OpenAPI 3.0.0 for backward compatibility
- **OpenAPI 3.1**: Use `--openapi-version 3.1` flag for 3.1.0 output
- **Input**: Automatically detects and handles both 3.0 and 3.1 input specs

#### Key Differences

When converting to OpenAPI 3.1:
- Nullable types use type arrays: `["string", "null"]` instead of `nullable: true`
- Supports new fields: `webhooks`, `jsonSchemaDialect`, `license.identifier`
- Full JSON Schema 2020-12 compatibility

When converting from 3.1 to 3.0:
- Type arrays are converted back to `nullable: true`
- 3.1-only features (webhooks, etc.) are dropped

### AsyncAPI Support

The tool supports AsyncAPI 2.6 and 3.0 for event-driven APIs:

- **Versions Supported**:
  - AsyncAPI 2.6 (default output)
  - AsyncAPI 3.0
  - Note: AsyncAPI 1.x is not supported
- **Protocols**: WebSocket (`ws`), MQTT (`mqtt`), Kafka (`kafka`), AMQP (`amqp`), HTTP (`http`)
- **Default Protocol**: WebSocket when not specified
- **Format Detection**: Automatically detects AsyncAPI version and format
- **Conversion Mappings**:
  - **AsyncAPI 2.6**: Channels contain publish/subscribe operations
  - **AsyncAPI 3.0**: Operations at root level with send/receive actions
  - AsyncAPI channels → API Blueprint paths
  - Subscribe/Receive operations → GET operations (receiving messages)
  - Publish/Send operations → POST operations (sending messages)

#### AsyncAPI CLI Flags

- `--output-format asyncapi` - Specify AsyncAPI as output format
- `--asyncapi-version <version>` - AsyncAPI version for output (2.6 or 3.0, default: 2.6)
- `--protocol <protocol>` - Set the protocol for AsyncAPI servers (default: `ws`)

### OpenAPI → API Blueprint

Convert OpenAPI 3.0 JSON to API Blueprint format:

```go
import "github.com/amer8/apibconv/converter"

openapiJSON := `{"openapi": "3.0.0", ...}`
apiBlueprint, err := converter.FromJSONString(openapiJSON)
if err != nil {
    log.Fatal(err)
}
fmt.Println(apiBlueprint)
```

### API Blueprint → OpenAPI

Convert API Blueprint to OpenAPI JSON:

```go
import "github.com/amer8/apibconv/converter"

apibContent := `FORMAT: 1A
# My API
...`

// Convert to OpenAPI 3.0 (default)
openapiJSON, err := converter.ToOpenAPIString(apibContent)
if err != nil {
    log.Fatal(err)
}
fmt.Println(openapiJSON)

// Convert to OpenAPI 3.1
opts := &converter.ConversionOptions{
    OutputVersion: converter.Version31,
}
spec, err := converter.ParseAPIBlueprintWithOptions([]byte(apibContent), opts)
if err != nil {
    log.Fatal(err)
}
// spec.OpenAPI is now "3.1.0"
```

### AsyncAPI → API Blueprint

Convert AsyncAPI to API Blueprint format:

```go
import "github.com/amer8/apibconv/converter"

asyncapiJSON := `{
  "asyncapi": "2.6.0",
  "info": {
    "title": "Chat API",
    "version": "1.0.0"
  },
  "channels": {
    "chat": {
      "subscribe": {
        "message": {
          "payload": {"type": "object"}
        }
      }
    }
  }
}`

spec, err := converter.ParseAsyncAPI([]byte(asyncapiJSON))
if err != nil {
    log.Fatal(err)
}

apiBlueprint := converter.AsyncAPIToAPIBlueprint(spec)
fmt.Println(apiBlueprint)
```

### API Blueprint → AsyncAPI

Convert API Blueprint to AsyncAPI format (v2.6 or v3.0):

```go
import "github.com/amer8/apibconv/converter"

apibContent := `FORMAT: 1A
# Events API

## /events [/events]

### Receive events [GET]

+ Response 200 (application/json)
`

// Parse API Blueprint
spec, err := converter.ParseAPIBlueprint([]byte(apibContent))
if err != nil {
    log.Fatal(err)
}

// Convert to AsyncAPI 2.6 with Kafka protocol
asyncSpec := converter.APIBlueprintToAsyncAPI(spec, "kafka")

// Marshal to JSON
data, err := json.MarshalIndent(asyncSpec, "", "  ")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))

// Or convert to AsyncAPI 3.0
asyncSpecV3 := converter.APIBlueprintToAsyncAPIV3(spec, "kafka")
dataV3, err := json.MarshalIndent(asyncSpecV3, "", "  ")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(dataV3))
```

### Parse and Format

```go
import "github.com/amer8/apibconv/converter"

// Parse OpenAPI
data := []byte(`{"openapi": "3.0.0", ...}`)
spec, err := converter.Parse(data)
if err != nil {
    log.Fatal(err)
}

// Modify the spec programmatically
spec.Info.Title = "My Custom API"

// Format to API Blueprint
apiBlueprint, err := converter.Format(spec)
if err != nil {
    log.Fatal(err)
}
```

### Use the streaming API for large files

```go
import (
    "os"
    "github.com/amer8/apibconv/converter"
)

// OpenAPI → API Blueprint
input, _ := os.Open("openapi.json")
output, _ := os.Create("output.apib")
defer input.Close()
defer output.Close()

err := converter.Convert(input, output)

// API Blueprint → OpenAPI
input2, _ := os.Open("input.apib")
output2, _ := os.Create("output.json")
defer input2.Close()
defer output2.Close()

err = converter.ConvertToOpenAPI(input2, output2)
```

### API Functions

#### Version Conversion Functions
- `ConvertToVersion(spec *OpenAPI, targetVersion Version, opts *ConversionOptions) (*OpenAPI, error)` - Convert between 3.0 and 3.1
- `DetectVersion(openapiField string) Version` - Detect OpenAPI version from spec
- `GetSchemaType(schema *Schema) string` - Get primary type from schema (handles both 3.0 and 3.1)
- `IsNullable(schema *Schema) bool` - Check if schema allows null values

#### Parsing Functions
- `Parse(data []byte) (*OpenAPI, error)` - Parse OpenAPI JSON bytes (3.0 or 3.1)
- `ParseWithConversion(data []byte, opts *ConversionOptions) (*OpenAPI, error)` - Parse and convert to target version
- `ParseReader(r io.Reader) (*OpenAPI, error)` - Parse OpenAPI from reader
- `ParseAPIBlueprint(data []byte) (*OpenAPI, error)` - Parse API Blueprint to OpenAPI 3.0
- `ParseAPIBlueprintWithOptions(data []byte, opts *ConversionOptions) (*OpenAPI, error)` - Parse with version options
- `ParseAPIBlueprintReader(r io.Reader) (*OpenAPI, error)` - Parse API Blueprint from reader

#### Formatting Functions
- `Format(spec *OpenAPI) (string, error)` - Format OpenAPI to API Blueprint string
- `FormatTo(spec *OpenAPI, w io.Writer) error` - Format OpenAPI to API Blueprint writer
- `MustFormat(spec *OpenAPI) string` - Format or panic

#### Conversion Functions (OpenAPI → API Blueprint)
- `FromJSON(data []byte) (string, error)` - JSON bytes to API Blueprint
- `FromJSONString(jsonStr string) (string, error)` - JSON string to API Blueprint
- `ToBytes(data []byte) ([]byte, error)` - JSON bytes to API Blueprint bytes
- `ConvertString(openapiJSON string) (string, error)` - Alias for FromJSONString

#### Conversion Functions (API Blueprint → OpenAPI)
- `ToOpenAPI(data []byte) ([]byte, error)` - API Blueprint bytes to OpenAPI JSON bytes
- `ToOpenAPIString(apibStr string) (string, error)` - API Blueprint string to OpenAPI JSON string

#### I/O Functions
- `Convert(r io.Reader, w io.Writer) error` - Stream OpenAPI → API Blueprint conversion
- `ConvertToOpenAPI(r io.Reader, w io.Writer) error` - Stream API Blueprint → OpenAPI conversion

#### AsyncAPI Functions
- `DetectAsyncAPIVersion(asyncapiVersion string) int` - Detect AsyncAPI version (2 or 3)
- `ParseAsyncAPIAny(data []byte) (interface{}, int, error)` - Parse any AsyncAPI version

**AsyncAPI 2.6 Functions:**
- `ParseAsyncAPI(data []byte) (*AsyncAPI, error)` - Parse AsyncAPI 2.x JSON
- `ParseAsyncAPIReader(r io.Reader) (*AsyncAPI, error)` - Parse AsyncAPI 2.x from reader
- `AsyncAPIToAPIBlueprint(spec *AsyncAPI) string` - Convert AsyncAPI 2.x to API Blueprint
- `APIBlueprintToAsyncAPI(spec *OpenAPI, protocol string) *AsyncAPI` - Convert to AsyncAPI 2.6
- `ConvertAsyncAPIToAPIBlueprint(r io.Reader, w io.Writer) error` - Stream AsyncAPI 2.x → API Blueprint
- `ConvertAPIBlueprintToAsyncAPI(r io.Reader, w io.Writer, protocol string) error` - Stream to AsyncAPI 2.6

**AsyncAPI 3.0 Functions:**
- `ParseAsyncAPIV3(data []byte) (*AsyncAPIV3, error)` - Parse AsyncAPI 3.x JSON
- `ParseAsyncAPIV3Reader(r io.Reader) (*AsyncAPIV3, error)` - Parse AsyncAPI 3.x from reader
- `AsyncAPIV3ToAPIBlueprint(spec *AsyncAPIV3) string` - Convert AsyncAPI 3.x to API Blueprint
- `APIBlueprintToAsyncAPIV3(spec *OpenAPI, protocol string) *AsyncAPIV3` - Convert to AsyncAPI 3.0
- `ConvertAsyncAPIV3ToAPIBlueprint(r io.Reader, w io.Writer) error` - Stream AsyncAPI 3.x → API Blueprint
- `ConvertAPIBlueprintToAsyncAPIV3(r io.Reader, w io.Writer, protocol string) error` - Stream to AsyncAPI 3.0

All functions use zero-allocation buffer pooling internally for optimal memory efficiency.

## Included Examples

The `examples/` directory now contains paired specification files, demonstrating various conversions. Each subdirectory represents a base API or specification, with `.json` and `.apib` files showing the input and converted output.

- **OpenAPI Examples (`examples/openapi/`)**
  - `petstore/`: Classic Petstore example, demonstrating OpenAPI to API Blueprint conversion.
  - `openapi-3.1-validation/`: OpenAPI 3.1 features to API Blueprint.
  - `openapi-advanced-params/`: Advanced OpenAPI parameter usage.
  - `openapi-webhooks/`: OpenAPI 3.1 webhooks conversion.

- **AsyncAPI Examples (`examples/asyncapi/`)**
  - `chat-asyncapi-v2.6/`: WebSocket chat application, demonstrating AsyncAPI 2.6 to API Blueprint conversion.
  - `events-asyncapi-v3.0/`: Kafka event streaming example, showing AsyncAPI 3.0 to API Blueprint conversion.
  - `iot-asyncapi-v2.6-mqtt/`: IoT MQTT telemetry, AsyncAPI 2.6 to API Blueprint conversion.

- **API Blueprint Examples (`examples/apib/`)**
  - `mson-example/`: Demonstrates API Blueprint with MSON data structures, converted to OpenAPI.

To view a conversion:

```sh
# Example: OpenAPI Petstore to API Blueprint
apibconv -f examples/openapi/petstore/petstore.json -o petstore.apib

# Example: API Blueprint with MSON to OpenAPI
apibconv -f examples/apib/mson-example/mson-example.apib -o mson-example.json
```


## Performance

This package is optimized for high performance and low memory usage, utilizing `sync.Pool` for buffer reuse to achieve **zero allocations** for buffer operations.

### Core Benchmarks

```
BenchmarkWriteAPIBlueprint-14     37.3M      32.19 ns/op      0 B/op    0 allocs/op
BenchmarkBufferPool-14            1B+         0.98 ns/op      0 B/op    0 allocs/op
```

### Conversion Throughput

| Conversion | Throughput (avg) |
|------------|------------------|
| OpenAPI 3.0 → API Blueprint | ~115 MB/s |
| API Blueprint → OpenAPI 3.0 | ~42 MB/s |
| API Blueprint → AsyncAPI 2.6 | ~60 MB/s |
| API Blueprint → AsyncAPI 3.0 | ~50 MB/s |

## GitHub Actions Integration

This tool is designed to integrate seamlessly into GitHub Actions workflows. See [ACTIONS.md](ACTIONS.md) for detailed examples.

### Quick Example

```yaml
- name: Convert OpenAPI to API Blueprint
  run: |
    go install github.com/amer8/apibconv@latest
    apibconv -f openapi.json -o api-blueprint.apib
```

### Reusable Workflow

```yaml
jobs:
  convert:
    uses: amer8/apibconv/.github/workflows/reusable-apibconv.yml@main
    with:
      openapi-file: 'openapi.json'
      output-file: 'docs/api-blueprint.apib'
```

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Benchmarks maintain zero allocations
4. Code coverage remains high

## License

See [LICENSE](LICENSE) file for details.
