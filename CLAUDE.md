# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gocc is a lightweight Claude Code launcher that provides LLM API protocol conversion and fallback capabilities. Each gocc process starts a local proxy on a random port using `llmapimux`, launches Claude Code as a child process pointing at the proxy, and exits when Claude exits. No daemon, no WebUI — each invocation is self-contained.

Predecessor project: `../goclaude`. Protocol conversion library: `../llmapimux`.

**Status:** Pre-development — design spec and implementation plan are complete, source code not yet written. Refer to the plan for implementation order.

## Build & Development Commands

Requires Go 1.21+.

```bash
# Build
go build ./...

# Install locally
go install .

# Run tests
go test ./...

# Run a single test (example)
go test ./internal/config/ -run TestFunctionName

# Run tests with verbose output
go test -v ./...

# Tidy dependencies
go mod tidy
```

## Architecture

### Core Flow

1. `main.go` — CLI entry point. Manually extracts `--goccprofile` flag (not cobra parsing) to avoid rejecting arbitrary Claude args. Loads config, resolves profile (via flag or TUI), then launches.
2. **Native profile** → `syscall.Exec` replaces process with claude directly (no proxy).
3. **Non-native profile** → starts HTTP proxy on `127.0.0.1:0`, sets env vars (ANTHROPIC_BASE_URL, model annotations, etc.), launches claude as child via `os/exec`, forwards signals, exits with claude's exit code.

### Package Layout

| Package | Role |
|---------|------|
| `internal/config` | Config/Profile structs, YAML load/save, model name annotation (`Sonnet(model-name)` format), env var generation |
| `internal/proxy` | Wraps `llmapimux.Mux` — builds CandidateFunc from profile + fallback chain, UUID-based auth, AnthropicHandler |
| `internal/launcher` | Finds claude binary, `ExecClaude` (syscall.Exec for native), `RunClaude` (os/exec for proxy mode) |
| `internal/tui` | bubbletea TUI — profile list, profile edit form, models/reasoning/headers/fallback sub-pages, global config, status bar |

### Key Design Decisions

- **Model annotation**: env vars use `Level(model)` format (e.g., `Sonnet(gpt-4o)`) so the proxy can always determine the requested model level for correct fallback routing, even when multiple levels map to the same model name.
- **ANTHROPIC_MODEL** is set to an alias string (e.g., `"sonnet"`), not an annotated name. Claude resolves it via `ANTHROPIC_DEFAULT_SONNET_MODEL`.
- **Flag parsing**: gocc manually scans argv for `--goccprofile`, removes it, and passes everything else through to claude. This avoids cobra/flag conflicts with Claude's own flags.
- **Native profile** (`id: "__native__"`) has fixed id/name, cannot be deleted, cannot appear in fallback chains. Only proxy fields are editable.
- **Profile IDs**: 10-char random hex, immutable after creation.

### Config

Stored at `~/.gocc/config.yaml`. Supports protocols: `openai_chat`, `openai_responses`, `anthropic`, `gemini` (aligned with llmapimux).

## Design Documents

- **Spec**: `docs/superpowers/specs/2026-03-14-gocc-design.md` — full design with config schema, TUI mockups, protocol details
- **Plan**: `docs/superpowers/plans/2026-03-14-gocc-implementation.md` — step-by-step implementation plan with file structure and task breakdown

