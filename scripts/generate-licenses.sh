#!/bin/sh
# Generate or check the license report embedded for each release target.
set -eu

if [ "$#" -eq 0 ] || [ "$#" -eq 1 ]; then
    mode=${1:---write}
    targets='linux/amd64 linux/arm64 darwin/amd64 darwin/arm64'
elif [ "$#" -eq 3 ]; then
    mode=$1
    targets="$2/$3"
else
    echo 'Usage: scripts/generate-licenses.sh [--write|--check] [GOOS GOARCH]' >&2
    exit 2
fi
case "$mode" in --write|--check) ;; *) echo "Unknown mode: $mode" >&2; exit 2;; esac
license_tool=${GO_LICENSES:-go-licenses}
if ! command -v "$license_tool" >/dev/null 2>&1 && [ -z "${GO_LICENSES:-}" ]; then
    gobin=$(go env GOBIN)
    if [ -z "$gobin" ]; then gobin=$(go env GOPATH)/bin; fi
    license_tool="$gobin/go-licenses"
fi
command -v "$license_tool" >/dev/null 2>&1 || { echo 'Install github.com/google/go-licenses/v2@v2.0.1 first.' >&2; exit 1; }
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/apibconv-licenses.XXXXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
export LC_ALL=C
goroot=$(go env GOROOT)
for name in LICENSE PATENTS; do
    case "$name" in LICENSE) local_file=licenses/GO-BSD-3-CLAUSE;; PATENTS) local_file=licenses/GO-PATENTS;; esac
    if ! cmp -s "$goroot/$name" "$local_file"; then
        echo "Review and update $local_file for the current Go toolchain" >&2
        exit 1
    fi
done
for target in $targets; do
    os=${target%/*}
    arch=${target#*/}
    case "$target" in linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) ;; *) echo "Unsupported target: $target" >&2; exit 2;; esac
    output="internal/licenses/reports/${os}_${arch}.txt"
    saved="$tmp/saved-${os}-${arch}"
    report="$tmp/report-${os}-${arch}.txt"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 "$license_tool" report ./cmd/apibconv \
        --ignore github.com/amer8/apibconv --template scripts/licenses-report.tmpl > "$tmp/inventory"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 "$license_tool" save ./cmd/apibconv \
        --ignore github.com/amer8/apibconv --save_path "$saved"
    # go-licenses save does not generally copy separate patent grants. Inspect
    # the source directories of the modules actually linked for this target.
    app_module=$(GOOS=$os GOARCH=$arch CGO_ENABLED=0 go list -f '{{.Module.Path}}' ./cmd/apibconv)
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go list -deps \
        -f '{{if .Module}}{{printf "%s\t%s" .Module.Path .Module.Dir}}{{end}}' \
        ./cmd/apibconv > "$tmp/modules-unsorted"
    sort -u "$tmp/modules-unsorted" > "$tmp/modules"
    if [ -d "$saved/go.yaml.in/yaml/v3" ]; then
        cmp -s "$saved/go.yaml.in/yaml/v3/LICENSE" licenses/GO-YAML && \
            cmp -s "$saved/go.yaml.in/yaml/v3/NOTICE" licenses/GO-YAML-NOTICE || {
            echo 'Review and update the copied YAML license and notice' >&2; exit 1;
        }
    fi
    if [ -d "$saved/golang.org/x/text" ]; then
        cmp -s "$saved/golang.org/x/text/LICENSE" licenses/GO-BSD-3-CLAUSE || {
            echo 'Review the golang.org/x/text license' >&2; exit 1;
        }
    fi
    # Other files may be required source distributions; never silently discard them.
    unexpected=$(find "$saved" -type f ! -iname 'LICENSE*' ! -iname 'LICENCE*' ! -iname 'NOTICE*' ! -iname 'COPYING*' ! -iname 'PATENTS*' -print -quit)
    if [ -n "$unexpected" ]; then
        echo "Dependency requires additional distribution material: $unexpected" >&2
        exit 1
    fi
    {
        printf 'Third-party licenses for %s/%s\n\n' "$os" "$arch"
        printf 'Generated Go dependency inventory (license classification may be incomplete; see full texts below):\n'
        cat "$tmp/inventory"
        printf '\nOpenAPI schemas and Go runtime attribution:\n\n'
        cat THIRD_PARTY_NOTICES.md
        for file in licenses/APACHE-2.0 licenses/GO-BSD-3-CLAUSE licenses/GO-PATENTS; do
            printf '\n\n===== %s =====\n\n' "$file"
            cat "$file"
        done
        find "$saved" -type f | sort | while IFS= read -r file; do
            printf '\n\n===== %s =====\n\n' "${file#"$saved"/}"
            cat "$file"
        done
        while IFS="$(printf '\t')" read -r module module_dir; do
            [ -n "$module" ] || continue
            [ "$module" != "$app_module" ] || continue
            find "$module_dir" -type f -iname 'PATENTS*' > "$tmp/patents-unsorted"
            sort "$tmp/patents-unsorted" > "$tmp/patents"
            while IFS= read -r file; do
                printf '\n\n===== %s/%s =====\n\n' "$module" "${file#"$module_dir"/}"
                cat "$file"
            done < "$tmp/patents"
        done < "$tmp/modules"
        printf '\n'
    } > "$report"
    if [ "$mode" = --check ]; then
        if ! cmp -s "$report" "$output"; then
            echo "$output is missing or out of date; run scripts/generate-licenses.sh --write" >&2
            exit 1
        fi
    else
        cp "$report" "$output"
    fi
    echo "$mode: $target license report verified"
done

# The fallback contains the union of the full release-target reports. It may
# not cover dependencies specific to an unsupported platform.
fallback="$tmp/fallback.txt"
{
    printf '%s\n\n' 'No target-specific dependency report is available for this platform. The following reports cover the supported release targets; platform-specific dependencies may be missing.'
    for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
        cat "internal/licenses/reports/$target.txt"
        printf '\n'
    done
} > "$fallback"
if [ "$mode" = --check ]; then
    cmp -s "$fallback" internal/licenses/reports/fallback.txt || { echo 'Fallback report is out of date; regenerate reports' >&2; exit 1; }
else
    cp "$fallback" internal/licenses/reports/fallback.txt
fi
