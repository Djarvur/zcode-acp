package acpfront

// Per-turn context ownership (RESEARCH Pattern 5, CORE-05): each prompt
// turn runs on a FRONT-OWNED context derived from the Front's base
// context — never the SDK request context. The SDK maps a handler's
// context.Canceled to a -32800 RequestCancelled ERROR response, while ACP
// requires session/cancel answered with stopReason:"cancelled" as a
// SUCCESS result — so the turn context is cancelled only when the prompt
// returns, and cancellation reaches the turn exclusively through
// machine.Cancel (idempotent, exactly once: Phase 3), which resolves
// StopCancelled via the observed stop outcome. A cancel fan-in goroutine
// selects {SDK ctx.Done → machine.Cancel; turn ctx.Done → return} so both
// cancel sources trigger exactly one machine.Cancel, and post-cancel
// delta drain still reaches the client because the tap writes on the turn
// context (alive until the prompt returns).
//
// GREEN (04-06) lands acquireTurn here: the front-wide slot guard plus the
// turn-context construction the tap and Prompt share.
