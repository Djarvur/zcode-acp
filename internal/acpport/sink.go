package acpport

import (
	"context"

	"github.com/coder/acp-go-sdk"
)

// NewUpdateSink binds an UpdateSink to the agent-side connection: each
// note becomes exactly one session/update notification on the ACP wire —
// message notes via acp.UpdateAgentMessageText, thought notes via
// acp.UpdateAgentThoughtText (RESEARCH Pattern 2 mapping, executed-probe
// shapes). Notes are never synthesized here; the sink is a passthrough
// (D-04 honesty), and an unknown note kind fails loud instead of dropping.
func NewUpdateSink(conn *acp.AgentSideConnection) UpdateSink {
	return func(ctx context.Context, note SessionNote) error {
		notification, err := noteToNotification(note)
		if err != nil {
			return err
		}

		return conn.SessionUpdate(ctx, notification)
	}
}

// noteToNotification translates one neutral note onto the SDK's session
// update vocabulary; an unknown note kind is a loud coded error, never a
// dropped chunk.
func noteToNotification(note SessionNote) (acp.SessionNotification, error) {
	switch note.Kind {
	case NoteMessage:
		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update:    acp.UpdateAgentMessageText(note.Delta),
		}, nil
	case NoteThought:
		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update:    acp.UpdateAgentThoughtText(note.Delta),
		}, nil
	case NoteTool:
		if note.Tool == nil {
			return acp.SessionNotification{}, NewInternal("acpport: tool note carries no ToolNote payload")
		}

		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update:    toolUpdate(note.Tool),
		}, nil
	case NoteCatalog:
		commands := make([]acp.AvailableCommand, 0, len(note.Catalog))
		for _, item := range note.Catalog {
			cmd := acp.AvailableCommand{Name: item.Name, Description: item.Description}
			if item.Hint != "" {
				cmd.Input = &acp.AvailableCommandInput{
					Unstructured: &acp.UnstructuredCommandInput{Hint: item.Hint},
				}
			}

			commands = append(commands, cmd)
		}

		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update: acp.SessionUpdate{AvailableCommandsUpdate: &acp.SessionAvailableCommandsUpdate{
				SessionUpdate:     "available_commands_update",
				AvailableCommands: commands,
			}},
		}, nil
	case NoteConfigOptions:
		selects := configOptionSelects(note.ConfigOptions)
		if selects == nil {
			selects = []acp.SessionConfigOption{}
		}

		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update: acp.SessionUpdate{ConfigOptionUpdate: &acp.SessionConfigOptionUpdate{
				SessionUpdate: "config_option_update",
				ConfigOptions: selects,
			}},
		}, nil
	default:
		return acp.SessionNotification{}, NewInternal("acpport: unknown session note kind %q", note.Kind)
	}
}

// toolUpdate translates one neutral ToolNote onto the SDK's tool
// vocabulary (TOOL-01, Don't-Hand-Roll: the SDK helpers set the
// sessionUpdate discriminants and the patch semantics).
//
//   - IsStart → acp.StartToolCall: the tool_call frame carrying
//     title (required), plus Name/Kind/Status/RawInput/Content(diff)/
//     Locations set from the note whenever populated. Option order is
//     load-bearing: locations are appended BEFORE the raw input so the
//     SDK helper's rawInput.path backfill (WithStartLocations) is wholly
//     overwritten by the note's own RawInput — the input rides verbatim,
//     never augmented (the D-04 honesty carry-over).
//   - !IsStart → acp.UpdateToolCall: the tool_call_update patch carrying
//     ONLY the populated fields (patch semantics per the ACP spec — no
//     empty patch objects on the wire). The diff rides via content when
//     the note re-emits it (the result leg), and RawOutput maps the
//     backend result object verbatim.
//
// The diff renders through acp.ToolDiffContent with oldText omitted when
// empty (the new-file Write shape per the ACP tool-calls spec).
func toolUpdate(t *ToolNote) acp.SessionUpdate {
	if t.IsStart {
		return acp.StartToolCall(acp.ToolCallId(t.ToolCallID), t.Title, startOpts(t)...)
	}

	var opts []acp.ToolCallUpdateOpt

	if t.Status != "" {
		opts = append(opts, acp.WithUpdateStatus(acp.ToolCallStatus(t.Status)))
	}

	if content := toolContent(t.Diff); content != nil {
		opts = append(opts, acp.WithUpdateContent(content))
	}

	if t.RawOutput != nil {
		opts = append(opts, acp.WithUpdateRawOutput(t.RawOutput))
	}

	return acp.UpdateToolCall(acp.ToolCallId(t.ToolCallID), opts...)
}

// startOpts assembles the start frame's populated-field options in the
// order that keeps rawInput verbatim (see toolUpdate). Only populated
// fields map — unset note fields leave the corresponding wire fields out.
func startOpts(t *ToolNote) []acp.ToolCallStartOpt {
	var opts []acp.ToolCallStartOpt

	if t.Kind != "" {
		opts = append(opts, acp.WithStartKind(acp.ToolKind(t.Kind)))
	}

	if t.Status != "" {
		opts = append(opts, acp.WithStartStatus(acp.ToolCallStatus(t.Status)))
	}

	if len(t.Locations) > 0 {
		locations := make([]acp.ToolCallLocation, len(t.Locations))
		for i, loc := range t.Locations {
			locations[i] = acp.ToolCallLocation{Path: loc.Path}
		}

		opts = append(opts, acp.WithStartLocations(locations))
	}

	if content := toolContent(t.Diff); content != nil {
		opts = append(opts, acp.WithStartContent(content))
	}

	if t.RawInput != nil {
		opts = append(opts, acp.WithStartRawInput(t.RawInput))
	}

	if t.Name != "" {
		opts = append(opts, acp.WithStartName(t.Name))
	}

	return opts
}

// toolContent renders the note's diff as the SDK's diff tool-call content
// ([]ToolCallContent for the WithStart*/WithUpdate* content opts); a nil
// diff yields nil so the content field stays unset on the wire.
func toolContent(d *ToolDiff) []acp.ToolCallContent {
	if d == nil {
		return nil
	}

	if d.OldText == "" {
		return []acp.ToolCallContent{acp.ToolDiffContent(d.Path, d.NewText)}
	}

	return []acp.ToolCallContent{acp.ToolDiffContent(d.Path, d.NewText, d.OldText)}
}
