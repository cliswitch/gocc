package tui

import (
	"fmt"
	"sort"
	"strings"
)

func parseEnvVars(text string) (map[string]string, error) {
	env := make(map[string]string)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("invalid env line (missing '='): %q", line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("empty env key in line: %q", line)
		}
		env[key] = value
	}
	return env, nil
}

func formatEnvVars(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + "=" + env[k])
	}
	return sb.String()
}
