# Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Code coverage remains high

Format Go code with `gofmt` and follow the conventions in
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

The release helper's policy tests are included in `go test ./...`. To run them
separately, use `go test ./internal/releasetool`.

After changing dependencies, embedded schemas, notices, or the Go toolchain,
run `make licenses`. Review the generated reports and third-party notices;
`make licenses-check` verifies the
reports for all supported release platforms. The tool's classification is a
starting point and can miss terms in mixed-license projects.
The generated reports are checked in so normal source builds and `go install`
also embed notices. CI, release builds, and container builds verify that the
relevant reports are current. Builds for other platforms use a clearly marked
fallback rather than a platform-specific dependency inventory.

For the release process, see the [release guide](docs/releasing.md).
