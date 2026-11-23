---
phase: "09"
slug: "user-input-bridge"
status: complete
---

# UAT — v0.22.0 Phase 9 (ask-the-user bridge)

## Test 1 — the choice dialog renders and the answer feeds the workflow (UIB-05, VM GUI pass)

**PASSED (2026-11-23, agent-executed per the standing VM delegation).** On the qa02
VM (Orca + the omp card → the wrapper → the fake backend with the scripted
`interaction/requestUserInput` reverse): sending a prompt opened the choice dialog —
the form message ("Pick one"), the question ("Which phase?"), both options with
labels and descriptions (1 Verify 1 / verify phase one; 2 Exit manager), the Skip
button, and the chat's "Awaiting user input" marker. Clicking "Verify 1" resolved
the dialog — the chat recorded "Asked … / Verify 1 / Resolved — Answered on this
computer", the turn completed, and the fake's scripted expect matched the bridged
answer EXACTLY (`{action: accept, content: {answers: {"Which phase?": "Verify 1"}}}`
— a mismatch fails the run). Evidence: `experiments/captures/v022-ask-dialog.png`
(the dialog) + `experiments/captures/v022-ask-answered.png` (the resolved state).

## Test 2 — the real-machine acceptance (the user's /gsd-manager)

awaiting — the user restarts their Orca chat session (the binary with the bridge is
installed) and re-runs /gsd-manager: the workflow's choice dialogs must now RENDER
in the card, and answering one feeds the workflow (the field symptom "окно в третий
раз отклонено" must be gone).
