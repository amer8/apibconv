# Capability analysis

CI and the release workflow use Capslock v0.3.4 and Go 1.27.3 to track
capabilities for all packages, including examples and release tooling, on
linux/amd64 with CGO disabled. The baseline records each package's direct and
transitive capability categories; it does not track individual functions or
call-path counts.

To check locally, install Go 1.27.3 and Python 3.9 or newer, then run
`./scripts/capslock.sh --check`.
When a change is expected, run `./scripts/capslock.sh --write`, review the
baseline diff, and commit it with the change. Set `CAPSLOCK_REPORT` to a file
path to retain the full JSON report, including example call paths; CI uploads
it as an artifact even when the baseline comparison fails. Update the pinned
versions in the script and both workflows together, then review and regenerate
the baseline when upgrading Capslock or Go.

Capslock's static analysis may report paths that cannot occur in practice and
cannot fully analyze every call. See the [Capslock caveats](https://github.com/google/capslock/blob/v0.3.4/docs/caveats.md).
