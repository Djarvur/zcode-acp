package acpfront

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The delta tap (RESEARCH Pattern 2): the Front's face on the conn's
// ordered observer fan-out, registered BEFORE any machine (front.go New —
// registration order is delivery order, RESEARCH Pattern 3). One backend
// session/event envelope with type part.delta becomes exactly ONE chunk in
// arrival order — field text maps to agent_message_chunk, field reasoning
// to agent_thought_chunk (CORE-03/CORE-04) — never synthesized, never an
// upsert (Pitfall 5: upserts carry cumulative part text and would
// duplicate every delta).
//
// TOOL-01 (plan 05-04): the tap extends to the tool.updated discriminated
// union — one backend envelope maps to exactly ONE neutral ToolNote
// (scheduled → the tool_call start with the Edit/Write diff recipe and the
// cwd-resolved absolute path; started/progress → in_progress patches;
// result → completed with the raw output and the diff re-emitted; error →
// failed), every client-visible tool card traceable to one received
// envelope (the TOOL-01 honesty carry-over). batch/raw kinds and
// subagent/background mirrors stay log-only — never client cards on the
// top-level turn (misattributed child cards are a violation).
//
// IN-05: the turn slot records the active turn's backend turnId
// (record-on-first-sight, the machine.go discipline) and the tap skips
// BOTH part.delta and tool.updated envelopes whose turnId differs from
// the slot's — a foreign turn's deltas never reach the client.
//
// WR-03 (plan 05-04 Task 3): the tap never blocks the backend read loop —
// classification stays on the read goroutine, and the emit goes through
// the slot's per-turn pump (turnctx.go), which drains to the ACP wire at
// wire pace. Until Task 3's pump lands the emit is the slot's inline
// emitter closure (the Phase-4 shape).

// part.delta kind and field literals (zcode-src schema, RESEARCH Pattern 2
// verbatim: field ∈ {"text","reasoning","input","output"}).
const (
	kindPartDelta = "part.delta"

	deltaFieldText      = "text"      // → agent_message_chunk (acpport.NoteMessage)
	deltaFieldReasoning = "reasoning" // → agent_thought_chunk (acpport.NoteThought, CORE-04)
	deltaFieldInput     = "input"     // tool inputs ride tool.updated — skipped here
	deltaFieldOutput    = "output"    // tool outputs ride tool.updated result — skipped here

	// The tool.updated discriminated-union kinds (05-RESEARCH.md Pattern 4
	// taxonomy table, zcode-src index.ts:1758-1812 verbatim literals).
	toolKindScheduled = "scheduled"
	toolKindStarted   = "started"
	toolKindProgress  = "progress"
	toolKindResult    = "result"
	toolKindError     = "error"
	toolKindBatch     = "batch"
	toolKindRaw       = "raw"

	// The subagent-mirror marker (05-RESEARCH.md Pattern 4: source
	// "subagent" or background true — log-only on the single-agent v1 card).
	toolSourceSubagent = "subagent"

	// Tool-input keys (zcode-src edit.ts:86 / write.ts:216 verbatim wire
	// keys — camelCase file_path per the capture-quoted input schemas).
	toolInputFilePath = "file_path"
	toolInputOldText  = "old_string"
	toolInputNewText  = "new_string"
	toolInputContent  = "content"
)

// observeEnvelope is the tap body. It runs on the conn's read goroutine
// (the hook contract: delivery is synchronous and must not block — the tap
// parses small params and classifies one note), AHEAD of the machine's
// ObserveEnvelope, so a terminal event can never resolve the turn ahead of
// earlier delta envelopes. Routing keys on the front's single active turn
// (the single-active-turn invariant): envelopes outside an active prompt
// turn are observation only.
func (f *Front) observeEnvelope(env zcode.Envelope) {
	if env.Method != zcode.MethodSessionEvent {
		return
	}

	// The envelope header keys dig out camelCase-safe (the tagliatelle
	// contract: production decode structs stay snake-safe — single-word
	// keys only here, the payload digs through raw maps below).
	var params struct {
		Type    string          `json:"type"`
		Field   string          `json:"field"`
		Delta   string          `json:"delta"`
		TurnID  string          `json:"turnId"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil {
		return
	}

	switch params.Type {
	case kindPartDelta:
		f.observePartDelta(params.Field, params.Delta, params.TurnID)
	case "tool.updated":
		f.observeToolUpdated(params.Payload, params.TurnID)
	}
}

// observePartDelta maps one part.delta envelope onto its chunk note
// (CORE-03/CORE-04): field text → NoteMessage, field reasoning →
// NoteThought. field input/output are tool surfaces that ride tool.updated
// — silently skipped here, never synthesized (D-04 honesty).
func (f *Front) observePartDelta(field, delta, turnID string) {
	var kind acpport.SessionNoteKind

	switch field {
	case deltaFieldText:
		kind = acpport.NoteMessage
	case deltaFieldReasoning:
		// CORE-04: backend reasoning deltas are the ONLY source of thought
		// chunks — the acpport sink maps the note onto
		// acp.UpdateAgentThoughtText (the executed-probe wire shape).
		kind = acpport.NoteThought
	default:
		return
	}

	emit, scoped := f.turnScope(turnID)
	if !scoped {
		return // pre-turn, foreign-turn, or slotless envelope — observation only
	}

	// Exactly ONE chunk per part.delta envelope, in arrival order, on the
	// turn context (alive until the prompt returns — post-cancel drain
	// still reaches the client, RESEARCH Pattern 5). A failed update write
	// (a stalled or closing ACP wire) is recorded structurally — never
	// with params/content — and the turn continues; the prompt answer
	// carries the honest outcome.
	note := acpport.SessionNote{Kind: kind, Delta: delta}
	if err := emit(note); err != nil {
		slog.Warn("acpfront: session update write failed", "err", err)
	}
}

// observeToolUpdated classifies one tool.updated payload into at most one
// neutral ToolNote and emits it through the turn scope (TOOL-01). The
// batch/raw kinds and subagent/background mirrors are held from the client
// surface with a structural log (kinds and ids only — never content); an
// undecodable payload or an unknown kind is observation only — nothing is
// ever synthesized.
func (f *Front) observeToolUpdated(payload json.RawMessage, turnID string) {
	p, ok := decodeToolPayload(payload)
	if !ok {
		return
	}

	if heldToolEnvelope(p) {
		// WR-06-adjacent structural posture: log-only surfaces carry kinds
		// and id keys only — never input payloads, never titles.
		slog.Debug("acpfront: tool envelope held from the client surface",
			"kind", p.Kind, "source", p.Source, "toolCallID", p.ToolCallID)

		return
	}

	emit, cwd, scoped := f.turnScopeWithCwd(turnID)
	if !scoped {
		return
	}

	note, held := f.toolNote(p, cwd)
	if note == nil {
		if held {
			slog.Warn("acpfront: unsupported tool.updated kind held from the client", "kind", p.Kind)
		}

		return
	}

	if err := emit(*note); err != nil {
		slog.Warn("acpfront: session update write failed", "err", err)
	}
}

// heldToolEnvelope reports the log-only arms (05-RESEARCH.md Pattern 4
// mapping table): batch/raw have no ACP equivalent, and subagent/background
// mirrors must never render as client tool cards on the top-level turn.
func heldToolEnvelope(p toolPayload) bool {
	return p.Kind == toolKindBatch || p.Kind == toolKindRaw ||
		p.Source == toolSourceSubagent || p.Background
}

// toolNote builds the neutral note for one client-visible tool.updated
// kind (scheduled/started/progress/result/error). It returns (nil, true)
// for kinds that must be held with a warn (unknown kinds — never
// synthesized), (nil, false) for silent observation, and the note
// otherwise. The scheduled arm records the diff recipe in the turn slot so
// the result arm can re-emit it (clients render early; ACP update content
// merges per spec).
func (f *Front) toolNote(p toolPayload, cwd string) (*acpport.SessionNote, bool) {
	switch p.Kind {
	case toolKindScheduled:
		note := &acpport.SessionNote{
			Kind: acpport.NoteTool,
			Tool: &acpport.ToolNote{
				ToolCallID: p.ToolCallID,
				IsStart:    true,
				Title:      firstNonEmpty(p.Description, p.ToolName),
				Name:       p.ToolName,
				Kind:       toolKindFor(p.ToolName),
				Status:     acpport.ToolStatusPending,
				RawInput:   p.Input,
			},
		}

		if diff := toolDiffFor(p.ToolName, p.Input, cwd); diff != nil {
			note.Tool.Diff = diff
			f.recordTurnDiff(p.ToolCallID, *diff)
		}

		if fp := inputString(p.Input, toolInputFilePath); fp != "" {
			note.Tool.Locations = []acpport.ToolLocation{{Path: resolveAbs(cwd, fp)}}
		}

		return note, false
	case toolKindStarted, toolKindProgress:
		if p.Kind == toolKindProgress {
			// v1-minimal mapping: the tail/preview fields (elapsedMs, pid,
			// stdout/stderr/output tails) are logged structurally, never
			// emitted (05-RESEARCH.md Pattern 4 progress row).
			slog.Debug("acpfront: tool progress detail held (v1 maps status only)", "toolCallID", p.ToolCallID)
		}

		return updateNote(p.ToolCallID, acpport.ToolStatusInProgress, nil, nil), false
	case toolKindResult:
		diff, _ := f.turnDiffFor(p.ToolCallID)

		return updateNote(p.ToolCallID, acpport.ToolStatusCompleted, p.Result, &diff), false
	case toolKindError:
		return updateNote(p.ToolCallID, acpport.ToolStatusFailed, nil, nil), false
	default:
		return nil, true
	}
}

// updateNote builds one tool_call_update note carrying only the populated
// fields (patch semantics — the sink maps populated fields onto the wire
// patch, everything else stays unset).
func updateNote(id, status string, rawOutput json.RawMessage, diff *acpport.ToolDiff) *acpport.SessionNote {
	return &acpport.SessionNote{
		Kind: acpport.NoteTool,
		Tool: &acpport.ToolNote{
			ToolCallID: id,
			Status:     status,
			RawOutput:  rawOutput,
			Diff:       diff,
		},
	}
}

// toolPayload is one decoded tool.updated payload (the discriminated union
// on kind). The camelCase wire keys dig out through a raw map — the
// tagliatelle contract keeps production decode structs snake-safe (the
// interact/permission.go discipline).
type toolPayload struct {
	Kind        string
	ToolCallID  string
	ToolName    string
	Description string
	Source      string
	Background  bool
	Input       json.RawMessage
	Result      json.RawMessage
}

// decodeToolPayload digs the classification-relevant keys out of one
// tool.updated payload; an undecodable payload decodes as not-ok —
// observation only, never a synthesized note.
func decodeToolPayload(raw json.RawMessage) (toolPayload, bool) {
	var obj map[string]json.RawMessage

	if err := json.Unmarshal(raw, &obj); err != nil {
		return toolPayload{}, false
	}

	p := toolPayload{
		Kind:        rawString(obj["kind"]),
		ToolCallID:  rawString(obj["toolCallId"]),
		ToolName:    rawString(obj["toolName"]),
		Description: rawString(obj["description"]),
		Source:      rawString(obj["source"]),
		Input:       obj["input"],
		Result:      obj["result"],
	}

	if b, err := rawBool(obj["background"]); err == nil {
		p.Background = b
	}

	return p, true
}

// toolKindFor maps the backend toolName onto the ACP ToolKind vocabulary
// (05-RESEARCH.md Pattern 4 kind table): case-insensitive match, the
// `other` default (the SDK decodes unknown kinds as ToolKindOther — the
// mapping table is data; unlisted names render acceptably).
func toolKindFor(name string) string {
	switch strings.ToLower(name) {
	case "read":
		return acpport.ToolKindRead
	case "edit", "write":
		return acpport.ToolKindEdit
	case "glob", "grep":
		return acpport.ToolKindSearch
	case "bash":
		return acpport.ToolKindExecute
	case "webfetch":
		return acpport.ToolKindFetch
	case "todoread", "todowrite":
		return acpport.ToolKindThink
	default:
		return acpport.ToolKindOther
	}
}

// toolDiffFor renders the Edit/Write input recipes (D-03, 05-RESEARCH.md
// Pattern 4): Edit {file_path, old_string, new_string} → a full diff;
// Write {file_path, content} → newText only (the new-file shape). The
// path resolves ABSOLUTE against the session cwd before emitting
// (Pitfall 5). Other tool names carry no diff — nil.
func toolDiffFor(name string, input json.RawMessage, cwd string) *acpport.ToolDiff {
	switch strings.ToLower(name) {
	case "edit":
		oldText := inputString(input, toolInputOldText)
		newText := inputString(input, toolInputNewText)

		if oldText == "" && newText == "" {
			return nil
		}

		return &acpport.ToolDiff{
			Path:    resolveAbs(cwd, inputString(input, toolInputFilePath)),
			OldText: oldText,
			NewText: newText,
		}
	case "write":
		content := inputString(input, toolInputContent)
		if content == "" {
			return nil
		}

		return &acpport.ToolDiff{
			Path:    resolveAbs(cwd, inputString(input, toolInputFilePath)),
			NewText: content,
		}
	default:
		return nil
	}
}

// resolveAbs resolves one backend-supplied path ABSOLUTE against the
// session cwd (the ACP diff/location paths MUST be absolute; relative
// paths render broken client cards — Pitfall 5). An already-absolute path
// passes through; without a known cwd the path passes through unchanged
// (the front's session/new validates an absolute cwd, so the known-cwd
// path is the production path).
func resolveAbs(cwd, path string) string {
	switch {
	case path == "", filepath.IsAbs(path):
		return path
	case cwd == "":
		return path
	default:
		return filepath.Join(cwd, path)
	}
}

// inputString digs one string member out of a raw tool-input object;
// absent or non-string members decode as "".
func inputString(input json.RawMessage, key string) string {
	if len(input) == 0 {
		return ""
	}

	var obj map[string]json.RawMessage

	if err := json.Unmarshal(input, &obj); err != nil {
		return ""
	}

	return rawString(obj[key])
}

// rawString decodes one raw JSON member as a string; non-string members
// decode as "" (the interact/permission.go discipline).
func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string

	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}

	return s
}

// rawBool decodes one raw JSON member as a bool.
func rawBool(raw json.RawMessage) (bool, error) {
	var b bool

	if err := json.Unmarshal(raw, &b); err != nil {
		return false, err
	}

	return b, nil
}

// firstNonEmpty returns the first non-empty string (the title recipe:
// description else toolName).
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}

	return ""
}
