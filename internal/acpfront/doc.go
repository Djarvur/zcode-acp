// Package acpfront implements the ACP agent policy layer (RESEARCH
// Pattern 1, ARCHITECTURE): the acpport.Handlers impl behind the D-01
// seam — the honest D-02 initialize report, the D-03 session/new contract
// (bare create + subscribe + setModel follow-up), the D-04 streamed
// prompt turn over the Phase-3 turn machine, and the D-05 exactly-once
// cancel. It never imports the SDK; every SDK type stays behind
// internal/acpport.
//
// Phase 4 lands the package with its test harness (harness_test.go,
// RESEARCH Pattern 6: recording tee + SDK test client + parameterized
// compose); the Front itself arrives with plan 04-06, whose tracer is
// this harness's first full-loop consumer.
package acpfront
