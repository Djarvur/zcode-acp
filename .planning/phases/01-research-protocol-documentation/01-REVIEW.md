---
phase: 01-research-protocol-documentation
reviewed: 2026-11-19T12:05:00Z
depth: deep
files_reviewed: 12
files_reviewed_list:
  - experiments/check-docs.sh
  - experiments/scan-secrets.sh
  - experiments/scrub-capture.mjs
  - experiments/scrub-capture.test.mjs
  - experiments/app-server-probe.mjs
  - experiments/README.md
  - docs/protocol/README.md
  - docs/protocol/app-server-dialect.md
  - docs/protocol/acp-client-dialect.md
  - docs/protocol/install-discovery.md
  - docs/protocol/workspace-key.md
  - docs/research/reference-adapters.md
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
  necessity: 0
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2026-11-19T12:05:00Z
**Depth:** deep
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Deep review of the Phase 1 wire-truth pipeline: both validation gates, the capture
scrubber and its tests, the app-server probe harness, the experiments ops guide, and
all six deliverable docs. Every claim below was verified against the current working
tree; both gates were executed (scan-secrets green, check-docs green), the scrubber
test suite was run (12/12 pass), and suspicious regex/expansion behavior was tested
empirically rather than assumed.

Two important notes for the record:

1. **Transient blocker observed and resolved mid-review.** While this review was
   running, `experiments/check-docs.sh` was being edited concurrently. An intermediate
   working-tree version added the T-1-03 `api_keys` guard WITHOUT excluding the gate
   script from its own scan, so `bash experiments/check-docs.sh` exited 1 on a clean
   tree — the "run before every commit" gate was permanently red (self-match on its
   own comment/lines). The same edit window also briefly shipped a `head -n 1`-over-a-glob
   variant of the T-1-07 header check that only inspected one capture file. Both were
   fixed in the current tree (`--exclude=check-docs.sh` with an explanatory comment at
   line 139-141; per-file iteration at lines 100-107) and the gate is green now. No
   finding is raised against the final state — but make sure this fixed version is
   what actually gets committed.
2. **Checks that came back clean (adversarial probes that failed to find anything):**
   - *Catastrophic regex backtracking:* none. All four scrubber/gate regexes use a
     single character class per quantifier separated by literal delimiters; an
     adversarial 48–192 KB `eyJ...`-heavy line was constructed and matched in 0 ms
     (verified twice). No nested quantifiers exist.
   - *Injection via captured wire data:* none. The probe spawns via array-form
     `spawn()` (no shell), the scrubber is pure string surgery, and scrubbed captures
     are only ever re-parsed as JSON downstream.
   - *Secret-shape residue / machine paths in committed material:* both gates green;
     manual grep for `/(home|Users|root)/` shapes over `docs/` + `experiments/` found
     zero hits; capture headers verified for all five experiment ids.
   - *Cross-references:* every markdown link and backticked `.md`/`.planning` path in
     the six deliverable docs resolves (automated check, zero broken).
   - *Known context confirmed intentional (not findings):* `check-docs.sh` PENDING
     lines + exit 0 in default mode; `--phase-gate` is the strict variant (verified
     both modes); no real machine paths in any script (all bundle/scratch paths are
     CLI args or `<placeholder>` forms); the scrubber's separator-blob RED→GREEN fix
     is present and tested.

The four warnings are gate/scrubber pattern divergence, a broken re-derivation recipe
in the dialect doc, a data-falsifying path-rewrite edge in the scrubber, and a hole in
the probe's stale-evidence header guard. All are small, concrete fixes.

## Critical Issues

None found in the final tree (see Summary note 1 regarding the transient
self-matching gate state observed mid-review).

## Warnings

### WR-01: Secret-scan gate misses separator-containing credential bodies (diverges from the scrubber's corrected rule)

**File:** `experiments/scan-secrets.sh:19`
**Severity:** warning
**Issue:** The gate's credential pattern is `enc:v[12]:(os:)?[A-Za-z0-9+/=_-]{20,}` —
its body class omits `.` and `:`. But the scrubber's own TDD cycle (RED→GREEN defect
fix, `scrub-capture.test.mjs:90-100`) established that real `enc:v2:` bodies contain
separators, and `scrub-capture.mjs:48` (CRED_RX) was corrected to
`[A-Za-z0-9+/=_:.-]{20,}`. Verified empirically: against
`enc:v2:os:Aa09+/=_Bb11.Cc22:Dd33Ee44Ff55Gg66.Hh77Ii88Jj99` the scan pattern finds
nothing while the scrubber pattern matches the whole blob. A credential blob that
leaks past the scrubber (or is hand-committed) in the exact shape the test suite
proves real would pass the per-commit secret gate that "Phase 2 CI reuses this script
verbatim". Same family: the gate's token pattern requires a closing quote while the
scrubber's TOKEN_RX does not, so a line-final token value is scrubbed but not scanned.
**Fix hint:** Align the gate with the scrubber's corrected class:
`'enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}'`; optionally drop the closing-quote
requirement from pattern (c) to match TOKEN_RX.

### WR-02: Error-code re-derivation recipe is broken for every documented code (leading-dash patterns)

**File:** `docs/protocol/app-server-dialect.md:146`
**Severity:** warning
**Issue:** Section 6's header promises the occurrence counts are re-derivable "via
`grep -o <code> <bundle>/resources/glm/zcode.cjs | wc -l`". Every code in the table is
negative, and GNU grep cannot take a leading-dash pattern as a bare argument:
verified with GNU grep 3.11 — `grep -o -32603 <file>` silently parses `-32603` as
numeric context options (`-3`, `-2`, … are all valid), then treats the bundle path as
the PATTERN and reads stdin, returning rc=1 with no output and no error. Quoting does
not help (`grep -o '-32603' file` fails identically). The recorded counts may well be
correct, but the recipe as written cannot produce them — and fails silently, which is
worse for a doc whose core promise is hand-re-derivability of every bundle-anchored
claim.
**Fix hint:** Record the working form, e.g.
`grep -o -e '-32603' <bundle>/resources/glm/zcode.cjs | wc -l` (verified working), or
`grep -o -- '-32603' …`.

### WR-03: Scrubber tilde-rewrite falsifies sibling-of-home paths

**File:** `experiments/scrub-capture.mjs:107-110`
**Severity:** warning
**Issue:** The home rule runs `split(home + '/')` then `split(home)` with no boundary
check on the second replacement. Verified empirically with `home=/home/<user>`: an
absolute path belonging to a SIBLING home entry, `/home/<user>x/otherproj/file.txt`,
is rewritten to `~x/otherproj/file.txt` — no longer a valid absolute path, not a
truthful tilde form, and it falsely implies the path is under the user's home. The
scrubbed captures are the provenance substrate for Phase 3 golden fixtures, so a
silently falsified path poisons evidence rather than just failing to redact. (Output
is still idempotent, so the fixed-point test cannot catch it.) Reachable in practice
whenever wire data contains a path sharing the home string as a strict prefix
(multi-user hosts, container bind-mounts, `-suffix` sibling accounts).
**Fix hint:** Make the bare-home replacement boundary-aware — apply only
`home + '/' → '~/` unconditionally, then replace bare `home` occurrences only when
NOT followed by a path/name character, e.g.
`line.replace(new RegExp(escapeRegExp(home) + '(?!/[A-Za-z0-9._-])', 'g'), '~')`
(or only at JSON string boundaries: followed by `"`, end-of-string, whitespace, or a
closing delimiter), and add a unit test for the sibling-prefix case.

### WR-04: Probe capture header hardcodes `desktopVersion` while claiming a capture-time stale-evidence guard

**File:** `experiments/app-server-probe.mjs:473`
**Severity:** warning
**Issue:** The header writes `desktopVersion: '3.18.2'` as a constant while
`builtinRevision` is deliberately read at capture time "READ AT CAPTURE TIME
(stale-evidence guard against desktop auto-update mid-phase)" (lines 31-33, 458-465) —
the same staleness concern. `install-discovery.md` documents that the revision is
NOT per-version unique (3.16.3–3.17.2 share revision 43), so a desktop auto-update
mid-phase can produce captures whose header asserts version 3.18.2 with a stale (or
even unchanged) revision and no in-header way to detect either. Docs then cite the
header as evidence: workspace-key.md states the capture "pins `desktopVersion` 3.18.2
and `builtinRevision` 44 read from the active builtin config at capture time" — the
version half of that pin is asserted, not read. For a phase whose deliverable is
honest evidence labeling (D-05), the header's version field should not be a constant.
**Fix hint:** Read the desktop version at capture time (e.g. from the bundle's
version metadata next to the builtin file, or accept a `--desktop-version` CLI arg
resolved by the runner), or rename the field to make clear it is the phase baseline
(e.g. `targetDesktopVersion`) rather than the probed truth.

## Info

### IN-01: Method-map extraction command (a) silently drops digit-bearing method strings (the entire v4 family)

**File:** `docs/protocol/app-server-dialect.md:40-41`
**Severity:** info
**Issue:** The §2 re-derivation pipeline ends with `grep -o '"[a-zA-Z/]*"$'`, whose
class has no digits. Verified empirically: `v4Subscribe:"v4/conversation/subscribe"`
yields no match (the `4` breaks the class), while `plugins/listUiSurfaces` survives.
All 43 v4 method strings contain a digit, so a reader re-deriving the map via the
recorded commands gets the 110-string host→server map but silently none of the v4
family the doc separately inventories. The primary 110-map claim is unaffected; the
re-derivability promise for the v4 block is not.
**Fix hint:** Widen the class to `'"[a-zA-Z0-9/]*"$'` in the recorded command, or add
a one-line v4 extractor (e.g. `grep -o '"v4/[^"]*"' <bundle> | sort -u`) to the v4
paragraph.

### IN-02: Singletons row label says "(8)" but lists 9 methods

**File:** `docs/protocol/app-server-dialect.md:65`
**Severity:** info
**Issue:** The method-map row `singletons (8)` enumerates nine methods
(`bot/channel/reply`, `offPeak/create`, `offPeak/list`, `runtime/capabilities`,
`skills/referenceCatalog`, `subagents/readRuntimeConfig`, `topic/resource/read`,
`topic/resource/cancel`, `usage/stats` — machine-counted). The headline "110 method
strings" is only arithmetically consistent with 9 (27+3+7+2+9+6+21+18+8+9 = 110), so
the list is right and the row label is the wrong half. In a doc genre built on exact
counts, an internal off-by-one invites readers to distrust the neighboring counts.
**Fix hint:** Change the row label to `singletons (9)`.

### IN-03: Gates assume repo-root CWD; scan-secrets silently narrows its scan from a subdirectory

**File:** `experiments/scan-secrets.sh:25` (and `experiments/check-docs.sh` generally)
**Severity:** info
**Issue:** `git grep -nE "$p" -- . ':(exclude).planning'` has a CWD-relative pathspec:
invoked from `experiments/` it silently scans only that subtree and exits 0 — the
per-commit secret gate passes while scanning less, with no signal. (`check-docs.sh`
fails loudly from a wrong CWD instead, which is acceptable.) Since Phase 2 CI reuses
scan-secrets.sh verbatim, make the cwd assumption structural rather than implicit.
**Fix hint:** Start both gate scripts with
`cd "$(git rev-parse --show-toplevel)" || exit 1`.

### IN-04: Scrubber passes line 1 through completely unscrubbed — leaks on header-less raw captures

**File:** `experiments/scrub-capture.mjs:163`
**Severity:** info
**Issue:** The byte-identical-header contract is deliberate (the probe always writes
line 1 itself), but the scrubber accepts ANY file, and a raw capture produced by
anything other than this probe (e.g. predecessor-shim captures, a hand-edited run) has
no header, so its first line — potentially a full wire message with credentials —
enters `experiments/captures/` verbatim. Mitigated for the known shapes: the
committed capture is still scanned by `scan-secrets.sh` and the path-shape scan in
`check-docs.sh` (e), so JWT/cred/token/home-path shapes would trip a gate; unknown
shapes would not. Worth a cheap structural guard or an explicit doc warning, since
the scrubber is documented as "the ONLY path into captures/".
**Fix hint:** Either validate that line 1 parses as JSON containing a `capture` key
(fail loudly otherwise, with a `--no-header` escape hatch), or state in README +
script comment that line 1 is trusted because the probe wrote it.

---

_Necessity review: the rubric ($HOME/.zcode/gsd-core/references/necessity-review.md)
was read; no necessity findings — the guarded states here (wire data at a trust
boundary, child-process lifecycle, user-supplied machine paths) are all reachable.
`necessity: 0`._

---

_Reviewed: 2026-11-19T12:05:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
