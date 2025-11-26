# Contributing to apibconv

Thank you for your interest in contributing to apibconv! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

This project follows a standard code of conduct. Please be respectful and professional in all interactions.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally: `git clone https://github.com/YOUR_USERNAME/apibconv.git`
3. Add the upstream repository: `git remote add upstream https://github.com/amer8/apibconv.git`

## Development Setup

### Prerequisites

- Go 1.20 or higher
- golangci-lint (for linting)
- Git

### Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install golangci-lint (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Build the Project

```bash
# Build the binary
go build -o apibconv .

# Or use make
make build
```

## Project Structure

```
apibconv/
├── converter/           # Core conversion logic
│   ├── api.go          # Public API functions
│   ├── converter.go    # OpenAPI → API Blueprint conversion
│   ├── parser.go       # API Blueprint → OpenAPI parsing
│   ├── asyncapi.go     # AsyncAPI support (v2.6 & v3.0)
│   ├── version*.go     # Version conversion (3.0 ↔ 3.1)
│   ├── pool.go         # Buffer pooling for performance
│   └── *_test.go       # Test files
├── examples/           # Example files and usage
├── main.go             # CLI implementation
├── Makefile            # Build and development tasks
└── README.md           # Project documentation
```

## Development Workflow

### Creating a Feature Branch

```bash
# Update your fork
git fetch upstream
git checkout main
git merge upstream/main

# Create a feature branch
git checkout -b feature/your-feature-name
```

### Making Changes

1. **Write Code**: Make your changes in the appropriate files
2. **Add Tests**: Ensure your changes are covered by tests
3. **Run Tests**: Verify all tests pass (see [Testing](#testing))
4. **Run Linter**: Ensure code passes linting
5. **Commit**: Write clear, descriptive commit messages

### Commit Message Guidelines

Use clear and descriptive commit messages:

```
<type>: <subject>

<optional body>

<optional footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

Examples:
```
feat: add support for AsyncAPI 3.0

fix: resolve shallow copy mutation in version conversion

test: add comprehensive tests for NormalizeSchemaType

perf: pre-compile regular expressions for 10% parser speedup
```

## Testing

### Running Tests

```bash
# Run all tests
make test
# OR
go test ./...

# Run tests with coverage
make test-coverage
# OR
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# View coverage report in browser
make coverage-html

# Run tests with race detector
make test-race
# OR
go test -v -race ./...
```

### Writing Tests

- Place tests in `*_test.go` files alongside the code they test
- Use table-driven tests for multiple test cases
- Aim for high test coverage (current: 87.4% in converter package)
- Include edge cases and error conditions

Example table-driven test:

```go
func TestYourFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid input",
			input:    "test",
			expected: "test",
			wantErr:  false,
		},
		// Add more test cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YourFunction(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
```

### Running Benchmarks

```bash
# Run all benchmarks
make bench
# OR
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkWriteAPIBlueprint -benchmem ./converter
```

## Code Style

### Linting

This project uses golangci-lint with the configuration in `.golangci.yml`.

```bash
# Run linter
make lint
# OR
golangci-lint run

# Auto-fix issues when possible
make lint-fix
# OR
golangci-lint run --fix
```

### Style Guidelines

1. **Follow Go conventions**: Use `gofmt` and standard Go style
2. **Document exports**: All exported functions, types, and constants must have doc comments
3. **Keep functions focused**: Each function should do one thing well
4. **Limit complexity**: Keep cyclomatic complexity under 15 (enforced by linter)
5. **Error handling**: Always check and properly handle errors
6. **Performance**: Use buffer pooling and avoid unnecessary allocations
7. **Thread safety**: Document thread-safety guarantees (all converter functions are safe for concurrent use)

### Documentation Format

```go
// FunctionName does something useful with the provided input.
//
// A longer description can provide additional context, usage examples,
// and important notes about behavior.
//
// Parameters:
//   - input: Description of the input parameter
//   - options: Description of options (can be nil)
//
// Returns:
//   - string: Description of the return value
//   - error: Error if the operation fails
//
// Example:
//
//	result, err := FunctionName("input", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result)
func FunctionName(input string, options *Options) (string, error) {
	// Implementation
}
```

## Pull Request Process

### Before Submitting

1. **Run all tests**: `make test`
2. **Run linter**: `make lint`
3. **Run benchmarks**: Verify no performance regressions for critical paths
4. **Update documentation**: Update README.md if adding new features
5. **Add examples**: Include usage examples for new features

### Submitting a Pull Request

1. **Push your branch**: `git push origin feature/your-feature-name`
2. **Create PR**: Open a pull request on GitHub
3. **Fill in template**: Provide a clear description of changes
4. **Link issues**: Reference any related issues
5. **Wait for review**: Maintainers will review your PR

### PR Description Template

```markdown
## Description
Brief description of what this PR does

## Motivation
Why is this change needed?

## Changes
- List key changes
- Use bullet points

## Testing
- Describe how you tested the changes
- Include test coverage information

## Checklist
- [ ] Tests pass locally
- [ ] Linter passes
- [ ] Documentation updated
- [ ] Benchmarks verified (if applicable)
```

### Review Process

- Maintainers will review your PR within a few days
- Address any feedback or requested changes
- Once approved, your PR will be merged

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

1. **Description**: Clear description of the bug
2. **Steps to reproduce**: Detailed steps to reproduce the issue
3. **Expected behavior**: What you expected to happen
4. **Actual behavior**: What actually happened
5. **Environment**: Go version, OS, apibconv version
6. **Input files**: Sample input files if relevant (or minimal reproduction)

### Feature Requests

For feature requests, please include:

1. **Use case**: Describe the problem you're trying to solve
2. **Proposed solution**: Your ideas for how to implement it
3. **Alternatives**: Other approaches you've considered
4. **Additional context**: Any other relevant information

## Performance Considerations

This project prioritizes performance:

- **Zero allocations**: Buffer operations use `sync.Pool` for zero allocations
- **Streaming API**: Large files can be processed without loading into memory
- **Benchmarks**: All performance-critical code has benchmark tests
- **Regression testing**: PRs should not introduce performance regressions

### Benchmark Targets

```
BenchmarkWriteAPIBlueprint: ~73 ns/op, 0 allocs/op
BenchmarkBufferPool:        ~1.75 ns/op, 0 allocs/op
```

## Questions?

If you have questions not covered in this guide:

- Check existing issues and pull requests
- Read the documentation in [README.md](README.md)
- Open a new issue with your question

Thank you for contributing to apibconv! 🎉
