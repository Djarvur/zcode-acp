# Phase 1: Research & Protocol Documentation - Research

**Researched:** 2026-11-19
**Domain:** Wire-truth documentation methodology for the ZCode 3.18.2 app-server dialect (JSONL over stdio), the chat-UI client's ACP v1 dialect, desktop install discovery, and `workspaceKey` semantics — plus the experiment harness that produces V-EXP evidence on this machine.
**Confidence:** HIGH — every load-bearing experiment mechanism was actually executed against the shipped 3.18.2 binary during this research session (spawn line, JSONL exchange, EOF behavior), the 3.17.x source clone and the local Orca chunks were read directly, and the two external dependencies (reference repo, Orca repo accessibility) were checked against the GitHub API.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 — Ground-Truth Source Hierarchy:** The installed unpacked 3.18.2 desktop bundle (`resources/glm/zcode.cjs` and its chunk files) is the V-SRC ground truth for shipped-binary behavior. The public GitHub tree (3.17.x local clone) is corroborating evidence, always cited with its version; never present 3.17.x-source behavior as 3.18.2 truth. If the GitHub tree carries a tag/commit matching 3.18.2, cite it additionally; otherwise document the 3.17↔3.18.2 drift baseline explicitly in `docs/protocol/app-server-dialect.md`. Load-bearing claims are additionally V-EXP against the running binary.

**D-02 — Experiment Artifacts:** Commit small harness scripts under `experiments/` and scrubbed JSONL captures under `experiments/captures/` (strip JWTs, `enc:v2:` blobs, PAT shapes, account identifiers). Raw unscrubbed captures stay outside the repo. Scrubbed transcripts are the seed for Phase 3 golden fixtures — this is why they are committed rather than summarized. Reversibility: costly — once captures are committed, scrubbing mistakes require history rewrite; hence scrub-first discipline and a CI secret-scan shape noted for Phase 2 (JWT / `enc:v2:` / PAT regexes).

**D-03 — Docs Layout:** Split per success criterion: `docs/protocol/app-server-dialect.md` (DISC-01), `docs/protocol/acp-client-dialect.md` (DISC-02), `docs/protocol/install-discovery.md` (DISC-03), `docs/protocol/workspace-key.md` (DISC-04), `docs/research/reference-adapters.md` (DISC-05), `docs/protocol/README.md` (index + shared provenance legend).

**D-04 — Provenance Format:** Inline tags per claim — `[V-SRC <path>#<anchor>]`, `[V-EXP <experiment-id>]`, `[V-RPT <source>]` — plus a Sources & provenance section at the end of each doc; the legend lives once in `docs/protocol/README.md`; `[V-EXP <id>]` resolves to an `experiments/captures/` filename.

**D-05 — Experiment Policy:** Targeted experiments only, for the open items: (1) `session/subscribe` / event-stream surface probe; (2) turn-completion terminal signals — one tiny real turn only if local account credentials exist, otherwise source-only + record the limitation; (3) `workspaceKey` derivation — source read + one experiment; (4) install discovery — enumerate layouts; (5) `--session-dir` session-file format + omp `modelDiscovery` kind `session` behavior. Quota discipline: minimal turns, manual tier only, never from CI. If credentials are unavailable, degrade to source-only and mark affected claims V-SRC-only rather than blocking the phase.

**D-06 — Orca Scope:** No live chat-UI integration in Phase 1. The omp contract table (V-SRC(parallel)) is spot-checked by re-reading the unpacked Orca AppImage chunks in this phase's own session — re-verified claims upgrade to V-SRC with in-repo citation; the two open items (`--session-dir` format, `modelDiscovery` kind `session`) are answered statically here; a live card run belongs to Phase 7 manual E2E.

### Claude's Discretion
- Exact experiment script implementations and capture formats (within D-02 rules)
- Doc prose, depth, and section structure (within D-03/D-04 rules)
- Fallback behavior when local credentials or a deb layout are unavailable (record limitations, do not block)
- Choice of which V-SRC(parallel) claims to re-verify first (prioritize load-bearing: launch line, `--version` shape, `agentCmdOverrides` semantics)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. (v2 items EXT-01 slash commands, SESSH-01 session history, EXT-04 additional persona cards remain tracked in REQUIREMENTS.md v2 section.)

Additional project constraints (from `.claude/CLAUDE.md` / PROJECT.md, binding on this phase):
- Docs language: English (all committed docs).
- Research policy: sources before experiments for ZCode and chat-UI-client facts.
- Hidden-goal rule: neutral ACP-server phrasing in all committed docs; no real machine paths of private research locations in any committed file ("real locations are supplied in agent prompts").
- No patching of the ZCode bundle; wrapper is read-only toward the desktop installation.
- CI must never call api.z.ai or spawn the real desktop (Phase 2 concern, but experiment scripts must already be shaped so they cannot be wired into CI accidentally).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DISC-01 | App-server 3.18.2 event surface documented from source (incl. `session/subscribe`/event-stream availability and turn-completion terminal signals), with V-SRC / V-EXP provenance per claim | Full 3.18.2 method map is extractable from the shipped bundle by grep (verified this session, §Code Examples); `session/subscribe` and `v4/conversation/subscribe` both present as strings; `session/delete` verifiably absent; 7 `interaction/*` server→host methods enumerated (incl. `requestUserInput` — new vs prior research docs); experiment harness validated live (E1/E2 designs in §Experiment Catalog) |
| DISC-02 | Client ACP dialect documented: preset-agent identity (resolved: omp), version floor, env quirks, load-bearing method set at card startup/per turn, full ACP method range | All five load-bearing omp contract strings (`17.0.5`, `agentCmdOverrides`, `acp-command-sessions`, `launchesWithModel`, `PI_NO_PTY`) confirmed present in the two named local chunks of the current Orca build — D-06 spot-check is fully executable locally; the GitHub fallback is dead (404, see Pitfall 4); current build carries ~22 preset ids (vs 8 documented) — card-registry re-enumeration is an explicit task |
| DISC-03 | Desktop discovery matrix (AppImage / deb / unpacked layouts, auto-update effects on builtin revision) | All three layouts have local evidence (unpacked squashfs-root probe-verified; AppImage file + symlink; deb 3.17.2-9661 rc-state in dpkg db + desktop-entry remnant); official docs confirm AppImage/deb/rpm downloads for x64+ARM64; deb layout extraction designed via `dpkg-deb -c` on the downloaded artifact (no install) |
| DISC-04 | `workspaceKey` derivation documented (source + experiment) | Derivation convention found and read verbatim in the 3.17.x source this session: `workspaceIdentity?.trim() \|\| workspacePath`, env override `ZCODE_WORKSPACE_IDENTITY`, schema `nonEmptyString` — the prior LOW-confidence item is source-resolved; one confirming experiment remains (E3) |
| DISC-05 | Reference implementation studied and compared into `docs/research/` | `william0wang/zcode-acp` verified accessible via GitHub API this session: public, Apache-2.0, 91 stars, pushed 2026-11-19 (active). Key-file list and adopt/avoid leads in REFERENCE-ADAPTERS.md carry over as the study checklist |
</phase_requirements>

## Summary

Phase 1 is a documentation phase whose raw material already exists on this machine, and this research validated every mechanism the phase depends on by *running* it, not by assuming it. The experiment harness design is proven: a ~25-line Node script spawns `ELECTRON_RUN_AS_NODE=1 <squashfs-root>/zcode <squashfs-root>/resources/glm/zcode.cjs app-server --stdio`, keeps stdin open, exchanges JSONL, and captures output — executed live during this research, it produced the first new wire fact of the phase (the app-server emits `startup/storageState` notifications on boot) and confirmed `provider/refreshRegistry → {"refreshed":true}`, clean exit 0 on stdin EOF, and zero orphaned processes. A critical harness lesson was learned the hard way and is now a documented pitfall: writing one request and closing stdin makes the app-server exit (cleanly, code 0) *before* answering — the drain window beats registry processing. Every E-experiment in this phase must hold stdin open.

The claim inventory for the five documents is mostly pre-assembled: `.planning/research/*` + the private field notes carry the V-EXP facts (dialect transport, snapshot semantics, PAT chain, Orca card contracts), and this research located their source-level anchors in the 3.18.2 bundle (full method map, interaction-method list, error-code literals, `desktop-continuous`/`deliveryKind` presence, `session/delete` absence). The biggest planning-relevant discovery beyond the prior docs: the `stablyai/orca` GitHub repository is **not** publicly accessible (404 on both anonymous fetch and authenticated API, 2026-11-19) — the CONTEXT.md fallback "re-verify from github.com/stablyai/orca source" is dead, but the local unpacked Orca AppImage chunks contain all load-bearing omp contract strings, so D-06's spot-check is fully executable locally. The workspaceKey open item collapsed from LOW-confidence to source-resolved: the convention `workspaceIdentity?.trim() || workspacePath` (env `ZCODE_WORKSPACE_IDENTITY`) was read verbatim from `zcode-protocol-entrypoint.ts` this session; only the confirming experiment remains.

**Primary recommendation:** Structure the phase as five claim-ledger documents written claim-first (every claim line born with its provenance tag), executed alongside a small experiment catalog E1–E5 whose harness pattern is already validated; run the Orca chunk re-verification against a freshly extracted AppImage copy (not volatile /tmp mounts), and gate every commit through a scrub-first secret scan so D-02's irreversibility warning is respected from the first capture.

## Architectural Responsibility Map

Phase 1 produces documents, experiments, and captures — the "tiers" are the artifact families later phases consume.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| App-server dialect documentation (DISC-01) | `docs/protocol/app-server-dialect.md` | `experiments/captures/*` (V-EXP evidence) | Claims live in the doc; evidence lives in named captures the doc's V-EXP ids resolve to (D-04) |
| Client ACP dialect documentation (DISC-02) | `docs/protocol/acp-client-dialect.md` | Orca chunk reads (source, not committed) | Chunk filenames + line anchors are cited in-doc; chunk contents are never committed |
| Install discovery matrix (DISC-03) | `docs/protocol/install-discovery.md` | `experiments/` (layout probes) | Matrix rows each carry V-SRC/V-EXP tags; probes are small committed scripts |
| workspaceKey semantics (DISC-04) | `docs/protocol/workspace-key.md` | one capture (E3) | Source citation + recorded experiment, per success criterion 4 |
| Reference-implementation comparison (DISC-05) | `docs/research/reference-adapters.md` | cloned repo (outside repo, read-only) | Adopt/avoid verdicts + Apache-2.0 attribution obligations documented; upstream repo is external material |
| Provenance legend + index | `docs/protocol/README.md` | — | Single legend location per D-04 |
| Capture hygiene (D-02) | `experiments/` scrub script + captures | Phase 2 CI secret-scan (noted, not built) | Scrub happens before any file enters `experiments/captures/`; raw captures stay outside the repo |

## Standard Stack

Phase 1 installs nothing; it uses preinstalled local tools. Verified versions on this machine this session:

### Core
| Tool | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Node.js | v26.13.1 | JSONL experiment harness scripts (`experiments/*.mjs`); spawn + stdin control + line capture | Child-process control (`spawn`, stdin `.end()` timing) is exactly what the EOF pitfall requires; JSONL handling is native; FINDINGS predecessor scripts are .cjs/.mjs already [VERIFIED: `node --version` this session] |
| bash + coreutils | system | Orchestration: grep/jq probes, capture scrubbing pipelines, `setsid`/process-group cleanup | All probes this session ran as one-liners; no framework needed [VERIFIED: session probes] |
| jq | 1.7 | Selective inspection of large JSON captures (extract `.method` counts, revision fields) without dumping secrets | Prints only the fields you ask for — the safe way to inspect captures [VERIFIED: `jq --version`] |
| GNU grep | system | Bundle anchor extraction (`grep -o` with context windows over the 15.7 MB minified `zcode.cjs`); provenance tag conformance checks | Method map extraction proven this session [VERIFIED: session probes] |
| Electron binary from the unpacked bundle | 3.18.2 build | The app-server runtime: `ELECTRON_RUN_AS_NODE=1 <root>/zcode <root>/resources/glm/zcode.cjs app-server --stdio` | The verified spawn mechanism; bundle confirmed byte-identical to its `.orig` backup (no patching) [VERIFIED: `cmp` this session] |
| `secret-tool` | /usr/bin | Keyring lookup probes for the credentials experiment (only if needed; Phase 1 does not need decrypted material) | Present with a live D-Bus session and unlocked login keyring [VERIFIED: `command -v`, DBUS addr, keyrings listed] |
| `gh` CLI | authenticated | Reference-repo accessibility checks (`gh api repos/...`) | Credible API truth where plain web fetch 404s are ambiguous [VERIFIED: worked this session] |

### Supporting
| Tool | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `dpkg-deb` | system | Inspect deb layout from the downloaded artifact: `dpkg-deb -c` / `-I` — no installation | DISC-03 deb row [VERIFIED: tool exists; artifact download pending] |
| `python3` | 3.15.2 | Fallback for scrub scripts if preferred over Node | Either is fine; pick one and keep one scrub implementation |
| `jq`-only scrub alternative | — | Not recommended: regex scrubbing (JWT/`enc:v2:` shapes) is easier in Node/Python | — |
| `--appimage-extract` | AppImage tool | Produce a fresh, version-true Orca extract for chunk re-verification | Avoids reliance on volatile /tmp mounts (Pitfall 3) [VERIFIED: extract dirs exist from prior use] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Node harness scripts | bash + jq pipes | Bash cannot hold stdin open while reading stdout with timeouts cleanly; Node's `spawn` is the proven pattern — keep Node |
| Fresh AppImage extract | Live /tmp mount points | Mounts vanish when the app exits and three different builds are currently mounted; fresh extract is reproducible |
| `strings` on the bundle | `grep -o` with windows | grep gives surrounding context in one step; `strings` loses offsets — use grep |

**Installation:** Nothing to install. All tools preinstalled.

**Package Legitimacy Audit:** Not applicable — Phase 1 installs zero external packages (no npm/pypi/crates dependencies). The only external code executed is the shipped ZCode 3.18.2 bundle (V-SRC ground truth per D-01) and read-only study of the reference repo. No `checkpoint:human-verify` install tasks needed.

## Architecture Patterns

### System Architecture Diagram

```
 PRIVATE (outside repo)                       IN-REPO ARTIFACTS
 ────────────────────────────                 ──────────────────────────────────────
 3.18.2 bundle (unpacked) ──grep/read──┐
 3.17.x source clone ──────read────────┤      docs/protocol/README.md  (legend)
 Orca chunk extract ───────grep/read───┤            │
 private field notes ──────facts───────┤            ▼
                                       │    docs/protocol/*.md + docs/research/*.md
 william0wang/zcode-acp ───clone/read──┤      (claim-ledger docs: every claim
 (public, Apache-2.0)                  │       carries V-SRC/V-EXP/V-RPT tag)
                                       │            ▲
 Official download site ───deb/rpm─────┤            │ resolve
                                       │            │ [V-EXP <id>]
 app-server child ──spawn/JSONL────┐   │            │
   (ELECTRON_RUN_AS_NODE=1 …)      │   │    experiments/captures/<id>.jsonl
                                   ▼   │    (scrubbed only — scrub-first)
                          harness script┘         ▲
                          experiments/*.mjs ──────┘ (scripts committed,
                                                    raw captures NOT)
                                                        │
                                                        ▼
                                          Phase 3 golden fixtures (seed)
```

Reading path: every claim in a doc names either a source anchor (V-SRC) or a capture id (V-EXP) or a reported source (V-RPT); captures are produced only by committed harness scripts; raw captures never enter the repo.

### Recommended Project Structure

```
docs/
├── protocol/
│   ├── README.md                  # index + provenance legend (D-04)
│   ├── app-server-dialect.md      # DISC-01
│   ├── acp-client-dialect.md      # DISC-02
│   ├── install-discovery.md       # DISC-03
│   └── workspace-key.md           # DISC-04
└── research/
    └── reference-adapters.md      # DISC-05
experiments/
├── README.md                      # how to run each experiment; scrub-first rules
├── app-server-probe.mjs           # reusable spawn/JSONL harness (validated pattern)
├── scrub-capture.mjs              # canonical scrubber (single implementation)
├── check-docs.sh                  # provenance-tag + structure conformance checks
├── scan-secrets.sh                # secret-shape gate over tracked files
└── captures/                      # scrubbed JSONL only; names = experiment ids
```

### Pattern 1: Claim-ledger documents

**What:** Each deliverable doc is written claim-first: one bullet/table-row per wire fact, each ending in its provenance tag, with a Sources & provenance section mapping every id to its origin (D-04). The doc leads with a header block: "Verified against: shipped 3.18.2 binary (bundle grep + experiments) / 3.17.x source clone (corroborating)" — the drift baseline of Pitfall 5.
**When to use:** All five documents.
**Structure per doc (blueprint the planner can turn into task actions):**

- `app-server-dialect.md` (DISC-01): 1. Transport facts (JSONL envelope grammar, no `jsonrpc` key, error shape, string-or-int ids) · 2. Full method map (extracted from bundle — see Code Examples) · 3. Server→host surface: `state.updated`, `startup/storageState` (new), 7 `interaction/*` methods · 4. `session/subscribe`/event-stream availability (E1 verdict) · 5. Turn-completion terminal signals (E2 verdict; quiet-window fallback documented) · 6. Error-code table (−32010 concurrency, −32031 headers, −32600 unrecognized_keys, −32601) · 7. Session lifecycle methods (`create`/`resume`/`close`/`stop`/`cancelBackgroundTask`; `session/delete` absent — verifiable absence) · 8. Snapshot semantics (`updateAccountConfig` ack≠applied, zod strict schema, revision string) · 9. **3.17↔3.18.2 drift baseline** (first-class section: `requestCredentialCipher` absent in 3.17.x source but in bundle; `startup/storageState` new; method-map diff)
- `acp-client-dialect.md` (DISC-02): 1. Preset-agent identity: omp, why chosen, alternatives ruled out (neutral wording, no hidden-goal phrasing) · 2. Card registry of the current build (re-enumerated from chunk — the build has ~22 preset ids, see Pitfall 6) · 3. Launch contract: command line Orca runs, env (`PI_NO_PTY=1`, agent home), `agentCmdOverrides` wiring semantics · 4. Version floor (17.0.5) and reported version policy (≥18.1.9) · 5. Load-bearing method set at card startup / per turn (re-verified from chunk, cited by filename+anchor) · 6. Full ACP method range the client sends · 7. Hb string contracts (exact error strings) · 8. Client capabilities (fs/terminal false, elicitation form, compaction) · 9. Open items resolved statically: `--session-dir` format (E5), `modelDiscovery` kind `session`
- `install-discovery.md` (DISC-03): 1. Layout table: unpacked AppImage (squashfs-root), AppImage file + symlink, deb (version scheme `<ver>-<build>` from dpkg db; layout from `dpkg-deb -c` on artifact), rpm (same technique), running mount (volatile — mark "do not rely") · 2. Per-layout: how to locate `<root>/resources/glm/zcode.cjs` · 3. Desktop-entry / URL-handler registration + the documented deb-vs-AppImage clash warning · 4. Auto-update: `app-update.yml` facts, runtime provider tree accretes per desktop version (`~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/`), revision value observed (44 for 3.18.2) · 5. Recommended discovery order for the Phase 3 supervisor (env/flag → desktop entry → AppImage paths → package paths; never /tmp mounts)
- `workspace-key.md` (DISC-04): 1. Wire schema fact (`workspaceKey: nonEmptyString` required in one shape, optional in another — cite both lines) · 2. Source convention, quoted verbatim: `workspaceIdentity?.trim() || workspacePath`, env `ZCODE_WORKSPACE_IDENTITY`; note the source comment that agent processes have no identity source so the key degrades to the path · 3. E3 experiment record (path-as-key accepted) · 4. Host-freedom note: in host-managed mode the wrapper *is* the host and chooses the key; document the convention as the compatible choice
- `reference-adapters.md` (DISC-05): 1. What the TypeScript reference covers (elicitation, slash commands, resume/fork, attachments, credential chain) — verified against the live repo (public, Apache-2.0, active) · 2. Adopt list (key files as golden references for Phase 6) · 3. Avoid list · 4. Attribution obligations (Apache-2.0 NOTICE requirement for ported code) · 5. Weaker adapters table (Rust/Python) · 6. Start-Plan headless limitation (their TROUBLESHOOTING → our DOCS-03 note)

### Pattern 2: Experiment harness (validated this session)

**What:** One reusable Node script spawns the app-server, keeps stdin open, writes requests, timestamps and collects stdout lines, then closes stdin for clean shutdown and hard-kills the process group as a backstop.
**When to use:** E1, E2, E3 — every app-server experiment.
**Why the shape is load-bearing:** the app-server exits cleanly on stdin EOF (verified: exit 0), and a request issued just before EOF gets no answer. See Code Examples for the validated script.

### Pattern 3: Scrub-first capture pipeline (D-02)

**What:** Raw captures land outside the repo; the scrubber (`experiments/scrub-capture.mjs`) is the *only* path into `experiments/captures/`; the scan script (`experiments/scan-secrets.sh`) gates every commit.
**Canonical scrub rules (define once, reuse everywhere):**
- JWTs: any `eyJ<base64url>.<base64url>.` prefix shape
- Credential blobs: `enc:v2:os:` and legacy `enc:v1:` values
- PAT/api-key material: long hex/alphanum token shapes adjacent to `apiKey`, `token`, `access_token`, `api-key` keys; 32-hex id shapes
- Account identifiers: UUID v4 shapes and numeric account ids; user_info payloads
- `secret-tool` output (never echo it; never capture it)
- Machine-specific paths of private research locations (neutral phrasing rule)
**Capture file header (first line, added by the harness, scrubber-safe):** `{"capture":"<experiment-id>","date":"...","desktopVersion":"3.18.2","builtinRevision":"<read at capture time>","host":"<os release>"}` — records the drift context so Phase 3 fixtures can be version-tagged (Pitfall 5 prevention).

### Pattern 4: Bundle anchor extraction (V-SRC against the shipped binary)

**What:** The minified bundle is greppable for wire surface: method-name maps survive minification because they are string literals (`sessionSubscribe:"session/subscribe"`). Extract the full map by grepping a wide context window around a known key (Code Examples), then prettify with jq after wrapping in `{}`.
**When to use:** DISC-01 method inventory; drift-baseline diff vs the 3.17.x `packages/shared/src/zcode-protocol/index.ts` method table.
**Caution:** `grep -c` counts *lines* (useless on a minified bundle) — always `grep -o pattern | wc -l` for occurrence counts. Presence/absence of a method string (0 hits) is meaningful evidence (e.g., `session/delete` absence).

### Anti-Patterns to Avoid
- **Reading the packed asar for client research:** the chat-UI client lives in `resources/app.asar.unpacked/out/main/chunks/`; the packed `index.js` contains no ACP strings (prior FINDINGS trap, §4).
- **Re-running private-notes facts as "new experiments":** D-05 says reuse existing V-EXP evidence where load-bearing but re-record it with reproducible in-repo experiment ids; do not redo the whole OAuth chain.
- **Minting new API keys during Phase 1:** the PAT chain's `api_keys` POST creates keys on the user's org. Phase 1's turn experiment needs at most an access-token refresh against an existing key — never a new key.
- **Writing unscrubbed captures anywhere inside the repo tree** (even temporarily): scrub-first means the raw file is created outside the repo from birth.
- **Treating the three currently-mounted Orca builds as interchangeable:** chunk hash suffixes differ per build; cite the extract you actually read.

## Experiment Catalog (E1–E5)

Exact runnable designs per D-05. Preconditions verified this session: OAuth + api-key credential keys present in `~/.zcode/v2/credentials.json` (keys enumerated, values never printed); keyring service, D-Bus session, login keyring all present; bundle pristine. Raw outputs go to a scratch dir outside the repo; only scrubbed files enter `experiments/captures/`.

**Common harness skeleton** (`experiments/app-server-probe.mjs`, pattern validated live this session):
1. Resolve paths from CLI args (no private paths hardcoded in committed scripts — pass the bundle root and builtin file as arguments at run time).
2. `spawn(electronBin, [cjs, 'app-server', '--stdio'], { env: {ELECTRON_RUN_AS_NODE:'1', ZCODE_BUILTIN_PROVIDER_CONFIG_FILE: builtinFile} })`, `stdio: ['pipe','pipe','pipe']`.
3. Write requests with delays; **keep stdin open** until the collection window ends (EOF pitfall).
4. Collect stdout lines with timestamps into a raw file outside the repo; mirror stderr capped.
5. End: `stdin.end()` → wait exit (observed: clean 0) → backstop `process.kill(-pid, 'SIGKILL')` on the process group.
6. Run scrubber → `experiments/captures/<experiment-id>.jsonl`.

**E1 — `session/subscribe` / event-stream surface probe** (DISC-01)
- Goal: does the shipped binary accept `session/subscribe` (and/or `v4/conversation/subscribe`), with which params (`deliveryKind:"desktop-continuous"`, `includeSnapshot` per 3.17.x source), and what event envelopes result?
- Steps: spawn → `provider/refreshRegistry` (warm-up; verified answer shape) → `session/create {workspace:{workspacePath:<scratch dir>, workspaceKey:<same path>}, mode:…, persistence:…}` (read the exact create schema from the 3.17.x `index.ts` before running — required/optional split partially verified this session) → `session/subscribe {sessionId, deliveryKind:"desktop-continuous"}` → record reply (result vs error code) → if accepted, send `session/send`-free idle window and record any event envelopes; repeat once with `v4/conversation/subscribe`; also probe plain `session/events` (present in the method map — a poll-style alternative the earlier research never named).
- Verdict fields for the doc: accepted/refused; envelope method names; sequence-number presence; `deliveryKind` enum values observed in bundle context.
- Fallback: error reply is itself the V-EXP fact ("refused with code X on 3.18.2") — a refusal still resolves the open item.
- Capture id: `EXP-SUBSCRIBE-01`.

**E2 — Turn-completion terminal signals** (DISC-01; the load-bearing experiment for Phase 4)
- Goal: enumerate exactly which server→host payloads end a turn on 3.18.2.
- Preconditions: local credentials exist (verified: OAuth + api-key keys present). Quota discipline: ONE tiny turn, manual tier, never CI. Use the default workspace model path (no model in create → personal config default; proven end-to-end by the predecessor shim, `NOMODEL_OK`). Do NOT mint API keys (reuse existing key material; at most refresh an access token).
- Steps: spawn → create session (subscribe first if E1 accepted it; otherwise poll `session/messages`) → `session/send {sessionId, content:"Reply with exactly: OK"}` → capture every line until quiet-window (2.5 s without new parts — the field-verified fallback heuristic) → also record `state.updated` patch shapes seen, any `turn.completed`-family payloads, and whether `interaction/requestPermission` or `requestUserInput` fire for a no-tool turn (expected: no).
- Doc output: terminal-signal table (signal → payload shape → reliability), quiet-window fallback documented as the proven secondary.
- Capture id: `EXP-TURN-01`. If credentials fail at run time: degrade per D-05 — source-only enumeration from bundle string neighborhoods (`turn.completed` ×22, `turn.failed` ×27 occurrences in the bundle) + reuse of private-notes V-EXP facts re-recorded as V-RPT-with-limitation, and record the limitation.
- Scrub emphasis: this capture contains model output and provider ids — safe; but strip any token-shaped strings and account identifiers defensively (scrubber runs regardless).

**E3 — `workspaceKey` derivation experiment** (DISC-04)
- Goal: confirm the source-read convention on the wire.
- Steps: spawn → `session/create` with `workspace:{workspacePath:<scratch dir>, workspaceKey:<same absolute path>}` → expect `result.session.sessionId` (acceptance) → optional second probe with `workspaceKey:"arbitrary-host-string"` to demonstrate host-opaque semantics → record both replies. Cheap: no turn, no quota.
- Capture id: `EXP-WORKSPACEKEY-01`.

**E4 — Install discovery enumeration** (DISC-03)
- Goal: fill the layout matrix with V-EXP rows; each row = a locate command + its verified output.
- Steps (all read-only, no installs):
  1. Unpacked AppImage: verify `<root>/zcode` + `<root>/resources/glm/zcode.cjs` exist and `zcode.cjs` matches its `.orig` sibling byte-for-byte (honesty-posture check; verified this session, re-run as experiment).
  2. AppImage file: locate `~/Applications/ZCode-*.AppImage` + symlink; extract to scratch (`--appimage-extract`) only if needed.
  3. deb: query dpkg db (rc-state entry observed: package `zcode`, version `3.17.2-9661`, empty file list) → download current .deb from the official install page into scratch → `dpkg-deb -c` + `-I` to record layout and version scheme without installing.
  4. rpm: same download-and-inspect (`rpm -qlp`) — optional if the deb row generalizes.
  5. Desktop entry: parse the remnant entry (Exec/AppImage path, `x-scheme-handler/zcode` registration) — evidence for the documented clash between deb and AppImage installs.
  6. Runtime/revision: enumerate `~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/` (all versions present), read `revision` from the active `zcode-builtin.json` (value 44 verified this session for 3.18.2), and record the `zcode-builtin:<rev>:<sha256(resolved path)>` formula from source (FINDINGS §3.1) as V-SRC.
  7. Auto-update: record `app-update.yml` contents from the bundle (verified: generic provider, localhost:8081 URL, updater cache dir name) and note what a desktop update does to the runtime tree (new `<ver>` dir → new revision) — mark the live-update behavior V-RPT if not directly observed.
- Capture id: `EXP-DISCOVERY-01` (a findings table + command transcript, scrubbed of paths per the neutral-phrasing rule — record *patterns* like `~/Applications/ZCode-*.AppImage`, not user-specific absolute paths).

**E5 — `--session-dir` format + omp `modelDiscovery` kind `session`** (DISC-02 static answers)
- Goal: answer the two open items without a live card run (D-06).
- Steps:
  1. Read the local pi/omp agent home `sessions/` directory (present this session): structure is `sessions/<path-slug-dir>/<ISO-timestamp>_<uuid>.jsonl` — record the naming scheme and (from the reference repo's omp sibling docs or the chunk's scanner logic) the JSONL record shape the scanner expects.
  2. Read `session-scanner-omp-subagent-transcripts-*.js` and the `transcriptImport` section of `structured-agent-session-host-assembly-*.js` in the current build's chunks: extract what file pattern the scanner matches, what fields `transcriptImport` (fork:'tip') reads, and the `acp-command-sessions` directory expectations (the string is present in the assembly chunk — verified).
  3. `modelDiscovery` kind `session`: extract the omp card block from the registry chunk; document that model listing happens in-session via `/model` mechanics (`listingNamesConfiguredModel:false`), citing chunk filename + anchor.
  4. Verdict for the doc: can a Go agent write omp-compatible history files? (Yes/No/Partial + exact format spec.) Note: our v1 does NOT advertise `loadSession`, so writing compatible files is a Phase 7+ nicety, not a v1 gate — say so explicitly to keep scope honest.
- Capture id: `EXP-SESSIONDIR-01` (quoted chunk excerpts + session-dir listing, scrubbed).

**Orca chunk re-verification pass** (DISC-02, D-06) — not a numbered experiment; a read-and-cite pass over a *fresh* extract of the installed Orca.AppImage:
- Priority order (per CONTEXT discretion note): launch command line for omp · `--version` probe mechanics · `agentCmdOverrides` semantics (prefix args, shell-quoting, `~/` expansion, metachar rejection) · version floor string · then the remaining card contracts (Hb strings, client capabilities, elicitation, compaction, retryBusyPrompts).
- Each re-verified claim: upgrade V-SRC(parallel) → V-SRC with `[V-SRC <chunk-filename>#<anchor or quoted snippet>]`.
- Deliverable delta: re-enumerate the card registry — the current build's preset id list (~22 ids, incl. `zcode`, `pi`, `hermes`, `devin`, …) is larger than the 8 documented from the parallel session's build; identify which are ACP vs line-dialect cards and note the delta in the doc (see Open Questions 1).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Spawn/stdin/capture orchestration | bash coprocess gymnastics | Validated Node harness pattern (Code Examples) | stdin-hold timing + process-group backstop is fiddly in bash; Node proven live |
| Secret scrubbing | Ad-hoc per-capture sed pipelines | One canonical scrubber script with the shared rule set | D-02 irreversibility: one implementation = auditable rules; per-capture hacks = guaranteed miss |
| Revision-string computation | Recomputing from memory | Formula from source + FINDINGS, `zcode-builtin:<rev>:<sha256(realpath)>` | Off-by-one path resolution breaks acceptance silently (Pitfall 1 class) |
| Deb layout discovery | Installing the deb on this machine | `dpkg-deb -c` / `-I` on the downloaded artifact | Installing mutates the machine and risks the documented URL-handler clash |
| Orca card contracts | Deriving from memory of REFERENCE-ADAPTERS | Fresh-extract chunk reads with quoted anchors | Build drift is real (three builds mounted; preset list grew) |
| Bundle searching | Reading the 15.7 MB bundle by eye | grep -o context windows + jq prettify | Method map extraction proven in minutes (Code Examples) |
| Repo accessibility checks | Trusting search-engine summaries | `gh api repos/<owner>/<repo>` | A search summary claimed the Orca repo was public/MIT; the API says 404 — API wins |

**Key insight:** every "don't hand-roll" here protects either honesty (provenance), safety (scrubbing), or reproducibility (drift context) — the three properties this phase exists to establish for all later phases.

## Runtime State Inventory

Not applicable — Phase 1 is documentation-only (no rename/refactor/migration). Omitted per template.

## Common Pitfalls

### Pitfall 1: stdin EOF kills the app-server before it answers (VERIFIED this session)
**What goes wrong:** Write one JSONL request, close stdin → app-server exits cleanly (code 0) with **zero output**; the request is never answered.
**Why:** On client EOF the app-server drains only ~100 ms then exits; registry processing outlives the drain window. (First probe attempt this session: exit 0, empty stdout.)
**How to avoid:** Hold stdin open until the collection window ends; close stdin only to trigger clean shutdown; keep the group-kill backstop.
**Warning signs:** Exit 0 + empty capture.

### Pitfall 2: Bundle-provenance ambiguity — FINDINGS says "patched", disk says pristine
**What goes wrong:** The private FINDINGS notes describe the experiment bundle as "patched locally, backup beside it"; this session `cmp` shows `zcode.cjs` and `zcode.cjs.orig` byte-identical (patch evidently reverted or note stale). Meanwhile the *running* desktop uses a different (read-only) unpack. Documenting claims against the wrong copy undermines the no-patching honesty claim.
**How to avoid:** Every bundle-based claim records *which copy* was probed; re-run the `cmp` as experiment E4.1 and record the result in the dialect doc's drift/provenance header.
**Warning signs:** docs claiming bundle hashes that don't reproduce.

### Pitfall 3: Orca chunk reads against volatile /tmp mounts
**What goes wrong:** Three different Orca builds are currently mounted under /tmp with different chunk-hash suffixes; mounts disappear when the app exits, and citations die with them.
**How to avoid:** Extract a fresh copy of the installed AppImage (`--appimage-extract` into scratch) at experiment time; cite chunk filenames from that extract; note the extraction recipe in `experiments/README.md`. (A stable prior extract also exists under /tmp — usable, but a fresh extract is the reproducible path.)

### Pitfall 4: Trusting web summaries for repo accessibility
**What goes wrong:** A web-search summary described `stablyai/orca` as public/MIT; both anonymous fetch and the authenticated GitHub API return 404 (2026-11-19). Planning D-06 re-verification against that repo would dead-end mid-phase.
**How to avoid:** The Orca re-verification pass targets local chunks only; if a GitHub source is ever wanted, re-check with `gh api` at execution time. Conversely, the reference repo `william0wang/zcode-acp` IS verified accessible (public, Apache-2.0, pushed today) — plan DISC-05 against it confidently.

### Pitfall 5: Shared `~/.zcode` state with the live desktop
**What goes wrong:** Experiments and the running desktop app share `~/.zcode` (credentials, runtime provider tree, logs). Concurrent app-server instances could interleave registry/runtime writes; desktop auto-update can swap the builtin revision mid-experiment.
**How to avoid:** Record the builtin revision in each capture header at capture time; keep experiments short; never run experiments that mutate account state (no key minting); treat registry state as read-and-record, not assert-and-hold.
**Warning signs:** revision differs between capture header and doc claim.

### Pitfall 6: Orca build drift vs the parallel-session contract table
**What goes wrong:** REFERENCE-ADAPTERS' 8-card list came from a different build; the current build's preset id list is ~22 entries (incl. `zcode`, `pi`, `hermes`, `devin`, `droid`, …) spotted inside the registry chunk. Assuming the 8-card list is current could mislead the compatibility matrix.
**How to avoid:** E5's chunk pass re-enumerates the ACP subset of the registry from the current build; the doc states the build it describes. The omp identity decision (locked) is unaffected — its contract strings are all present.

### Pitfall 7: Secrets in captures (D-02 irreversibility)
**What goes wrong:** `credentials.json` keys alone contain account-UUID-bearing identifiers (enumerated this session, values never printed); turn captures contain model output + provider ids; `state.updated` can carry user paths. One unscrubbed commit = history rewrite.
**How to avoid:** Scrub-first pipeline (Pattern 3); secret-scan gate on every commit; raw captures born outside the repo; never `cat` credentials or echo `secret-tool` output.
**Warning signs:** scan-secrets.sh hits; any `eyJ` / `enc:v2:` / UUID in `git diff`.

### Pitfall 8: `grep -c` lies on a minified bundle
**What goes wrong:** `grep -c -o pattern file` counts matching *lines* — on a single-line minified bundle every present string "counts" 1 and absence/presence is the only signal.
**How to avoid:** `grep -o pattern file | wc -l` for occurrence counts; `grep -o '.\{N\}pattern.\{M\}'` for context windows.

### Pitfall 9: Quota and key-creation discipline
**What goes wrong:** E2 is the phase's only quota-spending step. An careless harness could also mint API keys (the PAT chain's `api_keys` POST creates keys on the org).
**How to avoid:** One tiny turn max; no `api_keys` POST in any Phase 1 script; reuse existing key material only; record spend honesty in the experiment README ("turn count: 1").

### Pitfall 10: The desktop's localhost listener is not the app-server
**What goes wrong:** A `zcode-host-local-1` listener on a 127.0.0.1 port answers 404/400 to HTTP probes (FINDINGS §2) — wasting time treating it as a protocol surface.
**How to avoid:** The only channel is the stdio spawn line; ignore localhost listeners in discovery experiments (record the fact in the discovery doc as a known decoy).

## Code Examples

### Validated spawn + JSONL probe harness (executed live this session)
```javascript
// experiments/app-server-probe.mjs — pattern verified 2026-11-19 on 3.18.2
import { spawn } from 'node:child_process';

const [, , electronBin, cjs, builtinFile] = process.argv;
const child = spawn(electronBin, [cjs, 'app-server', '--stdio'], {
  env: { ...process.env,
         ELECTRON_RUN_AS_NODE: '1',
         ZCODE_BUILTIN_PROVIDER_CONFIG_FILE: builtinFile },
  stdio: ['pipe', 'pipe', 'pipe'],
});
child.stdout.on('data', d => appendRaw(d));           // raw file OUTSIDE the repo
child.stderr.on('data', d => appendErrCapped(d));
child.on('exit', (code, sig) => finish(code, sig));

send({ id: 1, method: 'provider/refreshRegistry', params: { reason: 'probe' } });

// E1/E3 continue here: session/create, session/subscribe, ... each with a delay
setTimeout(() => child.stdin.end(), 12000);           // EOF pitfall: hold stdin open first
setTimeout(() => {                                    // backstop: kill process group
  try { process.kill(-child.pid, 'SIGKILL'); } catch {}
  process.exit(0);
}, 18000);

function send(obj) { child.stdin.write(JSON.stringify(obj) + '\n'); }
```
Observed live output (shapes, abbreviated):
```json
{"method":"startup/storageState","params":{"schemaVersion":1,"attemptId":"<uuid>","sequence":1,"databaseId":"<sha256-like>","database..."}}
{"id":1,"result":{"refreshed":true}}
```
then clean exit 0 on `stdin.end()`; no orphaned processes (group backstop never needed).

### Spawn line (the phase's canonical command)
```bash
ELECTRON_RUN_AS_NODE=1 \
ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<newest zcode-builtin.json under ~/.zcode/v2/runtime/provider/linux-x86_64/> \
<squashfs-root>/zcode <squashfs-root>/resources/glm/zcode.cjs app-server --stdio
```
Note the executable is the Electron binary at the bundle root, with `zcode.cjs` as script argument (matches the field-verified predecessor line and this session's probe).

### Full 3.18.2 method-map extraction (V-SRC against the shipped binary)
```bash
# anchor on a known key, take a wide window, prettify
grep -o '.\{600\}sessionSubscribe:"session/subscribe".\{2500\}' <bundle>/resources/glm/zcode.cjs \
  | head -1 | tr ',' '\n' | grep -o '"[a-zA-Z/]*"$' | sort -u
# repeat with shifted windows until no new method strings appear; diff against the
# 3.17.x packages/shared/src/zcode-protocol/index.ts method table for the drift baseline
```
Verified hit count anchors this session (occurrences, `grep -o | wc -l`): `session/subscribe` 1 (map key), `workspaceKey` 141, `deliveryKind` 61, `desktop-continuous` 15, `turn.completed` 22, `turn.failed` 27; `session/delete` **0** (verifiable absence); `interaction/*` server→host set = exactly 7 methods: `requestPermission`, `requestUserInput`, `requestProviderRuntimeHeaders`, `requestOfficialMcpAuthHeaders`, `requestCredentialCipher`, `browserList`, `browserExecute`.

### workspaceKey source convention (read verbatim this session, 3.17.x clone)
`apps/zcode-cli/packages/bootstrap/src/zcode-protocol-entrypoint.ts` (~lines 222–237):
```ts
const workspaceIdentity = (options.env ?? process.env)[ZCODE_WORKSPACE_IDENTITY_ENV]?.trim();
...
// workspaceKey 必须遵守仓库约定 `workspaceIdentity?.trim() || workspacePath`，
// 否则同路径不同 identity 的远端 workspace 在审计上下文里无法区分。
// 注意：agent 进程当前没有 identity 来源，因此实际多为 undefined，key 退化为 path；
resolveWorkspace: ({ workspaceIdentity, workspacePath }) => {
  const path = workspacePath ?? options.cwd;
  if (!path) return undefined;
  const identity = workspaceIdentity?.trim();
  return { ...(identity ? { workspaceIdentity: identity } : {}),
           workspaceKey: identity || path,
           workspacePath: path };
},
```
(Translation: "workspaceKey must follow the repo convention `workspaceIdentity?.trim() || workspacePath`, else remote workspaces with different identities on the same path cannot be distinguished in audit context; note agent processes currently have no identity source, so the key degrades to the path.") Companion facts verified: `ZCODE_WORKSPACE_IDENTITY_ENV = "ZCODE_WORKSPACE_IDENTITY"` (`packages/shared/src/official-mcp-auth.ts:149`); wire schema `workspaceKey: nonEmptyString` (`packages/shared/src/zcode-protocol/index.ts:2762`) and an optional variant (`:2784`).

### Orca chunk re-verification greps (anchors verified this session)
```bash
C=<fresh-orca-extract>/resources/app.asar.unpacked/out/main/chunks
grep -c "agentCmdOverrides" "$C/agent-session-attachment-store-registry-*.js"
grep -o '"omp"[^}]*' "$C/agent-session-attachment-store-registry-*.js" | head
grep -o '.\{80\}acp-command-sessions.\{120\}' "$C/structured-agent-session-host-assembly-*.js"
```
All five load-bearing strings confirmed present in the two named chunks of the current build (`17.0.5`, `agentCmdOverrides`, `acp-command-sessions`, `launchesWithModel`, `PI_NO_PTY`).

### pi/omp session-file format sample (E5 input, verified this session)
```
<sessions-dir>/<path-slug>/<ISO-8601-local-ts>_<uuid-v4>.jsonl
e.g. sessions/--<a-b-c>--/2026-11-18T17-06-04-918Z_0123abcd-....jsonl
```
where `<path-slug>` is the session's working directory rendered as one token
(slashes become dashes; read the exact slug algorithm from the scanner chunk at
execution time and record it there). The committed doc records the *pattern*
only; the experiment records the JSONL record shape read from the scanner chunk.

### Deb layout inspection without installing (E4)
```bash
gh release download --repo <zcode desktop release source> --pattern '*.deb' -O /tmp/scratch/zcode.deb  # or the official download page URL
dpkg-deb -I /tmp/scratch/zcode.deb   # version scheme, maintainer, deps
dpkg-deb -c /tmp/scratch/zcode.deb   # full path list = the layout row
```
dpkg db remnant verified this session: package `zcode`, version `3.17.2-9661`, architecture amd64, installed-size ~800 MB, empty file list (rc state) — the artifact download is required to recover the layout.

### Secret-scan gate shape (committed at Phase 1; reused by Phase 2 CI)
```bash
#!/usr/bin/env bash
# experiments/scan-secrets.sh — fail on any tracked file matching live-credential shapes
patterns=('eyJ[A-Za-z0-9_-]{10,}\.'      # JWT
          'enc:v[12]:os:'                 # credential blobs
          '"(access_token|apiKey|api-key|token)"\s*:\s*"[A-Za-z0-9_-]{20,}"')
rc=0; for p in "${patterns[@]}"; do
  git grep -nE "$p" -- . ':(exclude).planning' && { echo "SECRET SHAPE: $p"; rc=1; }
done; exit $rc
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Private FINDINGS/field-notes as the fact store | In-repo `docs/` + `experiments/` with D-04 provenance ids | This phase | Facts become reproducible and committable; private notes become corroboration |
| Field note "app-server-stdio-protocol.md" (ends at the unsolved snapshot problem) | The later working recipe (oauth-coding-plan-shim.md + FINDINGS §3) record the solved state | 2026-11-19 | Docs must cite the *solved* facts; the older note is history, not ground truth |
| "Re-verify Orca claims from github.com/stablyai/orca" (CONTEXT canonical ref) | Local unpacked chunks are the only re-verification source; repo 404s | Discovered this session | Plan must not schedule GitHub-based Orca reads (Pitfall 4) |
| 8-card ACP registry (parallel session) | Current build lists ~22 preset ids; ACP subset re-derived at execution | Build drift | Compatibility notes must state the build they describe |
| `workspaceKey` semantics LOW/unconfirmed | Source-resolved convention + one cheap experiment remaining | This session | DISC-04 shrinks to confirmation, not discovery |

**Deprecated/outdated for this phase:** HTTP-surface probing of the desktop (the localhost listener is a decoy); live-card experiments (Phase 7); installing packages to inspect layouts.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The parallel-session omp contract details (launch argv order, `agentCmdOverrides` prefix-args semantics, exact Hb strings) match the currently installed Orca build | Experiment Catalog / DISC-02 | Doc states wrong launch contract; mitigated — the chunk re-verification pass is the first DISC-02 task and all five key strings verified present |
| A2 | Orca's `--version` probe shape under prefixed `agentCmdOverrides` invocations is unverified (probe is Orca→our future binary; irrelevant until Phase 7) | Open Questions | None for Phase 1 docs beyond recording the unknown |
| A3 | `state.updated` follows `provider/refreshRegistry` on 3.18.2 (not observed in the 12 s probe window this session) | E1 design | E1 waits longer; worst case the doc records "no event within window" honestly |
| A4 | `agentCmdOverrides` lives in an Electron storage/orcad store, not `~/.config/orca/*.json` (per parallel session; no key found in the JSON files visible this session) | DISC-02 open items | Recorded as unresolved; Phase 7 live concern |
| A5 | Desktop auto-update introduces a new runtime `<ver>` dir → new builtin revision (inferred from the per-version runtime tree + revision 44 today; no live update observed) | DISC-03 | Marked V-RPT in the doc; E4 records the tree evidence |
| A6 | `startup/storageState` notifications require no host reply (app-server proceeded to answer requests with no reply sent) | DISC-01 | E1/E2 captures will show whether any error/blocked state ever references them |
| A7 | `session/create` accepts `workspace{workspacePath,workspaceKey}` plus `mode`/`persistence`/`titleGenerationEnabled` per 3.17.x schema; exact 3.18.2 param surface re-read before E1/E3 runs | Experiment Catalog | E-probes reveal any rejection immediately (a rejection is itself recorded wire truth) |

## Open Questions (RESOLVED)

All six open questions are resolved — each routed to the plan task that retires it, or closed as a decided known-unknown. None remain open.

1. **Is there a native `zcode` preset card in the current Orca build, and is it ACP?** — RESOLVED by plan 01-04 Task 1 (registry re-enumeration from the current build's registry chunk; delta vs the 8-card list recorded). The omp identity decision stays locked and is not re-litigated either way.
2. **Exact turn-completion terminal set on 3.18.2** — RESOLVED by plan 01-02 Task 2 (E2 answers, or records the honest degradation per D-05).
3. **Does `session/subscribe` work on 3.18.2, and with which delivery kinds** — RESOLVED by plan 01-02 Task 1 (E1 answers; a refusal is a valid, publishable verdict).
4. **`interaction/requestUserInput` semantics** — RESOLVED by plan 01-03 Task 1 (read from its bundle/source context during DISC-01 writing, section 3); plan 01-02 Task 2 (E2) additionally records whether a plain turn triggers it (expected no).
5. **Orca settings-store location for `agentCmdOverrides`** — RESOLVED as a decided known-unknown (assumption A4): documented as [V-RPT] in plan 01-04 Task 2 (`Open items` section); Phase 7 live test resolves it.
6. **Deb/rpm artifact URLs for E4** — RESOLVED by procedure in plan 01-05 Task 1 (exact current download URLs fetched at execution time, never hardcoded).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Experiment harnesses | ✓ | v26.13.1 | python3 3.15.2 |
| jq | Capture inspection | ✓ | 1.7 | python3 -c json |
| Unpacked 3.18.2 desktop bundle | All V-SRC/V-EXP binary work | ✓ | 3.18.2, pristine (`cmp` vs `.orig` identical) | — (required; exists) |
| Electron binary at bundle root | Spawn line | ✓ | bundle build | bundled `node-runtime` untested for app-server mode — use the Electron binary |
| Active `zcode-builtin.json` (revision 44) | Spawn env + revision facts | ✓ | revision field = 44 [VERIFIED this session] | — |
| 3.17.x source clone | Corroborating source citations | ✓ | 3.17.1 (package.json) | upstream GitHub tree |
| OAuth + api-key credentials (keys present, values unread) | E2 tiny turn | ✓ | keys incl. `oauth:zai:access_token` present | D-05 degradation: source-only + limitation |
| Keyring (Secret Service) + D-Bus session | (not needed Phase 1; presence noted) | ✓ | login.keyring unlocked | n/a |
| Orca.AppImage + chunk extract | DISC-02 re-verification | ✓ | installed AppImage + stable extract; 5/5 key strings found | fresh `--appimage-extract` recipe |
| pi/omp agent home with sessions dir | E5 session format | ✓ | sample session file present | scanner-chunk read alone |
| william0wang/zcode-acp repo | DISC-05 | ✓ public, Apache-2.0, pushed 2026-11-19 [VERIFIED: gh api] | ~0.88.x per REFERENCE-ADAPTERS | none needed |
| stablyai/orca repo | (was a CONTEXT fallback) | ✗ HTTP 404 (anonymous + authenticated API) | — | local chunks only (Pitfall 4) |
| deb artifact | E4 deb layout row | ✗ not on disk (rc state, empty file list) | db remnant: 3.17.2-9661 | download at execution (`dpkg-deb -c`, no install) |
| rpm artifact | E4 rpm row | ✗ not on disk | — | same download-and-inspect; or mark row deb-generalized |
| `gh` CLI | repo accessibility checks | ✓ | authenticated | plain `git ls-remote` |
| api.z.ai network reachability | E2 turn | not probed this session | — | D-05 degradation if unreachable |

**Missing dependencies with no fallback:** none blocking — every unavailable item (deb/rpm artifacts, orca repo) has a designed fallback.
**Missing dependencies with fallback:** deb/rpm (download-and-inspect), orca GitHub source (local chunks).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Shell check scripts (no test framework exists yet; greenfield docs phase). Node available for helpers. |
| Config file | none — Wave 0 creates `experiments/check-docs.sh` + `experiments/scan-secrets.sh` |
| Quick run command | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` (<30 s, pure grep/jq) |
| Full suite command | Same + `git grep` provenance coverage audit (every V-EXP id in docs resolves to an existing `experiments/captures/` file) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DISC-01 | `docs/protocol/app-server-dialect.md` exists; contains transport + method map + subscribe verdict + terminal-signals + drift-baseline sections; every claim line carries a V-tag | structure+tag check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| DISC-02 | `docs/protocol/acp-client-dialect.md` names omp, floor 17.0.5, launch contract, method sets; neutral phrasing (no hidden-goal wording) | structure+tag check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| DISC-03 | `docs/protocol/install-discovery.md` matrix covers AppImage/deb/unpacked + auto-update section | structure check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| DISC-04 | `docs/protocol/workspace-key.md` contains ≥1 V-SRC citation and ≥1 V-EXP id | tag check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| DISC-05 | `docs/research/reference-adapters.md` exists with adopt/avoid + Apache-2.0 attribution section | structure check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| D-02 | No tracked file matches secret shapes; every `[V-EXP <id>]` in docs resolves to `experiments/captures/<id>*` | secret-scan + id-resolution | `bash experiments/scan-secrets.sh` | ❌ Wave 0 |
| D-04 | Legend present exactly once in `docs/protocol/README.md`; docs reference it rather than redefining | structure check | `bash experiments/check-docs.sh` | ❌ Wave 0 |
| Provenance honesty | Bundle-derived claims re-derivable: spot-check N method strings via grep | spot audit | `grep -o 'sessionSubscribe:"session/subscribe"' <bundle>` (manual, recorded in doc) | manual-only |

Notes: this phase has no executable unit tests (nothing is coded) — Nyquist sampling here is *documentation conformance*: structure checks, provenance-tag coverage, V-EXP id resolution, and the secret-scan gate. The one manual-only row is the bundle spot-check, justified because the bundle path is machine-private (supplied in agent prompts, never committed).

### Sampling Rate
- **Per task commit:** `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh`
- **Per wave merge:** both + a human skim of the changed doc sections (provenance tags are judgment-bearing)
- **Phase gate:** all six docs green, every V-EXP id resolves, secret-scan clean, success-criteria 1–5 each demonstrably satisfied before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `experiments/check-docs.sh` — structure + provenance-tag conformance for all six docs
- [ ] `experiments/scan-secrets.sh` — secret-shape gate (regex set above; Phase 2 CI reuses it)
- [ ] `experiments/scrub-capture.mjs` — canonical scrubber (must exist BEFORE the first capture is written)
- [ ] `experiments/README.md` — how to run E1–E5; scrub-first rules; "never in CI" warning
- [ ] `docs/protocol/README.md` — index + provenance legend (needed before claim-ledger docs land)

## Security Domain

Phase handles live credentials and commits capture files — the security surface is information-handling, not application code.

### Applicable ASVS Categories (level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth code this phase; auth docs only describe official paths |
| V3 Session Management | no | No sessions implemented |
| V4 Access Control | no | No access surface |
| V5 Input Validation | yes (adapted) | Canonical scrubber + secret-scan gate over everything committed; structured capture headers |
| V6 Cryptography | no | No crypto implemented; docs *describe* AES-256-GCM blob shapes without handling key material |
| V7 Errors/Logging | yes (adapted) | Never echo `secret-tool` output; raw captures outside repo; stderr-capped capture of child stderr |
| V9 Communications | no | No network code; E2 uses existing credentials over the official binary's own channel |

### Known Threat Patterns for This Phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Live credentials leaking into committed captures (JWT/`enc:v2:`/PAT/account UUIDs) | Information Disclosure | Scrub-first pipeline; secret-scan gate per commit; raw captures born outside the repo |
| Private machine paths committed into docs | Information Disclosure | Neutral phrasing rule; paths supplied only in agent prompts; check-docs greps for path shapes |
| Accidental API-key creation during experiments | Elevation of Privilege (account state) | Phase 1 scripts never POST `api_keys`; reuse existing key; access-token refresh only if needed |
| Quota burn via uncontrolled turns | Denial of Service (own plan) | One tiny turn (E2) max, manual tier, never CI; spend recorded in experiment README |
| Keyring material exposure while probing | Information Disclosure | Never print/redirect keyring lookups into captures; Phase 1 does not need decrypted values at all |
| Bundle tampering undermining the honesty claim | Tampering | E4.1 re-verifies `zcode.cjs` ≡ `zcode.cjs.orig`; doc records bundle copy provenance (Pitfall 2) |
| Research artifacts drifting from desktop auto-updates mid-phase | Repudiation (stale evidence) | Capture headers record builtin revision + date at capture time (Pattern 3) |

## Sources

### Primary (HIGH confidence — tool-verified this session, 2026-11-19)
- Live probe of the shipped 3.18.2 app-server (spawn line, `startup/storageState` observations, `provider/refreshRegistry` reply, EOF behavior, no orphans) — executed in this research session; raw transcript in research scratch (not committed)
- Bundle greps: method map (`sessionSubscribe:"session/subscribe"` neighborhood), `interaction/*` = 7 methods, occurrence counts, `session/delete` absence, error-code literals (−32010 ×4, −32031 ×5)
- `cmp` of `zcode.cjs` vs `zcode.cjs.orig` — identical (pristine)
- Active `zcode-builtin.json` read: keys `config/revision/schemaVersion`, `revision` = 44
- 3.17.x source clone (v3.17.1): `zcode-protocol-entrypoint.ts:222–237` (workspaceKey convention, read verbatim), `official-mcp-auth.ts:149` (env name), `zcode-protocol/index.ts:2762/2784` (workspaceKey schema), `index.ts:4265` (`sessionSubscribe` method constant)
- Orca stable-extract chunks: presence of `17.0.5`, `agentCmdOverrides`, `acp-command-sessions`, `launchesWithModel`, `PI_NO_PTY` in `agent-session-attachment-store-registry-*.js` / `structured-agent-session-host-assembly-*.js`; preset id list (~22) in the registry chunk; three distinct chunk-hash builds observed mounted
- `gh api repos/william0wang/zcode-acp` — public, Apache-2.0, 91 stars, pushed 2026-11-19
- `gh api repos/stablyai/orca` + anonymous fetch — both HTTP 404 (repo not publicly accessible)
- Local environment: dpkg db entry (zcode 3.17.2-9661, rc), desktop entry (Exec → AppImage, `x-scheme-handler/zcode`), credentials.json key enumeration (keys only), keyring/D-Bus/secret-tool presence, node/jq/python versions, pi agent home sessions layout
- Private research notes (outside this repo; paths in agent prompt): FINDINGS.md (§0–§7 read in full), field-notes `app-server-stdio-protocol.md`, `oauth-coding-plan-shim.md` (headers), predecessor scripts inventory

### Secondary (MEDIUM confidence)
- [zcode.z.ai/en/docs/install](https://zcode.z.ai/en/docs/install) — Linux installers: AppImage, deb, rpm; x64 + ARM64; fuse dependency note (fetched this session) [CITED: zcode.z.ai install docs]
- WebSearch summary referencing the linux-wsl troubleshooting page: deb + AppImage installs can clash over the system URL handler — corroborated locally by the desktop-entry remnant, but the page itself not fetched [CITED: search summary of zcode.z.ai linux-wsl guide]

### Tertiary (LOW confidence)
- A WebSearch result summary claiming stablyai/orca is "free and open source under the MIT License" — **refuted** by two direct checks (404 twice); recorded as a known-bad source

## Metadata

**Confidence breakdown:**
- Experiment mechanics: HIGH — the harness pattern was executed successfully against the real binary this session, including its failure mode
- Claim inventory per doc: HIGH — assembled from `.planning/research/*`, private notes read in full, and fresh source/bundle anchors
- Orca-side claims: MEDIUM until the chunk re-verification pass runs (build drift confirmed real; key strings verified present)
- Install discovery: HIGH for local layouts (probe-verified), MEDIUM for deb/rpm rows (artifact download pending) and auto-update live behavior (V-RPT)

**Research date:** 2026-11-19
**Valid until:** ~2026-12-19 for local-source facts (stable machine artifacts); Orca/build-dependent claims valid until the next Orca or ZCode desktop update — re-run the freshness greps if either updates.

## RESEARCH COMPLETE
