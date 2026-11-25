# Requirements: zcode-acp v0.24.0 macOS Support

**Defined:** 2026-11-25
**Core Value:** An ACP client can run a full turn end-to-end with ZCode on macOS — from `initialize` to streamed reply with tool-calls and permission requests — on a real z.ai account, without adding new ACP capabilities.

## v0.24.0 Requirements

Requirements for the macOS support milestone. Each maps to exactly one roadmap phase.

### Build & Toolchain

- [x] **MBLD-01**: The project cross-compiles with `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...`.
- [x] **MBLD-02**: Standard macOS build/test gates do not require `setsid`, GNU-only utilities, or the system Bash 3.2; required tool versions are explicit.
- [x] **MBLD-03**: CI builds and tests Linux plus macOS arm64/amd64, and a release snapshot produces checksummed universal macOS binary artifacts.

### Discovery & Backend Launch

- [x] **MDISC-01**: Discovery finds `/Applications/ZCode.app` and `$HOME/Applications/ZCode.app`, validates the bundle, and reads its version from `Info.plist`.
- [x] **MDISC-02**: `--app-server` accepts a macOS `.app`, its `Contents` directory, its `Contents/MacOS/<executable>`, and existing Linux layouts without regression.
- [x] **MDISC-03**: A discovered macOS bundle launches the desktop app-server with the correct executable, `Contents/Resources/glm/zcode.cjs` entrypoint, `ELECTRON_RUN_AS_NODE=1`, and bundled provider table.
- [x] **MDISC-04**: Runtime-provider discovery selects the macOS OS/architecture cache instead of `linux-x86_64`, and the builtin revision matches the path used by the backend.
- [x] **MDISC-05**: A GUI-launched Orca process gives the backend the user's tool `PATH` through a bounded, non-interactive login-shell union while preserving secret/proxy scrubbing.

### Authentication

- [x] **MAUTH-01**: The existing api-key authentication path completes on macOS without requiring Keychain.
- [x] **MAUTH-02**: OAuth coding-plan authentication decrypts `enc:v2:os` credentials through macOS Keychain using the existing service/account identity and bounded `/usr/bin/security` invocation.
- [x] **MAUTH-03**: Missing or locked Keychain items produce actionable coded errors, and key material is never written to logs or stdout.

### Tests & Orca Acceptance

- [x] **MTEST-01**: The full default test suite passes on macOS arm64 without an external `setsid` dependency.
- [x] **MTEST-02**: Automated fixtures cover macOS `.app` discovery/override, exact app-server argv, provider-cache selection, and the Keychain command contract without touching a real desktop or keychain in CI.
- [x] **MORCA-01**: `zcode-acp omp --print --mode text @file` completes through a fake macOS `.app` backend with correct stdout and exit status.
- [ ] **MORCA-02**: A live Orca card run on the current macOS machine completes the existing ACP flow against real ZCode with an available real account, and the evidence is recorded.

### Documentation

- [x] **MDOC-01**: README documents macOS build, ZCode discovery, Keychain prerequisites, Orca launch, and explicit platform limitations in English.

## Future Requirements

Deferred unless a later milestone explicitly adds them.

### Platform Expansion

- **FPLAT-01**: KDE and Windows support.
- **FPLAT-02**: macOS `.app` packaging, code signing, and notarization.
- **FPLAT-03**: Start Plan support on macOS.

## Out of Scope

| Feature | Reason |
|---------|--------|
| New ACP methods, tools, or client features | User explicitly requested a platform port, not functionality expansion. |
| ZCode CLI backend | Existing architecture supports desktop app-server only. |
| Rewriting Linux AppImage/FUSE behavior | macOS `.app` must bypass that path; Linux behavior remains regression-tested. |
| Plaintext or unofficial credential fallback | Existing security boundary forbids it. |
| Linux Secret Service replacement on macOS via D-Bus | macOS credentials live in Keychain; the existing `KeyLoader` seam gets a Darwin implementation. |

## Traceability

Populated during roadmap creation for v0.24.0.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MBLD-01 | Phase 11 | Complete |
| MBLD-02 | Phase 11 | Complete |
| MBLD-03 | Phase 11 | Complete |
| MDISC-01 | Phase 12 | Complete |
| MDISC-02 | Phase 12 | Complete |
| MDISC-03 | Phase 13 | Complete |
| MDISC-04 | Phase 13 | Complete |
| MDISC-05 | Phase 13 | Complete |
| MAUTH-01 | Phase 14 | Complete |
| MAUTH-02 | Phase 14 | Complete |
| MAUTH-03 | Phase 14 | Complete |
| MTEST-01 | Phase 11 | Complete |
| MTEST-02 | Phase 15 | Complete |
| MORCA-01 | Phase 15 | Complete |
| MORCA-02 | Phase 16 | Blocked — ZCode account quota reset 2026-11-27 18:05:02 |
| MDOC-01 | Phase 15 | Complete |

**Coverage:**

- v0.24.0 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0
- Duplicate mappings: 0

---

*Requirements defined: 2026-11-25*
*Last updated: 2026-11-25 after v0.24.0 roadmap creation*
