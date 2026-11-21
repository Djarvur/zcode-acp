package hostsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// readbackPollInterval is the Await poll cadence — the surface check is a
// cheap mutex-guarded read of the latched verdict.
const readbackPollInterval = 2 * time.Millisecond

// DefaultReadbackDeadline bounds the registry read-back when the caller's
// context carries no deadline (poll-with-deadline; tests inject shorter
// bounds via their context).
const DefaultReadbackDeadline = 30 * time.Second

// ErrRegistryNotConfirmed is the PITFALL-1 sentinel: the read-back deadline
// elapsed without the registry surfaces confirming the expected provider
// models — the received ack is not application (the schema's own comment:
// 收到账号结果不代表配套 Built-in 已到达).
var ErrRegistryNotConfirmed = errors.New("registry read-back not confirmed")

// Readback is the PITFALL-1 verdict over the registry surfaces: Applied is
// true only when an observed surface carried every wanted provider model;
// Revision carries the matched surface's revision echo in string form.
type Readback struct {
	Applied  bool
	Revision string
}

// WatchOption configures a RegistryWatcher at construction.
type WatchOption func(*RegistryWatcher)

// WithReadbackDeadline overrides the poll deadline (injectable, test-short).
func WithReadbackDeadline(d time.Duration) WatchOption {
	return func(w *RegistryWatcher) { w.deadline = d }
}

// RegistryWatcher awaits the registry read-back surfaces (PITFALL 1): the
// state.updated notification stream and, explicitly offered, fresh
// session/create replies (dialect §8's read-back surface). The verdict is
// the registry's — a received provider/updateAccountConfig ack is never
// consulted.
type RegistryWatcher struct {
	conn     *zcode.Conn
	want     []string
	deadline time.Duration

	mu       sync.Mutex
	applied  bool
	revision string
}

// WatchRegistry registers the watcher's surface observer and returns the
// watcher with its observer-remove func (transient observer; defer the
// remove when the auth flow completes). Registration must precede the
// traffic whose surfaces it awaits (the conn hook contract).
func WatchRegistry(conn *zcode.Conn, want []string, opts ...WatchOption) (w *RegistryWatcher, remove func()) {
	w = &RegistryWatcher{conn: conn, want: want, deadline: DefaultReadbackDeadline}

	for _, opt := range opts {
		opt(w)
	}

	return w, conn.AddObserver(w.observe)
}

// OfferCreateReply feeds one explicit create-reply surface: a fresh
// session/create result the caller already holds — the dialect §8 read-back
// surface (settings.model.available) which the conn's notification stream
// does not carry.
func (w *RegistryWatcher) OfferCreateReply(res json.RawMessage) {
	w.considerSurface(res)
}

// Applied reports whether any registry surface has already latched the
// verdict — the corrected authenticate chain's probe gate (plan 05-11):
// the probe-create observation runs only when no patch confirmed first
// (the state.updated observer stays for patches the desktop does emit; on
// real-desktop 3.18.2 none fires for push+refresh, EXP-READBACK-DIAG-01).
func (w *RegistryWatcher) Applied() bool {
	_, ok := w.latched()

	return ok
}

// Await polls the latched verdict with a deadline: the effective window is
// the shorter of the watcher deadline and the context's (injectable,
// test-short). Applied true the moment any surface matched; the coded
// ErrRegistryNotConfirmed error otherwise — never an ack-derived verdict.
func (w *RegistryWatcher) Await(ctx context.Context) (Readback, error) {
	if rb, ok := w.latched(); ok {
		return rb, nil
	}

	deadline := w.deadline
	if dl, ok := ctx.Deadline(); ok {
		if until := time.Until(dl); until < deadline {
			deadline = until
		}
	}

	timer := time.NewTimer(deadline)
	defer timer.Stop()

	tick := time.NewTicker(readbackPollInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return Readback{}, w.notConfirmed(ctx.Err())
		case <-timer.C:
			return Readback{}, w.notConfirmed(context.DeadlineExceeded)
		case <-tick.C:
			if rb, ok := w.latched(); ok {
				return rb, nil
			}
		}
	}
}

// observe is the conn-notification face — runs on the read goroutine and
// must not block (the machine.go observer discipline).
func (w *RegistryWatcher) observe(env zcode.Envelope) {
	w.considerSurface(env.Params)
}

// considerSurface checks one registry-surface payload; the first surface
// carrying every wanted model latches the verdict exactly once.
func (w *RegistryWatcher) considerSurface(raw json.RawMessage) {
	if !containsAllModels(registryModels(raw), w.want) {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.applied {
		return
	}

	w.applied = true
	w.revision = surfaceRevision(raw)
}

// latched snapshots the verdict under the mutex.
func (w *RegistryWatcher) latched() (Readback, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.applied {
		return Readback{}, false
	}

	return Readback{Applied: true, Revision: w.revision}, true
}

// notConfirmed renders the coded, actionable negative: what was awaited,
// the cause, and the PITFALL-1 discipline (the received ack is not
// application; the field-verified read-back surface on desktop 3.18.2 is
// the create reply's settings.model.available — no state.updated patch
// fires for push+refresh, EXP-READBACK-DIAG-01 — so check the snapshot's
// basedOn revision against the active builtin, the silent-discard shape
// dialect §8 records).
func (w *RegistryWatcher) notConfirmed(cause error) error {
	return fmt.Errorf("%w: models %v not observed within the read-back window (%v) — "+
		"the received ack is not application; the field-verified read-back surface is "+
		"the create reply's settings.model.available (no state.updated patch fires for "+
		"push+refresh on desktop 3.18.2, EXP-READBACK-DIAG-01); verify the snapshot's "+
		"basedOnZCodeBuiltinRevision matches the active builtin revision",
		ErrRegistryNotConfirmed, w.want, cause)
}

// AwaitRegistryContains is the one-shot convenience: register a transient
// watcher and poll-with-deadline until the registry surfaces carry want.
// Callers driving traffic through the same conn should WatchRegistry FIRST
// (registration before traffic — the hook contract): a surface emitted
// between the last request's reply and this call's registration is missed.
func AwaitRegistryContains(
	ctx context.Context,
	conn *zcode.Conn,
	want []string,
	opts ...WatchOption,
) (Readback, error) {
	w, remove := WatchRegistry(conn, want, opts...)
	defer remove()

	return w.Await(ctx)
}

// registryModels digs the provider model identifiers out of a registry
// surface payload, normalized to providerId/modelId strings: create replies
// carry settings.model.available[].ref.{providerId, modelId}; state.updated
// registry patches carry patch.providers.<id>.models[]. Undecodable
// payloads classify as no models (never a verdict).
func registryModels(raw json.RawMessage) []string {
	out := digCreateAvailable(raw)
	if out == nil {
		out = digPatchProviders(raw)
	}

	return out
}

// digCreateAvailable digs settings.model.available[].ref out of a create
// reply result (the map dig-out keeps the capture-verbatim camelCase wire
// keys — the tagliatelle contract).
func digCreateAvailable(raw json.RawMessage) []string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}

	settings, _ := obj["settings"].(map[string]any)

	model, _ := settings["model"].(map[string]any)

	available, _ := model["available"].([]any)
	if len(available) == 0 {
		return nil
	}

	out := make([]string, 0, len(available))

	for _, entry := range available {
		avail, _ := entry.(map[string]any)

		ref, _ := avail["ref"].(map[string]any)

		pid, _ := ref["providerId"].(string)
		mid, _ := ref["modelId"].(string)

		if pid == "" && mid == "" {
			continue
		}

		out = append(out, pid+"/"+mid)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// digPatchProviders digs patch.providers.<id>.models[] out of a
// state.updated registry patch.
func digPatchProviders(raw json.RawMessage) []string {
	var params struct {
		Patch struct {
			Providers map[string]struct {
				Models []string `json:"models"`
			} `json:"providers"`
		} `json:"patch"`
	}

	if err := json.Unmarshal(raw, &params); err != nil || len(params.Patch.Providers) == 0 {
		return nil
	}

	out := make([]string, 0, len(params.Patch.Providers))

	for pid, entry := range params.Patch.Providers {
		for _, m := range entry.Models {
			out = append(out, pid+"/"+m)
		}
	}

	return out
}

// surfaceRevision digs the surface's revision echo in string form (the
// numeric registry revision vocabulary on state.updated patches; a string
// revision passes through verbatim).
func surfaceRevision(raw json.RawMessage) string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}

	switch v := obj["revision"].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	}

	return ""
}

// containsAllModels reports whether got carries every want model.
func containsAllModels(got, want []string) bool {
	if len(want) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(got))
	for _, g := range got {
		set[g] = struct{}{}
	}

	for _, m := range want {
		if _, ok := set[m]; !ok {
			return false
		}
	}

	return true
}
