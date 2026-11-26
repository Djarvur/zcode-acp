//go:build !windows

package supervisor

import "os"

func allowlistKeys() []string {
	return []string{allowlistPathVar, allowlistHomeVar}
}

func envValue(parent map[string]string, key string) (string, bool) {
	value, ok := parent[key]

	return value, ok
}

func homeValue(parent map[string]string) string {
	if value, ok := envValue(parent, allowlistHomeVar); ok && value != "" {
		return value
	}

	value, _ := os.UserHomeDir()

	return value
}
