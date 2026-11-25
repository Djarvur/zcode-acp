#!/usr/bin/env bash
set -euo pipefail

if (( $# != 1 )); then
  printf 'Usage: %s user@windows-host\n' "$0" >&2
  exit 1
fi

guest=$1
repo_root=$(git rev-parse --show-toplevel)
archive=$(mktemp --suffix=.tar.gz)
trap 'rm -f -- "$archive"' EXIT

cd "$repo_root"
{
  while IFS= read -r -d '' path; do
    if [[ -f "$path" ]]; then
      printf '%s\0' "$path"
    fi
  done < <(git ls-files -z)

  while IFS= read -r -d '' path; do
    if [[ -f "$path" ]]; then
      case "$path" in
        cmd/*|internal/*|scripts/*|test/*|docs/*|.github/*)
          printf '%s\0' "$path"
          ;;
      esac
    fi
  done < <(git ls-files --others --exclude-standard -z)
} | tar --null -T - -czf "$archive"

# The destination is a dedicated build tree. Recreate it so deleted or renamed
# source files cannot survive an incremental sync and change the build.
ssh "$guest" "powershell.exe -NoProfile -Command \"Remove-Item -LiteralPath 'C:/src/zcode-acp' -Recurse -Force -ErrorAction SilentlyContinue; New-Item -ItemType Directory -Force -Path 'C:/src/zcode-acp' | Out-Null\""
scp "$archive" "$guest:C:/src/zcode-acp-source.tar.gz"
ssh "$guest" "tar.exe -xzf C:/src/zcode-acp-source.tar.gz -C C:/src/zcode-acp"
ssh "$guest" "powershell.exe -NoProfile -Command \"Remove-Item -LiteralPath 'C:/src/zcode-acp-source.tar.gz' -Force -ErrorAction SilentlyContinue\""

printf 'Synchronized source to %s:C:/src/zcode-acp\n' "$guest"
