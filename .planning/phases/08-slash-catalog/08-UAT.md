---
phase: "08"
slug: "slash-catalog"
status: complete
---

# UAT — v0.21.0 Phase 8 (slash catalog)

## Test 1 — the composer autocomplete lists the catalog (CAT-05, VM GUI pass)

**PASSED (2026-11-23, agent-executed per the standing VM-rendering delegation).**
On the qa02 clean VM (Orca + the omp card → the wrapper → the fake backend):
typing `/` in the composer opens the autocomplete menu listing
`/model — Choose the model` (the wrapper's entry with the `<selector>` hint
contract), the card's native commands, and the seeded catalog entries
`/gsd-capture — Capture and scrub a field experiment` and
`/gsd-manager [--analyze-deps] — Interactive command center for managing
multiple phases from one terminal` — names, descriptions and argument-hints
rendered. Evidence: `experiments/captures/v021-catalog-vm-menu.png`.

Prerequisites verified on the same pass: the card's agent entry shows OMP
available (the version floor holds with the new binary), the chat opens, the
catalog frame arrives within the discovery budget (the host-side card-argv
runs measured +2.1s…+5.5s, budget 9s).

## Test 2 — the real-machine acceptance (the user's pass)

awaiting — the user re-tests their Orca card (restart the chat session so a
fresh wrapper process spawns) and sees the autocomplete listing their real
commands/skills (139 items live-measured on this machine: the whole gsd-*
set, the user skills, `/model`).
