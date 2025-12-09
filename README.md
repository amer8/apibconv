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
go install github.com/amer8/apibconv@v0.1.4
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


## Usage

### CLI Usage

The tool automatically detects the input format and converts accordingly. It supports both file arguments and stdin.

```sh
# Usage
apibconv -f <input-file> -o <output-file>
apibconv <input-file> -o <output-file>

# Input Formats (auto-detected from file extension or content)
apibconv -f petstore.json -o petstore.apib          # OpenAPI JSON → API Blueprint
apibconv -f openapi.yaml -o petstore.apib           # OpenAPI YAML → API Blueprint
apibconv -f asyncapi.json -o api.apib               # AsyncAPI → API Blueprint
apibconv -f petstore.apib -o openapi.json           # API Blueprint → OpenAPI (default 3.0)
apibconv -f petstore.apib -o openapi.yaml

# Version Control
apibconv -f petstore.apib -o openapi.json --openapi-version 3.1
apibconv -f api.apib -o asyncapi.json --to asyncapi --asyncapi-version 3.0

# Protocol Selection (AsyncAPI only)
apibconv -f api.apib -o asyncapi.json --to asyncapi --protocol ws|kafka|mqtt|http|amqp

# Explicit Output Encoding
apibconv -f api.apib -o openapi.yaml -e yaml

# Validation Only
apibconv -f <file> --validate

# Pipe Support (Stdin)
cat openapi.json | apibconv -o api.apib
```

### OpenAPI Support

The tool supports both OpenAPI 3.0 and 3.1:

- **Default**: Outputs OpenAPI 3.0.0 for backward compatibility
- **OpenAPI 3.1**: Use `--openapi-version 3.1` flag for 3.1.0 output
- **Input**: Automatically detects and handles both 3.0 and 3.1 input specs

**Key Differences**

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

**CLI Flags**

- `--to asyncapi` - Specify AsyncAPI as output format
- `--asyncapi-version <version>` - AsyncAPI version for output (2.6 or 3.0, default: 2.6)
- `--protocol <protocol>` - Set the protocol for AsyncAPI servers

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

## Included Examples

The `examples/` directory now contains paired specification files, demonstrating various conversions. Each subdirectory represents a base API or specification, with `.json` and `.apib` files showing the input and converted output.

To view a conversion:

```sh
# Example: OpenAPI Petstore to API Blueprint
apibconv -f examples/openapi/petstore/petstore.json -o petstore.apib

# Example: API Blueprint with MSON to OpenAPI
apibconv -f examples/apib/mson-example/mson-example.apib -o mson-example.json
```

## GitHub Actions Integration

This tool is designed to integrate seamlessly into GitHub Actions workflows

```yaml
- name: Convert OpenAPI to API Blueprint
  run: |
    go install github.com/amer8/apibconv@latest
    apibconv -f openapi.json -o api-blueprint.apib
```


## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Code coverage remains high

## License

See [LICENSE](LICENSE) file for details.
