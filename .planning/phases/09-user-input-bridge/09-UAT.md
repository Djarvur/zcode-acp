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

**PASSED (2026-11-23 evening)** — the user confirmed in live Orca use: "вроде бы все
работает". The field trail the user exercised end-to-end on their machine:
- the choice dialogs render and feed the workflow (the "окно в третий раз отклонено"
  symptom closed);
- the model catalog: 19 entries right after start, provider labels on every entry,
  and the list STAYS FULL across model switches (deepseek) and turns — the
  switch-path merge (91e86f3) closed the last shrink.

The fixes that landed between the field report and the acceptance (all live-verified):
the probe poke's reasoning level, the session/new catalog preservation, the
coding-plan catalog binding, the silent auth for discovery sessions, the provider
labels, and the switch-path merge. Commits 817f83e / ed9b157 / d3dac1b / 91e86f3.
