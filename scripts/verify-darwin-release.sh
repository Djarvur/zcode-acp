#!/usr/bin/env bash
# Verify a GoReleaser macOS snapshot without executing any release binary.
set -euo pipefail

readonly EXPECTED_BINARY=zcode-acp
work_dir=""

fail() {
  printf 'verify-darwin-release: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "$work_dir" ]]; then
    rm -rf "$work_dir"
  fi
}
trap cleanup EXIT

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

if [[ "$#" -ne 1 ]]; then
  fail "usage: $0 <dist-directory>"
fi

dist=$1
[[ -d "$dist" && ! -L "$dist" ]] || fail "dist path is not a non-symlink directory: $dist"
dist=$(cd "$dist" && pwd -P)

for tool in shasum tar mktemp lipo; do
  require_command "$tool"
done

checksum_manifest="$dist/checksums.txt"
[[ -f "$checksum_manifest" && ! -L "$checksum_manifest" ]] ||
  fail "missing regular checksums.txt in $dist"

shopt -s nullglob
labels=(darwin_arm64 darwin_amd64 darwin_universal)
archives=()
declare -A expected_checksums=()
for label in "${labels[@]}"; do
  matches=("$dist"/zcode-acp_*_"$label".tar.gz)
  [[ "${#matches[@]}" -eq 1 ]] ||
    fail "expected exactly one $label archive, found ${#matches[@]}"
  archive=${matches[0]}
  [[ -f "$archive" && ! -L "$archive" ]] ||
    fail "archive is not a regular non-symlink file: $archive"
  archives+=("$archive")
  expected_checksums[${archive##*/}]=1
done

# Accept only the three expected archive basenames. This prevents a valid
# checksum for an unrelated file from standing in for a missing archive.
declare -A seen_checksums=()
checksum_count=0
checksum_pattern='^([0-9a-f]{64})  (\*?)(.+)$'
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ "$line" =~ $checksum_pattern ]] ||
    fail "malformed SHA-256 checksum line: $line"
  name=${BASH_REMATCH[3]}
  [[ "$name" != /* && "$name" != */* ]] ||
    fail "checksum entry is not an archive basename: $name"
  [[ -n "${expected_checksums[$name]+present}" ]] ||
    fail "checksum entry is not an expected Darwin archive: $name"
  [[ -z "${seen_checksums[$name]+present}" ]] ||
    fail "duplicate checksum entry: $name"
  seen_checksums[$name]=1
  checksum_count=$((checksum_count + 1))
done < "$checksum_manifest"

[[ "$checksum_count" -eq "${#archives[@]}" ]] ||
  fail "checksums.txt must cover exactly the three Darwin archives"
for name in "${!expected_checksums[@]}"; do
  [[ -n "${seen_checksums[$name]+present}" ]] ||
    fail "checksums.txt is missing $name"
done

(cd "$dist" && shasum -a 256 -c checksums.txt) ||
  fail "SHA-256 verification failed in $dist"

# Validate and extract private copies, not mutable dist paths. The second
# checksum pass closes the race between the dist check and staging.
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/zcode-acp-darwin-release.XXXXXX")
staged="$work_dir/archives"
mkdir -m 700 "$staged"
cp "$checksum_manifest" "$staged/checksums.txt"
for archive in "${archives[@]}"; do
  cp "$archive" "$staged/${archive##*/}"
done
(cd "$staged" && shasum -a 256 -c checksums.txt) ||
  fail "staged archive checksum verification failed"

verify_archive() {
  local archive=$1
  local label=$2
  local members listing binary archs
  local -a listing_lines actual_archs

  members=$(tar -tzf "$archive") || fail "cannot list archive members: $label"
  [[ "$members" == "$EXPECTED_BINARY" ]] ||
    fail "archive member list is not exactly $EXPECTED_BINARY: $label"

  listing=$(tar -tvzf "$archive") || fail "cannot inspect archive member types: $label"
  mapfile -t listing_lines <<< "$listing"
  [[ "${#listing_lines[@]}" -eq 1 ]] ||
    fail "archive must contain exactly one member: $label"
  [[ "${listing_lines[0]}" == -* ]] ||
    fail "archive member is not a regular file: $label"

  local extract_dir="$work_dir/extracted-$label"
  mkdir -m 700 "$extract_dir"
  tar -xzf "$archive" -C "$extract_dir" || fail "cannot extract validated archive: $label"
  binary="$extract_dir/$EXPECTED_BINARY"
  [[ -f "$binary" && ! -L "$binary" ]] ||
    fail "extracted zcode-acp is not a regular non-symlink file: $label"

  archs=$(lipo -archs "$binary") || fail "lipo could not inspect $label"
  read -r -a actual_archs <<< "$archs"
  case "$label" in
    darwin_arm64)
      [[ "${#actual_archs[@]}" -eq 1 && "${actual_archs[0]}" == arm64 ]] ||
        fail "$label has unexpected lipo architecture set: $archs"
      ;;
    darwin_amd64)
      [[ "${#actual_archs[@]}" -eq 1 && "${actual_archs[0]}" == x86_64 ]] ||
        fail "$label has unexpected lipo architecture set: $archs"
      ;;
    darwin_universal)
      [[ "${#actual_archs[@]}" -eq 2 ]] ||
        fail "$label has unexpected lipo architecture set: $archs"
      [[ " ${actual_archs[*]} " == *" arm64 "* && " ${actual_archs[*]} " == *" x86_64 "* ]] ||
        fail "$label has unexpected lipo architecture set: $archs"
      ;;
  esac

  printf 'PASS %s: SHA-256, one regular %s, lipo -archs %s\n' \
    "$label" "$EXPECTED_BINARY" "$archs"
}

for index in "${!archives[@]}"; do
  verify_archive "${archives[$index]}" "${labels[$index]}"
done
