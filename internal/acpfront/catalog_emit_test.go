package acpfront_test

import (
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// TestSessionNewEmitsCommandCatalog — CAT-01/CAT-03: after session/new the
// client receives exactly ONE session/update notification whose update is
// available_commands_update carrying the fixture skills + the /model item;
// a second session/new emits again (per-session emission).
func TestSessionNewEmitsCommandCatalog(t *testing.T) {
	sc := mustLoadScenario(t, "desktop-nodelta-messages")

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithCatalogRoots("testdata/catalog"),
		)

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	catalogLines := catalogUpdateLines(h.ACPLines(), string(sn.SessionId))
	if len(catalogLines) != 1 {
		t.Fatalf("client collected %d available_commands_update frames for session %s, want exactly ONE: %v",
			len(catalogLines), sn.SessionId, catalogLines)
	}

	for _, want := range []string{
		`"available_commands_update"`, `"gsd-manager"`, `"zed-tools"`, `"model"`, `"hint":"\u003cselector\u003e"`,
	} {
		if !strings.Contains(catalogLines[0], want) {
			t.Fatalf("the catalog frame lacks %s: %s", want, catalogLines[0])
		}
	}

	if _, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
		McpServers: []acp.McpServer{},
	}); err != nil {
		t.Fatalf("session/new two: %v", err)
	}

	// The corpus session/create answers the SAME session id for both
	// creates (sess_fix1), so the per-session filter matches both frames —
	// the lock is the COUNT: exactly one catalog frame per session/new.
	allLines := catalogUpdateLines(h.ACPLines(), "")
	if len(allLines) != 2 {
		t.Fatalf("two session/new calls collected %d catalog frames, want exactly 2 (one per session)", len(allLines))
	}
}

// TestSessionNewEmitsCommandRoots — the command manifests arm (CAT-02, the
// zcode CLI's ~/.zcode/commands layout): command .md files ride the SAME
// catalog frame with their frontmatter description and argument-hint.
func TestSessionNewEmitsCommandRoots(t *testing.T) {
	sc := mustLoadScenario(t, "desktop-nodelta-messages")

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithCommandRoots("testdata/catalog-cmds"),
		)

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	catalogLines := catalogUpdateLines(h.ACPLines(), string(sn.SessionId))
	if len(catalogLines) != 1 {
		t.Fatalf("client collected %d catalog frames, want exactly ONE", len(catalogLines))
	}

	for _, want := range []string{`"gsd-manager"`, `"[--analyze-deps]"`, `"model"`} {
		if !strings.Contains(catalogLines[0], want) {
			t.Fatalf("the catalog frame lacks %s: %s", want, catalogLines[0])
		}
	}
}

// TestSessionNewCatalogNoRoots — the degraded arm (CAT-02 honesty): with no
// catalog roots configured the emission carries ONLY the /model item —
// never an error, never a skipped frame.
func TestSessionNewCatalogNoRoots(t *testing.T) {
	sc := mustLoadScenario(t, "desktop-nodelta-messages")

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn, acpfront.WithVersion("0.0.0-dev"))

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	catalogLines := catalogUpdateLines(h.ACPLines(), string(sn.SessionId))
	if len(catalogLines) != 1 {
		t.Fatalf("client collected %d catalog frames with no roots, "+
			"want exactly ONE (the /model item)", len(catalogLines))
	}

	if !strings.Contains(catalogLines[0], `"model"`) || strings.Contains(catalogLines[0], "gsd-manager") {
		t.Fatalf("the no-roots catalog frame = %s, want only the /model item", catalogLines[0])
	}
}

// catalogUpdateLines picks the client-bound session/update lines carrying
// the available_commands_update variant for one session id.
func catalogUpdateLines(lines []TeeLine, sessionID string) []string {
	var out []string

	for _, l := range lines {
		if l.Dir != "in" || !strings.Contains(l.JSON, `"available_commands_update"`) {
			continue
		}

		if sessionID != "" && !strings.Contains(l.JSON, sessionID) {
			continue
		}

		out = append(out, l.JSON)
	}

	return out
}
