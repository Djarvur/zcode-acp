#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 || $# > 2 )); then
  printf 'Usage: %s /path/to/zcode-acp-tools.iso [directory-with-OpenSSH-packages]\n' "$0" >&2
  exit 1
fi

out=$(realpath -m -- "$1")
version=9.5.0.0
release=v${version}p1-Beta
base_url="https://github.com/PowerShell/Win32-OpenSSH/releases/download/$release"
msi="OpenSSH-Win64-v$version.msi"
zip="OpenSSH-Win64-v$version.zip"

work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

if (( $# == 2 )); then
  source_dir=$(realpath -- "$2")
  if [[ ! -d "$source_dir" ]]; then
    printf 'Source directory not found: %s\n' "$source_dir" >&2
    exit 1
  fi
  if [[ -f "$source_dir/$msi" ]]; then
    cp -- "$source_dir/$msi" "$work/$msi"
  fi
  if [[ -f "$source_dir/$zip" ]]; then
    cp -- "$source_dir/$zip" "$work/$zip"
  elif [[ -f "$source_dir/OpenSSH-Win64.zip" ]]; then
    cp -- "$source_dir/OpenSSH-Win64.zip" "$work/$zip"
  fi
  if [[ ! -f "$work/$msi" && ! -f "$work/$zip" ]]; then
    printf 'Neither %s nor %s found in %s\n' "$msi" "$zip" "$source_dir" >&2
    exit 1
  fi
else
  curl -fL --retry 3 -o "$work/$msi" "$base_url/$msi"
  curl -fL --retry 3 -o "$work/$zip" "$base_url/OpenSSH-Win64.zip"
fi

(
  cd "$work"
  for package in *.msi *.zip; do
    if [[ -f "$package" ]]; then
      sha256sum "$package"
    fi
  done >SHA256SUMS
)

mkdir -p -- "$(dirname -- "$out")"
genisoimage -quiet -J -R -V ZCODE_TOOLS -o "$out" "$work"
printf 'Created %s\n' "$out"
