// Package acpfront implements the ACP agent policy layer (RESEARCH
// Pattern 1, ARCHITECTURE): the acpport.Handlers impl behind the D-01
// seam — the honest D-02 initialize report, the D-03 session/new contract
// (bare create + subscribe + setModel follow-up), the D-04 streamed
// prompt turn over the Phase-3 turn machine, the D-05 exactly-once
// cancel, the TOOL-01 tool-note tap (stream.go), the TOOL-02 permission
// bridge (permission.go, plan 05-02), and the SESS-01 session lifecycle
// (lifecycle.go, plan 05-05). It never imports the SDK; every SDK type
// stays behind internal/acpport.
//
// The test harness (harness_test.go, RESEARCH Pattern 6: recording tee +
// SDK test client + parameterized compose) drives the Front — plan
// 04-06's tracer was the first full-loop consumer, and the composed-tier
// tests of plans 04-03..04-06 and 05-02..05-05 ride it since (IN-03
// present-tense refresh).
package acpfront
