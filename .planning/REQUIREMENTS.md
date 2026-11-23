# Requirements — v0.22.0 Ask-the-User Bridge

> From the user's field report (2026-11-23): the gsd-manager workflow's choice
> dialogs (`AskUserQuestion`) never reached the Orca UI — the wrapper's standing
> `requestUserInput` policy row auto-declined them (responder.go:532, the
> "elicitation deferred" Phase 5/6 stub).

## v0.22.0 Requirements

- [ ] **UIB-01**: `interaction/requestUserInput` with the AskUserQuestion shape
      (`questions: [{question, header, options: [{value=label, label,
      description?, preview?}], multiSelect?}]` — zcode-src
      `interaction-broker.ts` `mapAskUserQuestion` + `zcodeUserInputQuestionSchema`)
      is bridged to ONE ACP form elicitation rendered by the client (Orca declares
      `clientCapabilities.elicitation.form` for the omp card)
- [ ] **UIB-02**: the elicitation form carries one property per question
      (index-keyed, duplicate-question safe): single-select → string enum of the
      option labels (titled oneOf when descriptions exist); multiSelect → array of
      label enums; the prompt rides the form message
- [ ] **UIB-03**: the accept outcome maps back to the dialect's response
      `{action: "accept", content: {answers: {<question text>: <option label>}}}`
      (multi-select joined ", ") — the merge contract
      `userInputResponseToBrokerResult`/`normalizeAskUserQuestionResponseContent`
      merges `answers` over the tool input; decline/cancel map verbatim
- [ ] **UIB-04**: without a bound elicitation sender (legacy/tests) the honest
      decline stays; an emission failure never fails the turn — it answers decline
      with the reason and logs structurally
- [ ] **UIB-05**: field-verified: the choice dialog RENDERS in Orca and answering
      it feeds the workflow (VM GUI pass + the user's /gsd-manager)

## Traceability

| ID | Phase | Status |
|----|-------|--------|
| UIB-01 | Phase 9 | Pending |
| UIB-02 | Phase 9 | Pending |
| UIB-03 | Phase 9 | Pending |
| UIB-04 | Phase 9 | Pending |
| UIB-05 | Phase 9 | Pending |
