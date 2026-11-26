#!/usr/bin/env bash
set -euo pipefail

if (( $# != 1 )); then
  printf 'Usage: %s /path/to/autounattend-floppy.img\n' "$0" >&2
  exit 1
fi

connect=qemu:///system
vm_name=zcode-acp-win11
image=$(realpath -- "$1")
if [[ ! -f "$image" ]]; then
  printf 'Floppy image not found: %s\n' "$image" >&2
  exit 1
fi
if ! virsh -c "$connect" dominfo "$vm_name" >/dev/null 2>&1; then
  printf 'VM %s is not defined; run create-vm.sh first\n' "$vm_name" >&2
  exit 1
fi
if [[ "$(virsh -c "$connect" domstate "$vm_name")" != "shut off" ]]; then
  printf 'Stop %s before attaching the floppy\n' "$vm_name" >&2
  exit 1
fi

disks=$(virsh -c "$connect" domblklist "$vm_name")
if [[ "$disks" == *"$image"* ]]; then
  printf 'Floppy already attached: %s\n' "$image"
elif [[ "$disks" == *" fda "* ]]; then
  printf 'Floppy fda is already occupied; detach it before attaching %s\n' "$image" >&2
  exit 1
else
  xml=$(mktemp)
  trap 'rm -f -- "$xml"' EXIT
  virsh -c "$connect" dumpxml "$vm_name" >"$xml"
  python3 -I - "$xml" "$image" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()
if "type='fdc'" not in text:
    text = text.replace("  </devices>", "    <controller type='fdc' index='0'/>\n  </devices>", 1)
addition = f"""
    <disk type='file' device='floppy'>
      <driver name='qemu' type='raw'/>
      <source file='{sys.argv[2]}'/>
      <target dev='fda' bus='fdc'/>
      <readonly/>
    </disk>
"""
text = text.replace("  </devices>", addition + "  </devices>", 1)
path.write_text(text)
PY
  virsh -c "$connect" define "$xml"
fi

virt-xml -c "$connect" "$vm_name" --edit --boot cdrom,hd
virsh -c "$connect" start "$vm_name"
printf 'Floppy attached; console: %s\n' "$(virsh -c "$connect" domdisplay "$vm_name")"
