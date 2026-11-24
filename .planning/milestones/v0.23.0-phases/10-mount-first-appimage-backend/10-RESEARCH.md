# Phase 10: Mount-First AppImage Backend - Research

**Researched:** 2026-11-23
**Domain:** Linux AppImage acquisition for the ZCode desktop app-server — managed `--appimage-mount` helper lifecycle, version-ranked install discovery, fail-loud diagnostics, CI-safe fake runtime
**Confidence:** HIGH for the mount protocol, the spawn/supervision contracts, and the current-code facts; MEDIUM for the discovery-policy interpretations flagged in the Assumptions Log (notably A1-A5) and for the unverified prompt leg on ZCode 4.0.0

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** The app-server runs from a FUSE mount created by
  `<AppImage> --appimage-mount`. Extraction is **not** used for packed AppImages and
  no `squashfs-root`/`zcode-<ver>` copy is created by the wrapper.
- **D-02:** The mount helper is a **supervised child**: start it, read the mount
  path from its stdout, validate `<mount>/zcode` + `<mount>/resources/glm/zcode.cjs`,
  then spawn the app-server with that root. On every exit path (normal, protocol
  error, signal, client disconnect) stop the app-server first, then the helper, reap
  both, and verify the mount is gone. Zero orphans, zero stale mounts.
- **D-03:** **No silent extraction fallback.** Missing FUSE, helper exit before the
  mount line, readiness timeout, or an unspawnable tree inside the mount exits
  non-zero with an actionable message naming the AppImage path, the failing step, the
  observed output, and the remediation.
- **D-04:** `--app-server` / `ZCODE_ACP_APP_SERVER` accept both a packed `.AppImage`
  and an install root directory. A directory keeps today's invariant validation; a
  packed AppImage goes through the mount path. The precedence (flag > env > discovery)
  is unchanged.
- **D-05:** Mid-session mount loss (helper dies or mount disappears while the
  app-server is alive) ends the session with an explicit cause and kills the child —
  never respawn against a vanished root, never silently continue.
- **D-06:** Discovery evaluates AppImage-layout candidates **by version** and selects
  the newest `<ver>`. A stale unpacked `~/Applications/zcode-<ver>/squashfs-root`
  must lose to a newer `ZCode-<ver>-linux-<arch>.AppImage`. Version extraction follows
  the documented formats (`install-discovery.md:33`): AppImage filename `<ver>`,
  unpacked parent dir `<ver>`; compare on `<ver>` only (existing `compareVersions`
  semantics).
- **D-07:** Candidate handling is **per-candidate**: a dangling symlink,
  `EvalSymlinks` failure, volatile `/tmp/.mount_*` path, or invalid tree is recorded
  as a miss and the probe continues with the next candidate. Today one bad first
  candidate aborts the whole probe — that is the defect this phase closes.
- **D-08:** Deterministic tie-break when versions are equal: prefer the existing
  spawnable unpacked tree (no mount needed at equal version); prefer the user-facing
  symlink over a versioned AppImage file when both are packed; then lexicographic
  path. The rule must be documented and covered by a test.
- **D-09:** Diagnostics must report the selected backend's **layout, version, and
  resolved path**, and the miss row for every skipped candidate.
- **D-10:** Linux/AppImage only. deb/rpm/opt paths and the existing discovery order
  for them stay as they are.
- **D-11:** Wire behavior is untouched: same spawn line, same env pair
  (`ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` + `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE`),
  same JSONL dialect. Only the acquisition of the root changes.

### Claude's Discretion
- Exact package layout inside `internal/supervisor`, helper-process plumbing
  (readiness timeout value, stdout scanner, stderr capture), log field names, error
  wording, test seam names, and whether the helper is a separate type or a
  supervisor child variant.

### Deferred Ideas (OUT OF SCOPE)
- Extraction fallback for hosts without FUSE — explicitly rejected for v1; loud
  failure with remediation is the contract (MNT-V2-01).
- Backend protocol-drift gate / version negotiation for major desktop upgrades —
  separate milestone (DISC-V2-01).
- macOS/Windows backend acquisition — out of the v1 platform scope.
- Bundling/repacking the AppImage — never (honesty contract).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (verbatim from REQUIREMENTS.md) | Research Support |
|----|---------------------------------------------|------------------|
| MNT-01 | The wrapper starts the ZCode app-server from a packed AppImage through a managed `--appimage-mount` helper, with no extraction and no on-disk duplicate | Pattern 1 + Pattern 2; runtime source verifies the mount protocol; field probe ran `session/create` from a real 4.0.0 mount |
| MNT-02 | `--app-server <path.AppImage>` and `ZCODE_ACP_APP_SERVER` accept a packed AppImage; the wrapper mounts it, validates `<mount>/zcode` and `<mount>/resources/glm/zcode.cjs`, and spawns the app-server from that mount | Pattern 6 (override routing) + Pattern 2 (validation); current `overrideInstall` defect documented |
| MNT-03 | The mount helper is a supervised child started before the app-server and torn down after it on every exit path (normal, error, signal, client disconnect) — zero orphan processes and zero stale mounts | Pattern 1 + Pattern 4 + OS-signal gap in current `run`; `SpawnChild`/`Supervisor.shutdown` quotes |
| MNT-04 | Mount failures (missing FUSE, helper exit before the mount line, readiness timeout, unspawnable tree in the mount) fail loud with a non-zero exit and an actionable message naming the AppImage, the failing step, and remediation; never a silent extraction fallback | Pattern 2 error classification; exact runtime failure strings; exit-code conventions |
| MNT-05 | If the mount helper dies or the mount disappears while the app-server is alive, the supervisor terminates the session with an explicit cause instead of respawning against a vanished root | Pattern 5 (Lost watcher + spawn guard); `Supervisor.Start` restart-loop quote |
| MNT-06 | A mount-backed run creates no `squashfs-root`/`zcode-<ver>` directory and does not modify the AppImage | Pattern 3 (extraction seam removed); test map + real-test hash check |
| DISC-01 | Discovery selects the newest usable backend by version across layouts; a stale unpacked `squashfs-root` never beats a newer packed `.AppImage` | Pattern 3 algorithm; real-machine fixture (4.0.0 packed vs 3.18.2 unpacked) |
| DISC-02 | AppImage candidates are tried one by one: a dangling symlink, unresolvable path, volatile mount, or invalid tree is recorded as a miss and the probe continues with the next candidate | Pattern 3; current `packed[0]` abort defect quoted |
| DISC-03 | Discovery diagnostics report the selected backend's layout, version, and resolved path; version comes from the AppImage filename or the unpacked dir name | Pattern 3 diagnostics rows + Pattern 6 `desktopVersion` update |
| DISC-04 | Selection is deterministic for a given filesystem state, with documented version ordering and tie-break (symlink before versioned file, then lexicographic) | Pattern 3 comparator; D-08 tie-break rules and tests |
| VER-01 | CI-safe unit tests drive mount lifecycle via a scripted fake AppImage runtime (mount line, readiness, teardown, early death, missing FUSE) without requiring FUSE | Code Examples (fake script); existing reexec/`SpawnChild` test patterns |
| VER-02 | A guarded real-machine test (env-gated, never CI) mounts a real AppImage, spawns the app-server, performs `session/create`, and shuts down with zero leftovers | Realdesktop test pattern + field probe proof on 4.0.0 |
| VER-03 | Docs updated coherently: `docs/protocol/install-discovery.md` (managed mount is a first-class layout; volatile-mount rule revised), README (AppImage override and behavior), and the provenance note; no documented claim contradicts behavior | Docs Impact subsection; V-SRC tagging strategy and `check-docs.sh` constraints |
</phase_requirements>

## Summary

Phase 10 changes **only how the backend root is acquired** `[CITED: .planning/phases/10-mount-first-appimage-backend/10-CONTEXT.md D-01..D-11]`. Discovery must still be a pure-read operation that returns one `Install`; the app-server spawn line, the env pair, and the JSONL dialect stay byte-identical (D-11). The new acquisition path for a packed `.AppImage` is: spawn `<AppImage> --appimage-mount` as a supervised child, read the mount point from its stdout, validate the mount tree, and hand the mounted root to the existing supervisor.

Three facts dominate the design:

1. **The current discovery code cannot select 4.0.0 on this machine.** `[VERIFIED: internal/supervisor/discover.go:405-423; field probe 2026-11-23]` The appimage probe returns early whenever any unpacked tree exists — `"packed AppImage present but an unpacked squashfs-root exists — extraction skipped"` — and even without that, it resolves only `packed[0]` after sorting symlinks first; the dangling `~/Applications/ZCode.AppImage` symlink makes `EvalSymlinks` fail and the whole probe returns a miss. The unpacked probe then selects 3.18.2. Both defects are quoted in Pattern 3.
2. **The `--appimage-mount` protocol is simple but has traps.** `[VERIFIED: AppImageKit src/runtime.c@effcebc; field probe 2026-11-23]` The exact runtime shipped in ZCode 4.0.0 reports `Version: effcebc`. At that revision, the runtime prints the **realpath of the mount directory to stdout after the mount is ready**, flushes, and `pause()`s forever. Failure text from `notify()` also goes to **stdout**; `TMPDIR` relocates the mount; `APPIMAGE_EXTRACT_AND_RUN` in the environment silently bypasses the mount branch and extracts + runs the GUI. The helper's fuse daemon is daemonized into its own session, so a process-group kill does not reach it — a keepalive pipe unmounts when the helper dies.
3. **The real backend works from a mount.** `[VERIFIED: field probe 2026-11-23, one-shot test overlay; no repo files changed]` A one-shot field probe on 2026-11-23 mounted the real 4.0.0 AppImage (~23 ms to the mount line), ran `ELECTRON_RUN_AS_NODE=1 <mount>/zcode -e ...` (`node-ok v24.21.0 arch x64`), and completed `session/create` through the production `SpawnArgv`/`ChildEnv` contract in 9.0 s, leaving zero processes and zero mounts. This substantially de-risks VER-02 and success criterion 1.

**Primary recommendation:** add `internal/supervisor/mount.go` (a `Mount` owner built on the existing `SpawnChild` primitives, with a private `TMPDIR`, readiness line validation, `Lost` watcher, and staged `Stop`), extend `Install` with `AppImage`/`Version`, rewrite the AppImage discovery stage into a per-candidate version-ranked evaluation (delete `ExtractAppImage`), route `--app-server` `.AppImage` values through it, add OS-signal handling to `run`, and update the three doc surfaces. No new Go dependencies.

## Project Constraints (from CLAUDE.md)

Source: `.claude/CLAUDE.md` — the config points at it (`[VERIFIED: .planning/config.json:65]`):

```json
  "claude_md_path": "./.claude/CLAUDE.md",
```

Directives that constrain this phase:

- **Stack:** Go only; the single backend is the ZCode desktop app-server. No new dependencies: `go.mod` stays `go 1.27` + `acp-go-sdk v0.20.0`; `mise run tidy-diff` fails on a second `require` line. `[VERIFIED: .claude/CLAUDE.md "Tech stack"; go.mod]`
- **Toolchain single source:** versions come from `mise.toml` (Go 1.27, golangci-lint 2.15.0, goreleaser 2.20.2, node 26); CI runs `mise install` + `mise run ci`. Do not pin tools a second time. `[VERIFIED: .claude/CLAUDE.md Technology Stack; mise.toml]`
- **Strict lint:** golangci-lint v2.15.0 exact pin, `linters.default: all` with documented disables; new code must pass `mise run lint` (lll 120, gosec, etc.). `[VERIFIED: .claude/CLAUDE.md Technology Stack; .golangci.yml]`
- **Docs language:** English (README, recipes, `docs/`). `[VERIFIED: .claude/CLAUDE.md Constraints]`
- **Research policy:** ZCode facts source-first (`github.com/zai-org/ZCode`), then experiments; Orca facts source-first. This phase's AppImage facts use the shipped runtime source at its embedded commit plus field probes (see Sources). `[VERIFIED: .claude/CLAUDE.md Constraints]`
- **Platform:** cross-platform code in principle, v1 supports Ubuntu GNOME only; this phase is Linux/AppImage only (D-10). `[VERIFIED: .claude/CLAUDE.md Constraints; 10-CONTEXT.md D-10]`
- **Honesty:** never patch the desktop, never forge attestations/signatures/version identities, never report an unknown as a number (`docs/provenance.md`). The mount stays read-only and creates no extraction copy. `[VERIFIED: .claude/CLAUDE.md Project; docs/provenance.md:14-24]`
- **Workflow:** phase work goes through GSD commands; do not make direct repo edits outside a GSD workflow. `[VERIFIED: .claude/CLAUDE.md "GSD Workflow Enforcement"]`
- **Git:** feature branches cut from an up-to-date master; this worktree is already on `gsd/phase-10-appimage-mount-first`. `[VERIFIED: user AGENTS.md; git branch]`
- **Backend protocol hazards to preserve:** no JSON-RPC on the ZCode leg, no `bufio.Scanner` defaults on the protocol leg, no stdout writes except ACP frames, no PTY, no ORM/DI. New mount code must not violate these. `[VERIFIED: .claude/CLAUDE.md "What NOT to Use"]`

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Packed-AppImage mount lifecycle (spawn helper, readiness, teardown, reaping) | Supervisor / process tier (`internal/supervisor`) | — | Owns `os/exec`, process groups, pipes, and reaping; discovery must stay side-effect-free |
| Install discovery and version ranking | Supervisor / discovery (`internal/supervisor/discover.go`) | — | Pure filesystem reads plus the existing `compareVersions`; no mount, no FUSE |
| CLI/env precedence and override mapping | CLI/config (`cmd/zcode-acp/main.go`) | supervisor validators (`ExplicitRootInstall`, new packed validator) | Flag > env > discovery is a CLI concern; the supervisor owns the Install shape |
| Diagnostics | Supervisor probe rows | `cmd` slog (selected layout/version/path, mount path) | Probe rows are the actionable empty-result product; main logs the selection |
| App-server spawn/env/JSONL | Supervisor + `internal/zcode` + `internal/acpfront` (UNCHANGED) | — | D-11: same spawn line, same env pair, same dialect |
| Exit orchestration on signals | `cmd` `run` | `supervisor.Mount.Stop` / `Supervisor.shutdown` | One owner must turn SIGINT/SIGTERM into staged teardown |
| Docs (install-discovery, README, provenance) | `docs/`, `README.md` | — | VER-03; honesty/provenance rules |

## Standard Stack

### Core

| Library / component | Version | Purpose | Why Standard |
|---------------------|---------|---------|--------------|
| Go toolchain + stdlib | Go 1.27 (local 1.27.3) | All new code | No new modules may be added; `tidy-diff` enforces the dependency set. Needed stdlib: `os/exec`, `syscall`, `os/signal`, `context`, `bufio`, `time`, `log/slog`, `path/filepath`, `os` |
| `github.com/coder/acp-go-sdk` | v0.20.0 (existing) | ACP front leg | Unchanged; this phase never touches the ACP surface |
| Existing `internal/supervisor` primitives | in-repo | `SpawnChild`, `Child`, `Supervisor`, `SpawnArgv`, `ChildEnv` | Reuse the proven process-group, stderr-drain, and reap machinery; do not hand-roll a second spawn path |
| AppImage type-2 runtime (embedded in the user's AppImage) | `Version: effcebc` for ZCode 4.0.0 | `--appimage-mount` helper | The AppImage's own runtime is the only supported way to mount the squashfs without extracting; it is an OS artifact, not a Go dependency |
| Linux FUSE + libfuse2 | host: `libfuse2t64` 2.9.9-8.1build1, FUSE3 3.14.0 | Backs the legacy runtime's `dlopen("libfuse.so.2")` mount | Required by the classic runtime; Ubuntu 24.04 renamed the package to `libfuse2t64` |

The Ubuntu 24.04 `libfuse2` → `libfuse2t64` rename and the install command are documented at `[CITED: docs.appimage.org/user-guide/troubleshooting/fuse.html]`; the installed versions above are field-verified (`[VERIFIED: field probe 2026-11-23, dpkg/ldconfig]`). The AppImage runtime version `effcebc` is what the shipped 4.0.0 AppImage reports (`[VERIFIED: field probe 2026-11-23; AppImageKit src/runtime.c@effcebc]`).

**Version verification (this session):**

```text
$ go version
go version go1.27.3 linux/amd64
$ cat go.mod
module github.com/Djarvur/zcode-acp

go 1.27

require github.com/coder/acp-go-sdk v0.20.0
$ ~/Applications/ZCode-4.0.0-linux-x64.AppImage --appimage-version   # prints on stderr
Version: effcebc
$ dpkg -l 'libfuse*' | grep ^ii
libfuse2t64:amd64 2.9.9-8.1build1
libfuse3-3:amd64 3.14.0-5build1
libfuse3-dev:amd64 3.14.0-5build1
$ golangci-lint --version
golangci-lint has version 2.15.0 built with go1.27.0 ...
$ node --version
v26.13.1
```

The `Version: effcebc` value is the AppImageKit commit the embedded runtime reports; it is the commit whose `src/runtime.c` is quoted throughout this document (`[VERIFIED: field probe 2026-11-23; AppImageKit src/runtime.c@effcebc]`).

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `internal/supervisor.SpawnChild` | in-repo | Process-group spawn, capped stderr drain, exactly-one reap | Starting the mount helper and (unchanged) the app-server |
| `internal/supervisor.compareVersions` | in-repo | Numeric dotted-version ordering | Ranking AppImage-layout candidates (D-06) |
| `internal/supervisor.BuiltinRevision` | in-repo | `zcode-builtin:<rev>:<sha256(path)>` | Unchanged; post-mount `BuiltinFile` must feed it |
| `syscall.Kill` / `os.Stat` / `os.Lstat` | stdlib | Staged stop, mount presence/absence, symlink handling | Teardown verification and per-candidate resolution |
| `os/signal.NotifyContext` | stdlib | SIGINT/SIGTERM/SIGHUP → staged shutdown | Closing the D-02 "signal" exit path (absent today) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `<AppImage> --appimage-mount` | `<AppImage> --appimage-extract` into a private dir | Rejected by D-01: creates an ~850 MB duplicate, violates MNT-06, and is the current behavior this phase removes |
| AppImage runtime mount | `APPIMAGE_EXTRACT_AND_RUN=1` | Rejected: extracts and runs `AppRun` (GUI); `--appimage-mount` is bypassed. Must be scrubbed from the helper env |
| AppImage runtime mount | Read the squashfs from Go / cgo libfuse | Hand-rolled squashfs+FUSE implementation; huge, unsafe, and unnecessary |
| Existing `SpawnChild` | `os/exec` inline in `Mount` | Duplicates process-group/reap/stderr logic and invites the exact PITFALL-3/PITFALL-6 bugs the supervisor already solved |
| `os.Stat(<mount>/zcode)` for mount presence | `mount` + `grep` / `fusermount -u` | Shelling out is non-portable, racy, and `fusermount -u` fails `EBUSY` while the helper holds fd 1023 (field-verified) |
| `compareVersions` | Semver library | New dependency; D-06 explicitly says existing `compareVersions` semantics |

### Installation

No package installation. `go.mod` must remain exactly one `require` line; `mise run tidy-diff` enforces it. The AppImage and FUSE/libfuse2 are host prerequisites documented in the README, not vendored dependencies.

## Package Legitimacy Audit

**No external packages are added by this phase.** The work is stdlib-only inside the existing module. The AppImage runtime is an executable embedded in the user's desktop artifact, not a package-manager dependency; it is executed because the user points the wrapper at that AppImage (or discovery selects a `ZCode*.AppImage` under `~/Applications`).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| — | — | — | — | — | — | None — no package installs |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious [SUS]:** none

Guardrail: any proposed `go get` in this phase is a plan defect; `mise run tidy-diff` fails on a second `require` line.

## Architecture Patterns

### System Architecture Diagram

```text
                         ┌──────────────────────────────────────────────┐
  CLI flag / env  ──────▶│ config: flag > env > discovery (unchanged)   │
  (--app-server,         └───────────────┬──────────────────────────────┘
   ZCODE_ACP_APP_SERVER)                 │
                                         ▼
                         ┌──────────────────────────────────────────────┐
                         │ discovery (PURE READS — no mount, no FUSE)   │
                         │ 1 override                                   │
                         │ 2 xdg-entry (package tree wins here)         │
                         │ 3 AppImage stage:                            │
                         │    collect packed + unpacked + xdg seed      │
                         │    evaluate EACH (miss rows)                 │
                         │    rank by <ver> (compareVersions)           │
                         │    tie-break: unpacked > symlink > reg. file  │
                         │ 4 dpkg  5 opt-path  6 rpm   (unchanged)      │
                         │ 7 builtin-runtime (unchanged)                │
                         │ 8 builtin-bundled (deferred if packed)       │
                         └───────────────┬──────────────────────────────┘
                                         │ Install{Layout, Version, AppImage?}
                                         ▼
                    ┌────────────────────┴─────────────────────┐
                    │ AppImage == "" ?                          │
                    │   yes ──▶ existing supervisor path         │
                    │   no  ──▶ PrepareMount                     │
                    └────────────────────┬─────────────────────┘
                                         ▼
      ┌──────────────────────────────────────────────────────────────┐
      │ Mount owner (internal/supervisor, one per wrapper run)       │
      │  private TMPDIR (0700)                                       │
      │  SpawnChild(<AppImage> --appimage-mount, allowlisted env)    │
      │  read stdout line ──▶ validate <mount>/zcode + zcode.cjs     │
      │  fill bundled BuiltinFile if runtime cache missed            │
      │  watch helper.Done + poll tree accessibility (Lost)          │
      └───────────────────────────────┬──────────────────────────────┘
                                      │ mounted Install{Root=<mount>, ...}
                                      ▼
      ┌──────────────────────────────────────────────────────────────┐
      │ existing Supervisor (UNCHANGED restart policy)                │
      │  SpawnFunc guard: refuse to spawn/restart if mount lost      │
      │  SpawnChild(SpawnArgv, ChildEnv) ──▶ zcode.Conn ──▶ ACP front │
      └───────────────────────────────┬──────────────────────────────┘
                                      │ exit: ACP done / child lost /
                                      │ helper lost / SIGINT/SIGTERM
                                      ▼
      ┌──────────────────────────────────────────────────────────────┐
      │ staged teardown (ORDER IS LOAD-BEARING)                       │
      │  1 stop app-server (stdin end → grace → group SIGKILL → reap) │
      │  2 stop mount helper (SIGTERM → grace → group SIGKILL → reap) │
      │  3 poll <mount>/zcode until ENOENT (gone) or deadline         │
      │  4 remove the private TMPDIR (ours only)                      │
      └──────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```text
internal/supervisor/
├── discover.go        # candidate collection, version ranking, per-candidate probe rows
├── mount.go           # NEW: PrepareMount, Mount, readiness, validation, Stop, Lost
├── mount_test.go      # NEW: fake-runtime lifecycle tests (no FUSE)
├── spawn.go           # unchanged argv/env contract; optional stderr tap
├── supervise.go       # unchanged restart policy; mount guard lives in the SpawnFunc
├── discover_test.go   # updated ranking/per-candidate tests; extraction test removed
└── realdesktop_test.go# guarded real mount + session/create test (build tag + env)

cmd/zcode-acp/
├── main.go            # signal ctx, prepare-mount before startSupervised, override routing
└── main_test.go       # binary-level fake-AppImage e2e (CI-safe)

docs/protocol/install-discovery.md   # managed-mount row/section; rank order; :42/:114/:116
README.md                            # install prerequisites, --app-server meaning, troubleshooting
docs/provenance.md                   # read-only mount, no extraction copies, FUSE limitation
scripts/vm-e2e.sh                    # discovery narrative mirror (optional but coherent)
```

### Pattern 1: Mount lifecycle above the restart loop (not inside it)

**What:** The mount helper is a **supervised child in the `SpawnChild` sense** — own process group, wired pipes, capped stderr drain, exactly-one reap, explicit stop — but it must **not** be owned by `Supervisor.Start`. `Supervisor.Start` restarts on child exit and owns a crash window; restarting a mount helper would remount mid-session and violate D-05.

The existing `Child` already carries everything the mount owner needs:

```go
// internal/supervisor/spawn.go:82-107 (field names and types verbatim; comments omitted)
type Child struct {
	Cmd *exec.Cmd
	Stdin io.WriteCloser
	Stdout io.ReadCloser
	PgID int
	Exited <-chan ExitInfo
	Done <-chan struct{}
	StderrDone <-chan struct{}
}
```

`SpawnChild` performs the process-group start, the three stdio pipes, the monitor, and the exactly-once reap (`[VERIFIED: internal/supervisor/spawn.go:178-216]`):

```go
// internal/supervisor/spawn.go:236-273 (verbatim excerpts)
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
...
cmd.Cancel = func() error {
	if cmd.Process == nil {
		return nil
	}

	return killGroup(cmd.Process.Pid)
}
cmd.WaitDelay = opts.WaitDelay
```

`killGroup` is the unconditional group SIGKILL backstop (`[VERIFIED: internal/supervisor/spawn.go:323-329]`):

```go
func killGroup(pgid int) error {
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("zcode: group kill %d: %w", pgid, err)
	}

	return nil
}
```

**When to use:** Always for a packed-AppImage backend. One `Mount` per wrapper run, started before the first app-server spawn, stopped after the app-server's staged shutdown. If the app-server crashes and the supervisor restarts it, the mount stays up and the new child spawns against the same root.

**Design shape (recommended):**

```go
// internal/supervisor/mount.go — design sketch, not yet implemented
type MountOpts struct {
	ReadyTimeout time.Duration // default 15s
	StopGrace    time.Duration // default 5s
	ReapDeadline time.Duration // default 10s
	PollInterval time.Duration // default 50ms
}

type Mount struct {
	appImage string
	version  string
	root     string
	base     string        // private TMPDIR we created (0700)
	child    *Child
	lost     chan struct{}
	once     sync.Once
	// exit info, cleanup state under a mutex
}

func PrepareMount(ctx context.Context, inst Install, opts MountOpts) (*Mount, error)
func (m *Mount) Install() Install      // mounted Root/Binary/Entrypoint + BuiltinFile
func (m *Mount) AppImage() string
func (m *Mount) Root() string
func (m *Mount) Lost() <-chan struct{} // closed once on helper death or tree loss
func (m *Mount) Accessible() bool      // <root>/zcode stat succeeds
func (m *Mount) Guard() error          // ErrMountLost when !Accessible()
func (m *Mount) Stop() error           // idempotent staged teardown
```

**Spawn-env isolation:** build a dedicated helper env, never `ChildEnv`:

```go
// design sketch
env := []string{
	"PATH=" + parentPATH,
	"HOME=" + parentHOME,
	"TMPDIR=" + privateBase,
}
```

This is an allowlist by construction. It excludes `APPIMAGE_EXTRACT_AND_RUN`, `TARGET_APPIMAGE`, `APPDIR`, `APPIMAGE`, `NO_CLEANUP`, `VERBOSE`, and every proxy/credential variable. The runtime's behavior with `APPIMAGE_EXTRACT_AND_RUN` is not hypothetical — the branch precedes the mount branch (`[VERIFIED: AppImageKit src/runtime.c@effcebc:688]`):

```c
if (getenv("APPIMAGE_EXTRACT_AND_RUN") != NULL || (arg && strcmp(arg, "appimage-extract-and-run") == 0)) {
```

Inheriting that variable would extract an ~850 MB copy into `TMPDIR` and launch `AppRun` (the GUI) instead of mounting — a silent violation of MNT-06 and a protocol hang.

**Log plumbing (discretion):** emit one Info record when the mount is ready (`appimage`, `version`, `mount`, `pid`, `ready_ms`), one on teardown (`mount`, `helper_exit`, `mount_gone`, `teardown_ms`), and Debug records for captured stdout/stderr lines. Error records carry `appimage`, `step`, `exit`, and quoted observed output. Never log the helper env, provider config paths, or any credential; the mount path itself is not secret.

**Why a private `TMPDIR`:** it makes the mount point attributable. The runtime creates the mount directory as `$TMPDIR/.mount_<first-6-chars-of-argv0-basename>XXXXXX` (`[VERIFIED: AppImageKit src/runtime.c@effcebc:501-525, 603-609, 830-840]`):

```c
char temp_base[PATH_MAX] = P_tmpdir;
...
const char* const TMPDIR = getenv("TMPDIR");
if (TMPDIR != NULL)
    strcpy(temp_base, getenv("TMPDIR"));
...
strncpy(mount_dir + templen, "/.mount_", 8);
strncpy(mount_dir + templen + 8, path_basename, namelen);
strncpy(mount_dir+templen+8+namelen, "XXXXXX", 6);
```

Consequences the plan must encode:

- All `.mount_*` entries under our private base are ours. Mount-name prefix cannot be assumed (`ZCode-` vs `ZCode.` vs a symlink name); only the `.mount_` prefix and the private-base containment are stable.
- A user `TMPDIR` with spaces is legal and produces a mount path with spaces (field-verified below). Never parse the line with `strings.Fields`.
- Scanning a shared `/tmp` for `.mount_*` is unsafe — this host had multiple leaked `/tmp/.mount_Orca.*` mounts from unrelated processes during the probe session. Private base, or nothing.

### Pattern 2: Readiness protocol — read, validate, classify, never guess

**What:** The runtime prints exactly one line on success and then waits forever. The line is printed **after** the mount is ready and flushed (`[VERIFIED: AppImageKit src/runtime.c@effcebc:914-928]`):

```c
if(arg && strcmp(arg, "appimage-mount") == 0) {
    char real_mount_dir[PATH_MAX];

    if (realpath(mount_dir, real_mount_dir) == real_mount_dir) {
        printf("%s\n", real_mount_dir);
    } else {
        printf("%s\n", mount_dir);
    }

    // stdout is, by default, buffered (unlike stderr), therefore in order to allow other processes to read
    // the path from stdout, we need to flush the buffers now
    // this is a less-invasive alternative to setbuf(stdout, NULL);
    fflush(stdout);

    for (;;) pause();
```

Readiness is real: 15/15 field trials had `<mount>/zcode` stat-able immediately after the line, because the parent has already waited for the fuse daemon's keepalive signal, reaped the intermediate child, and opened the mount directory (`[VERIFIED: AppImageKit src/runtime.c@effcebc:889-906]`; `[VERIFIED: field probe 2026-11-23, 15/15]`). Still validate — the check is cheap and converts a future runtime change into a diagnostic instead of a spawn failure.

**Failure output goes to stdout too.** When `fusefs_main` fails, the runtime calls `notify()` (`[VERIFIED: AppImageKit src/runtime.c@effcebc:872-881]`), and `notify()` prints to `stdout` in both its tty and non-tty arms (`[VERIFIED: AppImageKit src/notify.c@effcebc:12-21]`):

```c
if (isatty(fileno(stdin))){
    /* We were launched from the command line. */
    printf("\n%s\n", title);
    printf("%s\n", body);
}
else
{
    /* We were launched from inside the desktop */
    printf("\n%s\n", title);
    printf("%s\n", body);
```

So on a FUSE failure the wrapper may see an **empty line first**, then `Cannot mount AppImage, please check your FUSE setup.` plus the extract hint. The readiness loop must not accept the first line blindly; it must look for a line that validates as the expected mount path, while capturing every non-path line for the error.

**Recommended algorithm:**

1. Create a private base: `os.MkdirTemp("/tmp", "zcode-acp-mount-")` (or `os.TempDir()` if `/tmp` is unusable), mode 0700. Reject a base containing `\n`, `\r`, or NUL.
2. Spawn `argv = []string{appImagePath, "--appimage-mount"}` through `SpawnChild`, with the allowlisted env and a context that is **not** canceled by the app-server shutdown (`context.WithoutCancel` or a dedicated ctx). `cmd.Cancel` remains the SIGKILL backstop.
3. Start a capped stderr capture (add `SpawnOpts.StderrTap io.Writer` or an equivalent tap; the existing `drainChildStderr` already forwards+cap-drains and is the right place) and a stdout scanner. The cap precedent is `stderrCapDefault = 64 << 10` (`[VERIFIED: internal/supervisor/spawn.go:64-68]`):

```go
stderrCapDefault = 64 << 10
```

The scanner:
   - reads lines with a bounded buffer (a mount path is < PATH_MAX; 8–16 KiB is ample);
   - ignores empty lines;
   - accepts the first line that is absolute, exists, is a directory (not a symlink), lies one component under the resolved private base, and whose basename starts with `.mount_`;
   - on a path-looking line that fails validation, fails immediately naming the missing file/step;
   - keeps draining after acceptance into a capped buffer (the helper normally prints nothing else; this prevents a blocked pipe if it does).
4. On acceptance: validate `<mount>/zcode` (regular file, executable bit) and `<mount>/resources/glm/zcode.cjs` (regular file). Resolve the mount path once (`filepath.EvalSymlinks`) and store it.
5. Timeout (default 15 s): stop the helper, reap, collect stdout/stderr, return an actionable error. Do **not** leave the helper running.
6. If the helper exits before the line: read `Done`/`Exited`, collect output, return `helper exited before the mount line (exit ...)` with the observed text.
7. If the helper exits after the line (or the tree vanishes): the lifecycle is lost; the watcher closes `Lost`.

**FUSE/libfuse2 classification (best-effort, from the observed output):** the legacy runtime's failure strings are exact and stable enough to classify (`[VERIFIED: embedded runtime strings + AppImage docs]`):

- `dlopen(): error loading libfuse.so.2` → libfuse2 missing. Ubuntu 24.04 remediation: `sudo apt install libfuse2t64` (older: `sudo apt install libfuse2`) `[CITED: docs.appimage.org/user-guide/troubleshooting/fuse.html]`.
- `Cannot mount AppImage, please check your FUSE setup.` / `AppImages require FUSE to run.` → FUSE setup (`/dev/fuse`, unprivileged mounts). Same remediation family.
- `Failed to get fs offset`, `This doesn't look like a squashfs image`, `Failed to open AppImage file` → not a valid/readable AppImage; name the path and step.
- `create mount dir error`, `open dir error`, `permission denied` → TMPDIR/permissions/executable-bit issues; name the path and remediation.

Do not hard-code only the libfuse2 string: newer static type-2 runtimes do not require libfuse2 (`[CITED: github.com/AppImage/type2-runtime README]`). Classify the known strings, but always include the raw observed output.

### Pattern 3: Version-ranked, per-candidate AppImage selection

**What:** Replace the two order-coupled probes (`appimage-glob`, `unpacked-glob`) with one AppImage-layout stage that collects candidates, evaluates each one, ranks them by `<ver>`, and yields at most one winner.

**Current defect (verbatim):**

```go
// internal/supervisor/discover.go:134-140 — fixed probe order, first match wins
add(p.probeOverride())
add(p.probeXDGEntry(ctx))
add(p.probeAppImageGlob(ctx, found))
add(p.probeUnpackedGlob())
add(p.probeDpkg(ctx))
...
// internal/supervisor/discover.go:127-132 — later hits never replace the first
add := func(row Probe, inst Install, ok bool) {
	probes = append(probes, row)
	if ok && !found {
		winner, found = inst, true
	}
}
```

The packed probe returns before it even looks at candidates whenever an unpacked tree exists (`[VERIFIED: internal/supervisor/discover.go:405-415]`):

```go
if len(unpacked) > 0 {
	row.Result = joinNotes(notes, "packed AppImage present but an unpacked "+unpackedDirName+
		" exists — extraction skipped (matrix step 3: extract only when a packed AppImage is all that exists)")

	return row, Install{}, false
}
```

And when it does look, it resolves only `packed[0]` after sorting symlinks first, so one dangling symlink aborts the probe (`[VERIFIED: internal/supervisor/discover.go:417-423, 736-752]`):

```go
sortAppImageCandidates(packed)
resolved, err := filepath.EvalSymlinks(packed[0])
if err != nil {
	row.Result = joinNotes(notes, fmt.Sprintf("AppImage %s unresolvable: %v", packed[0], err))

	return row, Install{}, false
}
```

Then it extracts (`[VERIFIED: internal/supervisor/discover.go:434]`):

```go
root, err := p.ExtractAppImage(ctx, resolved, "")
```

Meanwhile the unpacked probe sorts by version and returns the newest tree, with no knowledge of the packed candidates (`[VERIFIED: internal/supervisor/discover.go:480-493]`):

```go
slices.SortFunc(trees, func(a, b string) int { return compareVersions(dirVersion(b), dirVersion(a)) })
for _, tree := range trees {
	inst, ok := bundleInstall(tree, layoutUnpackedAppImage)
	if ok {
		row.Result = joinNotes(notes, "found: "+tree)

		return row, inst, true
	}
}
```

**Real-machine fixture (verified this session):** `~/Applications/ZCode-4.0.0-linux-x64.AppImage` (packed, executable, current), `~/Applications/zcode-3.18.2/squashfs-root` (stale research extract, with bundled builtin), and a **dangling** `~/Applications/ZCode.AppImage -> ~/Applications/ZCode-3.18.2-linux-x64.AppImage` (target deleted). Current discovery selects 3.18.2; the phase must select 4.0.0. `xdg-mime query default x-scheme-handler/zcode` returns `zcode.desktop`, whose user entry Exec is the 4.0.0 AppImage `[VERIFIED: field probe 2026-11-23]`.

**Recommended algorithm:**

```text
candidates = []

for each path in AppDirs() + xdgAppImageSeed:
    if mountDecoy(path): miss("volatile mount"); continue
    if path ends ".AppImage":
        viaSymlink = Lstat(path).Mode()&ModeSymlink != 0
        resolved = EvalSymlinks(path)           # per candidate, failure => miss
        stat resolved; regular; mode&0111 != 0  # else miss("not executable: chmod +x ...")
        version = appImageVersion(resolved)     # "" when the name is not ZCode-<ver>-linux-<arch>
        candidates += {display:path, resolved, layout:"appimage", version, rank:viaSymlink?1:2}
    else if base(path) == "squashfs-root":
        resolved = EvalSymlinks(path)           # per candidate, failure => miss
        inst, ok = bundleInstall(resolved, layoutUnpackedAppImage)
        if !ok: miss("missing zcode or resources/glm/zcode.cjs"); continue
        candidates += {display:path, resolved, layout:"unpacked-appimage", version:dirVersion(resolved), rank:0}

dedupe by resolved path: keep one entry; prefer a symlink display/rank over a regular-file display
sort:
    1. compareVersions(versionB, versionA)   # newest first; "" ranks below any numeric version
    2. rank asc (unpacked 0, symlink-packed 1, regular-packed 2)
    3. strings.Compare(displayA, displayB)
winner = first candidate
```

**Version extraction:**

- Packed: from the **resolved** basename. The documented shape is `ZCode-<ver>-linux-<arch>.AppImage` (`[VERIFIED: docs/protocol/install-discovery.md:33]`). A symlink named `ZCode.AppImage` has no version; resolve first, then parse the target.
- Unpacked: from the **parent directory** name `zcode-<ver>` using the existing `dirVersion` (`[VERIFIED: internal/supervisor/discover.go:756-761]`):

```go
func dirVersion(path string) string {
	name := filepath.Base(filepath.Dir(path))
	ver, _ := strings.CutPrefix(name, "zcode-")

	return ver
}
```

- Comparison: the existing comparator (`[VERIFIED: internal/supervisor/discover.go:767-782]`):

```go
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		av, bv := versionComponent(as, i), versionComponent(bs, i)
		if av != bv {
			if av < bv {
				return -1
			}

			return 1
		}
	}

	return strings.Compare(a, b)
}
```

A candidate whose version cannot be parsed is still usable but ranks below every versioned candidate, with an explicit "version unknown" note. Do not silently drop it: a user who renamed the file should still be able to run it, and determinism is preserved by the `""` ordering plus lexicographic tie-break.

**Tie-break (D-08), applied only when versions compare equal:**

1. Unpacked spawnable tree (rank 0) — no mount needed.
2. Packed candidate reached through a symlink (rank 1) — the user-facing entry.
3. Packed regular file (rank 2).
4. `strings.Compare(displayA, displayB)`.

Dedupe note: a symlink and its target are the same backend. Keep one candidate keyed by the resolved path and prefer the symlink representation, otherwise D-08's "symlink before versioned file" would produce two rows for one backend.

**Diagnostics (D-09, DISC-02, DISC-03, DISC-04):**

- `appimage-rank` row: `selected: <layout> version <ver> resolved <resolved path>` or `no AppImage-layout candidate`.
- One `appimage-candidate` row per candidate, in final rank order: `hit: ...` or `miss: <display>: <reason>`.
- Include the display path (symlink) and resolved path in the hit row; include the version and the failing step in every miss row.
- The existing `emptyResultError` already turns every row into an actionable line (`[VERIFIED: internal/supervisor/discover.go:180-187]`).
- The fixed probe-kind order becomes: `override`, `xdg-entry`, `appimage-rank` + candidate rows, `dpkg`, `opt-path`, `rpm`, `builtin-runtime`, `builtin-bundled`. Update the order assertion in `discover_test.go` accordingly. The current fixed names and capacity are (`[VERIFIED: internal/supervisor/discover_test.go:32-37; internal/supervisor/discover.go:53-54]`):

```go
"override", "xdg-entry", "appimage-glob", "unpacked-glob",
"dpkg", "opt-path", "rpm", "builtin-runtime", "builtin-bundled",
...
probeCount = 9
```

**Scope boundary (D-10):** only AppImage-layout candidates are version-ranked. A desktop entry that resolves a **package** tree (root not named `squashfs-root`) still wins at order step 2 and outranks the AppImage stage, exactly as today. This is deliberate: D-10 says deb/rpm/opt paths and their order stay as they are.

**XDG seeding:** if the entry's Exec target resolves to a packed `.AppImage` or to a `squashfs-root` tree, feed that path as a candidate into the ranked stage instead of returning a hard miss. Otherwise an entry pointing at an AppImage outside `~/Applications` (or at an older unpacked tree) would still be able to pin an older backend, contradicting DISC-01. If the target resolves to a package tree, return it as the step-2 winner (unchanged).

**No extraction:** delete `Probers.ExtractAppImage`, `defaultExtractAppImage`, and the `--appimage-extract` execution path. Extraction remains a **manual inspection** route in the docs, never a wrapper behavior.

### Pattern 4: Staged teardown — app-server, then helper, then prove the mount is gone

**What:** The existing supervisor already implements the app-server half (`[VERIFIED: internal/supervisor/supervise.go:168-193]`):

```go
func (s *Supervisor) shutdown(child *Child, policy Policy) {
	_ = child.Stdin.Close() // stage 1: stdin end (the child drains ~100 ms by design, dialect §1 :30)

	select { // stage 2: wait for exit within the grace window
	case <-child.Done:
	case <-time.After(policy.Grace):
	}

	// Stage 3: the group SIGKILL backstop — best-effort, logged, never
	// propagated (the shutdown contract).
	if err := killGroup(child.PgID); err != nil {
		s.log.Warn("supervisor: staged shutdown group kill", "pgid", child.PgID, "err", err)
	}

	select { // stage 4: the reap completes (never Wait twice — the monitor owns it)
	case <-child.Done:
	case <-time.After(shutdownReapDeadline):
		s.log.Warn("supervisor: staged shutdown reap did not complete", "pgid", child.PgID)
	}
}
```

The helper half must be **different in two ways**:

1. **Do not close stdin and expect an exit.** The runtime does not read stdin; it `pause()`s (`[VERIFIED: AppImageKit src/runtime.c@effcebc:928]`). Send `SIGTERM` first, wait for the grace window, then group-SIGKILL.
2. **The group kill does not reach the fuse daemon.** The runtime forks and the fuse process daemonizes into its own session; the parent `waitpid`s the intermediate child. Field observation during a live mount: the helper was `pgid == pid` with no children visible, and a second daemonized process with the same argv existed in its own session/pgid. Teardown of the mount is driven by the **keepalive pipe**, not by signals to the daemon (`[VERIFIED: AppImageKit src/runtime.c@effcebc:137-161]`):

```c
while (1) {
    /* Write until we block, on broken pipe, exit */
    res = write (keepalive_pipe[1], c, sizeof (c));
    if (res == -1) {
        kill (fuse_pid, SIGTERM);
        break;
    }
}
```

When the helper dies, its read end of the pipe closes; the daemon's next write fails, and it SIGTERMs itself. Field trials: `SIGINT` → helper exit 130, unmounted; `SIGTERM` → exit 143, unmounted; `SIGKILL` → exit 137, unmounted. The mountpoint directory disappeared within 50 ms in 5/5 trials for INT and TERM; immediate inspection after `wait` occasionally still showed the empty directory, so **verify the mount, not the directory**.

**Recommended `Mount.Stop` sequence:**

```text
1. If already stopped, return.
2. SIGTERM to -pgid (helper's own group; do not touch the app-server's group).
3. Wait for child.Done up to StopGrace (default 5s).
4. If still alive: killGroup(pgid) (SIGKILL), wait for Done up to ReapDeadline.
5. Poll mountGone(root):
      stat(<root>/zcode) == nil           -> still mounted, keep polling
      ENOENT / ENOTDIR                    -> gone
      ENOTCONN                            -> stale mount entry, keep polling / fail at deadline
      other                               -> keep polling, record the error
   deadline 5s, interval 50ms.
6. RemoveAll(privateBase) — safe: it is our 0700 dir and contains only our mount dirs.
7. If the mount is still present at the deadline, return an error naming the mount and remediation
   (unmount manually; the helper is reaped, so fusermount may succeed now).
```

**Why not a `fusermount` fallback while the helper lives:** the runtime duplicates an fd to the mount directory (`dup2(dir_fd, 1023)`) and holds it for its lifetime (`[VERIFIED: AppImageKit src/runtime.c@effcebc:895-906]`). Field tests on a live mount:

```text
$ fusermount3 -u /tmp/.mount_ZCode-RMYIcv
fusermount3: failed to unmount /tmp/.mount_ZCode-RMYIcv: Operation not permitted
$ fusermount -u /tmp/.mount_ZCode-mDK4R9
fusermount: failed to unmount /tmp/.mount_ZCode-mDK4R9: Device or resource busy
```

`fusermount3` cannot unmount this libfuse2 mount; `fusermount` is `EBUSY` because of the held fd. The only reliable teardown is killing the helper. Do not build a cleanup strategy that depends on external unmount while the helper is alive.

**Zero stale mounts is a verification, not an assumption:** the daemon is separate, so `kill(-pgid, 0) == ESRCH` proves only that the helper's group is gone. The mount check is the separate stage 5. Tests must assert both: helper reaped **and** `<mount>/zcode` no longer stats (or the stat fails with ENOENT).

### Pattern 5: Mid-session mount loss and the spawn guard

**What:** `run` currently selects only on the ACP client and the app-server child (`[VERIFIED: cmd/zcode-acp/main.go:707-712]`):

```go
select {
case <-agentConn.Done():
	slog.Info("zcode-acp: ACP client disconnected")
case <-child.Done:
	childLost = true
}
```

For a mount-backed run, add `<-mount.Lost()`. `Lost` closes once when either:

- the helper's `Done` fires (any exit, including exit 0), or
- the mount tree stops being accessible (`os.Stat(<root>/zcode)` returns `ENOENT`, `ENOTDIR`, or `ENOTCONN`), sampled by a watcher (default every 1 s; tests 25 ms).

**Never respawn against a vanished root.** The supervisor's restart loop calls the spawn seam again on a child crash (`[VERIFIED: internal/supervisor/supervise.go:113-166]`):

```go
child, spawnErr := s.spawn(ctx)
if spawnErr != nil {
	return fmt.Errorf("zcode: supervisor: spawn app-server: %w", spawnErr)
}
...
crashes++
if crashes > policy.CrashLimit {
	...
	return fmt.Errorf("zcode: supervisor: app-server crash loop: %d crashes within %v: %w", ...)
}
```

A spawn error is already fail-loud and immediate. Wrap the production `SpawnFunc` with a guard:

```go
// design sketch
sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
	if mount != nil {
		if err := mount.Guard(); err != nil {
			return nil, err // ErrMountLost; Start returns it without restarting
		}
	}
	return supervisor.SpawnChild(spawnCtx, supervisor.SpawnArgv(inst),
		supervisor.ChildEnv(inst, os.Environ()), supervisor.SpawnOpts{})
}, supervisor.DefaultPolicy())
```

`Mount.Guard` returns a wrapped `ErrMountLost` naming the AppImage and the mount path. The first app-server spawn is also guarded (defense in depth against a mount that vanished between validation and spawn).

**Cause precedence at exit:** if the app-server and the mount are both gone, report the mount cause — it explains the app-server exit. Only report `errChildLost` (`"app-server exited before the ACP client disconnected"`, `[VERIFIED: cmd/zcode-acp/main.go:83]`) when the mount is still accessible.

### Pattern 6: Override routing and the `Install` shape

**What:** `--app-server` / `ZCODE_ACP_APP_SERVER` must accept three shapes without changing precedence: directory root, app-server binary, packed `.AppImage`. Today any non-directory is treated as the binary itself (`[VERIFIED: cmd/zcode-acp/main.go:780-801]`):

```go
func overrideInstall(path string) supervisor.Install {
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		if inst, ok := supervisor.ExplicitRootInstall(path); ok {
			return inst
		}

		return supervisor.Install{
			Root:   path,
			Binary: filepath.Join(path, "zcode"),
			Layout: "explicit",
		}
	}

	root := filepath.Dir(path)

	return supervisor.Install{
		Root:       root,
		Binary:     path,
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     "explicit",
	}
}
```

With a packed AppImage this produces `argv = [<path>.AppImage, <dir>/resources/glm/zcode.cjs, app-server, --stdio]` and `ELECTRON_RUN_AS_NODE=1`. The AppImage runtime sees no `--appimage-mount` flag, mounts itself, and executes `AppRun` (the GUI) — the exact wrong behavior. The override must route the path through the mount.

**Extend `Install` (recommended):**

```go
// internal/supervisor/discover.go:69-75 (current, verbatim)
type Install struct {
	Root        string // dir containing the Electron binary
	Binary      string // <root>/zcode
	Entrypoint  string // <root>/resources/glm/zcode.cjs (invariant relative path)
	BuiltinFile string // ZCODE_BUILTIN_PROVIDER_CONFIG_FILE value
	Layout      string // "unpacked-appimage" | "appimage" | "deb" | "rpm" | "explicit"
}
```

Add:

```go
AppImage string // resolved packed AppImage path; non-empty => this backend is mount-first
Version  string // <ver> parsed from the AppImage filename or the unpacked dir name
```

Semantics:

- Packed candidate before mount: `Root=""`, `Binary=""`, `Entrypoint=""`, `AppImage=<resolved>`, `Layout="appimage"`, `Version="4.0.0"`.
- Mounted install: `Root=<mount>`, `Binary=<mount>/zcode`, `Entrypoint=<mount>/resources/glm/zcode.cjs`, `AppImage` preserved, `Layout="appimage"`, `Version` preserved.
- `func (i Install) RequiresMount() bool { return i.AppImage != "" && i.Root == "" }`.
- `SpawnArgv`/`ChildEnv` stay byte-identical (`[VERIFIED: internal/supervisor/spawn.go:128-130, 137-169]`); they must only ever see the **mounted** install. A pre-mount packed Install reaching `SpawnArgv` is a programming error — add an empty-`argv[0]` rejection in `SpawnChild` as a cheap guard.

**Override mapping:**

1. If `os.Stat(path)` succeeds and `IsDir()` → existing `ExplicitRootInstall` path (unchanged).
2. Else if the literal path ends in `.AppImage`, or `EvalSymlinks(path)` resolves to a path ending in `.AppImage` → `ExplicitAppImageInstall(path)`:
   - `AppImage = path` (or the resolved target when resolution succeeds);
   - `Layout = layoutAppImage`; `Version = appImageVersion(resolved)` best-effort;
   - `Root/Binary/Entrypoint` empty;
   - return `ok=true` even when resolution fails, so the mount step fails loud naming the dangling/missing path (never silently fall through to discovery).
3. Else → existing binary shape.

Consolidate this function in `supervisor` (e.g., `supervisor.OverrideInstall(path)`) and have both `cmd/zcode-acp/main.go` and `test/realdesktop/apicycle_test.go` call it. The real-tier test currently carries a **duplicate mirror** of this mapping (`[VERIFIED: test/realdesktop/apicycle_test.go:370-404]`); after discovery starts returning packed installs, that mirror is a live inconsistency.

**`desktopVersion` must change.** The PAT client-version seam digs the version out of `Binary`, `Root`, `Entrypoint` (`[VERIFIED: cmd/zcode-acp/main.go:508-526]`):

```go
func desktopVersion(inst supervisor.Install) string {
	paths := []string{inst.Binary, inst.Root, inst.Entrypoint}

	for _, path := range paths {
		segs := strings.Split(path, string(filepath.Separator))

		for i := len(segs) - 1; i >= 0; i-- {
			if v := desktopVersionPattern.FindString(segs[i]); v != "" {
				return v
			}
		}
	}

	return ""
}
```

The fallback is a discrete constant (`[VERIFIED: cmd/zcode-acp/main.go:500]`):

```go
const patDesktopVersionFallback = "0.0.0"
```

After mount, all three paths are under `/tmp/zcode-acp-mount-.../.mount_ZCode-XXXXXX` and contain **no version**. Without a fix, the PAT mint's `clientVersion` silently falls back to `"0.0.0"` — a wire-visible regression. Prefer `inst.Version`, then `inst.AppImage`, then the existing path scan:

```go
// design sketch
if inst.Version != "" {
	return inst.Version
}
paths := []string{inst.AppImage, inst.Binary, inst.Root, inst.Entrypoint}
...
```

**Precedence is unchanged** (`[VERIFIED: cmd/zcode-acp/main.go:405-410]`):

```go
cfg := config{
	appServer: firstNonEmpty(appServerFlag, os.Getenv(envAppServer)),
	apiKey:    firstNonEmpty(apiKeyFlag, os.Getenv(envAPIKey)),
}
```

Update both flag help strings (`main.go:94-95` and the omp `ompFlagSet` at `main.go:233-234`) to say "install root, app-server binary, or packed `.AppImage`". The README config table says the same.

### Error and exit-code conventions

The repo's convention is two defined codes: `2` for an invalid flag/env combination (`[VERIFIED: cmd/zcode-acp/main.go:69-73]`):

```go
	// exitInvalidConfig is the usage exit code: the flags/env combination is
	// malformed (distinct from the runtime-failure exit 1 — the fake
	// binary's ExitUsage precedent: invalid input exits non-zero, naming
	// the contract).
	exitInvalidConfig = 2
```

and `1` for any `run` failure (`[VERIFIED: cmd/zcode-acp/main.go:131-135]`):

```go
	if err := run(cfg); err != nil {
		slog.Error("zcode-acp: exiting after failure", "err", err)

		os.Exit(1)
	}
```

Mount failures are **runtime** failures: helper spawn/exec failure, early exit, readiness timeout, invalid mount tree, missing FUSE/libfuse2, mount lost mid-session, and a mount still present after teardown all return an error from `run` → exit 1. A clean signal-initiated shutdown returns nil → exit 0. Invalid `--model`/`--reasoning-level` combinations remain exit 2 and are untouched.

Every mount error should carry: the **AppImage path**, the **failing step** (`spawn helper`, `read mount line`, `validate tree`, `mount lost`, `teardown`), the **observed output** (quoted/capped), and a **remediation** (install FUSE/libfuse2, `chmod +x`, verify the AppImage, unmount after reaping). This is the D-03/MNT-04 contract.

### Anti-Patterns to Avoid

- **Mounting during discovery.** Discovery is documented as a pure read (`[VERIFIED: docs/protocol/install-discovery.md:110, 120]`); mounting candidates would create FUSE side effects, break idempotency, and make "run discovery twice" unsafe.
- **Extracting "just in case".** D-01 forbids it; `APPIMAGE_EXTRACT_AND_RUN` can reintroduce it silently if the helper env is inherited.
- **Putting the helper in `Supervisor.Start`.** Its restart loop would remount on app-server crashes and could remount against a vanished root.
- **Closing the helper's stdin to stop it.** The runtime `pause()`s; stdin EOF does nothing.
- **Reading `packed[0]` or aborting on the first candidate error.** The real fixture's dangling symlink is exactly this case.
- **Using `strings.Fields` on the mount line.** Spaces are legal in `TMPDIR` and in the mount name (field-verified).
- **Checking "mount gone" by directory absence.** The mountpoint directory can outlive the unmount by tens of milliseconds; check `<mount>/zcode` accessibility/mount state.
- **Logging raw helper output without bounds/quotes.** Cap it and use `%q`; a hostile/broken helper must not inject log lines.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mounting a squashfs AppImage | A Go squashfs reader, cgo libfuse binding, or `mount` shell-out | `<AppImage> --appimage-mount` | The runtime already implements the exact FUSE mount, offset handling, and unprivileged setup; anything else is a second, unreviewed implementation |
| Child process groups, reaping, stderr drain | A new `os/exec` wrapper in `mount.go` | Existing `SpawnChild` + `Child` | Process-group kill, `WaitDelay`, capped stderr drain, and exactly-one `Wait` are already solved and tested (`spawn.go:236-329`) |
| App-server restart policy | A second restart loop for mount-backed runs | Existing `Supervisor` with a mount guard | The crash window/backoff/reset semantics are already locked by tests; only the pre-spawn guard is new |
| Version comparison | Semver library or string comparison | Existing `compareVersions`/`versionComponent` | D-06 locks existing semantics; `0.0.0-dev` and `<build>` suffix behavior are already defined |
| Symlink resolution | Manual `readlink` recursion | `filepath.EvalSymlinks` per candidate | Handles chained/relative links and reports the exact failure |
| Mount presence detection | `mount` + `grep`, `/etc/mtab`, `fusermount -u` | `os.Stat(<mount>/zcode)` + `ENOENT`/`ENOTCONN` classification | No shell injection, no external dependency, works for the fake runtime; authoritative enough for "the tree is usable" |
| Env sanitization | A deny-list of bad variables | Allowlist env by construction (PATH/HOME/TMPDIR) | Same posture as `ChildEnv`; deny-lists miss `APPIMAGE_EXTRACT_AND_RUN` variants |
| Capture/output capping | Unbounded `bytes.Buffer` | Existing capped stderr pump + a capped tap | A failing helper can emit unbounded text; the 64 KiB precedent already exists (`stderrCapDefault`) |
| Signal-driven shutdown | Ad-hoc goroutines per child | `signal.NotifyContext` + the existing staged shutdown | One cancellation path, deterministic teardown order |

**Key insight:** the hard parts here are process lifecycle and FUSE teardown, not AppImage parsing. Reuse the supervisor's proven primitives and the runtime's own mount implementation; add only the selection policy and the readiness/teardown glue.

## Runtime State Inventory

> This is not a rename phase, but it does change acquisition behavior. The categories are answered explicitly so the planner can distinguish "researched and found nothing" from "not checked."

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no persisted string, key, collection, or user id is renamed | None |
| Live service config | None — the desktop entry, mimeapps rows, and runtime provider tree are read-only inputs | None; do not "fix" xdg state (`install-discovery.md:66`) |
| OS-registered state | None — no systemd unit, scheduler task, launchd plist, or pm2 entry | None |
| Secrets/env vars | No secret renamed. `ZCODE_ACP_APP_SERVER` gains a new accepted value shape (packed `.AppImage`); existing directory/binary values keep working | Code edit only; document the expanded meaning |
| Build artifacts / installed packages | Users may have old `squashfs-root`/`zcode-<ver>` extracts created by the previous extraction behavior (this machine has one at `~/Applications/zcode-3.18.2/squashfs-root`) | **Never delete user data.** Rank the stale extract below the newer packed AppImage and leave it on disk. Document that the wrapper stops creating new copies |
| Mount state | Unmanaged mounts may exist from other AppImages (multiple `/tmp/.mount_Orca.*` mounts were present during the probe session). The wrapper's managed mount lives under its private `TMPDIR` | Reject unmanaged mounts in discovery; verify the managed mount is gone on every teardown |
| In-flight sessions | Killing the app-server loses live sessions (existing supervisor blast radius) | Unchanged; the mount does not add a new persistence surface |

**Canonical question:** after every code change, what runtime state still carries the old behavior? Answer: old extracted trees on disk (intentionally left, version-ranked out) and possibly empty `.mount_*` directories from failed runtime attempts. Neither is a spawn root; neither should be deleted automatically.

## Common Pitfalls

### Pitfall 1: Treating the first stdout line as the mount path
**What goes wrong:** On a mount failure the runtime's `notify()` prints an empty line and the failure text to **stdout**; a naive `Scanner.Scan()` + `line` takes `""` or `"Cannot mount AppImage..."` as the root.
**Why it happens:** The implementation prints both success and failure through stdout; only `perror` paths use stderr.
**How to avoid:** Scan until a line validates as the expected mount path; capture every non-path line for the error; never `TrimSpace` away meaningful spaces in a path.
**Warning signs:** A root of `""`, a spawn error naming `Cannot mount AppImage` as a binary, or a readiness timeout on a host whose mount actually succeeded.

### Pitfall 2: Expecting the helper to exit on stdin EOF
**What goes wrong:** The wrapper closes stdin during staged shutdown, waits, and then hangs until the grace timeout because the helper is in `for (;;) pause()`.
**Why it happens:** `--appimage-mount` is not an app-server; it does not read stdin.
**How to avoid:** `Mount.Stop` sends SIGTERM (then SIGKILL) and waits on `Done`. Do not reuse the app-server's "stdin end first" stage for the helper.
**Warning signs:** Every shutdown takes the full grace window; the helper is still alive after stdin close.

### Pitfall 3: Killing only the helper's process group and assuming the mount died
**What goes wrong:** `kill(-pgid, 0)` reports ESRCH, but the fuse daemon is daemonized into its own session and may still hold the mount; the next launch sees a stale `/tmp/.mount_*`.
**Why it happens:** The runtime forks and the fuse process daemonizes; the group is not the mount's lifetime.
**How to avoid:** Verify `<mount>/zcode` is gone (or `ENOTCONN`) after reaping; rely on the keepalive pipe to make the daemon exit; never equate group death with mount death.
**Warning signs:** A mount table row with a dead FUSE daemon; `stat` on the mount returning `ENOTCONN`.

### Pitfall 4: Checking for the mountpoint directory, not the mount
**What goes wrong:** A test asserts `!os.IsNotExist(stat(mountRoot))` and flakes: the directory can persist briefly after a clean unmount, and its removal is not part of the unmount contract, while the mount is already gone.
**Why it happens:** The mountpoint is created by `mkdtemp`; unmount and directory removal are separate, asynchronous steps.
**How to avoid:** Check `<mount>/zcode`: present → still mounted; `ENOENT`/`ENOTDIR` → gone; `ENOTCONN` → stale. Poll; never one racy check.
**Warning signs:** Flaky teardown tests; a passing manual `ls` of an empty `.mount_*` directory.

### Pitfall 5: Inheriting `APPIMAGE_EXTRACT_AND_RUN` into the helper
**What goes wrong:** The runtime extracts an ~850 MB copy into `TMPDIR` and launches `AppRun` (the GUI) instead of mounting; the wrapper waits for a mount line that never comes, and MNT-06 is violated.
**Why it happens:** The env check is before the mount branch (`runtime.c:688`), and `os.Environ()` is the easy default.
**How to avoid:** Build the helper env by allowlist (PATH/HOME/TMPDIR only). Explicitly exclude `APPIMAGE_EXTRACT_AND_RUN`, `TARGET_APPIMAGE`, `APPDIR`, `APPIMAGE`, `NO_CLEANUP`, `VERBOSE`, `LD_PRELOAD`, `LD_LIBRARY_PATH`.
**Warning signs:** A GUI window appears; an `appimage_extracted_*` directory under TMPDIR; readiness timeout with no output.

### Pitfall 6: Parsing the mount line with `strings.Fields`
**What goes wrong:** `TMPDIR="/tmp/tmp dir probe"` yields `/tmp/tmp dir probe/.mount_ZCode mEObyM`; fields parsing returns `/tmp/tmp`, which fails validation or, worse, resolves an unrelated path.
**Why it happens:** Unix paths may contain spaces; the mount name derives from `basename(argv0)` and can contain them too.
**How to avoid:** Trim only `\r\n`; treat the entire line as the path; validate with `filepath.IsAbs` + `filepath.Rel` from the private base.
**Warning signs:** A mount path truncated at the first space; a valid mount reported as invalid on a custom TMPDIR.

### Pitfall 7: Rejecting the wrapper's own managed mount as a volatile decoy
**What goes wrong:** The existing `mountDecoy` rejects any `/tmp/.mount_*` path. If discovery or the mount validator applies that rule to the runtime's output, the managed mount is rejected.
**Why it happens:** The decoy rule was written for unmanaged mounts found by discovery (`[VERIFIED: internal/supervisor/discover.go:193-199]`).
**How to avoid:** Apply `mountDecoy` to **user-provided/discovered** paths only. Accept the runtime's line only when it lies under the private base we created; the private base itself must not match the decoy prefix.
**Warning signs:** A valid mount reported as "rejected volatile mount".

### Pitfall 8: Aborting the candidate scan on the first bad candidate
**What goes wrong:** The real fixture's dangling `ZCode.AppImage` is sorted first; `EvalSymlinks` fails; the probe returns a miss and never tries `ZCode-4.0.0-linux-x64.AppImage`.
**Why it happens:** `packed[0]` is the only candidate attempted (`discover.go:417-423`); there is no per-candidate loop.
**How to avoid:** Evaluate every candidate into a hit/miss record, then rank the hits. A dangling symlink is a miss row, not an abort.
**Warning signs:** `appimage-glob: AppImage ... unresolvable` while a working AppImage sits in the same directory.

### Pitfall 9: Suppressing the packed probe when an unpacked tree exists
**What goes wrong:** The stale 3.18.2 extract beats the current 4.0.0 AppImage.
**Why it happens:** The early return at `discover.go:405-415` treats "unpacked exists" as "no need to look at packed". It was written for the extraction policy, not for version selection.
**How to avoid:** Remove the early return; collect both candidate sets in one stage and rank by version.
**Warning signs:** Discovery selects an older version than a packed AppImage in the same directory.

### Pitfall 10: Extracting during discovery
**What goes wrong:** Discovery mutates the filesystem, takes seconds/minutes, creates an 850 MB copy, and breaks the documented pure-read/idempotency guarantee.
**Why it happens:** The current `ExtractAppImage` seam is wired into the packed probe.
**How to avoid:** Delete the seam and the default extractor; discovery returns a packed Install; the mount happens only after selection.
**Warning signs:** A `squashfs-root` directory appearing during discovery; discovery slower than a few hundred ms.

### Pitfall 11: Forgetting the PAT client version after mounting
**What goes wrong:** `desktopVersion` scans `Binary`/`Root`/`Entrypoint`; after mounting all three are under `/tmp/.../.mount_ZCode-XXXXXX` with no version, so the PAT mint silently uses `"0.0.0"`.
**Why it happens:** The version used to be in the unpacked parent dir; the mount path erases it.
**How to avoid:** Set `Install.Version` during discovery and prefer it (then `AppImage`) in `desktopVersion`.
**Warning signs:** The mint chain's `clientVersion` diagnostic changes to `0.0.0` only on mount-backed runs.

### Pitfall 12: Treating a `.AppImage` override as the app-server binary
**What goes wrong:** `--app-server ~/Applications/ZCode-4.0.0-linux-x64.AppImage` runs the AppImage's `AppRun` (GUI) with `ELECTRON_RUN_AS_NODE=1`, never mounts, and never serves stdio.
**Why it happens:** `overrideInstall` only distinguishes directory vs regular file; every regular file is "the binary".
**How to avoid:** Detect `.AppImage` on the literal path and on the resolved target; route to the packed Install/mount path; keep the binary fallback for non-AppImage files.
**Warning signs:** A GUI window, no ACP initialize answer, or `spawn <path>.AppImage` succeeding while the client times out.

### Pitfall 13: Letting the supervisor restart against a vanished mount
**What goes wrong:** The app-server exits because the mount disappeared; the crash window restarts it against a dead root, burning attempts and possibly spawning a zombie against `ENOTCONN`.
**Why it happens:** `Supervisor.Start` restarts on `child.Exited` without a root-validity check.
**How to avoid:** Guard every spawn with `mount.Guard()`; return `ErrMountLost` so `Start` fails loud immediately. Also select on `mount.Lost()` to end the session proactively.
**Warning signs:** Crash-loop errors for a session whose mount was unmounted externally; repeated spawn attempts with `ENOENT`.

### Pitfall 14: No OS signal handling in the wrapper
**What goes wrong:** SIGINT/SIGTERM kills the wrapper immediately; because children are in their own process groups, the terminal's signal does not reach them. The app-server and mount helper are orphaned, leaving a live mount.
**Why it happens:** `run` uses `context.Background()` (`[VERIFIED: cmd/zcode-acp/main.go:651-653]`) and there is no `signal.Notify` anywhere in `cmd/` or `internal/` (grep: zero hits).
**How to avoid:** `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)`; include `<-ctx.Done()` in the run select; drive the existing staged shutdown; stop the mount last. Optionally add Linux `Pdeathsig` hardening via a build-tagged `SysProcAttr` helper for SIGKILL/crash cases.
**Warning signs:** `pgrep -f zcode.cjs` still finds the app-server after Ctrl-C; `mount | grep .mount_ZCode` after the wrapper exits.

### Pitfall 15: Bundled builtin resolved too early for a packed install
**What goes wrong:** A fresh machine has no `~/.zcode/v2/runtime/provider` cache; for an unpacked tree the bundled `<root>/resources/config/provider/zcode-builtin.json` fills `BuiltinFile`, but for a packed install that file is only readable **inside the mount**. `BuiltinFile` stays empty, `ChildEnv` passes only the personal path, and `resolveNodeProviderRuntimePaths` kills the worker — the QA-02 field failure.
**Why it happens:** `probeBuiltinBundled` runs before any mount and cannot read a packed AppImage.
**How to avoid:** For a packed winner, record the bundled probe as "deferred until mount"; after validation, if `BuiltinFile == ""`, stat `<mount>/resources/config/provider/zcode-builtin.json` and fill it. The relative path is a discrete constant (`[VERIFIED: internal/supervisor/discover.go:45-49]`):

```go
bundledProviderRel = "resources/config/provider"
```

The revision hash then uses the mount path, which is consistent because the app-server is given the same env path.
**Warning signs:** A packed run on a fresh machine dying before serving; `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` absent from the child env.

### Pitfall 16: Assuming libfuse2 is the only FUSE failure
**What goes wrong:** Diagnostics say "install libfuse2" for a missing `/dev/fuse` or a host that forbids unprivileged mounts.
**Why it happens:** The legacy runtime's first failure is often the `dlopen` string, but `fusefs_main` can fail after the library loads.
**How to avoid:** Classify the known strings (`libfuse`, `Cannot mount AppImage`, `FUSE setup`, `fs offset`) but include the raw observed output and both remediation families (FUSE setup vs libfuse2 package). Newer static runtimes do not need libfuse2.
**Warning signs:** A user installs libfuse2 and still fails because `/dev/fuse` is absent.

### Pitfall 17: Unbounded or unsanitized helper output
**What goes wrong:** A broken helper streams text; the wrapper's error/log grows without bound, or newlines forge log entries.
**Why it happens:** The stderr pump already cap-drains, but a new stdout scanner/capture can forget the cap; error strings can embed raw output.
**How to avoid:** Cap both streams (reuse `stderrCapDefault`/`stderrChunkSize` semantics), quote observed lines with `%q`, and log full text only at Debug.
**Warning signs:** Huge error messages; multi-line log entries that look like separate events.

### Pitfall 18: Assuming `fusermount3` can clean up a fuse2 mount
**What goes wrong:** A fallback tries `fusermount3 -u` and gets `Operation not permitted`; `fusermount -u` gets `EBUSY` while the helper lives.
**Why it happens:** FUSE3 tooling does not own the libfuse2 mount; the helper holds fd 1023.
**How to avoid:** Do not make external unmount part of the normal path. Kill the helper, poll the mount, and only mention manual unmount as remediation after the helper is reaped.
**Warning signs:** Teardown that "sometimes" leaves mounts and logs a fusermount error.

## Code Examples

### Existing spawn contract (verified — must not change)

```go
// internal/supervisor/spawn.go:128-130 (verbatim)
func SpawnArgv(inst Install) []string {
	return []string{inst.Binary, inst.Entrypoint, appServerSubcommand, stdioFlag}
}
```

```go
// internal/supervisor/spawn.go:137-169 (verbatim excerpts)
func ChildEnv(inst Install, parentEnv []string) []string {
	...
	env = append(env, childRunAsNodeVar+"="+childRunAsNodeValue)

	if inst.BuiltinFile != "" {
		env = append(env, childBuiltinVar+"="+inst.BuiltinFile)
	}

	if home, ok := parent["HOME"]; ok && home != "" {
		env = append(env, childPersonalVar+"="+filepath.Join(home, ".zcode", "v2", "provider_config.json"))
	}

	return env
}
```

The contract facts are `ELECTRON_RUN_AS_NODE=1`, `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE`, and `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE` (`[VERIFIED: internal/supervisor/spawn.go:28-45]`). Every mount-backed run must spawn the app-server with exactly this argv/env once `Root` is the mount.

### Design sketch: readiness + validation (`internal/supervisor/mount.go`)

```go
// Design sketch — new code; values are recommendations, not verified constants.
func PrepareMount(ctx context.Context, inst Install, opts MountOpts) (*Mount, error) {
	opts.applyDefaults()

	base, err := os.MkdirTemp("/tmp", "zcode-acp-mount-")
	if err != nil {
		return nil, fmt.Errorf("zcode: mount %s: create private TMPDIR: %w", inst.AppImage, err)
	}

	m := &Mount{appImage: inst.AppImage, version: inst.Version, base: base,
		lost: make(chan struct{}), opts: opts}

	// The helper must not be killed by the app-server's shutdown context.
	helperCtx, helperCancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = helperCancel
	child, err := SpawnChild(helperCtx,
		[]string{inst.AppImage, "--appimage-mount"},
		helperEnv(base),
		SpawnOpts{StderrCap: stderrCapDefault, WaitDelay: waitDelayDefault})
	if err != nil {
		helperCancel()
		_ = os.RemoveAll(base)

		return nil, fmt.Errorf("zcode: mount %s: spawn the --appimage-mount helper: %w", inst.AppImage, err)
	}

	line, observed, err := readMountLine(ctx, child, opts.ReadyTimeout)
	if err != nil {
		m.child = child
		_ = m.Stop() // SIGTERM → grace → SIGKILL → reap → remove base

		return nil, fmt.Errorf("zcode: mount %s: %w%s", inst.AppImage, err, observedSuffix(observed))
	}

	root, err := validateMountTree(base, line)
	if err != nil {
		m.child = child
		_ = m.Stop()

		return nil, fmt.Errorf("zcode: mount %s: %w", inst.AppImage, err)
	}

	m.child, m.root = child, root

	if inst.BuiltinFile == "" {
		m.install.BuiltinFile = bundledBuiltin(root) // <root>/resources/config/provider/zcode-builtin.json
	}

	go m.watch(helperCtx)

	return m, nil
}
```

The readiness reader must keep draining after the accepted line. A bounded `bufio.Scanner` (8–16 KiB) plus a goroutine/select on `ctx.Done()` is sufficient; the mount path is at most `PATH_MAX`. Do not use `Scanner.Text()` with default 64 KiB limits on a hostile helper without a cap.

### Design sketch: candidate ranking (`discover.go`)

```go
// Design sketch — new code inside the package, reusing compareVersions/dirVersion.
type appCandidate struct {
	display  string // path as scanned (symlink kept for diagnostics)
	resolved string // EvalSymlinks result (the executable target)
	layout   string // layoutAppImage | layoutUnpackedAppImage
	version  string // "" when unparseable
	rank     int    // 0 unpacked, 1 packed-via-symlink, 2 packed regular
	install  Install
	miss     string
}

sort.SliceStable(cands, func(i, j int) bool {
	if c := compareVersions(cands[i].version, cands[j].version); c != 0 {
		return c > 0 // newest first; "" sorts below any numeric version
	}

	if cands[i].rank != cands[j].rank {
		return cands[i].rank < cands[j].rank
	}

	return strings.Compare(cands[i].display, cands[j].display) < 0
})
```

Per-candidate evaluation must never return early. A candidate that fails any step gets `miss` and stays in the diagnostics; the loop continues.

### Design sketch: staged stop

```go
// Design sketch — idempotent; safe to call from a defer and the normal path.
func (m *Mount) Stop() error {
	m.stopOnce.Do(func() {
		if m.child != nil {
			_ = syscall.Kill(-m.child.PgID, syscall.SIGTERM)
			select {
			case <-m.child.Done:
			case <-time.After(m.opts.StopGrace):
				_ = killGroup(m.child.PgID)
			}
			select {
			case <-m.child.Done:
			case <-time.After(m.opts.ReapDeadline):
			}
		}

		if m.root != "" {
			m.mountErr = waitMountGone(m.root, m.opts) // poll <mount>/zcode: ENOENT/ENOTDIR => gone
		}
		if m.cancel != nil {
			m.cancel()
		}
		_ = os.RemoveAll(m.base)
	})

	return m.mountErr
}
```

### CI-safe scripted fake AppImage runtime (design sketch)

The script is invoked by the **production** `PrepareMount` as `<fake.AppImage> --appimage-mount`, so it exercises the real exec/readiness/teardown code without FUSE. It creates a plain directory (not a mount); the production validator must therefore accept a directory with the right tree, not require `statfs == FUSE`.

```sh
#!/bin/sh
# fake.AppImage — lives in a t.TempDir() next to fake-app-server and scenario.json.
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
cleanup() { rm -rf "$dir"; }
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexec "%s" --scenario "%s" "$@"\n' \
	"$here/fake-app-server" "$here/scenario.json" > "$dir/zcode"
chmod 755 "$dir/zcode"
printf '\n' > "$dir/resources/glm/zcode.cjs"
[ -f "$here/zcode-builtin.json" ] && cp "$here/zcode-builtin.json" "$dir/resources/config/provider/"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
```

Variants (separate script bodies, not env-controlled, because the helper env is a closed allowlist):

| Variant | Behavior | Assertion |
|---------|----------|-----------|
| `ok` | creates tree, prints path, cleans on TERM | happy path; packaged ACP e2e |
| `early-exit` | prints nothing, exits 127 | error names step + exit; no stale mount |
| `no-line` | creates tree but never prints; sleeps | readiness timeout; helper reaped; tree/base cleaned |
| `bogus-line` | prints `not-a-path`, exits 1 | validated-line rejection with observed output |
| `missing-tree` | prints a dir without `zcode` | validation error names the missing path |
| `vanish` | prints path, removes the tree after 200 ms, stays alive | `Lost` fires; session ends with mount cause |
| `no-fuse` | prints the runtime failure text to stdout/stderr, exits 127 | error classifies FUSE/libfuse2 and names remedies |

### Design sketch: OS signal wiring (`cmd/zcode-acp/main.go`)

```go
// Design sketch — closes the D-02 "signal" exit path (currently absent).
ctx, stop := signal.NotifyContext(context.Background(),
	os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
defer stop()

// ... discovery, mount, startSupervised ...

select {
case <-agentConn.Done():
	slog.Info("zcode-acp: ACP client disconnected")
case <-child.Done:
	childLost = true
case <-mountLost:
	mountLostFlag = true
case <-ctx.Done():
	slog.Info("zcode-acp: signal received; shutting down")
}
```

Then run the existing staged shutdown, stop the mount last, and return `nil` on a clean signal shutdown (unless teardown failed). `signal.NotifyContext` exists in Go 1.27's stdlib; no dependency.

## State of the Art

| Old Approach | Current Approach (this phase) | When Changed | Impact |
|--------------|-------------------------------|--------------|--------|
| Extract a packed AppImage with `--appimage-extract` into `squashfs-root` | Mount it read-only via `<AppImage> --appimage-mount` and run the app-server from the mount | v0.23.0 | No ~850 MB duplicate; requires FUSE/libfuse2; mount is read-only |
| `/tmp/.mount_*` is always do-not-rely | Unmanaged mounts remain do-not-rely; the wrapper's **managed** mount under its private `TMPDIR` is the spawn root | v0.23.0 | Volatile-name rule preserved for discovery; managed lifetime is owned |
| Discovery takes the first layout that matches | AppImage-layout candidates are ranked by `<ver>`; ties use D-08 | v0.23.0 | A stale unpacked extract can no longer pin an older desktop |
| One bad candidate aborts the probe | Every candidate is evaluated; misses are recorded; iteration continues | v0.23.0 | Dangling symlinks and invalid trees are diagnostics, not dead ends |
| `--app-server` treats every non-directory as the binary | `.AppImage` (literal or resolved) routes through the mount | v0.23.0 | The override can address a packed AppImage honestly |
| Legacy libfuse2 runtime (`dlopen("libfuse.so.2")`) | Same protocol; newer static type-2 runtimes also exist and need no libfuse2 | runtime/vendor-dependent | Do not key diagnostics solely on libfuse2; the line/wait protocol is shared |

**Deprecated/outdated:**
- `docs/protocol/install-discovery.md:42` "Extraction is the reproducible route" — replaced for the wrapper by the managed mount; extraction stays a manual inspection route.
- `docs/protocol/install-discovery.md:114` "extract (`--appimage-extract`) only if a packed AppImage is all that exists" — replaced by version ranking.
- `Probers.ExtractAppImage`, `defaultExtractAppImage`, and the `probeAppImageGlob` early return — removed.
- Treating a `.AppImage` override as an app-server binary — removed.
- `fusermount3 -u` as a cleanup answer for a libfuse2 mount — not viable (field-verified `Operation not permitted`).

## Docs Impact (VER-03)

| Surface | Exact change | Honesty / provenance constraint |
|---------|--------------|---------------------------------|
| `docs/protocol/install-discovery.md` row 5 (`:25`) and bullet `:45` | Keep `/tmp/.mount_*` **unmanaged** mounts do-not-rely; add that the wrapper creates its own managed mount from a packed AppImage and owns its lifetime | Cite the existing `[V-EXP EXP-DISCOVERY-01]` for the volatile unmanaged row; the managed row is new |
| `docs/protocol/install-discovery.md` layout matrix | Add **row 6 — Managed mount (wrapper-created)**: locate `<packed AppImage> --appimage-mount` → `<mount>/zcode` + `<mount>/resources/glm/zcode.cjs`; status SUPPORTED (read-only, lifecycle-owned) | New claims tag `[V-SRC AppImageKit src/runtime.c@effcebc]`; add an `EXP-...` capture only via the scrub pipeline if field evidence is wanted |
| `docs/protocol/install-discovery.md:42` | Replace "Extraction is the reproducible route" with: the wrapper's route is the managed mount; `--appimage-extract` remains a manual inspection route only | Do not imply extraction is used or supported for the spawn root |
| `docs/protocol/install-discovery.md:33` | Add selection semantics: compare `<ver>` across AppImage-layout candidates; ties prefer unpacked, then symlink, then lexicographic | Mirrors D-06/D-08; no version formula change |
| `docs/protocol/install-discovery.md:110-116` (recommended order) | Step 3 becomes "collect packed + unpacked + xdg AppImage-family seeds, evaluate per candidate, rank by `<ver>`"; step 5 distinguishes unmanaged volatile mounts from the wrapper's managed mount | "Each step is a pure read" stays true because mounting happens after selection |
| `docs/protocol/install-discovery.md:120-122` (operational guarantees) | Add: discovery is pure-read; the managed mount is created only for the selected packed AppImage, after discovery; teardown is verified | No claim of idempotent mounting; the mount is one-shot per run |
| `docs/protocol/install-discovery.md:5` (confidence) | Add the managed-mount row to the HIGH list (source at the embedded commit; field-exercised on 2026-11-23) | If no capture is committed, prefer V-SRC and describe field exercise as metadata, not a V-EXP claim |
| `README.md` Install section | State that a packed AppImage is mounted read-only at startup via its own `--appimage-mount`; requires FUSE + libfuse2/libfuse2t64; no extraction copy | Match `docs/provenance.md`; no unsupported claims |
| `README.md` configuration table (`:41-47`) | `--app-server` / `ZCODE_ACP_APP_SERVER`: "install root, app-server binary, or packed `.AppImage`" | Precedence unchanged (flag > env > discovery) |
| `README.md` Troubleshooting | Add the exact failures: `AppImages require FUSE to run.`, `Cannot mount AppImage, please check your FUSE setup.`, `dlopen(): error loading libfuse.so.2`; remediation `sudo apt install libfuse2t64` (24.04) / `libfuse2` | Quote the runtime strings; mention no extraction fallback |
| `docs/provenance.md` | Add to the provenance statement/known limitations: the AppImage is mounted **read-only**, the app-server runs from the mount, no `squashfs-root`/`zcode-<ver>` copy is created; FUSE/libfuse2 is a host requirement and missing FUSE fails loud | Keep the "never patched / never forged" contract intact; no new attestation claims |
| `scripts/vm-e2e.sh` | Update the mirrored probe-order narrative (`:41-45`) and the discovery-stage detail (`:345-358`); add FUSE/libfuse2 dependency asserts if the VM may discover a packed AppImage | The wrapper's discovery remains authoritative; the script is a mirror |
| `experiments/check-docs.sh` gates | No new gate needed. New `[V-EXP ...]` tags would require a scrubbed capture with a matching first-line header; avoid hand-creating captures | `experiments/README.md:13-15`: the scrubber is the only path into `experiments/captures/` |
| Path hygiene | All new doc text must use `~`/glob/placeholder patterns, never a user-home absolute path (check e) | `experiments/check-docs.sh` path-shape scan |
| Scope honesty | 4.0.0 `session/create` from a mount is verified; a full prompt on 4.0.0 is not; the static type-2 runtime variant is source-only | Do not upgrade these to verified claims in docs |

**Do not document the extraction route as the wrapper's behavior.** The docs currently say extraction is "the reproducible route" (`install-discovery.md:42`) and "extract only if a packed AppImage is all that exists" (`:114`); both become false. Replace them, do not leave them alongside the new text.

## Assumptions Log

All rows below are `[ASSUMED]` policy interpretations or extrapolations unless the row explicitly cites a verified fact; each rests on the verified code/runtime facts in the referenced section. They need user confirmation before becoming locked decisions.

| # | Claim (all `[ASSUMED]` unless noted) | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "Usable" for a packed candidate is validated at the file level during discovery (exists, regular, executable, resolvable, decoy-free, version parsed); the actual tree is validated once after mounting the selected candidate. Discovery does not mount-probe candidates. | Pattern 3 | If the user expects discovery to fall back to the next candidate when the newest packed AppImage is corrupt, the phase would need multi-mount probing (side effects, FUSE in discovery) — a materially different design |
| A2 | XDG targets that resolve to AppImage-family paths (packed or `squashfs-root`) join the ranked candidate stage; a package tree from the entry still wins at order step 2. | Pattern 3 | If XDG AppImage targets should win outright, a stale entry could pin an old backend (contradicts DISC-01); if they should be ignored, an AppImage outside `~/Applications` is unreachable |
| A3 | `Install` gains `AppImage`/`Version` fields rather than a separate `Backend` type; `Layout="appimage"` means the packed/mounted layout. | Pattern 6 | A separate type is cleaner but touches more call sites; overloading `Root` with a file path risks executing the AppImage as the binary |
| A4 | The helper env is exactly PATH, HOME, and a private TMPDIR. | Pattern 1 | If libfuse resolves `fusermount` via PATH, omitting PATH breaks mounts; if it needs another variable, the real test surfaces it |
| A5 | A teardown that leaves a mount should turn an otherwise clean run into exit 1. | Pattern 4 | A warning-only policy hides a leak; a hard failure may annoy users whose host is slow to unmount |
| A6 | ZCode 4.0.0 serves `session/create` from a mount but a full prompt on 4.0.0 is not proven in this phase. | Summary / Validation | Success criterion 1 mentions "session/create plus prompt"; if 4.0.0 drifts, the prompt proof needs a separate decision (pin 3.18.2 for the prompt leg or accept a documented drift gap) |
| A7 | Post-mount bundled `BuiltinFile` with a volatile mount path yields a revision consistent with the app-server (same env path) and does not break the registry match. | Pitfall 15 | If the app-server resolves a different canonical path, the `basedOnZCodeBuiltinRevision` match could silently drop; QA-02 proved the bundled content path works on an unpacked tree, not on a mount |
| A8 | A candidate with an unparseable version is kept and ranked last (with a diagnostics note) rather than dropped. | Pattern 3 | Dropping it could make a renamed-but-working AppImage undiscoverable; keeping it could let an unversioned file win when no versioned candidate exists (deterministic, but the user may prefer loud failure) |
| A9 | Default timeouts: readiness 15 s, stop grace 5 s, reap 10 s, mount-gone 5 s, loss poll 1 s. | Pattern 2/4/5 | Too short on a loaded host yields false readiness failures; too long delays Ctrl-C. All should be constants/options, not magic numbers |
| A10 | The `--appimage-mount` line/wait protocol is stable across runtime families (legacy libfuse2 and newer static type-2). | Pattern 2 | A future runtime could print extra lines; scanning for a validated path line (not the first line) mitigates it, but semantics are only source-verified for `effcebc` |
| A11 | Signal exit paths are SIGINT, SIGTERM, SIGHUP; a clean signal shutdown returns exit 0. | Pitfall 14 | The ACP client may prefer 130/143; the project's exit-code convention only defines 1 (runtime) and 2 (usage) |
| A12 | No `Pdeathsig` in v1; a SIGKILL of the wrapper can orphan the helper and leave a mount. | Pitfall 14 | D-02 lists "signal" (catchable) not SIGKILL; hardening requires Linux build tags and a separate design decision |
| A13 | Stale extracts from the old behavior are never deleted; they are ranked below newer packed candidates. | Runtime State Inventory | Deleting user data would violate the honesty posture; leaving them is safe but users may see a stale directory forever |
| A14 | Symlink and resolved target are one candidate keyed by resolved path; the symlink representation is preferred when equal. | Pattern 3 | Without dedupe, D-08 produces two rows for one backend and possibly two mount attempts |
| A15 | The real test may pin an AppImage via a new env (e.g., `ZCODE_ACP_REAL_APPIMAGE`) when the newest packed candidate is unsuitable. | Validation | Without a pin, the test is at the mercy of whatever auto-update left in `~/Applications` |

## Open Questions (RESOLVED)

All nine questions are resolved by binding decisions D-12..D-14 in `10-CONTEXT.md`
and by the plan set's exit-code/teardown decisions. The inline recommendations are
kept as the reasoning trail; the **Resolution** line is the contract execution
follows.

1. **How deep should "usable" go during discovery?**
   - What we know: discovery must stay pure-read (`install-discovery.md:110, 120`); mounting candidates would violate idempotency.
   - What's unclear: whether a corrupt newest packed AppImage should fall back to the next candidate or fail loud.
   - Recommendation: file-level validation + fail loud on the selected candidate (D-03), explicitly documented.
   - **Resolution (D-12):** pure-read file-level validation only — resolvable path, regular executable file, parseable `<ver>`, not a volatile mount. The newest candidate is acquired; a mount failure fails loud and never silently falls back to an older candidate. An unversioned packed file is a miss; the explicit `--app-server` override is the route for renamed files.

2. **Do XDG AppImage-family targets join the version ranking?**
   - What we know: today the entry's packed target is rejected and deferred to the glob; on this machine the entry points at 4.0.0 anyway.
   - What's unclear: whether an entry target outside `~/Applications` should be a candidate.
   - Recommendation: yes, seed the ranked stage; package tree targets still win at step 2.
   - **Resolution (D-13):** yes — an XDG entry whose target is an AppImage joins the AppImage version ranking; an entry resolving to a package tree (`/opt/ZCode`) keeps the existing package-path behavior and precedence.

3. **Does success criterion 1 require a real prompt on ZCode 4.0.0?**
   - What we know: `session/create` from a 4.0.0 mount works (field probe); VER-02 only names `session/create`.
   - What's unclear: whether a 4.0.0 turn's wire drift allows a prompt in this milestone (DISC-V2-01 defers drift).
   - Recommendation: make VER-02 assert `session/create` + clean teardown; record the prompt as a separate guarded acceptance or explicitly scope it out.
   - **Resolution (D-14):** the real tier stops at `session/create` + clean teardown. A live 4.0.0 prompt is out of scope; the success-criterion-1 turn proof is carried by the CI fake-runtime path, and 4.0.0 wire drift is deferred to DISC-V2-01. The gap is documented, never papered over.

4. **What exit code should a signal-initiated clean shutdown use?**
   - What we know: the repo defines 1 (runtime failure) and 2 (usage); `run` returns `nil` on intentional shutdown.
   - Recommendation: return `nil` after a clean signal shutdown; only a teardown failure returns an error (exit 1).
   - **Resolution:** adopted as recommended and implemented in 10-02 — a clean SIGINT/SIGTERM/SIGHUP shutdown exits 0 (`run` returns nil); only a teardown failure returns exit 1.

5. **Should a mount-teardown failure fail the run?**
   - Recommendation: yes (A5), with a message naming the mount and the manual remediation.
   - **Resolution:** yes — fail the run with exit 1, naming the mount path and the manual-unmount remediation; no warning-only arm. Implemented and tested in 10-02 (stubbed never-gone mount).

6. **Docs evidence: new capture or V-SRC?**
   - What we know: `experiments/README.md` forbids hand-creating captures; the runtime source at `effcebc` is authoritative and the doc tag grammar accepts `[V-SRC ...]`.
   - Recommendation: tag the managed-mount facts `[V-SRC AppImageKit src/runtime.c@effcebc]`; add an `EXP-APPIMAGE-MOUNT-01` capture only if the user wants field evidence in the docs (it requires the scrub pipeline).
   - **Resolution:** use `[V-SRC AppImageKit src/runtime.c@effcebc]` tags. No new capture is created in this phase; the field probe stays a research/session observation and is reused by the guarded VER-02 run.

7. **Should `scripts/vm-e2e.sh` assert FUSE/libfuse2?**
   - What we know: the QA-02 VM used an unpacked copy; a VM that discovers a packed AppImage now needs FUSE/libfuse2.
   - Recommendation: update the dependency asserts and the discovery narrative, then run `node --test experiments/`.
   - **Resolution:** yes — 10-06 updates the vm-e2e dependency asserts (FUSE/libfuse2) and the discovery narrative, and runs `node --test experiments/`.

8. **Is Linux `Pdeathsig` hardening in scope?**
   - What we know: SIGKILL of the wrapper cannot run Go defers; children are in separate process groups.
   - Recommendation: defer to a follow-up unless zero stale mounts on wrapper SIGKILL is a hard acceptance criterion. If in scope, use a build-tagged `SysProcAttr` helper.
   - **Resolution:** deferred during planning; zero stale mounts on normal and handled-signal paths is the acceptance. The wrapper-SIGKILL residual risk is documented (T-10-04/T-10-09), not claimed away.

9. **What version should the real test target?**
   - Recommendation: newest packed AppImage from discovery, with `ZCODE_ACP_REAL_APPIMAGE` as an explicit pin. On this machine that is 4.0.0; the probe proves it serves `session/create`.
   - **Resolution:** newest packed AppImage from discovery, with `ZCODE_ACP_REAL_APPIMAGE` as the explicit pin (4.0.0 on this host). The real tier asserts `session/create` and zero leftovers per D-14.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | Build/tests | ✓ | go1.27.3 (module `go 1.27`) | mise pin |
| golangci-lint | CI/local gate | ✓ | 2.15.0 | mise install |
| Node (experiments suite) | `mise run test-experiments` | ✓ | v26.13.1 | mise install |
| `/dev/fuse` | Real mount backend | ✓ | char 10:229 | None; fake runtime for CI |
| `libfuse2.so.2` | Legacy AppImage runtime | ✓ | `libfuse2t64` 2.9.9-8.1build1 | Install `libfuse2t64` (24.04) / `libfuse2` |
| `fusermount` (FUSE2 path) | Runtime unprivileged mount | ✓ | `/usr/bin/fusermount` (reports 3.14.0) | None; the runtime requires it |
| `fusermount3` | — | ✓ | 3.18.3 (brew) | Not usable to unmount the fuse2 mount |
| Real packed AppImage | Real test / field proof | ✓ | `~/Applications/ZCode-4.0.0-linux-x64.AppImage` (264,460,979 bytes, 0755) | Skip the real test on hosts without one |
| Stale unpacked tree | Ranking fixture | ✓ | `~/Applications/zcode-3.18.2/squashfs-root` | Create a fixture tree in tests |
| Dangling symlink fixture | Ranking fixture | ✓ | `~/Applications/ZCode.AppImage -> ...ZCode-3.18.2-linux-x64.AppImage` (target deleted) | Create a dangling symlink in tests |
| `xdg-mime` | Discovery step 2 | ✓ | `/usr/bin/xdg-mime`; default `zcode.desktop` → 4.0.0 AppImage | Missing seam records a miss |
| `dpkg` | Discovery step 4 | ✓ | `/usr/bin/dpkg` | Missing seam records a miss |
| `rpm` | Discovery step 4c | ✗ | — | Probe records "rpm seam not configured / not an rpm host"; not blocking |
| `mount`, `ps`, `pgrep` | Diagnostics/tests only | ✓ | coreutils/procps (note: `mount` on PATH is linuxbrew's) | Production code must not shell out to them |
| XDG runtime provider cache | Builtin selection | ✓ | `~/.zcode/v2/runtime/provider/linux-x86_64/4.0.0/...` | Bundled builtin post-mount; otherwise empty (honest) |

**Missing dependencies with no fallback:**
- None for development/CI. The real-machine tier requires FUSE + libfuse2 + a real AppImage, and is env-gated/skipped otherwise.

**Missing dependencies with fallback:**
- `rpm` — the rpm probe records an honest miss; deb/opt/AppImage paths are unaffected.
- FUSE in CI — the scripted fake AppImage runtime exercises the lifecycle without FUSE.

## Validation Architecture

> `workflow.nyquist_validation` is `true` in `.planning/config.json`; `tdd_mode` is also `true` and `granularity` is `"fine"`.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), `-race`; existing fake-app-server binary and reexec helpers |
| Config file | `mise.toml` (tool pins) + `.golangci.yml`; no separate test config |
| Quick run command | `go test -race -count=1 ./internal/supervisor/ -run 'TestMount|TestDiscover'` |
| Binary/integration command | `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestAppImage|TestOmp'` |
| Full suite command | `mise run test` (`go test -race -count=1 ./...`) |
| CI gate | `mise run ci` (build + vet + lint + test + test-experiments) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| MNT-01 | Packed AppImage run completes a session from the mount | integration (fake runtime) + guarded real | `go test -race ./cmd/zcode-acp/ -run TestAppImageMountEndToEnd` | ❌ Wave 0 |
| MNT-02 | Override/env packed path mounts, validates both files | unit + binary | `go test -race ./internal/supervisor/ -run TestMountValidatesTree` / `./cmd/zcode-acp/ -run TestOverridePacked` | ❌ Wave 0 |
| MNT-03 | Helper started first, stopped last; zero survivors | unit + integration | `go test -race ./internal/supervisor/ -run TestMountStopOrder` | ❌ Wave 0 |
| MNT-04 | Early exit / timeout / no FUSE / invalid tree fail loud | table unit + binary exit-code test | `go test -race ./internal/supervisor/ -run TestMountFailures` | ❌ Wave 0 |
| MNT-05 | Helper death / tree vanish ends the session with cause | unit | `go test -race ./internal/supervisor/ -run TestMountLost` | ❌ Wave 0 |
| MNT-06 | No extract dirs; AppImage unmodified | integration + real | `go test -race ./cmd/zcode-acp/ -run TestAppImageMountEndToEnd` (hash/stat asserts) | ❌ Wave 0 |
| DISC-01 | Newest packed beats stale unpacked | unit | `go test -race ./internal/supervisor/ -run TestDiscoverRanksPackedOverStaleUnpacked` | ❌ Wave 0 (update `discover_test.go`) |
| DISC-02 | Dangling symlink/invalid candidate is a miss, scan continues | unit | `... -run TestDiscoverPerCandidateMisses` | ❌ Wave 0 |
| DISC-03 | Diagnostics carry layout/version/resolved path | unit + binary stderr | `... -run TestDiscoverDiagnostics` | ❌ Wave 0 |
| DISC-04 | Deterministic tie-break (unpacked > symlink > file > lexicographic) | unit | `... -run TestDiscoverTieBreak` | ❌ Wave 0 |
| VER-01 | Fake runtime drives mount lifecycle without FUSE | unit/integration | `... -run TestMount` + `TestAppImageMountEndToEnd` | ❌ Wave 0 (fixture helper) |
| VER-02 | Real mount + `session/create` + zero leftovers | manual, build-tagged | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/` | ❌ Wave 0 (new guarded test) |
| VER-03 | Docs coherent; no claim contradicts behavior | static gate + review | `bash experiments/check-docs.sh` + doc diff review | ✅ (gate exists) |

### Sampling Rate

- **Per task commit:** the narrow package command for the touched area (`go test -race -count=1 ./internal/supervisor/`).
- **Per wave merge:** `mise run test` (full Go suite with race).
- **Phase gate:** `mise run ci` green before `/gsd:verify-work`; `bash experiments/check-docs.sh`; the guarded real test run once on the real machine with the output recorded.

### Wave 0 Gaps

- [ ] `internal/supervisor/mount.go` + `mount_test.go` — the mount owner and the fake-runtime harness (all MNT rows).
- [ ] `internal/supervisor/discover_test.go` — replace the extraction fixture with packed/ranking/tie-break/per-candidate tests; update `documentedProbeNames`/`assertProbeOrder`.
- [ ] `cmd/zcode-acp/main_test.go` — binary-level fake-AppImage e2e (happy path, failure exit codes, signal shutdown, no `squashfs-root`, AppImage hash unchanged).
- [ ] `internal/supervisor/realdesktop_test.go` (or a new `appimage_realdesktop_test.go`) — guarded real mount + `session/create`.
- [ ] `test/realdesktop/apicycle_test.go` — update the shared `discoverRealInstall`/`overrideInstall` mirror to mount packed installs (manual tier, but must not rot).
- [ ] No framework install; no new dependencies.

**Existing seams to reuse:** `Probers` (discovery: `Override`, `XDGQuery`, `AppDirs`, `DpkgQuery`, `RpmQuery`, `Home`, `OptDir`) keeps discovery hermetic; `SpawnFunc` is the app-server spawn seam; the reexec `TestSuperviseHelper` pattern covers process-group/staged-shutdown behavior; the `e2eBuildFakeBinary` + `e2eWriteFakeInstallRoot` shim pattern covers binary-level tests. Only the mount readiness/stop timings need a new option seam (`MountOpts`); the fake runtime itself needs no seam because production `PrepareMount` executes it directly.

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`, `security_block_on: "high"` (`[VERIFIED: .planning/config.json:50-52]`).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No new authentication surface; ACP auth is unchanged |
| V3 Session Management | no | No cookies/tokens; stdio sessions unchanged |
| V4 Access Control | yes | Private 0700 `TMPDIR`; only the user-specified/discovered AppImage and its mount are read/executed; never delete arbitrary paths |
| V5 Input Validation | yes | Validate the override path shape, every candidate, the mount line, the mount tree, and the helper env allowlist |
| V6 Cryptography | no | `BuiltinRevision` and credential code unchanged; never hand-roll new crypto |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via the AppImage path or mount path | Tampering / Elevation | `exec.CommandContext` with argv, never a shell; pass the mount path as one argv element; quote only for logs |
| Env poisoning (`APPIMAGE_EXTRACT_AND_RUN`, `TARGET_APPIMAGE`, `LD_PRELOAD`, proxy vars) | Tampering | Explicit helper env allowlist (PATH/HOME/TMPDIR); never `os.Environ()` |
| Symlink / path traversal in discovery or the mount line | Spoofing / Tampering | `EvalSymlinks` per candidate; reject volatile-mount decoys; require the mount path to be one component under the resolved private base; `Lstat` the mount root |
| Adopting an arbitrary path printed by a helper | Spoofing | Validate absolute, existing, directory, not a symlink, under the private base, with the expected tree; the helper is the executed binary, not an oracle |
| TOCTOU: the AppImage is replaced between discovery and exec | Tampering | Resolve once, exec the resolved path; mount is read-only; accept the residual risk (the user controls the path) |
| Log injection via helper output | Tampering | Cap both streams; render observed lines with `%q`; full text only at Debug |
| Temp-dir squatting / cross-user mounts | Elevation / DoS | 0700 private base created by us; never scan/delete other users' `/tmp/.mount_*`; `RemoveAll` only the base we own |
| Orphaned helper/daemon and stale mounts | DoS / Resource exhaustion | Staged Stop, keepalive-pipe teardown, `Lost` watcher, bounded polls, failure on a remaining mount |
| Untrusted AppImage execution | Elevation | Executing the AppImage is the explicit product contract (override or a `ZCode*.AppImage` under `~/Applications`); no auto-execution from `/tmp`; signature verification is out of scope and must not be claimed |
| Secret leakage into helper logs/env | Information disclosure | The helper env excludes credentials; diagnostics never print env values |

**Execution boundary:** the wrapper executes exactly two paths per mount-backed run — the user-selected/discovered AppImage (as the `--appimage-mount` helper) and `<mount>/zcode` (as the app-server binary, with `<mount>/resources/glm/zcode.cjs` as its script argument). Validation only `stat`s the two expected files; it never runs anything else from the mount. Do not add "run `<mount>/zcode --version`" as a validation probe — that is a third execution and a protocol risk.

## Sources

### Primary (HIGH confidence)

- `internal/supervisor/discover.go` — layout constants (`:29-34`), suffix/rel constants (`:37-59`), `Install` (`:69-75`), `Probe` (`:80-84`), `Probers` (`:90-113`), `Discover` order (`:122-156`), `mountDecoy` (`:189-199`), `bundleInstall` (`:204-222`), `ExplicitRootInstall` (`:230-232`), `probeOverride` (`:236-257`), `probeAppImageGlob`/early return/`packed[0]`/extraction (`:390-449`), `classifyAppDirs` (`:454-467`), `probeUnpackedGlob` (`:471-493`), `sortAppImageCandidates` (`:736-752`), `dirVersion` (`:756-761`), `compareVersions` (`:767-804`), `DefaultProbers` (`:821-832`), `defaultExtractAppImage` (`:898-916`)
- `internal/supervisor/spawn.go` — spawn facts/env names (`:28-45`), `Child` (`:82-107`), `SpawnOpts` (`:116-123`), `SpawnArgv` (`:128-130`), `ChildEnv` (`:137-169`), `SpawnChild` (`:178-216`), `newSpawnCommand` (`:236-273`), `killGroup` (`:323-329`)
- `internal/supervisor/supervise.go` — policy defaults (`:24-40`), `SpawnFunc`/`Supervisor` (`:90-106`), `Start` (`:113-166`), `shutdown` (`:168-193`)
- `cmd/zcode-acp/main.go` — env names/exit codes/sentinels (`:43-84`), flag help (`:94-95`), config precedence (`:405-434`), `desktopVersion` (`:508-526`), `run` (`:648-738`), `discoverInstall` (`:740-771`), `overrideInstall` (`:773-801`), `startSupervised` (`:803-847`)
- `docs/protocol/install-discovery.md` — layout matrix (`:19-35`), entrypoint invariant (`:39-45`), spawn line (`:47-53`), version formats (`:33`), empty result (`:34`), discovery order (`:108-122`)
- `docs/provenance.md` — honesty contract (`:10-38`)
- `test/realdesktop/apicycle_test.go` — shared `discoverRealInstall` (`:335-368`), duplicate `overrideInstall` (`:370-404`), `startSupervisedChild` (`:436-470`)
- `cmd/zcode-acp/main_test.go` — `e2eWriteFakeInstallRoot` shim pattern (`:501-519`)
- `internal/supervisor/realdesktop_test.go` — `ZCODE_ACP_REAL_E2E=1` + `-tags realdesktop` guard (`:1-19, 79-101`)
- AppImageKit `src/runtime.c` at commit `effcebc` — mount help (`:215-224`), `EXIT_EXECERROR` (`:65`), `TARGET_APPIMAGE` mount name (`:501-525`), `TMPDIR` (`:603-609`), `APPIMAGE_EXTRACT_AND_RUN` (`:688`), `mkdtemp`/fork (`:821-881`), keepalive (`:137-161`), readiness print/flush/pause (`:914-928`)
- AppImageKit `src/notify.c` at `effcebc` — failure text to stdout (`:12-21`)
- `.planning/config.json` — `tdd_mode: true` (`:53`), `granularity: "fine"` (`:81`), `nyquist_validation: true` (`:26`), security settings (`:50-52`)
- `mise.toml` — `test` task (`go test -race -count=1 ./...`); `tidy-diff` one-require guard
- `.github/workflows/pr-sanity.yml` — `mise run test`, `mise run lint`, `mise run tidy-diff`
- `go.mod` — `go 1.27`, one require (`acp-go-sdk v0.20.0`)
- `experiments/check-docs.sh` — path-shape scan (`:25-27` comment) and V-EXP resolution rules
- `experiments/README.md` — "never hand-edit or hand-create capture files" (`:13-15`)

### Secondary (MEDIUM confidence)

- `[CITED: docs.appimage.org/user-guide/troubleshooting/fuse.html]` — "AppImages require a Linux technology called Filesystem in Userspace (or short FUSE)"; "In Ubuntu 24.04, the libfuse2 package was renamed to libfuse2t64"; install `libfuse2`/`libfuse2t64`; the FUSE error text.
- `[CITED: github.com/AppImage/type2-runtime README]` — newer static runtime "mounts the payload via FUSE"; "Since the runtime is linked statically, libfuse2 is no longer required on the target system."
- `[VERIFIED: field probe 2026-11-23]` — local observations: mount line hex/newline, ~23 ms readiness, 15/15 immediately-ready, mount options `ro,nosuid,nodev,relatime`, `ELECTRON_RUN_AS_NODE=1 <mount>/zcode -e` → `node-ok v24.21.0 arch x64`, SIGINT 130 / SIGTERM 143 / SIGKILL 137 all unmount, 0/5 directory residue after 50 ms, spaces/TMPDIR relocation, `fusermount3` `Operation not permitted`, `fusermount` `EBUSY`, 4.0.0 `session/create` success in 9.0 s, zero leftovers.
- `[VERIFIED: /usr/include/linux/magic.h:39]` — `#define FUSE_SUPER_MAGIC 0x65735546` (only relevant if an optional statfs check is added).

### Tertiary (LOW confidence)

- The exact behavior of the newest static type-2 runtime under `--appimage-mount` on a FUSE-less host (source/README only; not field-tested).
- Real ZCode 4.0.0 prompt/turn behavior from a mount (only `session/create` was probed; drift is explicitly out of scope).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; versions and the module contents were read this session.
- Mount protocol/readiness/teardown: HIGH — exact runtime source at the embedded commit plus field trials.
- Architecture (discovery ranking, override routing, install shape): MEDIUM-HIGH — grounded in the current code and the locked decisions, but A1-A3 are policy interpretations the user should confirm.
- Pitfalls: HIGH — every pitfall is tied to a quoted code path, runtime source line, or field observation.
- Validation: HIGH for the fake-runtime and unit tiers; MEDIUM for the real tier (guarded, host-dependent, 4.0.0 prompt unproven).

**Research date:** 2026-11-23
**Valid until:** 2026-12-23 for this project's stable stack; re-check the AppImage runtime if ZCode ships a newer major (the `effcebc` source is version-pinned, not floating).

