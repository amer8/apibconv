# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

1. **Do not** open a public GitHub issue for security vulnerabilities
2. Email your findings to the repository maintainers via GitHub's private vulnerability reporting feature
3. Alternatively, use [GitHub's Security Advisory feature](https://github.com/amer8/apibconv/security/advisories/new) to report privately

### What to Include

Please include the following information in your report:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact of the vulnerability
- Any suggested fixes (optional)

## Review Model

This repository is currently maintained by a single active maintainer.

Because there is not yet a second active reviewer with write access, independent human approval cannot be required for every change. This is a known project limitation and should not be interpreted as equivalent to multi-maintainer review.

Where practical, changes should still be proposed through pull requests and merged through protected-branch or repository-rule workflows rather than by direct pushes. If additional maintainers are added in the future, the project intends to require independent approval for protected branch changes.

## Security Practices

This project currently uses the following security controls:

- Dependency updates via Dependabot
- Static analysis with `gosec` and `golangci-lint` in CI
- CodeQL and OpenSSF Scorecard workflows
- Private vulnerability reporting through GitHub Security Advisories

Thank you for helping keep this project secure.
