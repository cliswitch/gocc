package tui

import "strings"

func parseArgs(text string) []string {
	var args []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			args = append(args, line)
		}
	}
	return args
}

