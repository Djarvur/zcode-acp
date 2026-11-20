// Package realdesktop hosts the GUARDED real-tier tests (Phase 5 plan
// 05-09): manual-only compositions against the real ZCode desktop install
// and the real z.ai api-key account. The realdesktop build tag excludes the
// package from default builds, vet, lint, and CI; each test's body
// additionally skips without ZCODE_ACP_REAL_E2E=1 (the 03-07 double-guard
// discipline — CI never contacts the desktop or the account).
//
// This file carries no code: it exists so the directory is a buildable
// package for the default ./... gates (a directory whose only file is
// build-tag-excluded falls out of the ./... package pattern entirely).
package realdesktop
