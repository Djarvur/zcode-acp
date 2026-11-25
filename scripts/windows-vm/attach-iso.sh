#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 || $# > 2 )); then
  printf 'Usage: %s /path/to/windows.iso [/path/to/autounattend.iso]\n' "$0" >&2
  exit 1
fi

connect=qemu:///system
vm_name=zcode-acp-win11
sources=("$(realpath -- "$1")")
targets=(sdb)
if (( $# == 2 )); then
  sources+=("$(realpath -- "$2")")
  targets+=(sdc)
fi

for iso in "${sources[@]}"; do
  if [[ ! -f "$iso" ]]; then
    printf 'ISO not found: %s\n' "$iso" >&2
    exit 1
  fi
done
if ! virsh -c "$connect" dominfo "$vm_name" >/dev/null 2>&1; then
  printf 'VM %s is not defined; run create-vm.sh first\n' "$vm_name" >&2
  exit 1
fi
if [[ "$(virsh -c "$connect" domstate "$vm_name")" != "shut off" ]]; then
  printf 'Stop %s before attaching installation media\n' "$vm_name" >&2
  exit 1
fi

disks=$(virsh -c "$connect" domblklist "$vm_name")
for index in "${!sources[@]}"; do
  iso=${sources[$index]}
  target=${targets[$index]}

  if [[ "$disks" == *"$iso"* ]]; then
    printf 'ISO already attached: %s\n' "$iso"
  elif [[ "$disks" == *" $target "* ]]; then
    printf 'CD-ROM %s is already occupied; detach it before attaching %s\n' "$target" "$iso" >&2
    exit 1
  else
    virsh -c "$connect" attach-disk "$vm_name" "$iso" "$target" \
      --targetbus sata --type cdrom --mode readonly --config
  fi
done

virt-xml -c "$connect" "$vm_name" --edit --boot cdrom,hd
virsh -c "$connect" start "$vm_name"
printf 'Windows installation media attached; console: %s\n' "$(virsh -c "$connect" domdisplay "$vm_name")"
