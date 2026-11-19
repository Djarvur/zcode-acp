// Package supervisor implements the ZCode-desktop back-leg plumbing: install
// discovery per the Phase 1 matrix, child spawning, and process supervision
// (BRIDGE-03). Plan 03-04 ships the discovery half; the seams on Probers keep
// the whole matrix testable without a real desktop install.
package supervisor

import (
	"context"
	"errors"
)

// errSkeleton marks the not-yet-implemented discovery bodies. The RED-stage
// contract tests fail on their own assertions against these stubs; the GREEN
// commit replaces every use.
var errSkeleton = errors.New("supervisor: discovery skeleton: not implemented")

// ErrNoInstall is the empty-result outcome: no documented layout matched.
// install-discovery.md :34 makes the empty set a first-class result — the
// returned probe rows are the actionable diagnostics, never a guess.
var ErrNoInstall = errors.New("zcode: no ZCode desktop install found")

// Install describes one located ZCode desktop install. Field semantics follow
// docs/protocol/install-discovery.md: the relative entrypoint path
// resources/glm/zcode.cjs under Root is invariant across all layouts (:38).
type Install struct {
	Root        string // dir containing the Electron binary
	Binary      string // <root>/zcode
	Entrypoint  string // <root>/resources/glm/zcode.cjs (invariant relative path)
	BuiltinFile string // ZCODE_BUILTIN_PROVIDER_CONFIG_FILE value
	Layout      string // "unpacked-appimage" | "appimage" | "deb" | "rpm" | "explicit"
}

// Probe is one diagnostics row: the locate command a probe ran and what it
// returned, whether it hit or missed. The rows are the product of an empty
// discovery result (install-discovery.md :34).
type Probe struct {
	Name   string // documented probe name (fixed set, fixed order)
	Query  string // the locate command pattern from the matrix
	Result string // what the probe returned (hit, miss, or rejection)
}

// Probers is the seamed discovery unit: one injectable seam per OS contact.
// Tests stub the seams against t.TempDir() fixture trees and canned query
// outputs; production fills them with DefaultProbers, whose seams shell out
// per the matrix commands. A nil seam records an honest miss.
type Probers struct {
	Override        func() (Install, bool)
	XDGQuery        func(ctx context.Context) (string, error)
	AppDirs         func() []string
	DpkgQuery       func(ctx context.Context) (string, error)
	RpmQuery        func(ctx context.Context) (string, error)
	ExtractAppImage func(ctx context.Context, appImagePath, workDir string) (string, error)
	Home            string
	OptDir          string
}

// Discover resolves the ZCode desktop install in the documented probe order
// (install-discovery.md §Recommended discovery order): override, desktop
// entry, AppImage paths, package paths, runtime provider tree. The first
// probe that yields a spawnable install wins and probes never merge; every
// probe appends a Probe row whether it hits or misses, and an empty result
// errors with ErrNoInstall carrying every row as diagnostics. Volatile
// /tmp/.mount_* decoys are recorded-and-rejected, never resolved (row 5).
func (p Probers) Discover(ctx context.Context) (Install, []Probe, error) {
	return Install{}, nil, errSkeleton
}

// BuiltinRevision formats the zcode builtin revision string for an active
// zcode-builtin.json path and its recorded revision number:
// zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>
// (install-discovery.md :82). The hash input is the resolved absolute PATH
// STRING, not the file contents.
func BuiltinRevision(builtinFile string, rev int) string {
	return ""
}
