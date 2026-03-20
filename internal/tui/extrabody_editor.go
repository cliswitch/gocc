package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func parseExtraBody(text string) (map[string]any, error) {
	result := make(map[string]any)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			return nil, fmt.Errorf("invalid line (missing ': '): %q", line)
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			return nil, fmt.Errorf("empty key in line: %q", line)
		}
		valStr := line[idx+2:]

		var val any
		if err := json.Unmarshal([]byte(valStr), &val); err != nil {
			// Values that look like JSON structures must parse correctly
			if len(valStr) > 0 && (valStr[0] == '{' || valStr[0] == '[' || valStr[0] == '"') {
				return nil, fmt.Errorf("invalid JSON value for key %q: %s", key, valStr)
			}
			// Bare string: treat as string value
			val = valStr
		}
		result[key] = val
	}
	return result, nil
}

func formatExtraBody(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		b, _ := json.Marshal(m[k])
		sb.WriteString(k + ": " + string(b))
	}
	return sb.String()
}

