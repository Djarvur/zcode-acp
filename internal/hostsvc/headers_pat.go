// The mode-aware PAT runtime-headers source (plan 06-07, AUTH-02 on the
// coding-plan path — 06-RESEARCH.md Pattern 4): the ask-side half of the
// D-05 reactive lifecycle. Selection is the recorded auth mode (D-11): the
// coding-plan mode serves the PAT answer, every other mode (api-key, and
// the unauthenticated None — the static arm's unconfigured posture covers
// the pre-auth ask) delegates byte-compatibly to the existing static-key
// handler. Inside one coding-plan ask: the cached PAT answers with zero
// mints, a rejectedProjectTokenFingerprint equal to the cached fingerprint
// drops the cache and re-mints EXACTLY ONCE, a second consecutive rejection
// surfaces the honest coded error (never an infinite loop), and an
// expectedAccountScope mismatch refuses before any token is served (the
// desktop re-login case — a stale PAT would ride the old entitlement). Every
// failure answers the coded project_token-class error envelope — never the
// failure-shaped headersApplied:false success (PITFALL 3).

package hostsvc

import (
	"context"
	"errors"

	"github.com/Djarvur/zcode-acp/internal/interact"
)

// patModeCodingPlan is the auth-mode value the coding-plan arm records (the
// acpfront.AuthModeCodingPlan literal; hostsvc cannot import acpfront — the
// front imports hostsvc — so the selector contract is the documented string
// value and cmd wires front.AuthMode as the selector).
const patModeCodingPlan = "coding-plan"

// The ask-side coded stable errors (the project_token family — consumers
// match the sentinels, never the message). Tokens never appear in any
// message; fingerprints/scopes are digests by construction.
var (
	// ErrPatRejectionLoop: the re-minted token's fingerprint was rejected
	// AGAIN — the D-05 one-shot recovery is exhausted and the honest coded
	// error answers (never an infinite re-mint loop, T-06-23).
	ErrPatRejectionLoop = errors.New("project_token_rejected_after_re_mint")

	// errPatSourceUnwired: the coding-plan mode with no minter wired — the
	// coded missing-wiring posture (parity with the front's nil-Pat arm),
	// never a panic.
	errPatSourceUnwired = errors.New("project_token_source_unwired (no PAT minter wired)")
)

// patMintSource is the minter surface the source consumes — 06-02's locked
// Cached/Ensure/Invalidate orchestration surface (the D-05 re-mint mapping).
// *PatMinter satisfies it structurally; tests script doubles.
type patMintSource interface {
	// Cached returns the in-process cached PAT without any network I/O.
	Cached(ctx context.Context) (Pat, bool)

	// Ensure returns a current PAT, minting through the official chain when
	// the cache is stale or absent (single-flight).
	Ensure(ctx context.Context) (Pat, error)

	// Invalidate drops the cached token so the next Ensure re-mints.
	Invalidate(reason string)
}

// PatHeadersSourceDeps wires the mode-aware source: the minter surface, the
// auth-mode selector, and the api-key fallback (the existing
// RuntimeHeadersHandler-wrapped static source). All fields required — a nil
// field answers the coded missing-wiring error on the arm that needs it.
type PatHeadersSourceDeps struct {
	// Minter is the PAT cache/mint surface (production: *PatMinter).
	Minter patMintSource

	// Mode returns the recorded auth mode (production: front.AuthMode).
	Mode func() string

	// APIKeyFallback is the static-key source the non-coding-plan modes
	// delegate to byte-compatibly.
	APIKeyFallback interact.RuntimeHeadersKeySource
}

// PatHeadersSource builds the mode-aware RuntimeHeadersKeySource (plan
// 06-07): the coding-plan mode serves the PAT answer (cache, exactly-one
// re-mint, scope refusal), every other mode delegates to the api-key
// fallback. The exactly-one-re-mint bookkeeping is PER-ASK — a local
// drop-and-remint sequence inside the one handler invocation, never shared
// mutable retry state across asks.
func PatHeadersSource(deps PatHeadersSourceDeps) interact.RuntimeHeadersKeySource {
	return func(ctx context.Context, req interact.Request) (string, string, string, error) {
		// Skeleton posture (RED): full delegation to the static arm — the
		// coding-plan arms land in GREEN (plan 06-07 Task 2).
		return deps.APIKeyFallback(ctx, req)
	}
}
