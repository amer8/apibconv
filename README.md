![apibconv](https://repository-images.githubusercontent.com/1104692802/990c0067-778c-40bd-be06-a5148304fe3f)

[![CI](https://github.com/amer8/apibconv/actions/workflows/ci.yml/badge.svg)](https://github.com/amer8/apibconv/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/amer8/apibconv.svg)](https://pkg.go.dev/github.com/amer8/apibconv)
[![Go Version](https://img.shields.io/github/go-mod/go-version/amer8/apibconv)](go.mod)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/amer8/apibconv/badge)](https://scorecard.dev/viewer/?uri=github.com/amer8/apibconv)
[![License](https://img.shields.io/github/license/amer8/apibconv)](LICENSE)

Convert between API Blueprint (*.apib), OpenAPI 2.0/3.0/3.1, and AsyncAPI 2.x/3.0 specifications.

- [API Blueprint](https://apiblueprint.org/documentation/specification.html) and [MSON](https://apiblueprint.org/documentation/mson/specification.html)
- [AsyncAPI v3.0](https://www.asyncapi.com/docs/reference/specification/v3.0.0)
- [AsyncAPI v2.6](https://v2.asyncapi.com/docs/reference/specification/v2.6.0)
- [OpenAPI v3.1.x](https://swagger.io/specification/)
- [OpenAPI v3.0.x](https://swagger.io/specification/v3/)
- [OpenAPI v2.0 (Swagger)](https://swagger.io/specification/v2/)

## Installation

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
docker run --rm -v $(pwd):/data -w /data ghcr.io/amer8/apibconv -o openapi.json api.apib
```

### Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/amer8/apibconv/releases).

Prebuilt macOS binaries require **macOS 13 Ventura or newer**.

- published release assets cover Linux, macOS, and Windows on `x64` and `arm64`
- each platform archive ships with a matching `*_checksums.txt` file for targeted verification
- signed releases also ship a matching `*.sigstore.json` bundle for the checksum file
- each release also includes a CycloneDX `bom.json` SBOM

Older releases may only have the checksum file. In that case, verify the checksum directly.

Example verification:

```sh
cosign verify-blob \
  --bundle apibconv_X.Y.Z_Linux_x86_64.tar.gz_checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  apibconv_X.Y.Z_Linux_x86_64.tar.gz_checksums.txt

sha256sum -c apibconv_X.Y.Z_Linux_x86_64.tar.gz_checksums.txt
```

Releases produced by the provenance-enabled release workflow additionally include
`apibconv_X.Y.Z_provenance.sigstore.json` and `apibconv_X.Y.Z_provenance.intoto.jsonl`.
The native Sigstore bundle contains signed SLSA build provenance covering every platform
archive, checksum file, checksum signature bundle, and the SBOM. The JSONL file exports
the same signed in-toto envelope for tools such as OpenSSF Scorecard.

For an immutable release, verify GitHub's release attestation and a downloaded archive:

```sh
gh release verify vX.Y.Z --repo amer8/apibconv
gh release verify-asset vX.Y.Z apibconv_X.Y.Z_Linux_x86_64.tar.gz --repo amer8/apibconv
```

Verify the archive's build provenance using its downloaded native bundle:

```sh
gh attestation verify apibconv_X.Y.Z_Linux_x86_64.tar.gz \
  --repo amer8/apibconv \
  --bundle apibconv_X.Y.Z_provenance.sigstore.json \
  --cert-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

Add `--source-digest <expected-commit-sha>` to require a particular source commit.
Build provenance identifies the source and build workflow. GitHub's release attestation
binds the published tag, commit, and release assets. Older releases may lack either
attestation; use the checksum and signature checks available for those releases.

Container provenance is published with the image in GHCR. After authenticating to GHCR,
verify an image by its exact digest, replacing `IMAGE_DIGEST` with the SHA-256 digest:

```sh
gh attestation verify oci://ghcr.io/amer8/apibconv@sha256:IMAGE_DIGEST \
  --repo amer8/apibconv --bundle-from-oci \
  --cert-identity "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```


## Usage

### CLI Usage

The tool automatically detects the input format based on file extension and content. It supports both file arguments and stdin.

```sh
Usage: apibconv [OPTIONS] [INPUT_FILE]

Arguments:
  INPUT_FILE
      Input specification file (OpenAPI, AsyncAPI, or API Blueprint)

Options:
  -o, --output FILE
      Output file path (required for conversion)
  
  --to FORMAT
      Target format: openapi, asyncapi, apib
      Auto-detected from --output extension if not specified
  
  -e, --encoding FORMAT
      Output encoding: json, yaml (default: auto-detected from output extension)
  
  --validate
      Validate input without converting. Pinned OpenAPI and AsyncAPI versions
      are checked against specification JSON Schemas plus model validation
  
  -v, --version
      Print version information
  
  -h, --help
      Show this help message

AsyncAPI Options:
  --protocol PROTO
      Protocol: ws, wss, mqtt, kafka, amqp, http, https, auto (required)
  
  --asyncapi-version VERSION
      Version: 2.6, 3.0 (default: "2.6")

OpenAPI Options:
  --openapi-version VERSION
      Version: 2.0, 3.0, 3.1 (default: "3.0")

Examples:
  apibconv -o output.json spec.apib
  apibconv -o output.yaml --protocol ws spec.apib
  apibconv -o output.json --to openapi --openapi-version 3.1 < spec.apib
  apibconv --validate spec.json
```

### GitHub Actions

For GitHub Actions, prefer the first-party install action over `go install ...@latest`.
Pin the action itself to a full commit SHA and pin the binary version to a release tag.

```yaml
- name: Install apibconv
  uses: amer8/apibconv@<full-commit-sha>
  with:
    version: vX.Y.Z

- name: Convert OpenAPI to API Blueprint
  run: apibconv -o api-blueprint.apib openapi.json
```

The action downloads the published release asset for the current runner, verifies its SHA-256 checksum,
and when the release includes a Sigstore bundle it also verifies the checksum signature with cosign
against this repository's release workflow identity.
Older releases without a bundle fall back to checksum-only verification with a warning.
Published action assets target `x64` and `arm64` runners.

### Go projects

This tool is designed for seamless integration with servers and CLI tools built in Go.

- Basic [example](./examples/basic/main.go)
- Advanced [example](./examples/advanced/main.go)
- Plugin [example](./examples/plugin/main.go)

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Linter passes: `golangci-lint run`
3. Code coverage remains high

The release helper's policy tests are included in `go test ./...`. To run them
separately, use `go test ./internal/releasetool`.

## Releasing

Enable **Settings → Releases → Enable release immutability** before publishing with
the provenance-enabled workflow. Immutability applies to future releases. Merge the
release workflow changes before creating the next version tag.

The workflow builds the standard-library-only Go release helper once per job.
Run `go run ./internal/releasetool --help` to inspect its commands locally.

The release workflow runs in this order:

1. Build and sign all platform archives once, upload them to a draft release, and
   smoke-test the same artifacts through the install action.
2. Generate build provenance, attach the verification bundle and signed in-toto
   envelope, then download and verify the complete draft asset inventory.
3. Publish the completed draft. GitHub freezes its assets and tag and generates
   the release attestation.
4. Download the published assets and verify their checksums, Cosign signatures,
   build provenance, and release attestation. Verification requires the expected
   workflow identity, tag, and source commit, and rejects a modified archive.
5. Build and push the container image, generate provenance for its exact digest,
   and verify the provenance from GHCR.

If **Build and Verify Draft** fails, leave the release as a draft and rerun the
failed jobs. The next attempt replaces the incomplete draft and rebuilds its assets.
If **Publish Completed Draft** loses its response after publishing, rerunning it
recognizes the existing release and continues to verification.

If **Verify Published Release** or the Docker job fails, rerun only the failed jobs
from the original workflow run. The build job refuses to rebuild a published version.
Published asset corrections require a new version tag. Container attestations can
be retried by rerunning the Docker job; consumers should pin and verify image digests.

GitHub release attestations are generated automatically after immutability is enabled.
OpenSSF Scorecard currently discovers build provenance through the attached
`*.intoto.jsonl` asset and evaluates the last five releases, so its Signed-Releases
score improves as releases with provenance enter that window.

## License

See [LICENSE](LICENSE) file for details.
