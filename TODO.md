# ToDo List

## Context Handling Improvements

- [x] Add `ctx.Err()` checks at the beginning of `Parse` and `Write` methods in:
    - `pkg/format/apiblueprint/parser.go`
    - `pkg/format/apiblueprint/writer.go`
    - `pkg/format/asyncapi/parser.go`
    - `pkg/format/asyncapi/writer.go`
    - `pkg/format/openapi/writer.go`
- [x] Investigate replacing `io.ReadAll` with context-aware reading mechanisms (e.g., using `bufio.Reader` with context checks in a loop) in:
    - `pkg/format/openapi/parser.go`
    - `pkg/format/asyncapi/parser.go`
    - `internal/detect/detect.go`
- [x] Re-evaluate the `ctx.Err()` check in `pkg/format/openapi/v2.go`. If `io.ReadAll` is replaced upstream, this check might become effective; otherwise, it's currently redundant due to `io.ReadAll` being called before it.