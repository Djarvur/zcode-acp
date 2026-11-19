#!/usr/bin/env bash
# experiments/scan-secrets.sh — secret-shape gate over tracked files (D-02, T-1-01).
#
# Fails (exit 1, printing "SECRET SHAPE: <pattern>") when any tracked file outside
# .planning matches a live-credential VALUE shape:
#   (a) three-part JWT token: 10+ char base64url segment before the first dot,
#       then two more base64url segments;
#   (b) credential-blob value: enc:v1: / enc:v2:os: prefix followed by a 20+ char
#       base64ish body (legacy enc:v1: form included; a bare prefix mention in
#       prose never matches — the payload body is required);
#   (c) long (20+ char) token value keyed by access_token / apiKey / api-key / token.
#
# Shapes match VALUES, not documentation prose — legitimate doc text must stay green.
# .planning/ is excluded (planning artifacts are not shipped material).
# Phase 2 CI reuses this script verbatim as the per-commit secret gate.

patterns=(
  'eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+'
  'enc:v[12]:(os:)?[A-Za-z0-9+/=_-]{20,}'
  '"(access_token|apiKey|api-key|token)"\s*:\s*"[A-Za-z0-9_-]{20,}"'
)

rc=0
for p in "${patterns[@]}"; do
  out=$(git grep -nE "$p" -- . ':(exclude).planning' 2>/dev/null || true)
  if [ -n "$out" ]; then
    printf '%s\n' "$out" | head -5
    echo "SECRET SHAPE: $p"
    rc=1
  fi
done
exit $rc
