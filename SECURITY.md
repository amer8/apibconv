# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

Please report security vulnerabilities privately through
[GitHub Security Advisories](https://github.com/amer8/apibconv/security/advisories/new).
Do not open a public issue.

### What to Include

Please include the following information in your report:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact of the vulnerability
- Any suggested fixes (optional)

## Review Model

This repository is currently maintained by a single active maintainer.

There is no second active reviewer with write access, so independent approval is
not required for every change. This is a limitation of the project’s review process.

Where practical, changes should be proposed through pull requests and merged
through protected-branch or repository-rule workflows. If more maintainers join,
the project intends to require independent approval for protected branches.

## Security Practices

This project currently uses the following security controls:

- Dependency updates via Dependabot
- Static analysis with `gosec` and `golangci-lint` in CI
- CodeQL and OpenSSF Scorecard workflows
- Private vulnerability reporting through GitHub Security Advisories
- Sigstore signatures for release archive checksums

The provenance-enabled release workflow also generates SLSA build provenance for
release assets and GHCR container images. It verifies signatures, artifact digests,
the release workflow identity, source tag, and commit before accepting a release.
Published immutable releases additionally have GitHub-generated release attestations.
These controls apply to releases produced by that workflow; older releases may have
only checksums or checksum signatures. See the README for verification and release
recovery instructions.
