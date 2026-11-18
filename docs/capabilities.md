# Capability analysis

CI and the release workflow use Capslock v0.3.4 and Go 1.27.3 to track
capabilities for all packages, including examples and release tooling, on
linux/amd64 and darwin/arm64 with CGO disabled. Each platform has its own
baseline of package-level direct and transitive capability categories; it does
not track individual functions or call-path counts. CI cross-analyzes macOS on
a Linux runner, using the macOS target environment and standard library.

To check locally, install Go 1.27.3 and Python 3.9 or newer, then run
`./scripts/capslock.sh --check` for Linux and
`./scripts/capslock.sh --check darwin/arm64` for macOS. When a change is
expected, replace `--check` with `--write` for the affected platform, review
the baseline diff, and commit it with the change. Set `CAPSLOCK_REPORT` to a file
path to retain the full JSON report, including example call paths; CI uploads
it as an artifact even when the baseline comparison fails. Update the pinned
versions in the script and both workflows together, then review and regenerate
the baseline when upgrading Capslock or Go.

Capslock's static analysis may report paths that cannot occur in practice and
cannot fully analyze every call. See the [Capslock caveats](https://github.com/google/capslock/blob/v0.3.4/docs/caveats.md).
