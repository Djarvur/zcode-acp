package acpfront

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"log/slog"
)

// The slash catalog (v0.21.0, CAT-02): the ZCode skills the wrapper can
// resolve from disk, surfaced as the card's composer autocomplete items.
// The scan is a pure function of the roots — deterministic order (sorted
// by name), silent skips for anything unreadable or frontmatter-less
// (a broken skill never fails the catalog), and a missing root is simply
// no items (a machine without user skills still gets the /model entry
// from the emitter, never an error).

// skillManifestHeadCap bounds the SKILL.md head read the frontmatter
// parse sees — frontmatter lives in the first screenful; the body never
// influences the catalog item.
const skillManifestHeadCap = 4096

// CatalogItem is one neutral autocomplete item: the command name the
// client renders, its description, and the optional input hint (the
// /model entry carries "<selector>").
type CatalogItem struct {
	Name        string
	Description string
	Hint        string
}

// DefaultCatalogRoots resolves the catalog scan roots for a home
// directory (the zcode source's roots, zcode-src skillsService.ts:65-77):
// the user skills dir, the `~/.agents/skills` compat root, then the
// installed plugin-cache skills glob (the zcode CLI's plugin layout:
// plugins/cache/<marketplace>/<plugin>/<version>/skills/<skill>).
func DefaultCatalogRoots(home string) []string {
	return []string{
		filepath.Join(home, ".zcode", "skills"),
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".zcode", "cli", "plugins", "cache", "*", "*", "*", "skills"),
	}
}

// DefaultCommandRoots resolves the command scan roots for a home
// directory (the zcode source's user-level pair, commandDataSync.ts:38):
// `~/.zcode/commands` first, the `~/.agents/commands` compat root second —
// a stem collision keeps the first root's entry.
func DefaultCommandRoots(home string) []string {
	return []string{
		filepath.Join(home, ".zcode", "commands"),
		filepath.Join(home, ".agents", "commands"),
	}
}

// CommandItems scans the command roots: every top-level `*.md` file is ONE
// command (the zcode CLI's slash-command layout). The item name is the
// file STEM — the form the runtime provably invokes (`/gsd-manager`, the
// 2026-11-23 probe) — while the description and argument-hint ride the
// frontmatter; a file without a usable frontmatter head skips silently.
// Entries sort by name; stem collisions across roots keep the first
// root's entry (.zcode outranks the compat dir).
func CommandItems(roots []string) []CatalogItem {
	seen := map[string]bool{}
	var items []CatalogItem

	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "*.md"))
		if err != nil {
			slog.Debug("acpfront: command root glob failed", "root", root, "err", err)

			continue
		}

		sort.Strings(matches)

		for _, manifest := range matches {
			name := strings.TrimSuffix(filepath.Base(manifest), ".md")

			_, description, hint, hasFrontmatter := parseCommandManifest(manifest)
			if !hasFrontmatter {
				continue
			}

			key := strings.ToLower(strings.TrimSpace(name))
			if seen[key] {
				continue
			}

			seen[key] = true
			items = append(items, CatalogItem{Name: name, Description: description, Hint: hint})
		}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	return items
}

// parseCommandManifest reads one command .md's frontmatter description and
// argument-hint (the capture-verbatim fields of the gsd-core command
// files); the invocation name stays the caller's stem decision.
func parseCommandManifest(path string) (name, description, hint string, hasFrontmatter bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Debug("acpfront: command manifest unreadable", "path", path, "err", err)

		return "", "", "", false
	}

	head := string(data)
	if len(head) > skillManifestHeadCap {
		head = head[:skillManifestHeadCap]
	}

	rest, ok := strings.CutPrefix(head, "---\n")
	if !ok {
		return "", "", "", false
	}

	fence := strings.Index(rest, "\n---")
	if fence < 0 {
		return "", "", "", false
	}

	for _, line := range strings.Split(rest[:fence], "\n") {
		if v, ok := strings.CutPrefix(line, "name:"); ok && name == "" {
			name = strings.TrimSpace(v)
		}

		if v, ok := strings.CutPrefix(line, "description:"); ok && description == "" {
			description = strings.Trim(strings.TrimSpace(v), `"`)
		}

		if v, ok := strings.CutPrefix(line, "argument-hint:"); ok && hint == "" {
			hint = strings.Trim(strings.TrimSpace(v), `"`)
		}
	}

	return name, description, hint, true
}

// WorkspaceCommandRoots resolves the per-session workspace command roots
// (the zcode source's workspace-level commands pair,
// commandsService.ts:56-57): the session cwd's `.zcode/commands` and its
// `.agents/commands` compat root.
func WorkspaceCommandRoots(cwd string) []string {
	return []string{
		filepath.Join(cwd, ".zcode", "commands"),
		filepath.Join(cwd, ".agents", "commands"),
	}
}

// WorkspaceCatalogRoots resolves the per-session workspace roots (the
// zcode source's workspace-level pair, skillsService.ts:60-70): the
// session cwd's `.zcode/skills` and its `.agents/skills` compat root —
// later roots lose to earlier ones on a name collision.
func WorkspaceCatalogRoots(cwd string) []string {
	return []string{
		filepath.Join(cwd, ".zcode", "skills"),
		filepath.Join(cwd, ".agents", "skills"),
	}
}

// SkillItems scans the roots for skill manifests: every direct child
// directory's SKILL.md (the zcode CLI skill layout) contributes one item
// keyed by its frontmatter name (the child dir's name when the frontmatter
// carries none), described by the frontmatter description. Entries sort by
// name; duplicates (a plugin skill shadowing a user skill) keep the FIRST
// occurrence — user skills outrank plugin caches by root order.
func SkillItems(roots []string) []CatalogItem {
	type entry struct {
		name        string
		description string
	}

	seen := map[string]bool{}
	var entries []entry

	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "*", "SKILL.md"))
		if err != nil {
			slog.Debug("acpfront: catalog root glob failed", "root", root, "err", err)

			continue
		}

		sort.Strings(matches)

		for _, manifest := range matches {
			name, description, hasFrontmatter := parseSkillManifest(manifest)
			if !hasFrontmatter {
				continue // not a skill manifest — never a catalog item
			}

			if name == "" {
				name = filepath.Base(filepath.Dir(manifest))
			}

			key := strings.ToLower(strings.TrimSpace(name))
			if seen[key] {
				continue
			}

			seen[key] = true
			entries = append(entries, entry{name: name, description: description})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	items := make([]CatalogItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, CatalogItem{Name: e.name, Description: e.description})
	}

	return items
}

// parseSkillManifest reads one SKILL.md's frontmatter name/description:
// the --- fenced document head, `name:`/`description:` scalar lines. The
// boolean reports a usable frontmatter head at all — false for unreadable
// files and files without the fence (the caller skips them; a head WITH
// fields but no name line still qualifies and takes the caller's
// directory-name fallback).
func parseSkillManifest(path string) (name, description string, hasFrontmatter bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Debug("acpfront: catalog manifest unreadable", "path", path, "err", err)

		return "", "", false
	}

	head := string(data)
	if len(head) > skillManifestHeadCap {
		head = head[:skillManifestHeadCap]
	}

	rest, ok := strings.CutPrefix(head, "---\n")
	if !ok {
		return "", "", false
	}

	fence := strings.Index(rest, "\n---")
	if fence < 0 {
		return "", "", false
	}

	for _, line := range strings.Split(rest[:fence], "\n") {
		if v, ok := strings.CutPrefix(line, "name:"); ok && name == "" {
			name = strings.TrimSpace(v)
		}

		if v, ok := strings.CutPrefix(line, "description:"); ok && description == "" {
			description = strings.Trim(strings.TrimSpace(v), `"`)
		}
	}

	return name, description, true
}
