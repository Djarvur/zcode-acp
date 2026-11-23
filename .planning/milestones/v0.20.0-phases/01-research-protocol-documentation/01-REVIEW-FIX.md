---
phase: 01-research-protocol-documentation
fixed_at: 2026-11-19T21:45:00Z
review_path: .planning/phases/01-research-protocol-documentation/01-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

Fixed at: 2026-11-19T21:45:00Z. Source review: 01-REVIEW.md (re-review). Iteration: 1.
Findings in scope: 7. Fixed: 7. Skipped: 0.

Commits landed via branch `gsd-reviewfix/01-2735242` (base 6771e3d), integrated into
`gsd/phase-03-zcode-back-leg-codec-fake-app-server-supervisor` by merge `8a1933d`.

## Fixed Issues

### CR-01: Scrubber `user_info` redaction never fires on the wrapped/escaped wire-line form the probe writes
TDD RED→GREEN: 641502f (failing wrapped-form regression tests, 21/22) → 38bbc20
(USER_INFO_RX now escape-aware `/(\\?)"user_info(\\?)"\s*:\s*\{/g` + `findObjectEnd`
level-by-level escape decoder; replacement mirrors the escaping so the envelope stays
parseable). Empirical: the review's exact planted wrapped payload scrubs to
`[REDACTED-USERINFO]`, envelope parses, fixed point holds. Requires human verification
(two-level escape decoder encodes judgment about hostile input shapes).

### WR-01: Probe exits 0 (success) on a zero-capture run whenever the child exits non-zero
8bb924f — guard keyed on `capturedLines === 0` alone. Empirical: crash-exit stub with zero
lines exits 1 with warning post-fix (exit 0 pre-fix); clean-exit-0 stub also exits 1.

### WR-02: Documented example `--window-ms 12000` expires before the last step of both 01-02 flows fires (13000 / 13500 ms)
cb350c5 — example window 12000→30000 in probe usage/header comment and
experiments/README.md; plus a pre-spawn guard: `windowMs <= max(step.atMs)` exits 1
naming the last step, before any spawn/header write. Verified: 12000 rejected for
subscribe/turn, 30000 accepted.

### IN-01: Header claims "the two remaining known-unknowns" — the Open items section lists three
f199946 — header now says three known-unknowns and names them.

### IN-02: Build-match claim stated at two different strengths — "byte-identical" vs "byte-count-identical"
9e9406d — both spots unified to byte-count-identical (the strength with recorded
evidence); two remaining "byte-identical" hits are different, correct claims, untouched.

### IN-03: `EXP-ORCASRC-01` is cited in prose only — its capture header escapes the resolution and continuity gate
1efe48c — `[V-EXP EXP-ORCASRC-01]` tag added; capture exists and its line-1 header
carries the id; checks (b)+(f) now cover it (gate sweep re-run green).

### IN-04: `observeReply` conflates server→host requests with replies (latent verdict falsification path)
1be656b — observeReply ignores any method-bearing frame (comment cites
docs/protocol/app-server-dialect.md §1); request branch no longer calls it. Empirical
falsification reproduced pre-fix (verdict reported `session/subscribe: accepted` from a
stub `{id:3, method:"interaction/requestPermission"}`), honest `no-reply` post-fix.
Requires human verification.

## Verification

Gates re-run green on the merged main checkout: `check-docs.sh --phase-gate` exit 0,
`scan-secrets.sh` exit 0, `node --test experiments/` 22/22, run before every commit in
the fix worktree too (suite excepted at the RED commit by design). `go test ./...` at
merge time fails in `internal/zcode` — that is the parallel session's in-flight TDD RED
phase for plan 03-02 (commits 3eef389/a4c631b/f67e21d, dirty conn.go/call_test.go),
not this fix batch: the merge touches no Go files.

## Notes for follow-up

- `TOKEN_RX` in experiments/scrub-capture.mjs still matches bare-quote keys only — an
  escaped-form `\"access_token\":\"<20+ chars>\"` inside the probe envelope is not
  caught (JWT/credential/UUID rules are shape-based and safe). Worth a follow-up
  finding in the next review pass.
