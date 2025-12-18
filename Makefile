.PHONY: test test-coverage test-race coverage-html bench bench-all lint lint-install lint-fix doc doc-server build validate clean help

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

bench-all: bench
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
	go build -o apibconv ./cmd/apibconv

# Generate expected test data
gen-expected: build
	./scripts/gen_expected.sh

clean:
	rm -f apibconv coverage.txt

# Validation (using the tool itself)
validate:
	@echo "Run validation on a spec file:"
	@echo "  ./apibconv --validate <spec-file>"

# Documentation
doc:
	@echo "Package documentation:"
	@go doc github.com/amer8/apibconv/pkg/converter
	@go doc github.com/amer8/apibconv/pkg/model
	@go doc github.com/amer8/apibconv/pkg/format

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
	@echo ""
	@echo "Linting:"
	@echo "  lint-install      - Install golangci-lint"
	@echo "  lint              - Run linter"
	@echo "  lint-fix          - Run linter with auto-fix"
	@echo ""
	@echo "Building:"
	@echo "  build             - Build the CLI tool"
	@echo "  gen-expected      - Generate expected test data for integration tests"
	@echo "  clean             - Remove build artifacts"
	@echo ""
	@echo "Documentation:"
	@echo "  doc               - View package documentation"
	@echo "  doc-server        - Start local documentation server on :6060"
	@echo ""
	@echo "  help              - Show this help message"
