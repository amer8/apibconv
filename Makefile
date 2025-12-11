.PHONY: test test-coverage test-race coverage-html bench bench-converter bench-compare bench-sizes bench-memory lint lint-install lint-fix doc doc-server doc-converter doc-all build build-cmd validate help

# Running Tests
test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-race:
	go test -v -race ./...

coverage-html: test-coverage
	go tool cover -html=coverage.txt

# Running Benchmarks
bench:
	go test -bench=. -benchmem ./...

bench-converter:
	go test -bench=BenchmarkWriteAPIBlueprint -benchmem ./converter

bench-compare:
	@echo "Run this after saving benchmark results to old.txt:"
	@echo "  go test -bench=. -benchmem ./... > new.txt"
	@echo "  benchcmp old.txt new.txt"
	go test -bench=. -benchmem ./... > new.txt

bench-sizes:
	@echo "Running benchmarks with different spec sizes..."
	go test -bench=BenchmarkConvertOpenAPIToAPIBlueprint_Sizes -benchmem ./converter
	go test -bench=BenchmarkParseOpenAPI_Sizes -benchmem ./converter
	go test -bench=BenchmarkMarshalYAML_Sizes -benchmem ./converter

bench-memory:
	@echo "Running memory profiling benchmarks..."
	go test -bench=BenchmarkMemoryProfile -benchmem ./converter

bench-throughput:
	@echo "Running throughput benchmarks..."
	go test -bench=BenchmarkThroughput -benchmem ./converter

bench-concurrent:
	@echo "Running concurrent benchmarks..."
	go test -bench=BenchmarkConcurrent -benchmem ./converter

bench-all: bench bench-sizes bench-memory bench-throughput bench-concurrent
	@echo "All benchmarks completed"

# Linting
lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

# Building
build:
	go build -o apibconv .

# Validation (using the tool itself)
validate:
	@echo "Run validation on a spec file:"
	@echo "  go run . --validate -f <spec-file>"

# Documentation
doc:
	@echo "Package documentation:"
	@go doc github.com/amer8/apibconv

doc-converter:
	@echo "Converter package documentation:"
	@go doc github.com/amer8/apibconv/converter

doc-all:
	@echo "=== Main Package ==="
	@go doc github.com/amer8/apibconv
	@echo "\n=== Converter Package ==="
	@go doc github.com/amer8/apibconv/converter
	@echo "\n=== Converter Types ==="
	@go doc github.com/amer8/apibconv/converter.Spec
	@go doc github.com/amer8/apibconv/converter.Parse
	@go doc github.com/amer8/apibconv/converter.OpenAPI
	@echo "\n=== OpenAPI → API Blueprint Functions ==="
	@go doc github.com/amer8/apibconv/converter.Convert
	@go doc github.com/amer8/apibconv/converter.FromJSON
	@go doc github.com/amer8/apibconv/converter.FromJSONString
	@echo "\n=== API Blueprint → OpenAPI Functions ==="
	@go doc github.com/amer8/apibconv/converter.ParseAPIBlueprint
	@go doc github.com/amer8/apibconv/converter.ToOpenAPI
	@go doc github.com/amer8/apibconv/converter.ConvertToOpenAPI
	@echo "\n=== Validation Functions ==="
	@go doc github.com/amer8/apibconv/converter.ValidateOpenAPI
	@go doc github.com/amer8/apibconv/converter.ValidateBytes
	@echo "\n=== YAML Functions ==="
	@go doc github.com/amer8/apibconv/converter.MarshalYAML
	@go doc github.com/amer8/apibconv/converter.FormatOpenAPIAsYAML
	@echo "\n=== Error Types ==="
	@go doc github.com/amer8/apibconv/converter.ParseError
	@go doc github.com/amer8/apibconv/converter.ValidationError

doc-server:
	@echo "Starting documentation server at http://localhost:6060"
	@echo "View package docs at: http://localhost:6060/pkg/github.com/amer8/apibconv/"
	@echo "Press Ctrl+C to stop"
	@command -v godoc >/dev/null 2>&1 || { echo "Installing godoc..."; go install golang.org/x/tools/cmd/godoc@latest; }
	godoc -http=:6060

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-race         - Run tests with race detector"
	@echo "  coverage-html     - Generate HTML coverage report"
	@echo ""
	@echo "Benchmarks:"
	@echo "  bench             - Run all benchmarks"
	@echo "  bench-converter   - Run converter benchmarks"
	@echo "  bench-sizes       - Run benchmarks with different spec sizes"
	@echo "  bench-memory      - Run memory profiling benchmarks"
	@echo "  bench-throughput  - Run throughput benchmarks"
	@echo "  bench-concurrent  - Run concurrent benchmarks"
	@echo "  bench-all         - Run all benchmark suites"
	@echo "  bench-compare     - Save benchmark results to new.txt"
	@echo ""
	@echo "Linting:"
	@echo "  lint-install      - Install golangci-lint"
	@echo "  lint              - Run linter"
	@echo "  lint-fix          - Run linter with auto-fix"
	@echo ""
	@echo "Building:"
	@echo "  build             - Build from root main.go"
	@echo ""
	@echo "Documentation:"
	@echo "  doc               - View main package documentation"
	@echo "  doc-converter     - View converter package documentation"
	@echo "  doc-all           - View all package documentation"
	@echo "  doc-server        - Start local documentation server on :6060"
	@echo ""
	@echo "  help              - Show this help message"
