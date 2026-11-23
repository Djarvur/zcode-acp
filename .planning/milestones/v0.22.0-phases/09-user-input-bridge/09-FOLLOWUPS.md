# Phase 9 follow-ups — the field-driven fix set (2026-11-23, same day)

## The /model catalog fix (the user's report: "меню /model содержит ТОЛЬКО текущую модель")

Root-caused to TWO stacked behaviors, both wrapper-side, fixed together:

1. **The authenticate probe's setModel poke omitted the reasoning level** — the
   backend now refuses account models without one (`-32603 "Reasoning level is
   required"`) → the account provider never materialized → the registry held no
   account models. Fix: the poke digs the want model's own declared level from the
   probe create reply's settings snapshot (DefaultLevel, else the first level).
2. **The session/new reply's configOptions came from the REBIND reply's settings
   view** — the backend scopes `settings.model.available` to the session's CURRENT
   provider, so the rebind reply narrowed the catalog to the bound model's scope,
   overwriting the create snapshot's FULL unscoped catalog (19 entries: all the
   account families + zai-api + deepseek). Fix: `mergeCatalogEntries` — the
   confirmed current/last-used/thought-level ride the follow-up reply; the
   AVAILABLE catalog rides the create snapshot.

Verified live: session/new after authenticate carries **19 model entries** on the
real account (the full catalog, the bound Flash confirmed as current); v0.20.0's
binary shows 1 entry on the same probe (the pre-fix state, NOT a regression of
v0.21/22 — the probe-without-authenticate shape behaves identically on both).

The deferred family-materialization idea (poking EVERY family model to warm the
registry) was tried and reverted: the catalog preservation delivers the picker
without it; the backend materializes a model on demand when the user switches.
