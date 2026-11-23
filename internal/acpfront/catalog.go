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

// CatalogItem is one neutral autocomplete item: the command name the
// client renders, its description, and the optional input hint (the
// /model entry carries "<selector>").
type CatalogItem struct {
	Name        string
	Description string
	Hint        string
}

// DefaultCatalogRoots resolves the catalog scan roots for a home
// directory: the user skills dir first, then the installed plugin-cache
// skills glob (the zcode CLI's plugin layout:
// plugins/cache/<marketplace>/<plugin>/<version>/skills/<skill>).
func DefaultCatalogRoots(home string) []string {
	return []string{
		filepath.Join(home, ".zcode", "skills"),
		filepath.Join(home, ".zcode", "cli", "plugins", "cache", "*", "*", "*", "skills"),
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

			if seen[name] {
				continue
			}

			seen[name] = true
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
	if len(head) > 4096 {
		head = head[:4096]
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
			description = strings.TrimSpace(strings.Trim(v, `"`))
		}
	}

	return name, description, true
}
