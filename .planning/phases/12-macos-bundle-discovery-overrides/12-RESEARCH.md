# Phase 12 Research: macOS Bundle Discovery & Overrides

**Researched:** 2026-11-26
**Phase:** 12 — macOS Bundle Discovery & Overrides
**Requirements:** MDISC-01, MDISC-02

## Verified machine facts

The installed bundle on this Mac is:

```
/Applications/ZCode.app
  Contents/Info.plist
  Contents/MacOS/ZCode
  Contents/Resources/glm/zcode.cjs
  Contents/Resources/config/provider/zcode-builtin.json
```

`Info.plist` reports:

- `CFBundleExecutable = ZCode`
- `CFBundleShortVersionString = 4.0.0`
- `CFBundleVersion = 4.0.0.10870`
- `CFBundleIdentifier = dev.zcode.app`

The provider runtime cache already uses a Darwin platform directory:

```
~/.zcode/v2/runtime/provider/darwin-aarch64/<version>/endpoint-*/zcode-builtin.json
```

The bundled table is revision 44; the 4.0.0 runtime cache is revision 48. Runtime
cache precedence therefore matters, but selecting it is Phase 13 (MDISC-04).

## Existing implementation

`internal/supervisor/discover.go` currently models only Linux layouts:

- `bundleInstall(root, layout)` requires `<root>/zcode` and
  `<root>/resources/glm/zcode.cjs`.
- `OverrideInstall(path)` handles directories, `.AppImage` files, and arbitrary
  app-server binaries.
- `DefaultProbers` wires `xdg-mime`, `dpkg`, `rpm`, and
  `~/.zcode/v2/runtime/provider/linux-x86_64`.
- `probeBuiltinBundled` looks under `<root>/resources/config/provider/`.

None of those shapes match an `.app`.

## Design

1. Add `layoutMacApp = "mac-app"`.
2. Add an XML-plist reader for `Contents/Info.plist` using `encoding/xml`; if the
   plist is binary and `plutil` is available, convert it to XML first. Tests use
   XML fixtures and do not invoke `plutil`.
3. Add `macAppInstall(appPath) (Install, reason string)`:
   - accept only an existing `.app` directory;
   - require `Info.plist` with `CFBundleExecutable` and a version
     (`CFBundleShortVersionString`, falling back to `CFBundleVersion`);
   - reject path separators or traversal in `CFBundleExecutable`;
   - require an executable regular `Contents/MacOS/<exe>`;
   - require `Contents/Resources/glm/zcode.cjs`;
   - require `Contents/Resources/config/provider/zcode-builtin.json`;
   - return `Install{Root: appPath, Binary: ..., Entrypoint: ..., Layout:
     "mac-app", Version: ...}`.
4. Add a `MacApps` probe seam and `probeMacApp` fixed probe after `override` and
   before the Linux entry/AppImage probes. Each candidate gets an actionable
   `mac-app-candidate` row; a valid newest version wins.
5. `DefaultProbers` scans `/Applications/ZCode*.app` and
   `$HOME/Applications/ZCode*.app` only on `darwin`.
6. Extend `OverrideInstall`:
   - `<path>.app` → app install;
   - `<path>.app/Contents` → app install;
   - `<path>.app/Contents/MacOS/<exe>` → app install;
   - all existing Linux directory/AppImage/binary behavior unchanged.
7. Teach `probeBuiltinBundled` the app-relative provider path:
   `Contents/Resources/config/provider/zcode-builtin.json`.
8. A mac-app install has `AppImage == ""`, so `RequiresMount()` is false and no
   FUSE path can run.

## Test strategy

- Fixture builder writes a fake `.app` with XML `Info.plist`, executable, stub
  entrypoint, and bundled provider table.
- Table tests cover valid discovery from both standard locations, missing or
  malformed `Info.plist`, missing executable, non-executable binary, missing
  entrypoint, missing provider table, empty version, and traversal executable.
- Override tests cover `.app`, `Contents`, `Contents/MacOS/<exe>`, missing app,
  and all existing Linux override cases.
- Probe-order tests include `mac-app` in the fixed sequence.
- No test reads the real `/Applications` or real ZCode install.
