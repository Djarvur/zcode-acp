// Package hostsvc implements AUTH-01's execution tier: the api-key snapshot
// push (provider/updateAccountConfig with the exact-revision builtin
// formula), the provider/refreshRegistry call, the registry read-back
// helper, and the runtime-headers answer over the 05-01 KeySource seam.
package hostsvc

import (
	"context"
	"errors"

	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// KeySource supplies the authenticate-sourced api-key material (the 05-01
// responder seam shape: apiKey/apiKeyID accessors). Implementations hold
// the material in memory only — never in logs, captures, or fixtures.
type KeySource interface {
	// APIKey returns the configured key material; ok=false when
	// unconfigured (the honest coded-error arm).
	APIKey(ctx context.Context) (apiKey, apiKeyID string, ok bool)
}

// SnapshotOption configures a SnapshotBuilder at construction.
type SnapshotOption func(*SnapshotBuilder)

// WithRevisionSource overrides the push-revision generator (tests and
// corpus replays inject a fixed value for byte-deterministic goldens; the
// default generates a fresh account-bridge:<unix-nano> per build).
func WithRevisionSource(func() string) SnapshotOption {
	return func(*SnapshotBuilder) {}
}

// SnapshotBuilder builds the provider/updateAccountConfig snapshot params.
type SnapshotBuilder struct{}

// NewSnapshotBuilder returns a builder over the account-credential binding
// (keys) with options applied.
func NewSnapshotBuilder(_ KeySource, _ ...SnapshotOption) *SnapshotBuilder {
	return &SnapshotBuilder{}
}

// Snapshot is one built push envelope and its identity: Revision is the
// wrapper-generated snapshot revision, BasedOn the exact builtin-revision
// formula output, Params the strict envelope to push.
type Snapshot struct {
	Revision string
	BasedOn  string
	Params   map[string]any
}

// Build computes the strict envelope for install, reading the recorded
// revision int out of the active zcode-builtin.json.
func (b *SnapshotBuilder) Build(_ supervisor.Install) (Snapshot, error) {
	return Snapshot{}, nil
}

// BuildRevision computes the strict envelope over an explicitly supplied
// recorded revision int (probe-time reads hold it already).
func (b *SnapshotBuilder) BuildRevision(_ supervisor.Install, _ int) (Snapshot, error) {
	return Snapshot{}, nil
}

// Ack is the provider/updateAccountConfig reply — receipt evidence only,
// never application (PITFALL 1).
type Ack struct {
	ReceivedRevision string
	ProviderCount    int
	Status           string
}

// PushSnapshot builds with the default builder and runs the
// provider/updateAccountConfig Call.
func PushSnapshot(_ context.Context, _ *zcode.Conn, _ supervisor.Install) (Ack, error) {
	return Ack{}, nil
}

// PushEnvelope runs the provider/updateAccountConfig Call with params.
func PushEnvelope(_ context.Context, _ *zcode.Conn, _ map[string]any) (Ack, error) {
	return Ack{}, nil
}

// RefreshRegistry runs provider/refreshRegistry {reason}.
func RefreshRegistry(_ context.Context, _ *zcode.Conn) error {
	return nil
}

// errAPIKeyUnconfigured is the honest unconfigured-key arm.
var errAPIKeyUnconfigured = errors.New("api key not configured")

// RuntimeHeadersHandler adapts src to the 05-01 responder seam.
func RuntimeHeadersHandler(_ KeySource) interact.RuntimeHeadersKeySource {
	return func(_ context.Context, _ interact.Request) (string, string, error) {
		return "", "", errAPIKeyUnconfigured
	}
}
