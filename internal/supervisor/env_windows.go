//go:build windows

package supervisor

import (
	"os"
	"strings"
)

func allowlistKeys() []string {
	return []string{
		allowlistPathVar,
		allowlistHomeVar,
		"USERPROFILE",
		"HOMEDRIVE",
		"HOMEPATH",
		"SystemRoot",
		"APPDATA",
		"LOCALAPPDATA",
		"TEMP",
		"TMP",
		"ComSpec",
		"PATHEXT",
	}
}

// Windows environment names are case-insensitive; os.Environ may preserve
// the shell's spelling (for example Path rather than PATH).
func envValue(parent map[string]string, key string) (string, bool) {
	if value, ok := parent[key]; ok {
		return value, true
	}

	for candidate, value := range parent {
		if strings.EqualFold(candidate, key) {
			return value, true
		}
	}

	return "", false
}

func homeValue(parent map[string]string) string {
	if value, ok := envValue(parent, allowlistHomeVar); ok && value != "" {
		return value
	}
	if value, ok := envValue(parent, "USERPROFILE"); ok && value != "" {
		return value
	}
	if drive, ok := envValue(parent, "HOMEDRIVE"); ok && drive != "" {
		if path, ok := envValue(parent, "HOMEPATH"); ok && path != "" {
			return drive + path
		}
	}

	value, _ := os.UserHomeDir()

	return value
}
