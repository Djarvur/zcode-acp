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
// ownership). The subscribe-before-any-turn gate is enforced by the
// D-03 sequence in acpfront.NewSession (front.go) — the registry carries
// no subscription flag (IN-02: a dead field is an API lie; delete unless
// a consumer reads it — Phase 5's resume path reads neither).
type Session struct {
	ID      string
	Machine *turn.Machine
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

// Register stores (or last-wins replaces) one session's machinery.
func (r *SessionRegistry) Register(s Session) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions == nil {
		r.sessions = make(map[string]Session)
	}

	r.sessions[s.ID] = s
}

// Get resolves one session's machinery, failing loud with ErrNoSession
// on a lookup miss — a prompt or cancel for an unknown id is an error
// the caller must surface, never an anonymous zero Session.
func (r *SessionRegistry) Get(sessionID string) (Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[sessionID]
	if !ok {
		return Session{}, ErrNoSession
	}

	return s, nil
}

// Remove drops one session's machinery; removing an unknown id is a
// no-op (cancel-after-complete symmetry — repeats never panic).
func (r *SessionRegistry) Remove(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, sessionID)
}
