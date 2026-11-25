#!/usr/bin/env bash
set -euo pipefail

connect=qemu:///system
network=zcode-acp-dev
vm_name=zcode-acp-win11
pool=images
volume=zcode-acp-win11.qcow2
mac=52:54:00:ac:01:10
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

if ! virsh -c "$connect" net-info "$network" >/dev/null 2>&1; then
  virsh -c "$connect" net-define "$script_dir/network.xml"
  virsh -c "$connect" net-autostart "$network"
  virsh -c "$connect" net-start "$network"
fi

network_info=$(virsh -c "$connect" net-info "$network")
if [[ "$network_info" != *"Active:"*"yes"* ]]; then
  virsh -c "$connect" net-start "$network"
fi

if virsh -c "$connect" dominfo "$vm_name" >/dev/null 2>&1; then
  printf 'VM %s already exists; leaving it unchanged\n' "$vm_name" >&2
  exit 0
fi

if ! virsh -c "$connect" pool-info "$pool" >/dev/null 2>&1; then
  pool_target=/var/lib/libvirt/zcode-acp-images
  if ! mkdir -p "$pool_target"; then
    printf 'Cannot create libvirt pool target %s; create it with appropriate permissions and rerun\n' "$pool_target" >&2
    exit 1
  fi
  virsh -c "$connect" pool-define-as "$pool" dir --target "$pool_target"
  virsh -c "$connect" pool-autostart "$pool"
  virsh -c "$connect" pool-start "$pool"
fi

pool_info=$(virsh -c "$connect" pool-info "$pool")
if [[ "$pool_info" != *"State:"*"active"* ]]; then
  virsh -c "$connect" pool-start "$pool"
fi

if ! virsh -c "$connect" vol-info "$volume" --pool "$pool" >/dev/null 2>&1; then
  virsh -c "$connect" vol-create-as "$pool" "$volume" 100G --format qcow2
fi

args=(
  --connect "$connect"
  --name "$vm_name"
  --machine q35
  --vcpus 6
  --memory 10240
  --cpu host-passthrough
  --boot cdrom,hd
  --tpm "backend.type=emulator,backend.version=2.0,model=tpm-crb"
  --disk "pool=$pool,vol=$volume,format=qcow2,bus=sata"
  --network "network=$network,model=e1000,mac=$mac"
  --graphics "spice,listen=127.0.0.1"
  --video qxl
  --noautoconsole
  --osinfo win11
)

if (( $# > 0 )); then
  iso=$(realpath -- "$1")
  if [[ ! -f "$iso" ]]; then
    printf 'ISO not found: %s\n' "$iso" >&2
    exit 1
  fi
  args+=(--cdrom "$iso")
fi

xml=$(mktemp)
trap 'rm -f -- "$xml"' EXIT
virt-install --dry-run --print-xml "${args[@]}" >"$xml"
virsh -c "$connect" define "$xml"

if (( $# > 0 )); then
  virsh -c "$connect" start "$vm_name"
  printf 'Started %s with %s\n' "$vm_name" "$iso"
else
  printf 'Defined %s without installation media; attach an ISO and start it when ready\n' "$vm_name"
fi
