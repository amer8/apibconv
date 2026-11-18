#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

mode=${1:---check}
if [[ $# -gt 1 || ( "$mode" != "--check" && "$mode" != "--write" ) ]]; then
  echo "Usage: $0 [--check|--write]" >&2
  exit 2
fi

# Keep these versions in sync with the CI job and docs/capabilities.md.
export GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOEXPERIMENT= CGO_ENABLED=0 GOAMD64=v1
if [[ $(go env GOVERSION) != go1.27.3 ]]; then
  echo "Capslock requires Go 1.27.3 for a reproducible baseline." >&2
  exit 2
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
report="$tmpdir/report.json"
baseline=scripts/capslock-baseline.txt

# Build for the host, then set the analysis environment to match CI. Capslock
# also loads the standard library without its -goos/-goarch flags.
GOOS=$(go env GOHOSTOS) GOARCH=$(go env GOHOSTARCH) GOBIN="$tmpdir" \
  go install github.com/google/capslock/cmd/capslock@v0.3.4
GOOS=linux GOARCH=amd64 "$tmpdir/capslock" \
  -packages=./... -force_local_module -goos=linux -goarch=amd64 \
  -output=json > "$report"
if [[ -n ${CAPSLOCK_REPORT:-} ]]; then
  cp "$report" "$CAPSLOCK_REPORT"
fi

# Record each package's capabilities and whether each is direct or transitive.
python3 - "$report" "$tmpdir/baseline.txt" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as source:
    report = json.load(source)

entries = {
    (item["packageDir"], item["capabilityName"], item["capabilityType"].removeprefix("CAPABILITY_TYPE_"))
    for item in report.get("capabilityInfo", [])
}
with open(sys.argv[2], "w", encoding="utf-8") as output:
    for package, capability, reach in sorted(entries):
        print(f"{package} {capability} {reach}", file=output)
PY

if [[ "$mode" == "--write" ]]; then
  cp "$tmpdir/baseline.txt" "$baseline"
  echo "Updated $baseline; review the changes before committing."
elif diff -u "$baseline" "$tmpdir/baseline.txt"; then
  echo "Capslock capabilities match $baseline."
else
  echo "Capslock capabilities changed. Review the report and run $0 --write if expected." >&2
  exit 1
fi
