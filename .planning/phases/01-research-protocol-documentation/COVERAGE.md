# API Coverage Decision — Phase 1

**Detector verdict (plan time, 2026-11-19):**

- Roadmap Phase 1 section alone: `{"detected":false,"signals":[]}` (exit 0).
- Roadmap section + all six PLAN bodies: `{"detected":true,"signals":[{"verb":"(surface)","noun":"api", …}]}` —
  the detector fired AFTER the plan bodies were included.

**Honest verdict after re-reading the phase scope: the detected=true signal is a lexical false
positive; this phase genuinely integrates no external API.** The strings that tripped it are:

1. Threat-register prohibition prose — "No flow ever POSTs api_keys", the quota-discipline rules,
   and the api.z.ai reachability *fallback note* in E2. These name the endpoint in order to
   FORBID touching it (T-1-03 mitigation), not to integrate it.
2. "adopt-list" phrasing in the DISC-05 plan — "adopt" is a lexical verb match; it refers to
   adopt/avoid verdicts about a reference implementation studied read-only, not library adoption.
3. "surface" in section names ("event surface") — a documentation subject, not a wiring verb.

**Declaration:** No external API integration: this phase writes documentation and local
experiment harnesses; it implements no client against any external API/SDK/service. The only
machine interfaces it touches are (a) the shipped ZCode 3.18.2 app-server spawned locally over
stdio — the protocol being *documented*, not integrated — and (b) read-only study of external
repositories cloned to scratch (no code consumed at build or run time). Experiment E2's single
tiny turn travels the official binary's own channel; no HTTP/API client code is authored; the
one credential-adjacent endpoint in scope appears only inside a prohibition (never called).
Phase 1 installs zero external packages (no npm/pypi/cargo installs — RESEARCH.md
"Package Legitimacy Audit: not applicable").

Capability surface considered and consciously not integrated: none — there is no external API
client surface in this phase to enumerate or opt out of. The INTEGRATE-by-default rule has no
surface to apply to; an OPT-OUT matrix would enumerate a client that does not exist.

*This declaration is the seal-time input for `api-coverage.verify-pre` (which accepts a reasoned
declaration in place of a matrix when the detector fires but the scope genuinely integrates no
external API).*
