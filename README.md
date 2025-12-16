![apibconv](https://repository-images.githubusercontent.com/1104692802/8033120f-b80a-456c-a993-8dbd79c41716)

[![CI](https://github.com/amer8/apibconv/actions/workflows/ci.yml/badge.svg)](https://github.com/amer8/apibconv/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/amer8/apibconv/branch/main/graph/badge.svg)](https://codecov.io/gh/amer8/apibconv)
[![Go Reference](https://pkg.go.dev/badge/github.com/amer8/apibconv.svg)](https://pkg.go.dev/github.com/amer8/apibconv)
[![Go Report Card](https://goreportcard.com/badge/github.com/amer8/apibconv)](https://goreportcard.com/report/github.com/amer8/apibconv)
[![Go Version](https://img.shields.io/github/go-mod/go-version/amer8/apibconv)](go.mod)
[![Docker Image](https://ghcr-badge.egpl.dev/amer8/apibconv/latest_tag?label=latest&ignore=latest,0,0.1)](https://github.com/amer8/apibconv/pkgs/container/apibconv)
[![License](https://img.shields.io/github/license/amer8/apibconv)](LICENSE)

Convert between API Blueprint (*.apib), OpenAPI 2.0/3.0/3.1, and AsyncAPI 2.x/3.0 specifications.

- [API Blueprint](https://apiblueprint.org/documentation/specification.html) and [MSON](https://apiblueprint.org/documentation/mson/specification.html)
- [AsyncAPI v3.0](https://www.asyncapi.com/docs/reference/specification/v3.0.0)
- [AsyncAPI v2.6](https://v2.asyncapi.com/docs/reference/specification/v2.6.0)
- [OpenAPI v3.1.x](https://swagger.io/specification/)
- [OpenAPI v3.0.x](https://swagger.io/specification/v3/)
- [OpenAPI v2.0 (Swagger)](https://swagger.io/specification/v2/)

## Installation

### Using Go

```sh
go install github.com/amer8/apibconv/cmd/apibconv@latest
```

### Using Docker

**Pull the image**
```sh 
docker pull ghcr.io/amer8/apibconv:latest
```

**Run directly**
```sh
docker run --rm -v $(pwd):/data -w /data ghcr.io/amer8/apibconv -o openapi.json api.apib
```

### Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/amer8/apibconv/releases):


## Usage

### CLI Usage

The tool automatically detects the input format based on file extension and content. It supports both file arguments and stdin.

```sh
Usage: apibconv [OPTIONS] [INPUT_FILE]

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
  apibconv -o output.json spec.apib
  apibconv -o output.yaml --protocol ws spec.apib
  apibconv -o output.json --to openapi --openapi-version 3.1 < spec.apib
  apibconv --validate spec.json
```

### GitHub Actions

This tool is designed to integrate seamlessly into GitHub Actions workflows

```yaml
- name: Convert OpenAPI to API Blueprint
  run: |
    go install github.com/amer8/apibconv@latest
    apibconv -o api-blueprint.apib openapi.json
```

### Go projects

This tool is designed for seamless integration with servers and CLI tools built in Go.

- Basic [example](./examples/basic/main.go)
- Advanced [example](./examples/advanced/main.go)
- Plugin [example](./examples/plugin/main.go)

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Code coverage remains high

## License

See [LICENSE](LICENSE) file for details.
