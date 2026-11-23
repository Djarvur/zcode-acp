---
phase: 01-research-protocol-documentation
reviewed: 2026-11-19T17:59:20Z
depth: deep
files_reviewed: 8
files_reviewed_list:
  - docs/protocol/acp-client-dialect.md
  - docs/protocol/app-server-dialect.md
  - experiments/app-server-probe.mjs
  - experiments/check-docs.sh
  - experiments/scan-secrets.sh
  - experiments/scrub-capture.mjs
  - experiments/scrub-capture.test.mjs
  - .gitignore
findings:
  critical: 1
  warning: 2
  info: 4
  total: 7
  necessity: 0
status: issues_found
---

# Phase 1: Code Review Report (re-review)

**Reviewed:** 2026-11-19T17:59:20Z
**Depth:** deep
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Re-review of Phase 01, scoped to the 8 files changed since the prior review commit
(7f1c4ab, "docs(phase-1): code review report + disposition ledger"). The changes came
from later-phase work: the Phase 2 CI reuse of the gates, the Phase 3 experiment
extensions (`--desktop-version`, `--end-mode quiet-window`), the doc drift-guard
updates, and the fixes for every finding in the prior review (WR-01..04, IN-01..04).
This document replaces the prior review for the current tree.

**Disposition verification — all 8 prior findings are fixed and were verified
empirically, not by reading the diff:**

- WR-01 (gate/scrubber credential-class divergence): fixed. `scan-secrets.sh:29` now
  carries the separator-inclusive class `[A-Za-z0-9+/=_:.-]`; verified the gate pattern
  matches the dotted blob the scrubber test proved real (`enc:v2:os:Aa09…Hh77Ii88Jj99`)
  while the old pattern matched nothing; the token pattern is now quote-free (line-final
  tokens scanned).
- WR-02 (broken leading-dash grep recipe): fixed. `app-server-dialect.md:146` documents
  the `-e` form; verified `grep -o -e '-32603'` returns counts on synthetic input.
- WR-03 (sibling-of-home path falsification): fixed. Boundary-aware rewrite at
  `scrub-capture.mjs:126-133` with three dedicated unit tests; suite green.
- WR-04 (asserted `desktopVersion` in the capture header): fixed. `--desktop-version`
  CLI flag added (`app-server-probe.mjs:384`), empty/unset stays `unknown` (line 417-419).
- IN-01 (digit-less extraction class dropped the v4 family): fixed. Class widened to
  `[a-zA-Z0-9/]*` and a one-line v4 extractor added (`app-server-dialect.md:41,69`);
  verified on synthetic minified text that the fixed class extracts
  `v4/conversation/subscribe` while the old class dropped it.
- IN-02 (singletons off-by-one): fixed — row now says (9).
- IN-03 (CWD-dependent gates): fixed. Both gates `cd "$(git rev-parse --show-toplevel)"`;
  verified green from a subdirectory.
- IN-04 (unvalidated scrubber header): fixed. Line-1 JSON+`capture`-key validation with
  `--no-header` escape hatch (`scrub-capture.mjs:194-211`), three unit tests.

The CR-01 from the prior review cycle (USER_INFO_RX `lastIndex` regression) is also
fixed (`scrub-capture.mjs:78`) with two regression tests.

**Checks that ran clean (adversarial probes that found nothing):** both gates green in
all modes (`scan-secrets`, `check-docs` default and `--phase-gate`, from repo root and
from `experiments/`); scrubber suite 20/20; `node --check` clean on all three JS files;
capture-header continuity verified for all six captures (each first line carries
`{"capture":"<id>"}`); broader secret sweep (`ghp_`/`AKIA`/`sk-`/`xox`/`Bearer eyJ`)
zero hits; `\s` verified recognized by `git grep -E` on this system (the gate's token
pattern depends on it); scrubber brace-matcher survives braces and escaped quotes
inside user_info strings in the bare form; no key-minting reference in any experiments
script; all doc link targets exist (README, workspace-key.md, install-discovery.md,
`.planning/research/REFERENCE-ADAPTERS.md`, `experiments/README.md`).

**The one critical finding:** the scrubber's `user_info` wholesale-redaction rule never
fires on the wrapped/escaped line form that `app-server-probe.mjs` writes for every
wire line — verified by running the scrubber on a wrapped capture with a planted
`user_info` payload; the email and a marker string survived verbatim. The rule's own
comment claims the wrapped form is handled; it is not. No committed capture currently
contains a `user_info` payload (swept both bare and escaped forms), so nothing has
leaked — but the rule is dead for this pipeline's primary input format, and the gate
layer has no email/PII shape to catch the residue downstream. Fix before the next
turn-capture run (Phase 4+ exercises real turns).

The two warnings are a probe exit-code hole (a crashed child with zero captured lines
exits 0) and a documented example `--window-ms` that expires before the last step of
both E1/E2 flows fires. The four info items are doc-internal precision defects in the
new acp-client-dialect material and one latent probe conflation.

## Critical Issues

### CR-01: Scrubber `user_info` redaction never fires on the wrapped/escaped wire-line form the probe writes

**File:** `experiments/scrub-capture.mjs:55` (USER_INFO_RX) and `64-66` (false claim in the rule comment)
**Issue:** USER_INFO_RX is `/"user_info"\s*:\s*\{/g` — it requires unescaped quotes
around `user_info`. But `app-server-probe.mjs:507` stores every wire line as
`JSON.stringify({ ts, line })`, i.e. the wire JSON appears as an escaped string inside
an envelope, where the key looks like `\"user_info\":{`. The regex cannot match that
sequence (`\"` after `user_info` breaks the pattern), so the wholesale-redaction rule —
one of the canonical rules the scrubber exists to enforce — is dead for the primary raw
format of this pipeline. Verified empirically: running the scrubber on
`{"ts":"…","line":"{\"method\":\"state.updated\",\"params\":{\"user_info\":{\"email\":\"leak@example.test\",\"marker\":\"WRAPPED_MARKER_XYZ\"},\"sequence\":3}}" }`
left both the email and the marker verbatim in the output. The rule comment
(`scrub-capture.mjs:64-66`) explicitly claims the rule "works regardless of whether the
line is bare JSON or a wrapped/timestamped capture line" — that claim is false for the
wrapped+escaped form. Downstream exposure is real: the residue is an email/PII payload,
which none of the three `scan-secrets.sh` shapes match, so a leak would land in
`experiments/captures/` silently. Swept all committed captures for `user_info` in both
bare and escaped forms: zero occurrences today — no leak has happened yet, but Phase 4+
turn captures are exactly where such payloads are anticipated. The bare form does work
(verified, including braces/escaped quotes inside the payload strings), so the
existing unit tests — all of which use bare lines — cannot catch this.
**Fix:**
Make the rule escaped-form aware, and add a unit test that feeds the probe's actual
envelope form (`JSON.stringify({ ts, line: JSON.stringify(wire) })`):

```js
// match both "user_info": and the escaped \"user_info\": sequence
const USER_INFO_RX = /\\?"user_info\\?"\s*:\s*(\\\\?\{)/g;
```

and make the string-aware brace scanner treat a backslash before `"` as content (an
escaped quote is never a string boundary) so the closing-brace search stays correct
inside the escaped envelope. Add the wrapped-form regression test alongside the two
existing CR-01 tests in `scrub-capture.test.mjs`.

## Warnings

### WR-01: Probe exits 0 (success) on a zero-capture run whenever the child exits non-zero

**File:** `experiments/app-server-probe.mjs:661`
**Issue:** The EOF-pitfall guard is `const earlyZero = exitInfo.code === 0 && capturedLines === 0;` —
it covers only the exit-0 path. Verified empirically with stub electrons: a child that
exits 0 with zero lines → probe exits 1 with the warning; a child that exits 1
immediately (wrong `--cjs` path, crash-on-boot) with zero lines → probe prints
`lines=0` on stderr but **exits 0**. Both runs produced no wire truth; only one is
treated as a failure. The capture in the crash case contains the header plus honest
no-reply verdicts, so the evidence is not falsified — but the harness's own failure
signal (exit code) reports success for the same no-wire-truth condition the guard was
built to catch, and the guard's warning text ("or a bad --electron/--cjs path")
promises exactly this case is flagged.
**Fix:**
```js
if (capturedLines === 0) {   // drop the exit-code condition
```
so any zero-line run warns and exits 1 regardless of how the child exited.

### WR-02: Documented example `--window-ms 12000` expires before the last step of both 01-02 flows fires (13000 / 13500 ms)

**File:** `experiments/app-server-probe.mjs:431-433` (usage), `:15` (header comment), vs `:284` (subscribe `session/events` at 13000 ms) and `:342` (turn `session/send` at 13500 ms)
**Issue:** The usage text pairs every flow — including `subscribe` and `turn` — with
`--window-ms 12000`. Both post-01-02 flows schedule their final (load-bearing) probe
beyond that window. Traced statically: at 12 s the window backstop wins the race
(`endReason: 'window-expired'`), verdicts are appended, and `child.stdin.end()` runs;
when the 13.5 s timer then fires inside the 4 s grace window, its `send()` writes to an
already-ended stdin and is swallowed by the EPIPE error handler (`:496`) — the
`session/send` (or `session/events`) probe is never delivered, and the run completes
exit 0. The recorded captures prove the real E1/E2 runs used longer windows (the send
reply is in `EXP-TURN-01`), so committed evidence is unaffected — but anyone following
the script's own usage for a fresh E2 run gets a quietly useless run of the
quota-spending flow. The probe knows its steps; it should not accept a window shorter
than its own last step.
**Fix:**
Raise the documented example (e.g. `--window-ms 30000`) and validate after the flow
resolves:

```js
const lastAtMs = Math.max(...steps.map((s) => s.atMs));
if (flags.windowMs <= lastAtMs) {
  process.stderr.write(`app-server-probe: --window-ms must exceed the flow's last step (${lastAtMs} ms)\n`);
  process.exit(1);
}
```

## Info

### IN-01: Header claims "the two remaining known-unknowns" — the Open items section lists three

**File:** `docs/protocol/acp-client-dialect.md:10-11` (vs `:475-490`)
**Issue:** The header says "the two formerly open items are answered statically … the
two remaining known-unknowns stay V-RPT-labeled." The "Still unresolved" block records
**three** V-RPT-labeled known-unknowns (settings-store persistence path, end-to-end
`--version` probe, interruption semantics). The count is one stale — the
interruption-semantics bullet was presumably added after the header was written. In a
doc whose genre is exact counts and honesty labeling, the abstract miscounting its own
open items invites exactly the distrust the discipline is meant to prevent.
**Fix:** Change the header to "the three remaining known-unknowns stay V-RPT-labeled"
or restructure to "two answered statically; the remaining known-unknowns (settings
store, live `--version`, interruption semantics) stay V-RPT-labeled."

### IN-02: Build-match claim stated at two different strengths — "byte-identical" vs "byte-count-identical"

**File:** `docs/protocol/acp-client-dialect.md:16` (vs `:519`)
**Issue:** The header says v1.4.262 is "the newest release whose card registry is
byte-identical to the installed build's 8-card chunk registry"; the Sources section
says the same registry "is byte-count-identical to v1.4.261 and v1.4.262". These are
different claim strengths (content-equal vs count-equal); both cannot be the operative
verification. Everything downstream (the "v1.4.262 is the citation state" decision,
the build-window bracketing argument) inherits whichever is true. The claim itself is
externally sourced and not re-verifiable from this repo — which is precisely why the
doc must state one strength consistently.
**Fix:** Pick the claim the cross-check actually established (likely "same 8-entry
registry content" for v1.4.262, with the weaker count-only phrasing reserved for
v1.4.261) and use the same wording in both places.

### IN-03: `EXP-ORCASRC-01` is cited in prose only — its capture header escapes the resolution and continuity gate

**File:** `docs/protocol/acp-client-dialect.md:517` (and `experiments/check-docs.sh:92-112`)
**Issue:** The repo-source citation half of the acp doc names `capture EXP-ORCASRC-01`
as its evidence origin, but never in bracketed `[V-EXP EXP-ORCASRC-01]` form, so
check-docs checks (b) and (f) — V-EXP resolution and capture-header continuity — never
see it (verified: the gate's own id sweep contains no ORCASRC tag). The capture exists
today with a valid header, so nothing is broken; the gap is that a re-scrub or edit
that mangled that header would not fail the gate, unlike the other five captures.
Half of the doc's load-bearing `[V-SRC repo …]` claims rest on it.
**Fix:** Either tag the repo-source claims that rest on that capture with
`[V-EXP EXP-ORCASRC-01]` (it is an experiment capture and already V-EXP-shaped), or
extend check (f) with an explicit list of capture ids whose headers must stay valid
regardless of tag form.

### IN-04: `observeReply` conflates server→host requests with replies (latent verdict falsification path)

**File:** `experiments/app-server-probe.mjs:92-105, 512-521`
**Issue:** In the stdout handler, a line carrying `method` + `id` — a server→host
REQUEST — is passed to `observeReply`, which stores it in `state.replies[wire.id]` and
checks `PROBED_METHODS[wire.id] === 'session/send'`. If the server ever sent a reverse
request whose id collided with a numeric probe id (1–5), the request object would
shadow the real reply: `classifyReply` would report `accepted` (requests have neither
`result` nor `error`) for a call the server never answered, or `sendReplyTs` would
anchor early and open the quiet-window latency floor prematurely. Latent, not live:
the recorded captures show reverse requests arrive with string ids (`"server-1"`, …),
which cannot collide — the same convention `app-server-dialect.md` §1 records as wire
truth. Flagged because the two wire directions flowing through one observer is the
kind of accident that survives until a server version changes the id scheme.
**Fix:** In the request branch, skip `observeReply` (or make it ignore lines that carry
a `method`): only lines without a `method` are candidate replies.

---

_Necessity review: the rubric ($HOME/.zcode/gsd-core/references/necessity-review.md)
was read; no necessity findings. The guarded states in scope (wire data at a trust
boundary, child-process lifecycle, user-supplied machine paths, header-validation
null-vs-object ordering) are all reachable; nothing defends against a provably
impossible state. `necessity: 0`._

---

_Reviewed: 2026-11-19T17:59:20Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
