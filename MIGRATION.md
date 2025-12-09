# Migration Guide

This document outlines important changes and how to migrate your usage of `apibconv` to newer versions.

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
