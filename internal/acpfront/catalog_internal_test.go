package acpfront

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSkillItems locks the catalog scanner (CAT-02): every direct child
// dir's SKILL.md frontmatter name/description becomes one item, entries
// sorted by name, unreadable/frontmatter-less children skipped silently,
// a missing root contributes nothing and never an error.
func TestSkillItems(t *testing.T) {
	roots := []string{filepath.Join("testdata", "catalog")}

	items := SkillItems(roots)

	if len(items) != 2 {
		t.Fatalf("skill items = %d (%v), want exactly 2 (the broken and "+
			"SKILL-less children skipped)", len(items), items)
	}

	const wantAlphaDesc = "Manage GSD milestones and phases from the chat"
	if items[0].Name != "gsd-manager" || items[0].Description != wantAlphaDesc {
		t.Fatalf("item[0] = %+v, want the gsd-manager frontmatter (sorted first)", items[0])
	}

	const wantZetaDesc = "Tool helper skills"
	if items[1].Name != "zed-tools" || items[1].Description != wantZetaDesc {
		t.Fatalf("item[1] = %+v, want the zed-tools frontmatter", items[1])
	}

	if items := SkillItems([]string{filepath.Join("testdata", "catalog-missing")}); len(items) != 0 {
		t.Fatalf("a missing root produced %v, want no items and no error", items)
	}
}

// TestSkillItemsDirNameFallback locks the name fallback: a SKILL.md with
// frontmatter but no name line takes the child directory's name.
func TestSkillItemsDirNameFallback(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "unnamed-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	const manifest = "---\ndescription: only a description\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	items := SkillItems([]string{root})
	if len(items) != 1 || items[0].Name != "unnamed-skill" || items[0].Description != "only a description" {
		t.Fatalf("items = %+v, want the directory-name fallback with the description", items)
	}
}

// TestDefaultCatalogRoots locks the root resolution (CAT-02; the zcode
// source's root set, skillsService.ts:65-77): the user skills dir, the
// `~/.agents/skills` compat root, then the installed plugin-caches glob.
func TestDefaultCatalogRoots(t *testing.T) {
	const home = "/home/tester"
	roots := DefaultCatalogRoots(home)

	if len(roots) != 3 {
		t.Fatalf("roots = %v, want exactly the user dir, the compat dir, and the plugin-cache glob", roots)
	}

	wantUser := filepath.Join(home, ".zcode", "skills")
	wantCompat := filepath.Join(home, ".agents", "skills")
	wantPlugins := filepath.Join(home, ".zcode", "cli", "plugins", "cache", "*", "*", "*", "skills")

	if roots[0] != wantUser || roots[1] != wantCompat || roots[2] != wantPlugins {
		t.Fatalf("roots = %q, want %q, %q, then %q", roots, wantUser, wantCompat, wantPlugins)
	}
}

// TestCommandItems locks the command-root scan (v0.21.0 CAT-02/CAT-03, the
// zcode source's command layout commandsService.ts:56-57): every top-level
// *.md file is ONE command; the item name is the file STEM (the form the
// runtime provably invokes — /gsd-manager), description and argument-hint
// ride the frontmatter; stem collisions across roots keep the first root's
// entry (.zcode outranks the compat dir); non-frontmatter files skip.
func TestCommandItems(t *testing.T) {
	items := CommandItems([]string{"testdata/catalog-cmds"})

	if len(items) != 2 {
		t.Fatalf("command items = %d (%v), want exactly 2 (the broken file skipped)", len(items), items)
	}

	const (
		wantGsdDesc = "Interactive command center for managing multiple phases from one terminal"
		wantGsdHint = "[--analyze-deps]"
		wantPlain   = "A command without a name or hint"
	)

	if items[0].Name != "gsd-manager" || items[0].Description != wantGsdDesc ||
		items[0].Hint != wantGsdHint {
		t.Fatalf("item[0] = %+v, want the gsd-manager stem with the frontmatter description and hint", items[0])
	}

	if items[1].Name != "plain-cmd" || items[1].Description != wantPlain || items[1].Hint != "" {
		t.Fatalf("item[1] = %+v, want the stem with the description and no hint", items[1])
	}
}

// TestCommandItemsRootPrecedence locks the compat-root collision rule: the
// same stem in two roots keeps the FIRST root's entry (.zcode outranks
// ~/.agents, mirroring commandDataSync's merge order).
func TestCommandItemsRootPrecedence(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()

	const (
		zcodeManifest  = "---\ndescription: from zcode\n---\n"
		agentsManifest = "---\ndescription: from agents\n---\n"
	)

	if err := os.WriteFile(filepath.Join(first, "dup.md"), []byte(zcodeManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(second, "dup.md"), []byte(agentsManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	items := CommandItems([]string{first, second})

	const wantFirst = "from zcode"
	if len(items) != 1 || items[0].Description != wantFirst {
		t.Fatalf("items = %+v, want ONE entry with the first root's description", items)
	}
}

// TestDefaultCommandRoots locks the command roots (the zcode source's
// user-level pair, commandDataSync.ts:38): ~/.zcode/commands then
// ~/.agents/commands.
func TestDefaultCommandRoots(t *testing.T) {
	const home = "/home/tester"
	roots := DefaultCommandRoots(home)

	if len(roots) != 2 {
		t.Fatalf("roots = %v, want exactly the two user command dirs", roots)
	}

	if roots[0] != filepath.Join(home, ".zcode", "commands") || roots[1] != filepath.Join(home, ".agents", "commands") {
		t.Fatalf("roots = %q, want .zcode/commands then .agents/commands", roots)
	}
}
