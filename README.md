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
go install github.com/amer8/apibconv@latest
```

### Using Docker

**Pull the image**
```sh 
docker pull ghcr.io/amer8/apibconv:latest
```

**Run directly**
```sh
docker run --rm -v $(pwd):/data -w /data ghcr.io/amer8/apibconv api.apib -o openapi.json
```

### Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/amer8/apibconv/releases):


## Usage

### CLI Usage

The tool automatically detects the input format based on file extension and content. It supports both file arguments and stdin.

```sh
Usage: apibconv [INPUT_FILE] [OPTIONS]

Arguments:
  INPUT_FILE
      Input specification file (OpenAPI, AsyncAPI, or API Blueprint)

Options:
  -o, --output FILE
      Output file path (required for conversion)
  
  --to FORMAT
      Target format: openapi, asyncapi, apib
      Auto-detected from --output extension if not specified
  
  -e, --encoding FORMAT
      Output encoding: json, yaml (default: auto-detected from output extension)
  
  --validate
      Validate input without converting
  
  -v, --version
      Print version information
  
  -h, --help
      Show this help message

AsyncAPI Options:
  --protocol PROTO
      Protocol: ws, wss, mqtt, kafka, amqp, http, https, auto (required)
  
  --asyncapi-version VERSION
      Version: 2.6, 3.0 (default: "2.6")

OpenAPI Options:
  --openapi-version VERSION
      Version: 3.0, 3.1 (default: "3.0")

Examples:
  apibconv spec.apib -o output.json
  apibconv spec.apib -o output.yaml --protocol ws
  apibconv -o output.json --to openapi --openapi-version 3.1 < spec.apib
  apibconv spec.json --validate
```

### OpenAPI Support

The tool supports both OpenAPI 3.0 and 3.1:

- **Versions Supported**:
  - OpenAPI 3.0 (default output)
  - OpenAPI 3.1
  - Note: OpenAPI/Swagger 2.x is not supported
- **OpenAPI 3.1**: Use `--openapi-version 3.1` flag for 3.1.0 output
- **Input**: Automatically detects and handles both 3.0 and 3.1 input specs

**Key Differences**

When converting to OpenAPI 3.1:
- Nullable types use type arrays: `["string", "null"]` instead of `nullable: true`
- Supports new fields: `webhooks`, `jsonSchemaDialect`, `license.identifier`
- Full JSON Schema 2020-12 compatibility


### AsyncAPI Support

The tool supports AsyncAPI 2.6 and 3.0 for event-driven APIs:

- **Versions Supported**:
  - AsyncAPI 2.6 (default output)
  - AsyncAPI 3.0
  - Note: AsyncAPI 1.x is not supported
- **Protocols**: WebSocket (`ws`, `wss`), MQTT (`mqtt`), Kafka (`kafka`), AMQP (`amqp`), HTTP (`http`, `https`), Auto (`auto`)
- **Conversion Mappings**:
  - **AsyncAPI 2.6**: Channels contain publish/subscribe operations
  - **AsyncAPI 3.0**: Operations at root level with send/receive actions
  - AsyncAPI channels → API Blueprint paths
  - Subscribe/Receive operations → GET operations (receiving messages)
  - Publish/Send operations → POST operations (sending messages)


## Examples

The `examples/` directory contains paired specification files, demonstrating various conversions. Each subdirectory represents a base API or specification, with `.json`, `.yml` and `.apib` files showing the input and converted output.

<details>
<summary>Unified Parsing (OpenAPI, AsyncAPI, Blueprint)</summary>

```go
import "github.com/amer8/apibconv/converter"

// OpenAPI (JSON)
spec, _ := converter.Parse([]byte(`{"openapi": "3.0.0", ...}`))
if openapiSpec, ok := spec.AsOpenAPI(); ok {
    fmt.Println(openapiSpec.Info.Title)
}

// AsyncAPI (YAML)
spec, _ := converter.Parse([]byte(`asyncapi: 2.6.0\n...`))
if asyncapiSpec, ok := spec.AsAsyncAPI(); ok {
    fmt.Println(asyncapiSpec.Info.Title)
}

// API Blueprint
spec, _ := converter.Parse([]byte(`FORMAT: 1A...`))
// Access via API Blueprint AST
if apibSpec, ok := spec.AsAPIBlueprint(); ok {
    fmt.Println(apibSpec.Name)
}
```
</details>

<details>
<summary>API Blueprint → OpenAPI</summary>

```go
import "github.com/amer8/apibconv/converter"

apibContent := `FORMAT: 1A
# My API
...`

// Parse API Blueprint
spec, err := converter.Parse([]byte(apibContent))
if err != nil {
    log.Fatal(err)
}

// Access as API Blueprint AST
apibSpec, ok := spec.AsAPIBlueprint()
if !ok {
    log.Fatal("Expected API Blueprint spec")
}

// Convert to OpenAPI 3.0 (default)
openapiSpec, err := apibSpec.ToOpenAPI()
if err != nil {
    log.Fatal(err)
}

// Serialize to JSON
data, err := json.MarshalIndent(openapiSpec, "", "  ")
fmt.Println(string(data))
```
</details>

<details>
<summary>API Blueprint → AsyncAPI</summary>

```go
import "github.com/amer8/apibconv/converter"

apibContent := `FORMAT: 1A
# Events API
...`

// Parse API Blueprint
spec, err := converter.Parse([]byte(apibContent))
if err != nil {
    log.Fatal(err)
}

// Access as API Blueprint AST
apibSpec, ok := spec.AsAPIBlueprint()
if !ok {
    log.Fatal("Expected API Blueprint spec")
}

// Convert to AsyncAPI 2.6 with Kafka protocol
asyncSpec, err := apibSpec.ToAsyncAPI(converter.ProtocolKafka)
if err != nil {
    log.Fatal(err)
}

// Marshal to JSON
data, err := json.MarshalIndent(asyncSpec, "", "  ")
fmt.Println(string(data))
```
</details>

<details>
<summary>OpenAPI → API Blueprint</summary>

```go
import "github.com/amer8/apibconv/converter"

openapiJSON := `{"openapi": "3.0.0", ...}`

// Parse OpenAPI
spec, err := converter.Parse([]byte(openapiJSON))
if err != nil {
    log.Fatal(err)
}

if openapiSpec, ok := spec.AsOpenAPI(); ok {
    // Convert to API Blueprint
    apibSpec, err := openapiSpec.ToBlueprint()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(apibSpec)
}
```
</details>


<details>
<summary>AsyncAPI → API Blueprint</summary>

```go
import "github.com/amer8/apibconv/converter"

asyncapiJSON := `{
  "asyncapi": "2.6.0",
  ...
}`

// Parse AsyncAPI (auto-detects version)
spec, err := converter.Parse([]byte(asyncapiJSON))
if err != nil {
    log.Fatal(err)
}

if asyncApiSpec, ok := spec.AsAsyncAPI(); ok {
    // Convert to API Blueprint
    apibSpec, err := asyncApiSpec.ToBlueprint()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(apibSpec)
}
```
</details>


## GitHub Actions Integration

This tool is designed to integrate seamlessly into GitHub Actions workflows

```yaml
- name: Convert OpenAPI to API Blueprint
  run: |
    go install github.com/amer8/apibconv@latest
    apibconv openapi.json -o api-blueprint.apib
```


## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Code coverage remains high

## License

See [LICENSE](LICENSE) file for details.
