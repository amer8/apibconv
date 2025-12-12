# Migration Guide

This document outlines important changes and how to migrate your usage of `apibconv` to newer versions.

## From 0.4.x to 0.5.x

Introducing a dedicated `APIBlueprint` struct (AST) for representing API Blueprint documents. Previously, parsing an API Blueprint resulted in an `*OpenAPI` struct.

**Example Migration:**

```go
// Old (0.4.x):
spec, _ := converter.Parse([]byte(apibContent), converter.FormatBlueprint)
// API Blueprint was parsed into an OpenAPI struct
if openapi, ok := spec.AsOpenAPI(); ok {
    fmt.Println(openapi.Info.Title)
}

// New (0.5.x):
spec, _ := converter.Parse([]byte(apibContent), converter.FormatBlueprint)
// Access the dedicated APIBlueprint AST
if bp, ok := spec.AsAPIBlueprint(); ok {
    fmt.Println(bp.Name)
}

// If you need the OpenAPI representation, convert it explicitly:
if bp, ok := spec.AsAPIBlueprint(); ok {
    openapi, _ := bp.ToOpenAPI()
    fmt.Println(openapi.Info.Title)
}
```

## From 0.3.x to 0.4.x

Introducing an unified `converter.Parse` and `converter.Spec` Interface.

**Example Migration (Parsing and Conversion):**

```go
// Old: Specific parse and direct conversion functions
// OpenAPI to API Blueprint
// openapiJSON := `{"openapi": "3.0.0", ...}`
// apiBlueprint, err := converter.FromJSONString(openapiJSON)

// API Blueprint to OpenAPI
// apibContent := `FORMAT: 1A\n# My API\n...`
// openapiJSON, err := converter.ToOpenAPIString(apibContent)

// AsyncAPI to API Blueprint
// asyncapiJSON := `{"asyncapi": "2.6.0", ...}`
// specAsync, err := converter.ParseAsync([]byte(asyncapiJSON))
// bpAsync := specAsync.ToBlueprint()

// New: Unified Parse and Spec interface methods
// OpenAPI to API Blueprint
spec, err := converter.Parse([]byte(openapiJSON), converter.FormatOpenAPI)
if err != nil { ... }
apiBlueprint, err = spec.ToBlueprint()

// API Blueprint to OpenAPI
spec, err := converter.Parse([]byte(apibContent), converter.FormatBlueprint)
if err != nil { ... }
openapiSpec, err := spec.ToOpenAPI()

// AsyncAPI to API Blueprint
spec, err := converter.Parse([]byte(asyncapiJSON), converter.FormatAsyncAPI) // Auto-detects 2.x or 3.x
if err != nil { ... }
apiBlueprint, err = spec.ToBlueprint()

// To access specific fields of the underlying struct (e.g., spec.Info.Title)
// a type assertion is required:
if openapiSpecConcrete, ok := openapiSpec.(*converter.OpenAPI); ok {
    fmt.Println(openapiSpecConcrete.Info.Title)
}
```

### Strongly Typed `converter.Protocol`

The `protocol` parameter in `ToAsyncAPI()` and `ToAsyncAPIV3()` methods now uses the new `converter.Protocol` type, with predefined constants for clarity and type safety.

**Example Migration:**

```go
import "github.com/amer8/apibconv/converter"

// Old:
// asyncSpec, err := specAPIB.ToAsyncAPI("kafka")

// New:
asyncSpec, err := specAPIB.ToAsyncAPI(converter.ProtocolKafka)
```


## From 0.2.x to 0.3.x

### API Refactoring

In v0.3.0, the Go API has been refactored to be more idiomatic, using methods on struct types and cleaner function names.

**Function Renaming:**

| Old Function | New Function |
|--------------|--------------|
| `ParseAPIBlueprint` | `ParseBlueprint` |
| `ParseAPIBlueprintReader` | `ParseBlueprintReader` |
| `ParseAPIBlueprintWithOptions` | `ParseBlueprintWithOptions` |
| `ParseAsyncAPI` | `ParseAsync` |
| `ParseAsyncAPIReader` | `ParseAsyncReader` |
| `ParseAsyncAPIV3` | `ParseAsyncV3` |
| `ParseAsyncAPIV3Reader` | `ParseAsyncV3Reader` |

**Method Conversions:**

Conversion functions have been converted to methods on the respective structs:

| Old Function | New Method |
|--------------|------------|
| `AsyncAPIToAPIBlueprint(spec)` | `spec.ToBlueprint()` |
| `AsyncAPIV3ToAPIBlueprint(spec)` | `spec.ToBlueprint()` |
| `APIBlueprintToAsyncAPI(spec, protocol)` | `spec.ToAsyncAPI(protocol)` |
| `APIBlueprintToAsyncAPIV3(spec, protocol)` | `spec.ToAsyncAPIV3(protocol)` |
| `Format(spec)` (OpenAPI) | `spec.ToBlueprint()` (returns string) or `spec.WriteBlueprint(w)` |

**Example Migration:**

```go
// Old:
spec, _ := converter.ParseAsyncAPI(data)
bp := converter.AsyncAPIToAPIBlueprint(spec)

// New:
spec, _ := converter.ParseAsync(data)
bp := spec.ToBlueprint()
```

The old functions are kept as deprecated aliases for backward compatibility but may be removed in future versions.


## From 0.1.x to 0.2.x

### CLI Flag Changes

Starting with v0.2.0, several CLI flags have been updated to provide a more consistent and user-friendly experience, including short-form aliases and clearer naming.

**Key Changes:**

-   **Input File:**
    -   Old: `-f <file>`
    -   New: `-f <file>`, `--file <file>` (positional argument also supported: `apibconv <file> ...`)

-   **Output File:**
    -   Old: `-o <file>`
    -   New: `-o <file>`, `--output <file>`

-   **Output Encoding Format (JSON/YAML):**
    -   Old: `--format <format>`
    -   New: `-e <format>`, `--encoding <format>`

-   **Output Specification Format (OpenAPI/AsyncAPI/APIB):n**
    -   Old: `--output-format <format>`
    -   New: `--to <format>`

-   **Show Version:**
    -   Old: `-version`
    -   New: `-v`, `--version`

-   **Show Help:**
    -   New: `-h`, `--help`

**Example Migrations:**

```sh
# Old:
apibconv -f api.apib -o openapi.json --format yaml
apibconv -f api.apib -o asyncapi.json --output-format asyncapi --asyncapi-version 3.0
apibconv -version

# New:
apibconv api.apib -o openapi.yaml -e yaml
apibconv api.apib -o asyncapi.json --to asyncapi --asyncapi-version 3.0
apibconv -v
```

### Stdin Support

`apibconv` now supports reading the input specification from `stdin` when no input file is specified via `-f` or as a positional argument.

**Example:**

```sh
cat openapi.json | apibconv -o api.apib
```
