#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 || $# > 3 )); then
  printf 'Usage: %s /path/to/new-artifact-directory [public-key-path] [bios|uefi]\n' "$0" >&2
  exit 1
fi

out=$(realpath -m -- "$1")
public_key_path=${2:-"$HOME/.ssh/id_ed25519.pub"}
public_key_path=$(realpath -- "$public_key_path")
format=${3:-bios}
case "$format" in
  bios|uefi) ;;
  *)
    printf 'Unsupported firmware format: %s\n' "$format" >&2
    exit 1
    ;;
esac
if [[ -e "$out" && -n "$(ls -A "$out")" ]]; then
  printf 'Artifact directory must be new or empty: %s\n' "$out" >&2
  exit 1
fi
mkdir -p "$out"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
password=$(openssl rand -hex 24)
python3 -I - \
  "$script_dir/Autounattend-$format.xml.in" \
  "$script_dir/configure-ssh.ps1.in" \
  "$out/Autounattend.xml" \
  "$out/configure-ssh.ps1" \
  "$password" \
  "$public_key_path" <<'PY'
from pathlib import Path
import sys

template = Path(sys.argv[1]).read_text(encoding="utf-8")
ssh_template = Path(sys.argv[2]).read_text(encoding="utf-8")
public_key = Path(sys.argv[6]).read_text(encoding="utf-8").strip()
if not public_key:
    raise SystemExit("public key file is empty")
ssh_script = ssh_template.replace("__PUBLIC_KEY__", public_key)
ssh_command = r"powershell.exe -NoProfile -ExecutionPolicy Bypass -File A:\configure-ssh.ps1"
answer = template.replace("__PASSWORD__", sys.argv[5]).replace("__SSH_COMMAND__", ssh_command)
Path(sys.argv[3]).write_text(answer, encoding="utf-8")
Path(sys.argv[4]).write_text(ssh_script, encoding="utf-8")
PY

printf 'username=zcode\npassword=%s\n' "$password" >"$out/credentials.txt"
chmod 600 "$out/Autounattend.xml" "$out/credentials.txt"
genisoimage -quiet -J -R -V ZCODE_ACP_UNATTEND -o "$out/autounattend.iso" \
  "$out/Autounattend.xml" "$out/configure-ssh.ps1"
floppy="$out/autounattend-$format-floppy.img"
truncate -s 1440K "$floppy"
mkfs.vfat -F 12 -n ZCODE_UNAT "$floppy" >/dev/null
mcopy -o -i "$floppy" "$out/Autounattend.xml" ::/Autounattend.xml
mcopy -o -i "$floppy" "$out/configure-ssh.ps1" ::/configure-ssh.ps1
sha256sum "$out/autounattend.iso" "$floppy" >"$out/SHA256SUMS"

printf 'Created %s/autounattend.iso\nCreated %s\nCredentials: %s/credentials.txt\n' "$out" "$floppy" "$out"
