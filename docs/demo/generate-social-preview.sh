#!/bin/sh
# Rebuild the social preview on macOS with Swift and ffmpeg installed.
set -eu
cd "$(dirname "$0")/../.."
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
mkdir -p "$tmp_dir/swift-cache"
CLANG_MODULE_CACHE_PATH="$tmp_dir/swift-cache" swift docs/demo/social-variants/design.swift 10 "$tmp_dir/background.png"
ffmpeg -v error -y -loop 1 -i "$tmp_dir/background.png" -i docs/demo/apibconv.gif \
  -filter_complex '[1:v]fps=15,scale=650:448:flags=lanczos[demo];[0:v][demo]overlay=50:18:shortest=1:format=rgb,split[frames][palette_input];[palette_input]palettegen=stats_mode=diff[palette];[frames][palette]paletteuse=dither=sierra2_4a' \
  -loop 0 docs/demo/social-preview.gif
