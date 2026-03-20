package tui

import (
	"fmt"
	"sort"
	"strings"
)

func parseHeaders(text string) (map[string]string, error) {
	headers := make(map[string]string)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			return nil, fmt.Errorf("invalid header line (missing ':'): %q", line)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("empty header key in line: %q", line)
		}
		headers[key] = value
	}
	return headers, nil
}

func formatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + ": " + headers[k])
	}
	return sb.String()
}
