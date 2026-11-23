---
phase: 02-scaffold-ci-bootstrap
reviewed: 2026-11-19T14:53:06Z
depth: deep
files_reviewed: 16
files_reviewed_list:
  - mise.toml
  - .golangci.yml
  - go.mod
  - cmd/zcode-acp/main.go
  - .gitignore
  - .github/workflows/pr-sanity.yml
  - .github/workflows/release.yml
  - .github/workflows/e2e-matrix.yml
  - .github/workflows/security-scheduled.yml
  - experiments/scan-secrets.sh
  - experiments/check-docs.sh
  - experiments/scrub-capture.mjs
  - experiments/scrub-capture.test.mjs
  - experiments/app-server-probe.mjs
  - README.md
  - .planning/phases/02-scaffold-ci-bootstrap/protection-body.json
findings:
  critical: 1
  warning: 2
  info: 6
  total: 9
  necessity: 0
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-11-19T14:53:06Z
**Depth:** deep
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Deep review of the Phase-2 scaffold, CI surface, and the capture pipeline scripts, with
every adversarial claim below verified by execution, not by reading alone. The CI and
protection wiring — the phase's stated "must not hang future PRs" requirement — is
**verified sound**: the three required contexts in `protection-body.json` match the
`pr-sanity` job names character-for-character, the live `GET /branches/master/protection`
returns exactly the body's configuration (app_id 15368), and `gh run list` shows
`pr-sanity` green on both the push and pull_request legs. `scan-secrets.sh` and
`check-docs.sh` both run green from the repo root **and** from a subdirectory (toplevel
pin verified), no untrusted event field reaches any `run:` block, `GITHUB_TOKEN` write
scope exists only in `release.yml`, mise tasks were empirically confirmed to execute at
the project root (so `tidy-diff` pathspecs are correct from any invocation directory),
and all 18 scrubber unit tests pass.

The one genuine defect is severe: **the capture scrubber — the only path into
`experiments/captures/` — leaks every `user_info` payload object after the first on a
line**, caused by a stateful global regex. I reproduced the leak end-to-end with the real
script (two `user_info` objects on one line: the first is redacted, the second survives
verbatim with its `email` field), and the output is not a fixed point, violating the
scrubber's documented idempotency contract. This re-opens T-2-01 (information
disclosure) in `02-SECURITY.md`, whose `closed` disposition rests on a test suite that
has no multi-object line coverage. No committed capture contains `user_info` data today,
so the leak is latent, not yet realized.

Two warnings (job timeouts on the required-check workflows; probe verdict integrity under
host-request id reuse) and six advisory items round out the report.

Verified-clean highlights: check-name ↔ protection-context match; protection body == live
state; CWD independence of both gates; mise task root-cwd behavior; workflow injection
surface (0 untrusted interpolations); token scoping (read-only everywhere but release);
`.gitignore` coverage of raw captures and the root binary; go.mod/mise version agreement;
scan patterns byte-identical to the scrubber's CRED_RX/JWT_RX classes as claimed.

## Critical Issues

### CR-01: Stateful global regex leaks every `user_info` payload after the first on a line — PII can enter committed captures

**File:** `experiments/scrub-capture.mjs:55,71-73` (loop `67-99`)
**Issue:** `USER_INFO_RX` is a module-level `/g` regex driven by `exec()` inside the
`replaceUserInfos()` loop. `RegExp.prototype.exec` on a global regex is **stateful**:
`lastIndex` persists across calls, but the loop re-assigns `rest` to a *shorter* suffix of
the line after each redaction. The next `exec(rest)` then starts scanning from a
`lastIndex` computed in the *previous* string's coordinates, so `user_info` occurrences
near the start of the remaining suffix are skipped, and when `lastIndex > rest.length`
the scan fails outright (returning `out + rest` without re-scanning from position 0).
Reproduced with the real script:

```
input : {"params":{"user_info":{"email":"leak-one@example.com"},"prev":{"user_info":{"email":"leak-two@example.com"}}}}
pass 1: {"params":{"user_info":"[REDACTED-USERINFO]","prev":{"user_info":{"email":"leak-two@example.com"}}}}
pass 2: {"params":{"user_info":"[REDACTED-USERINFO]","prev":{"user_info":"[REDACTED-USERINFO]"}}}
```

Consequences, all verified by execution:
1. `leak-two@example.com` survives the canonical scrub — the scrubber is the ONLY ingress
   into `experiments/captures/`, so PII (the field the project's own test asserts must not
   survive: `scrub-capture.test.mjs:194`) can be committed silently. `scan-secrets.sh`
   scans credential *value shapes* only — an unredacted `user_info` matches none of its
   three patterns, so no other gate catches it.
2. Output is not a fixed point (`cmp` pass1 vs pass2 differs), violating the header
   comment's idempotency guarantee (`scrub-capture.mjs:32`) and the scrub-first
   provenance model.
3. A three-object line loses only the second (`[RED]`, leak, `[RED]`) — nondeterministic
   coverage that makes the failure easy to miss in spot checks.
4. `02-SECURITY.md` T-2-01 (information disclosure, high) is marked `closed` on the
   strength of 18/18 tests; no test exercises two `user_info` objects in one line, so the
   disposition is premature.

No committed capture currently contains `user_info` lines, so no leak has landed yet —
this is a broken security boundary, not a realized incident. Fix before any further
`state.updated` captures are scrubbed (the E2 turn flow is exactly where multi-payload
lines are plausible).

**Fix:**

```js
for (;;) {
  USER_INFO_RX.lastIndex = 0; // exec() on a /g regex is stateful; rest shrank this iteration
  const m = USER_INFO_RX.exec(rest);
  if (!m) return out + rest;
  ...
}
```

(The loop stays terminating because `rest` shrinks past each match.) Add a regression
test: one line with two or three `user_info` objects → all redacted, and re-assert the
fixed-point property on that output. The failure signature is silent, so the test is the
only guard.

## Warnings

### WR-01: Required-check jobs have no `timeout-minutes` — a hung gate holds a runner for the 6-hour default

**File:** `.github/workflows/pr-sanity.yml:48-53,92-95,119-122`; `.github/workflows/release.yml:40-43`; `.github/workflows/security-scheduled.yml:21-24`
**Issue:** Per the CI/CD rubric ("a job with `runs-on:` and no job-level
`timeout-minutes:`"), five jobs lack a timeout: all three `pr-sanity` jobs (two of which
— `govulncheck` and `secret-scan` — are branch-protection required checks), the release
job, and the scheduled govulncheck job. `e2e-matrix.yml:45` gets this right
(`timeout-minutes: 150`). A hung `mise install`, curl, or govulncheck DB fetch on the
required checks stalls the check in "running" — which is precisely the "future PR hangs"
failure mode this phase was asked to prevent, just via a different mechanism than a name
mismatch (names were verified correct).
**Fix:** Add e.g. `timeout-minutes: 15` to each of the five jobs (the observed end-to-end
runs complete in ~30 s; 15 minutes leaves generous headroom).

### WR-02: Probe stores server→host *requests* in the replies map — an id-space collision turns a no-reply verdict into a false "accepted"

**File:** `experiments/app-server-probe.mjs:94,128-142`
**Issue:** `observeReply()` records **every** wire object with `id != null` into
`state.replies` — including server→host REQUESTS (which carry `method`), not just
replies. `classifyReply()` then treats any stored entry as a reply: a request has no
`error` and no `result`, so it classifies as `verdict: "accepted", resultKeys: []`. When
the server's outbound-request id counter lands in the probe's numeric 1–5 id space, a
host request under the same key as a probed method makes a never-answered probe report
"accepted" in the committed per-method verdict line. The probe's own stale-evidence
posture treats desktop version drift as in-scope (header reads `builtinRevision` at
capture time), but the 3.18.2 server observed in `EXP-SUBSCRIBE-01.jsonl` uses prefixed
string ids (`"server-1"`…), which do not collide with the numeric keys — so today the
verdicts are honest; a server version that numbers host requests `1..N` would silently
corrupt them, and verdict lines are load-bearing evidence for the Phase-3 protocol docs.
**Fix:** One line — route method-bearing wires away from the reply store:

```js
if (wire.id != null && typeof wire.method !== 'string') state.replies[wire.id] = wire;
```

(server→host requests are still answered and recorded as `hostSent` lines downstream, so
nothing is lost from the capture). Also consider a unit-style red test for the harness.

## Info

### IN-01: WR-03 boundary rewrite is ASCII-only — a non-ASCII char directly after home falsifies a sibling path into `~`

**File:** `experiments/scrub-capture.mjs:124`
**Issue:** The negative lookahead `(?!/?[A-Za-z0-9._-])` treats non-ASCII name characters
as "not a name character", so a sibling directory distinguished by one rewrites anyway.
Verified: with `HOME=/nonascii-user-home`, input `/nonascii-user-homeé/x` scrubs to
`~é/x` — exactly the falsified tilde path the WR-03 fix (`scrub-capture.mjs:117-124`)
exists to prevent ("a path that merely shares the home prefix must pass through
verbatim"). The ASCII sibling cases (`home + 'x'`) are correct and tested; only
non-ASCII continuation escapes. Narrow (requires an accented username/path char
immediately after home), but it breaks the stated invariant in the direction of
falsified provenance.
**Fix:** Use Unicode classes with the `u` flag: `new RegExp(escapeRegExp(home) + '(?!/?[\\p{L}\\p{N}._-])', 'gu')`, and add a non-ASCII sibling test alongside the two WR-03 tests.

### IN-02: `--redact-path` with a literal `~` prefix silently redacts nothing

**File:** `experiments/scrub-capture.mjs:160-162`
**Issue:** `path.resolve(p)` does not expand `~`; a user naturally passing
`--redact-path '~/private'` gets `<cwd>/~/private`, which never matches, and the private
path flows into the capture unredacted (verified: `/tmp/tilde-private/f.md` survived a
run with `--redact-path '~/tilde-private'`). Silent no-op on a redaction flag is the
worst failure direction. The usage text (`scrub-capture.mjs:12-13`) doesn't mention the
restriction.
**Fix:** Expand a leading `~` against `os.homedir()` before `path.resolve`, or exit
non-zero with an explicit "tilde not expanded — pass an absolute path" error.

### IN-03: `govulncheck@latest` is a required-check dependency on an unpinned upstream

**File:** `.github/workflows/pr-sanity.yml:117`; `.github/workflows/security-scheduled.yml:38`
**Issue:** The branch-protection-required `govulncheck` context is produced by
`go run golang.org/x/vuln/cmd/govulncheck@latest` — an unversioned upstream binary. A
regression in a future `@latest` release (not the vuln DB, which is data, but the tool)
turns the required check red for every open PR until manual intervention — a PR-hang
path the comments (`pr-sanity.yml:108-115`) consider toolchain drift but not tool
breakage. The deliberate rationale (no double pinning, D-02; always-current scanner) is
real, so this is advisory.
**Fix:** Pin the govulncheck version in `mise.toml` `[tools]` for the PR gate
(reproducible required check), keep `@latest` only in the weekly
`security-scheduled` run as the drift detector that tells you when to bump the pin.

### IN-04: Secret gate fails open if `git grep` itself errors

**File:** `experiments/scan-secrets.sh:34`
**Issue:** `out=$(git grep ... 2>/dev/null || true)` collapses git grep's exit 2
(invocation error — e.g. a bad pattern after a future edit, an index problem) into "no
match": stderr is suppressed and `|| true` zeroes the status, so the scan reports green.
The three patterns are static strings today, so this needs an external trigger — but a
credential gate should fail closed, and the sibling gate `check-docs.sh` accumulates
failures instead.
**Fix:** Drop `|| true` and distinguish exits: `git grep -nE "$p" ... >/tmp/out 2>/dev/null; st=$?; [ $st -eq 1 ] && continue; [ $st -ne 0 ] && { echo "SECRET SCAN ERROR: git grep exit $st for $p"; exit 1; }` then inspect the hits.

### IN-05: `curl https://mise.run | sh` is the trust root for every job — including the one with `contents: write`

**File:** `.github/workflows/pr-sanity.yml:60,99`; `.github/workflows/release.yml:53`; `.github/workflows/security-scheduled.yml:28`
**Issue:** All four workflows bootstrap the toolchain by piping a remote script into
`sh` before any pin in `mise.toml` can apply; whatever `mise.run` serves that minute
controls the compiler, linter, and — in `release.yml` — goreleaser running with the only
write-scoped `GITHUB_TOKEN` in the set. The comments document the deliberate choice
(official channel, ephemeral runners, read-only rights) and `actions/checkout@v4` is a
version pin (not a rubric violation), so this is a hardening note, not a defect: the
release job is where the tradeoff is weakest.
**Fix:** Acceptable to keep as-is; if hardened, checksum-pin the installer (commit the
script + its sha256, verify before `sh`) or install mise from the mise
GitHub-release tarball pinned by digest. Apply the release.yml case first if ever.

### IN-06: Capture-header format contract is stricter downstream than at the boundary

**File:** `experiments/check-docs.sh:107` vs `experiments/scrub-capture.mjs:186-203,205`
**Issue:** The scrubber validates the header as "JSON object with a `capture` key" and
passes line 1 through byte-identical — accepting spaced forms like the phase's own test
fixture (`scrub-capture.test.mjs:31`: `"capture": "EXP-SCRUBTEST-01"`), and accepting
non-string values (`"capture":null`, `123`) at the boundary. But check-docs (f) greps the
committed capture for the exact compact form `"capture":"<id>"` (BRE, unescaped id), so
a hand-authored raw capture with header spacing — or a non-string capture value —
produces a capture that the scrubber accepts and the docs gate then fails (loudly, and
id metacharacters in a `[V-EXP]` tag would weaken the grep). All six committed captures
are compact, so the discipline currently holds; the two components just disagree on the
dialect.
**Fix:** In the scrubber, require `typeof header.capture === 'string' && header.capture` (fail loudly otherwise); in check-docs (f), relax the grep to `"capture":\s*"$id"` with `$id` regex-escaped (`$(printf '%s' "$id" | sed 's/[][.\\*^$]/\\&/g')`).

---

*No necessity-category findings: nothing in scope provably guards against a state that
cannot occur (the guards present sit on trust boundaries — env, filesystem, event
payloads — which are excluded from necessity review by rule).*

*Verified non-findings (checked, sound): required check names ↔ `protection-body.json`
contexts character-exact (live `GET` verified, app_id 15368); live protection state ==
body (enforce_admins, linear history, no force-push/delete, 0-approval PR requirement);
`pr-sanity` green on both trigger legs (`gh run list`); both gates green from repo root
and subdirectory; mise tasks execute at the project root (empirically probed with a
`pwd` task from a nested dir), so `tidy-diff` pathspecs are CWD-safe; zero untrusted
event-field interpolation in `run:` blocks; `contents: write` confined to `release.yml`;
scan patterns byte-identical to scrubber CRED_RX/JWT_RX as claimed; 18/18 scrubber tests
pass; token fixtures in the test file provably never appear adjacent in tracked source
(scan stays green).*

---

_Reviewed: 2026-11-19T14:53:06Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

## REVIEW COMPLETE

- Critical: 1
- Warning: 2
- Info: 6
- Total: 9
- Necessity: 0
