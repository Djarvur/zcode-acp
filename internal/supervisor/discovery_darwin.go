//go:build darwin

package supervisor

import "context"

// platformBinaryName is the macOS desktop executable name. An .app bundle is
// resolved through macAppInstall; this name only serves the explicit-root
// fallback for a non-bundle tree.
func platformBinaryName() string { return "ZCode" }

// probeInstallStage is the macOS install-layout stage: evaluate the standard
// ZCode*.app candidates and select the newest valid bundle.
func (p Probers) probeInstallStage(ctx context.Context, found bool) ([]Probe, Install, bool) {
	return p.probeMacApp(ctx, found)
}

// probeMacApp evaluates /Applications and $HOME/Applications candidates,
// keeps one diagnostics row per candidate, and selects the newest valid
// bundle. An earlier override wins and the stage records that it was not
// consulted.
func (p Probers) probeMacApp(ctx context.Context, found bool) ([]Probe, Install, bool) {
	row := Probe{Name: "mac-app", Query: "/Applications/ZCode*.app + $HOME/Applications/ZCode*.app"}
	if found {
		row.Result = "not consulted; an earlier probe resolved the install"

		return []Probe{row}, Install{}, false
	}
	if p.MacApps == nil {
		row.Result = "macOS app seam not configured (or not a macOS host)"

		return []Probe{row}, Install{}, false
	}

	apps := dedupeStrings(p.MacApps())
	rows := make([]Probe, 0, len(apps)+1)
	rows = append(rows, row)

	var winner Install
	for _, app := range apps {
		inst, reason := macAppInstall(ctx, app)
		if reason != "" {
			rows = append(rows, Probe{Name: "mac-app-candidate", Query: app,
				Result: "miss: " + app + ": " + reason})

			continue
		}

		rows = append(rows, Probe{Name: "mac-app-candidate", Query: app,
			Result: "hit: mac-app version " + inst.Version + " " + inst.Root})
		if winner.Root == "" || compareVersions(inst.Version, winner.Version) > 0 {
			winner = inst
		}
	}

	if winner.Root == "" {
		if len(apps) == 0 {
			rows[0].Result = "no ZCode*.app in the standard application dirs"
		} else {
			rows[0].Result = "no valid ZCode*.app; see mac-app-candidate rows"
		}

		return rows, Install{}, false
	}

	rows[0].Result = "selected: mac-app version " + winner.Version + " " + winner.Root

	return rows, winner, true
}

// dedupeStrings preserves first-seen order.
func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}

// DefaultProbers returns the Darwin seam set: the explicit override plus the
// standard .app scan and the host runtime-provider cache.
func DefaultProbers(home, optDir string) Probers {
	return Probers{
		Override:                defaultOverride,
		MacApps:                 defaultMacApps(home),
		Home:                    home,
		OptDir:                  optDir,
		RuntimeProviderPlatform: hostProviderPlatform(),
	}
}

// builtinProviderRelForPlatform names the Darwin runtime-provider cache dir.
func builtinProviderRelForPlatform() (string, bool) {
	platform := hostProviderPlatform()
	if platform == "" {
		return "", false
	}

	return builtinProviderRoot + "/" + platform, true
}

// DefaultOptDir is empty on macOS: the Linux package layout does not exist.
func DefaultOptDir() string { return "" }
