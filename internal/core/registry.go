// Package core holds the session-boundary state shared across the ACP
// front (ARCHITECTURE "Session registry (ACP sid ↔ machine)"): one
// SessionRegistry maps ACP session ids to their turn machinery under a
// mutex, so every front handler resolves the same Session for a session
// id no matter which request carries it.
//
// ARCHITECTURE also places the TurnTranslator in this package in later
// phases (Open Question 5 resolution, 04-RESEARCH): the registry lives in
// core from Phase 4 onward; the translator joins it when the front grows
// past the minimal chat loop (RESEARCH recommendation, plan 04-02).
package core

import (
	"context"
	"errors"
	"sync"

	"github.com/Djarvur/zcode-acp/internal/turn"
)

// ErrNoSession is the lookup-miss sentinel: Get fails loud instead of
// returning an anonymous zero Session — a prompt or cancel for an
// unknown id is a programming/wire error the caller must surface.
var ErrNoSession = errors.New("core: no session registered under id")

// Session is one ACP session's machinery: the backend-side turn machine
// and the per-turn cancel func the front owns (Pattern 5 turn-context
// ownership). Subscribed records whether session/subscribe has been
// issued — session/event pushes are gated on it (RESEARCH Pitfall 1).
type Session struct {
	ID         string
	Machine    *turn.Machine
	CancelTurn context.CancelFunc
	Subscribed bool
}

// SessionRegistry is the mutex-guarded session map (ARCHITECTURE
// Component Responsibilities). Register is last-wins: re-registering an
// id replaces its machinery — the front is the only writer and ids are
// app-server-generated opaque strings (1:1, no alias table).
type SessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]Session
}

// NewSessionRegistry returns an empty registry.
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{}
}

// RED stub (04-02 TDD): the map lands in the GREEN step.
func (r *SessionRegistry) Register(s Session) {}

// RED stub (04-02 TDD): answers the sentinel session until GREEN.
func (r *SessionRegistry) Get(sessionID string) (Session, error) {
	return Session{ID: "stub"}, nil
}

// RED stub (04-02 TDD): Remove is a no-op until GREEN.
func (r *SessionRegistry) Remove(sessionID string) {}
