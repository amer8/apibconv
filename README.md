# apibconv

![API Blueprint to OpenAPI migration demo](docs/demo/apibconv.gif)

[![CI](https://github.com/amer8/apibconv/actions/workflows/ci.yml/badge.svg)](https://github.com/amer8/apibconv/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/amer8/apibconv.svg)](https://pkg.go.dev/github.com/amer8/apibconv)
[![Go Version](https://img.shields.io/github/go-mod/go-version/amer8/apibconv)](https://github.com/amer8/apibconv/blob/main/go.mod)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/amer8/apibconv/badge)](https://scorecard.dev/viewer/?uri=github.com/amer8/apibconv)
[![License](https://img.shields.io/github/license/amer8/apibconv)](LICENSE)

Migrate API Blueprint and OpenAPI specifications to newer versions.

- [API Blueprint](https://apiblueprint.org/documentation/specification.html) and [MSON](https://apiblueprint.org/documentation/mson/specification.html)
- [OpenAPI v3.1.x](https://swagger.io/specification/)
- [OpenAPI v3.0.x](https://swagger.io/specification/v3/)
- [OpenAPI v2.0 (Swagger)](https://swagger.io/specification/v2/)

## Migrating from Apiary

Oracle announced the end of life of Apiary Cloud Service effective October 31, 2026,
and recommends converting API Blueprint specifications to OpenAPI. See the
[Oracle service change announcement](https://docs.oracle.com/en-us/iaas/Content/servicechanges.htm#oracle-apiary)
for details. apibconv can help convert saved API Blueprint files to OpenAPI.

Supported conversions through the CLI and `Converter.Convert`:

- API Blueprint → OpenAPI 3.1
- OpenAPI 2.0 or 3.0 → OpenAPI 3.1
- OpenAPI → the same major/minor version, including JSON/YAML conversion

Downgrades and other cross-format conversions are rejected. The default output
version is OpenAPI 3.1. For a same-version conversion to an
older version, specify `--openapi-version` explicitly.

**Conversion is not lossless**, even when only changing the encoding. The API
Blueprint parser supports a subset of the format, and some fields may be omitted
or changed. Keep the originals, validate the output, and review it against the
source before using it as an API contract. For supported conversions, reference
handling, and known limitations, see [conversion behavior](docs/conversion.md).

## Installation

The CLI supports Linux and macOS. On Windows, use the Linux CLI from
[Windows Subsystem for Linux (WSL)](https://learn.microsoft.com/en-us/windows/wsl/install).
Native Windows binaries and Windows GitHub Actions runners are not supported;
use a Linux runner for the install action. In WSL, keep input and output files
in the Linux filesystem for the most consistent file-permission behavior.

### Using Homebrew

Install from the [amer8/homebrew-tap](https://github.com/amer8/homebrew-tap):

```sh
brew tap amer8/tap
brew install amer8/tap/apibconv
```

### Using Go

Building from source requires Go 1.26 or newer. CI tests Go 1.26 and Go 1.27.

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
docker run --rm -v "$(pwd):/data" -w /data ghcr.io/amer8/apibconv -o openapi.json api.apib
```

### Download a binary

Download pre-built binaries from [GitHub Releases](https://github.com/amer8/apibconv/releases).

Prebuilt macOS binaries require **macOS 13 Ventura or newer**. See
[Release verification](docs/release-verification.md) for checksums and attestations.

## Usage

### CLI Usage

The input format is detected from the content, or can be set with `--from`.
Output files default to OpenAPI; when writing to stdout, specify `--to openapi`.
The output file extension selects JSON or YAML unless `--encoding` or an
environment variable overrides it. See `apibconv --help` for all options.

```sh
apibconv --api-version 1.0.0 -o openapi.yaml spec.apib
apibconv -o openapi.json swagger.yaml
apibconv --to openapi --openapi-version 3.0 -o output.json input.yaml
apibconv --to openapi -e json < spec.apib > openapi.json
apibconv --validate spec.json
apibconv --validate --report json spec.json
apibconv --validate-output -o openapi.yaml api.apib
```

`--validate` checks input without converting it. By default, generated OpenAPI
is also validated before it is written. Validation cannot guarantee a lossless
conversion; review the generated contract and its API version. API Blueprint
input has no API version, so use `--api-version` to provide one.

See the [CLI reference](docs/cli.md) for validation reports, option precedence,
and file-replacement behavior.

### GitHub Actions

For GitHub Actions, prefer the first-party install action over `go install ...@latest`.
Pin the action itself to a full commit SHA and pin the binary version to a release tag.

```yaml
- name: Install apibconv
  uses: amer8/apibconv@<full-commit-sha>
  with:
    version: vX.Y.Z

- name: Migrate API Blueprint to OpenAPI
  run: apibconv -o openapi.json api-blueprint.apib
```

The action downloads the published release asset for the current runner, verifies its SHA-256 checksum,
and when the release includes a Sigstore bundle it also verifies the checksum signature with cosign
against this repository's release workflow identity.
Older releases without a bundle fall back to checksum-only verification with a warning.
Published action assets target `x64` and `arm64` runners.

### Go projects

Use the Go packages to integrate conversion into your own application. See the
[Go integration guide](docs/go-integration.md) and the
[Go reference](https://pkg.go.dev/github.com/amer8/apibconv/pkg/converter).

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development
checks and the [release guide](docs/releasing.md) for maintainer procedures.

## License

apibconv is licensed under the [MIT License](LICENSE). It includes third-party
software and OpenAPI validation schemas under their respective licenses. See
[third-party notices](THIRD_PARTY_NOTICES.md) for details. Run
`apibconv --licenses` to display the notices and license texts embedded in the
executable.
