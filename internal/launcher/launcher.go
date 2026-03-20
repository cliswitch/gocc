package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func FindClaude() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH: %w", err)
	}
	return path, nil
}

func ExecClaude(claudePath string, envVars []string, args []string) error {
	env := mergeEnv(os.Environ(), envVars)
	argv := append([]string{claudePath}, args...)
	return syscall.Exec(claudePath, argv, env)
}

func RunClaude(claudePath string, envVars []string, args []string) (int, error) {
	cmd := exec.Command(claudePath, args...)
	cmd.Env = mergeEnv(os.Environ(), envVars)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start claude: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			cmd.Process.Signal(sig)
		}
	}()

	err := cmd.Wait()
	signal.Stop(sigCh)
	close(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func ExtractGoccFlags(args []string) (profile string, remaining []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--goccprofile" && i+1 < len(args) {
			profile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--goccprofile=") {
			profile = strings.TrimPrefix(arg, "--goccprofile=")
			continue
		}
		remaining = append(remaining, arg)
	}
	return
}

// mergeEnv merges base and extra environment variable slices.
// For duplicate keys, the last occurrence wins (extra overrides base).
// Insertion order is preserved: base keys first in their original order,
// then any new keys from extra appended at the end.
func mergeEnv(base, extra []string) []string {
	env := make(map[string]string)
	order := make([]string, 0)
	for _, e := range base {
		k, _, _ := strings.Cut(e, "=")
		if _, exists := env[k]; !exists {
			order = append(order, k)
		}
		env[k] = e
	}
	for _, e := range extra {
		k, _, _ := strings.Cut(e, "=")
		if _, exists := env[k]; !exists {
			order = append(order, k)
		}
		env[k] = e
	}
	result := make([]string, 0, len(order))
	for _, k := range order {
		result = append(result, env[k])
	}
	return result
}

