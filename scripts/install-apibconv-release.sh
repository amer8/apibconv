#!/bin/sh

set -eu

require_env() {
    name=$1
    eval "value=\${$name-}"
    if [ -z "${value}" ]; then
        echo "Required environment variable '${name}' is not set." >&2
        exit 1
    fi
    printf '%s\n' "$value"
}

require_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Required command '$1' is not available." >&2
        exit 1
    fi
}

download_file() {
    url=$1
    destination=$2

    case "$url" in
        file://*)
            source_path=${url#file://}
            if [ ! -f "$source_path" ]; then
                echo "Failed to download $url (file not found)." >&2
                exit 1
            fi
            cp "$source_path" "$destination"
            return 0
            ;;
    esac

    http_code=$(curl -sSL -w '%{http_code}' -o "$destination" "$url")
    if [ "$http_code" != "200" ]; then
        echo "Failed to download $url (HTTP $http_code)." >&2
        exit 1
    fi
}

try_download_file() {
    url=$1
    destination=$2

    case "$url" in
        file://*)
            source_path=${url#file://}
            if [ -f "$source_path" ]; then
                cp "$source_path" "$destination"
                return 0
            fi
            rm -f "$destination"
            return 1
            ;;
    esac

    http_code=$(curl -sSL -w '%{http_code}' -o "$destination" "$url")
    case "$http_code" in
        200)
            return 0
            ;;
        404)
            rm -f "$destination"
            return 1
            ;;
        *)
            echo "Failed to download $url (HTTP $http_code)." >&2
            exit 1
            ;;
    esac
}

get_expected_checksum() {
    checksum_path=$1
    archive_name=$2

    checksum=$(awk -v archive_name="$archive_name" '
        $2 == archive_name || $2 == "*" archive_name {
            print tolower($1)
            exit
        }
        NF == 1 && $1 ~ /^[[:xdigit:]]+$/ {
            print tolower($1)
            exit
        }
    ' "$checksum_path")
    if [ -n "$checksum" ]; then
        printf '%s\n' "$checksum"
        return 0
    fi

    echo "Could not find $archive_name in $checksum_path." >&2
    exit 1
}

write_output_value() {
    name=$1
    value=$2
    printf '%s=%s\n' "$name" "$value" >> "$(require_env GITHUB_OUTPUT)"
}

runner_os=$(require_env RUNNER_OS)
runner_arch=$(require_env RUNNER_ARCH)

case "$runner_os" in
    Linux)
        platform=Linux
        extension=tar.gz
        binary_name=apibconv
        checksum_cmd=sha256sum
        ;;
    macOS)
        platform=Darwin
        extension=tar.gz
        binary_name=apibconv
        checksum_cmd=shasum
        ;;
    *)
        echo "Unsupported runner OS '$runner_os'." >&2
        exit 1
        ;;
esac

case "$runner_arch" in
    X64)
        arch=x86_64
        ;;
    ARM64)
        arch=arm64
        ;;
    ARM)
        arch=armv7
        ;;
    *)
        echo "Unsupported runner architecture '$runner_arch'." >&2
        exit 1
        ;;
esac

require_command curl
require_command tar
require_command "$checksum_cmd"

version=$(require_env APIBCONV_VERSION)
case "$version" in
    v*)
        ;;
    *)
        echo "Expected APIBCONV_VERSION to be a tag like v0.7.0, got '$version'." >&2
        exit 1
        ;;
esac

clean_version=${version#v}
archive_name="apibconv_${clean_version}_${platform}_${arch}.${extension}"
checksum_name="${archive_name}_checksums.txt"
bundle_name="${checksum_name}.sigstore.json"
base_url="https://github.com/amer8/apibconv/releases/download/${version}"
asset_base_url=${APIBCONV_ASSET_BASE_URL:-$base_url}
temp_root="$(require_env RUNNER_TEMP)/apibconv-${clean_version}-${platform}-${arch}"
extract_dir="${temp_root}/extract"
install_dir="${temp_root}/bin"
archive_path="${temp_root}/${archive_name}"
checksum_path="${temp_root}/${checksum_name}"
bundle_path="${temp_root}/${bundle_name}"
installed_binary_path="${install_dir}/${binary_name}"

rm -rf "$temp_root"
mkdir -p "$extract_dir" "$install_dir"

echo "Downloading $archive_name from $asset_base_url"
download_file "$asset_base_url/$archive_name" "$archive_path"
download_file "$asset_base_url/$checksum_name" "$checksum_path"

signature_verified=false
if try_download_file "$asset_base_url/$bundle_name" "$bundle_path"; then
    require_command cosign
    certificate_identity="https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/${version}"

    echo "Verifying checksum signature with cosign"
    cosign verify-blob \
        --certificate-identity "$certificate_identity" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        --bundle "$bundle_path" \
        "$checksum_path"

    signature_verified=true
else
    echo "Warning: No Sigstore bundle found for $checksum_name. Falling back to checksum-only verification." >&2
fi

expected_checksum=$(get_expected_checksum "$checksum_path" "$archive_name")
case "$runner_os" in
    Linux)
        actual_checksum=$(sha256sum "$archive_path" | awk '{print $1}')
        ;;
    macOS)
        actual_checksum=$(shasum -a 256 "$archive_path" | awk '{print $1}')
        ;;
esac
actual_checksum=$(printf '%s\n' "$actual_checksum" | tr '[:upper:]' '[:lower:]')

if [ "$actual_checksum" != "$expected_checksum" ]; then
    echo "Checksum mismatch for $archive_name. Expected $expected_checksum, got $actual_checksum." >&2
    exit 1
fi

echo "Verified checksum for $archive_name"
tar -xzf "$archive_path" -C "$extract_dir"

binary_path=$(find "$extract_dir" -type f -name "$binary_name" -print -quit)
if [ -z "$binary_path" ]; then
    echo "Could not find $binary_name after extracting $archive_name." >&2
    exit 1
fi

mv "$binary_path" "$installed_binary_path"
chmod +x "$installed_binary_path"

printf '%s\n' "$install_dir" >> "$(require_env GITHUB_PATH)"
write_output_value binary-path "$installed_binary_path"
write_output_value version "$version"
write_output_value signature-verified "$signature_verified"
