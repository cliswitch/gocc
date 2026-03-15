package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/cliswitch/gocc/internal/config"
	"github.com/cliswitch/gocc/internal/launcher"
	"github.com/cliswitch/gocc/internal/proxy"
	"github.com/cliswitch/gocc/internal/stats"
	"github.com/cliswitch/gocc/internal/tui"
)

// Version is set by go build -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gocc: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	profileFlag, claudeArgs := launcher.ExtractGoccFlags(os.Args[1:])

	claudePath, err := launcher.FindClaude()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	config.EnsureNative(cfg)

	var profile config.Profile
	if profileFlag != "" {
		p, ok := config.FindProfile(cfg, profileFlag)
		if !ok {
			return fmt.Errorf("profile %q not found", profileFlag)
		}
		profile = p
	} else {
		selected, err := runTUI(cfg)
		if err != nil {
			return err
		}
		if selected == "" {
			return nil
		}
		p, ok := config.FindProfile(cfg, selected)
		if !ok {
			return fmt.Errorf("profile %q not found", selected)
		}
		profile = p
	}

	cfg.Global.LastProfile = profile.ID
	if err := config.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "gocc: warning: failed to save last profile: %v\n", err)
	}

	resolved := config.Resolve(cfg.Global, profile)

	allArgs := make([]string, 0, len(resolved.ClaudeArgs)+len(claudeArgs))
	allArgs = append(allArgs, resolved.ClaudeArgs...)
	allArgs = append(allArgs, claudeArgs...)

	if profile.Protocol == config.ProtocolNative {
		return launcher.ExecClaude(claudePath, resolved.EnvVars, allArgs)
	}

	return launchWithProxy(cfg, claudePath, profile, allArgs, resolved.EnvVars)
}

func launchWithProxy(cfg *config.Config, claudePath string, profile config.Profile, args []string, resolvedEnvVars []string) error {
	allProfiles := make(map[string]config.Profile)
	for _, p := range cfg.Profiles {
		allProfiles[p.ID] = p
	}

	// Create session logger (best-effort — failure does not block proxy).
	configDir, _ := config.Dir()
	logsDir := filepath.Join(configDir, "logs")

	logger, err := stats.NewSessionLogger(logsDir, profile.ID, profile.Name, Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create session logger: %v\n", err)
	}
	if logger != nil {
		defer logger.Close()
	}

	token := uuid.New().String()
	port, shutdown, err := proxy.StartProxy(profile, allProfiles, token, logger)
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}
	defer shutdown()

	// Proxy-specific vars are appended last so they always override any custom
	// env entries with the same key (e.g., ANTHROPIC_BASE_URL).
	envVars := make([]string, len(resolvedEnvVars))
	copy(envVars, resolvedEnvVars)
	envVars = append(envVars,
		fmt.Sprintf("ANTHROPIC_BASE_URL=http://127.0.0.1:%d", port),
		fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", token),
	)

	exitCode, err := launcher.RunClaude(claudePath, envVars, args)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

func runTUI(cfg *config.Config) (string, error) {
	profiles := buildDisplayProfiles(cfg)
	initialCursor := findInitialCursor(cfg, profiles)
	callbacks := buildCallbacks(cfg)

	m := tui.NewModel(profiles, initialCursor, callbacks)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("tui: %w", err)
	}
	return result.(tui.Model).Chosen(), nil
}

func buildDisplayProfiles(cfg *config.Config) []tui.DisplayProfile {
	var dps []tui.DisplayProfile
	for _, p := range cfg.Profiles {
		dps = append(dps, tui.DisplayProfile{
			ID:       p.ID,
			Name:     p.Name,
			Protocol: p.Protocol,
		})
	}
	return dps
}

func findInitialCursor(cfg *config.Config, profiles []tui.DisplayProfile) int {
	for i, p := range profiles {
		if p.ID == cfg.Global.LastProfile {
			return i
		}
	}
	return 0
}

func buildCallbacks(cfg *config.Config) tui.Callbacks {
	return tui.Callbacks{
		Reload: func() ([]tui.DisplayProfile, error) {
			newCfg, err := config.LoadConfig()
			if err != nil {
				return nil, err
			}
			config.EnsureNative(newCfg)
			*cfg = *newCfg
			return buildDisplayProfiles(cfg), nil
		},
		GetProfile: func(id string) (config.Profile, bool) {
			return config.FindProfile(cfg, id)
		},
		SaveProfile: func(p config.Profile, isNew bool) error {
			if isNew {
				if p.ID == "" {
					p.ID = config.GenerateProfileID()
				}
				cfg.Profiles = append(cfg.Profiles, p)
			} else {
				for i, existing := range cfg.Profiles {
					if existing.ID == p.ID {
						cfg.Profiles[i] = p
						break
					}
				}
			}
			return config.SaveConfig(cfg)
		},
		DeleteProfile: func(id string) error {
			for i, p := range cfg.Profiles {
				if p.ID == id {
					cfg.Profiles = append(cfg.Profiles[:i], cfg.Profiles[i+1:]...)
					break
				}
			}
			return config.SaveConfig(cfg)
		},
		MoveProfile: func(id string, delta int) error {
			for i, p := range cfg.Profiles {
				if p.ID == id {
					j := i + delta
					if j >= 0 && j < len(cfg.Profiles) {
						cfg.Profiles[i], cfg.Profiles[j] = cfg.Profiles[j], cfg.Profiles[i]
					}
					break
				}
			}
			return config.SaveConfig(cfg)
		},
		CopyProfile: func(id string) error {
			p, ok := config.FindProfile(cfg, id)
			if !ok {
				return fmt.Errorf("profile %q not found", id)
			}
			dup := config.CloneProfile(p)
			dup.ID = config.GenerateProfileID()
			dup.Name = p.Name + " (copy)"
			cfg.Profiles = append(cfg.Profiles, dup)
			return config.SaveConfig(cfg)
		},
		SaveGlobal: func(g config.GlobalConfig) error {
			cfg.Global = g
			return config.SaveConfig(cfg)
		},
		GetGlobal: func() config.GlobalConfig {
			return cfg.Global
		},
	}
}
