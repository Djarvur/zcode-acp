#!/usr/bin/env bash
# experiments/check-docs.sh — structure + provenance-tag conformance gate for docs/
# (D-03/D-04, per 01-VALIDATION.md Wave 0). Pure grep/jq; must stay <30 s.
#
# Modes:
#   (default)     conformance checks over the docs that exist
#   --phase-gate  additionally requires ALL SIX deliverable docs to exist and each
#                 present doc to carry its required section headings
#
# Output contract:
#   CHECK FAIL: <msg>              -> violation; exit 1
#   PENDING: <path> (n/6 present)  -> deliverable doc not written yet; affects exit
#                                     code only under --phase-gate
#
# Checks:
#   (a) the heading "## Provenance legend" appears exactly once across docs/, and
#       only in docs/protocol/README.md (other docs reference the README);
#   (b) every [V-EXP <id>] token in any docs/**/*.md resolves to an existing
#       experiments/captures/<id>* file;
#   (c) tag grammar: every bracketed provenance tag is [V-SRC|V-EXP|V-RPT] plus a
#       non-empty argument;
#   (d) per-doc required-section check (exact headings; --phase-gate only, so
#       partially-written docs keep per-task commits green). Heading lists are the
#       ones specified by each doc-writing task in plans 01-01..01-06;
#   (e) path-shape scan over docs/ + experiments/ (captures included): any absolute
#       path resolving under a user home directory is a violation (neutral-phrasing
#       rule; the regex lives in this script only, never in committed prose).

set -u
MODE=conformance
[ "${1:-}" = "--phase-gate" ] && MODE=phase-gate
rc=0
fail() { echo "CHECK FAIL: $*"; rc=1; }

docs=(
  docs/protocol/README.md
  docs/protocol/app-server-dialect.md
  docs/protocol/acp-client-dialect.md
  docs/protocol/install-discovery.md
  docs/protocol/workspace-key.md
  docs/research/reference-adapters.md
)

# Required section headings per doc (exact heading lines, "|" separated).
# Source of truth: the doc-writing tasks of plans 01-01..01-06.
declare -A REQUIRED_SECTIONS
REQUIRED_SECTIONS["docs/protocol/README.md"]="## Provenance legend"
REQUIRED_SECTIONS["docs/protocol/workspace-key.md"]="## Wire schema|## Derivation convention|## Experiment record|## Host-freedom note|## Sources & provenance"
REQUIRED_SECTIONS["docs/protocol/app-server-dialect.md"]="## 1. Transport facts|## 2. Method map|## 3. Server-to-host surface|## 4. session/subscribe and event-stream availability|## 5. Turn-completion terminal signals|## 6. Error-code table|## 7. Session lifecycle methods|## 8. Snapshot semantics|## 9. 3.17 vs 3.18.2 drift baseline|## Sources & provenance"
REQUIRED_SECTIONS["docs/protocol/acp-client-dialect.md"]="## Preset-agent identity|## Card registry of this build|## Launch contract|## Version floor|## Load-bearing method set|## Full ACP method range the client sends|## String contracts|## Client capabilities|## Open items|## Sources & provenance"
REQUIRED_SECTIONS["docs/protocol/install-discovery.md"]="## Layout matrix|## Locating the entrypoint per layout|## Desktop entry and URL handler|## Auto-update and the builtin revision|## Recommended discovery order|## Sources & provenance"
REQUIRED_SECTIONS["docs/research/reference-adapters.md"]="## What the reference covers|## Adopt|## Avoid|## Attribution|## Weaker adapters|## Sources & provenance"

# --- missing-doc report (exit impact only under --phase-gate) -----------------
present=0
for f in "${docs[@]}"; do [ -f "$f" ] && present=$((present + 1)); done
for f in "${docs[@]}"; do
  if [ ! -f "$f" ]; then
    echo "PENDING: $f ($present/6 present)"
    [ "$MODE" = "phase-gate" ] && rc=1
  fi
done

# --- (a) legend defined exactly once, only in docs/protocol/README.md --------
legend_total=0
legend_file=""
while IFS=: read -r f c; do
  legend_total=$((legend_total + c))
  legend_file="$f"
done < <(grep -rc --include='*.md' '^## Provenance legend' docs/ 2>/dev/null | grep -v ':0$' || true)

if [ "$legend_total" -eq 0 ]; then
  fail "provenance legend missing: expected exactly one '## Provenance legend' heading, in docs/protocol/README.md"
elif [ "$legend_total" -gt 1 ]; then
  fail "provenance legend defined $legend_total times; it must be defined exactly once (docs/protocol/README.md), other docs reference the README"
elif [ "$legend_file" != "docs/protocol/README.md" ]; then
  fail "provenance legend must live only in docs/protocol/README.md (found in: $legend_file)"
fi

# --- (b) every [V-EXP <id>] resolves to experiments/captures/<id>* -----------
# Arguments wrapped in angle brackets (e.g. <experiment-id>) are definition-site
# placeholder forms from the legend — not concrete ids — and are skipped.
while IFS= read -r tag; do
  [ -n "$tag" ] || continue
  id=${tag#\[V-EXP }
  id=${id%\]}
  case "$id" in
    '<'*) continue ;;
  esac
  if ! ls experiments/captures/"$id"* >/dev/null 2>&1; then
    fail "V-EXP id does not resolve: $tag -> expected experiments/captures/$id*"
  fi
done < <(grep -rhoE '\[V-EXP [^]]+\]' --include='*.md' docs/ 2>/dev/null | sort -u)

# --- (c) tag grammar ----------------------------------------------------------
bad_tags=$(grep -rhoE '\[V-[A-Za-z]+[^]]*\]' --include='*.md' docs/ 2>/dev/null \
  | grep -vE '^\[V-(SRC|EXP|RPT) [^]]+\]$' | sort -u)
if [ -n "$bad_tags" ]; then
  while IFS= read -r t; do
    fail "malformed provenance tag (need [V-SRC|V-EXP|V-RPT <non-empty arg>]): $t"
  done <<< "$bad_tags"
fi

# --- (d) per-doc required sections (--phase-gate only) ------------------------
if [ "$MODE" = "phase-gate" ]; then
  for f in "${docs[@]}"; do
    [ -f "$f" ] || continue
    IFS='|' read -ra heads <<< "${REQUIRED_SECTIONS[$f]}"
    for h in "${heads[@]}"; do
      grep -Fqx "$h" "$f" || fail "$f missing required section heading: $h"
    done
  done
fi

# --- (e) path-shape scan: no user-home absolute paths anywhere ----------------
path_rx='/(home|Users|root)/[A-Za-z0-9._-]+'
path_hits=$(grep -rnE "$path_rx" docs/ experiments/ 2>/dev/null || true)
if [ -n "$path_hits" ]; then
  printf '%s\n' "$path_hits" | head -5
  fail "absolute path under a user home directory committed (neutral-phrasing rule; use tilde or <placeholder> forms)"
fi

exit $rc
