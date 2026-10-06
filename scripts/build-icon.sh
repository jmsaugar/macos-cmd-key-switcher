#!/bin/bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: bash scripts/build-icon.sh <source.png> <output.icns>" >&2
    exit 1
fi

source_png="$1"
output_icns="$2"
width=""
height=""
dimensions="$(sips -g pixelWidth -g pixelHeight "$source_png")"
while read -r property value; do
    case "$property" in
        pixelWidth:) width="$value" ;;
        pixelHeight:) height="$value" ;;
    esac
done <<< "$dimensions"

if [[ -z "$width" || -z "$height" || "$width" != "$height" ]]; then
    echo "App icon must be a square image." >&2
    exit 1
fi
if [[ "$width" -lt 1024 ]]; then
    echo "App icon must be at least 1024 × 1024 pixels." >&2
    exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/cmd-key-switcher-icon.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
iconset="$work_dir/AppIcon.iconset"
mkdir -p "$iconset" "$(dirname "$output_icns")"

for size in 16 32 128 256 512; do
    sips -s format png -z "$size" "$size" "$source_png" \
        --out "$iconset/icon_${size}x${size}.png" >/dev/null
    retina_size=$((size * 2))
    sips -s format png -z "$retina_size" "$retina_size" "$source_png" \
        --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done

iconutil -c icns "$iconset" -o "$output_icns"
