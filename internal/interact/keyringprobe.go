package interact

import (
	"context"
	"errors"
)

// ErrKeyringNoEntry is the keyring probe's coded log-in-via-desktop hint:
// the keyring holds no matching ZCode credential.
var ErrKeyringNoEntry = errors.New(
	"keyring probe: no ZCode desktop login found in the keyring (log in via the ZCode desktop first)")

// KeyringProber is the proactive keyring availability seam (OQ3 lock): a
// function type on the authenticate config. Each platform constructor builds
// it over its own credential store; tests inject canned verdicts (the
// KeyLoader injection discipline). nil error = the keyring path is usable;
// every non-nil error is a coded sentinel (possibly wrapped with coarse
// context — never a value).
type KeyringProber func(ctx context.Context) error

// The structural-log codes shared by every platform's prober.
const (
	probeClassOK      = "ok"
	probeClassNoEntry = "no-entry"
)
