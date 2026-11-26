//go:build !windows

package acpfront

import "time"

// autoAuthBudget bounds the silent authenticate at session/new: the
// client's discovery scans wait ~9 s, so the all-local chain (the keyring
// probe, the PAT mint, the read-back) must fit well inside; a slower
// surface answers the honest builtin scope instead.
const autoAuthBudget = 7 * time.Second
