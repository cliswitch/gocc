package config

import (
	"maps"
	"strings"
)

// ResolvedLaunchConfig holds the merged arguments and environment variables
// that should be used when launching Claude Code.
type ResolvedLaunchConfig struct {
	ClaudeArgs []string // merged claude arguments
	EnvVars    []string // merged environment variables (KEY=VALUE format)
}

// Resolve merges global config and profile settings into a ResolvedLaunchConfig.
//
// Args merge rules:
//   - If profile.ShouldInheritGlobalArgs() is true: global.ClaudeArgs + profile.ClaudeArgs
//   - Otherwise: only profile.ClaudeArgs
//
// Env merge rules:
//  1. Start with structured env vars from profile.EnvVars()
//  2. Build custom env map:
//     - If profile.ShouldInheritGlobalEnv(): start with global.CustomEnv, overlay profile.CustomEnv (profile wins)
//     - Otherwise: only profile.CustomEnv
//  3. Append custom env (KEY=VALUE strings) after structured env
//  4. Deduplicate final slice using last-wins semantics
func Resolve(global GlobalConfig, profile Profile) ResolvedLaunchConfig {
	// Merge args
	var args []string
	if profile.ShouldInheritGlobalArgs() {
		args = append(args, global.ClaudeArgs...)
	}
	args = append(args, profile.ClaudeArgs...)

	// Build structured env vars
	envs := profile.EnvVars()

	// Build custom env map
	customEnv := make(map[string]string)
	if profile.ShouldInheritGlobalEnv() {
		maps.Copy(customEnv, global.CustomEnv)
	}
	// Profile custom env overlays (wins over global)
	maps.Copy(customEnv, profile.CustomEnv)

	// Append custom env as KEY=VALUE strings
	for k, v := range customEnv {
		envs = append(envs, k+"="+v)
	}

	// Deduplicate using last-wins semantics
	envs = dedupEnv(envs)

	return ResolvedLaunchConfig{
		ClaudeArgs: args,
		EnvVars:    envs,
	}
}

// dedupEnv deduplicates a slice of KEY=VALUE strings using last-wins semantics.
// The returned slice preserves the order of the last occurrence of each key.
func dedupEnv(envs []string) []string {
	// Track the last index for each key
	lastIndex := make(map[string]int, len(envs))
	for i, s := range envs {
		key, _, _ := strings.Cut(s, "=")
		lastIndex[key] = i
	}

	// Build result preserving insertion order of the last occurrence
	result := make([]string, 0, len(lastIndex))
	for i, s := range envs {
		key, _, _ := strings.Cut(s, "=")
		if lastIndex[key] == i {
			result = append(result, s)
		}
	}
	return result
}
