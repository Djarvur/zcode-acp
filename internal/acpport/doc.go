// Package acpport is the D-01 seam: the only package in this module that
// imports github.com/coder/acp-go-sdk. The ACP v1 front leg never lets SDK
// types past this boundary — the adapter here translates between the SDK's
// generated surface and the neutral vocabulary the agent policy layer
// (internal/acpfront, plans 04-02..04-06) consumes, so a v0.x SDK break is a
// one-package migration (ARCHITECTURE seam rule; RESEARCH Pattern 1).
//
// Dependency pin: the module requires github.com/coder/acp-go-sdk v0.20.0
// exactly (D-01 audit PASS — 04-RESEARCH.md "D-01 Verdict": Apache-2.0, coder
// org, zero transitive deps, daily-active maintenance). The pin is held
// tidy-stable by the blank import in pin_test.go and asserted by the mise
// tidy-diff gate (T-4-SC pin discipline).
//
// This file landed with the dependency (plan 04-01) so the require line
// was tidy-stable from day one; the adapter surface (types.go, agent.go,
// sink.go) landed with plan 04-02, together with TestSDKImportBoundary —
// the structural seam lock (no non-test SDK import outside this package).
package acpport
