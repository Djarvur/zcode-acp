---
status: complete
phase: 10-mount-first-appimage-backend
source: [10-VERIFICATION.md]
started: 2026-11-24T04:15:00.000Z
updated: 2026-11-24T04:22:00.000Z
---

## Current Test

number: 2
name: Real read-only/no-extraction proof
expected: |
  AppImage sha256 unchanged and the parent directory gains zero extraction-shaped entries while pre-existing extracts survive.
awaiting: user response

## Tests

### 1. Real packed-AppImage acceptance (VER-02)
expected: Mount the real packed AppImage through the production mount owner, spawn the real app-server from the mount, complete session/create, and tear down with zero leftover processes or mounts.
result: passed — `ZCODE_ACP_REAL_E2E=1 ZCODE_ACP_REAL_APPIMAGE=/home/nil/Applications/ZCode-4.0.0-linux-x64.AppImage go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v` → PASS (5.45s) on 2026-11-24. Evidence: layout=appimage version=4.0.0 mount=/tmp/zcode-acp-mount-548128178/.mount_ZCode-4mDu6E; app-server spawned from `<mount>/zcode` (pgid 3738384); session/create ok (sess_085c6d06-0b7d-4087-93a1-ee1c60df66ab); app-server group ESRCH; helper group ESRCH; mount entry and private base gone.

### 2. Real read-only/no-extraction proof
expected: AppImage sha256 unchanged and the parent directory gains zero extraction-shaped entries while pre-existing extracts survive.
result: passed — sha256 `2cedc5fc…fefcc47` identical before and after; extraction check under `/home/nil/Applications`: 1 pre-existing entry (`zcode-3.18.2`) unchanged, **0 added**.

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
