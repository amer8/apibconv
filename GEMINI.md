# apibconv

`apibconv` is a command-line tool written in Go for converting between various API specification formats. It acts as a bridge between synchronous (REST) and asynchronous (Event-driven) API descriptions.

## Supported Formats

The tool supports bidirectional conversion between the following formats:

*   **API Blueprint** (*.apib)
*   **OpenAPI**
    *   v2.0 (Swagger)
    *   v3.0.x
    *   v3.1.x
*   **AsyncAPI**
    *   v2.6
    *   v3.0

## Architecture

The project follows a standard Go project layout:

*   **`cmd/apibconv/`**: Contains the `main.go` entry point for the CLI application.
*   **`internal/`**: Private application code.
    *   `cli/`: CLI application logic, flag parsing, and execution runner.
    *   `detect/`: Logic for auto-detecting file formats and versions.
    *   `transform/`: Core transformation logic.
*   **`pkg/`**: Library code that could potentially be used by other projects.
    *   `converter/`: The high-level conversion orchestration.
    *   `model/`: The internal intermediate representation (IR) used to map between different formats. This is the core "hub" of the conversion process.
    *   `format/`: Contains parsers and writers for each specific format (apiblueprint, openapi, asyncapi).
    *   `validator/`: Validation logic for specifications.
*   **`test/`**: Integration tests and a rich set of test data in `testdata/`.

## Build and Run

### Prerequisites

*   Go (latest version recommended)
*   Make (optional, for convenience commands)

### Common Commands

The project uses a `Makefile` to simplify common development tasks:

*   **Build:**
    ```bash
    make build
    # Output binary will be 'apibconv'
    ```

*   **Run via Go:**
    ```bash
    go run cmd/apibconv/main.go [flags] [input_file]
    ```

*   **Test:**
    ```bash
    make test
    # Or manually: go test -v ./...
    ```

*   **Lint:**
    ```bash
    make lint
    # Requires golangci-lint
    ```

*   **Clean:**
    ```bash
    make clean
    ```

## Usage Examples

**Convert API Blueprint to OpenAPI 3.0 (JSON):**
```bash
./apibconv -o output.json input.apib
```

**Convert with specific version target:**
```bash
./apibconv -o output.yaml --to openapi --openapi-version 3.1 input.apib
```

**Validate a spec without converting:**
```bash
./apibconv --validate spec.json
```

## Key Files

*   `README.md`: General documentation and usage guide.
*   `Makefile`: Build and test automation.
*   `go.mod`: Go module definitions and dependencies.
*   `pkg/model/api.go`: Defines the central data structures for the API model.
