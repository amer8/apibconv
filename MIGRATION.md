# Migration Guide

This document outlines important changes and how to migrate your usage of `apibconv` to newer versions.

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

-   **Output Specification Format (OpenAPI/AsyncAPI/APIB):**
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
apibconv -f api.apib -o asyncapi.json --to asyncapi --asyncapi-version 3.0
apibconv -v
```

### Stdin Support

`apibconv` now supports reading the input specification from `stdin` when no input file is specified via `-f` or as a positional argument.

**Example:**

```sh
cat openapi.json | apibconv -o api.apib
```
