---
plan: "09-01"
phase: "09"
slug: "user-input-bridge"
---
# 09-01 — The ask-the-user bridge: requestUserInput ↔ form elicitation

## Deliverable

Shipped: `interaction/requestUserInput` bridges to ONE client-rendered form
elicitation (the omp card declares `clientCapabilities.elicitation.form`), and the
outcome maps back into the dialect's `{action, content.answers}` merge contract
(accept content keyed by QUESTION TEXT with the option labels; multi joined ", ";
decline/cancel verbatim; an ask-surface failure declines with the reason — never
fails the turn; no policy deadline when bridged — a human dialog takes minutes).

## Wire truth chain

- zcode sources: the request/response schemas (the questions/options shape, options
  `value=label` via `mapAskUserQuestion`), the broker's dual-channel flow, and the
  merge contract (`userInputResponseToBrokerResult` + `normalizeAskUserQuestionResponseContent`).
- orca sources: the omp card declares `clientCapabilities.elicitation.form` — the
  card renders form elicitations natively.
- SDK v0.20.0: `CreateElicitationRequest{Form}` / `CreateElicitationResponse`
  accept/decline/cancel + the titled oneOf/array property variants.

## Verification

- Unit locks (TDD RED→GREEN): the decode (full shape, free-text, broken JSON), the
  build (single/multi/free-text, titled options, session scope), the outcome map
  (accept by question text, multi joined, decline/cancel), the responder row
  (bridged accept, asker-error decline, the legacy nil-asker decline).
- The composed binary E2E: the scripted requestUserInput reverse crosses as a FORM
  elicitation on the client surface (recorded + asserted) and the scripted decline
  rides back — production stdio, real binary.
- The field pass (VM GUI): the dialog rendered (message/question/labels/
  descriptions/Skip + "Awaiting user input"), the click resolved it, and the fake's
  scripted expect matched the bridged answer EXACTLY. Screenshots:
  `experiments/captures/v022-ask-dialog.png`, `v022-ask-answered.png`.
- `mise run ci` green; whole-tree lint 0 issues.
