# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

lore is a Claude Code PostToolUse hook that captures structured decision reasoning ("Decision Shadow") from agent sessions and stores it as git notes alongside commits. Full session transcripts are preserved on an orphan branch. It runs automatically -- no user action needed beyond `lore init` in a project.

## Development commands

```bash
# Build
go build ./cmd/lore

# Run tests
go test ./...

# Vet
go vet ./...

# Install locally
go install ./cmd/lore

# Run the CLI
go run ./cmd/lore status
```

No external tools beyond `go` are required for development. The project uses Go modules with vendoring disabled.

## Architecture

Go CLI built with cobra. The hook fires on `git commit` during a Claude Code session and runs two steps sequentially:

1. **Capture** (`internal/git/plumbing.go`): writes the full JSONL transcript to a `lore/transcripts` orphan branch using git plumbing (no checkout, no working tree disruption).
2. **Distill** (`internal/distill/distill.go`): windows the transcript via `internal/transcript/window.go`, combines it with the commit diff, pipes both to `claude -p` with the system prompt in `prompts/distill.md`, and writes the output as a git note on `refs/notes/lore`.

### Package structure

- `cmd/lore/` -- CLI entry point and cobra commands (init, hook, show, status, push, export, disable, tui)
- `internal/config/` -- env var configuration
- `internal/git/` -- git plumbing (orphan branch ops), notes CRUD, helpers
- `internal/transcript/` -- JSONL parsing and windowing (port of the original Python extract_window.py)
- `internal/distill/` -- prompt assembly, claude CLI call, fallback note
- `internal/settings/` -- .claude/settings.json manipulation (hook install/remove)
- `internal/tui/` -- bubbletea v2 TUI for browsing notes
- `internal/export/` -- JSONL and Markdown export
- `prompts/` -- embedded distill prompt and skill definition (go:embed)
- `testdata/` -- fixture JSONL transcripts for tests

### Key design decisions

- All orphan branch writes use git plumbing (`hash-object`, `read-tree`, `update-index`, `write-tree`, `commit-tree`, `update-ref`) to avoid touching the working tree or index.
- `internal/transcript/window.go` windows the transcript between the previous commit's tool call and the current one, keeping only the relevant deliberation. It strips verbose tool outputs, keeping just tool names and key arguments.
- Hook errors are non-fatal -- both capture and distill failures are caught and logged but don't block the agent.
- `distill.go` calls `claude -p` to prevent hook recursion and skip project context.
- The distill prompt and skill definition are embedded in the binary via `go:embed`.

## Configuration

Environment variables: `LORE_MODEL` (distillation model, default `opus`), `LORE_DEBUG` (set to `1` for debug logging).

## Conventions

- Standard Go project layout with `cmd/` and `internal/`.
- Tests use `t.TempDir()` for git repo fixtures.
- Git operations that touch the orphan branch must use plumbing commands only -- never `git checkout` or `git stash`.

## Versioning

Semver, with a narrow definition of "minor":

- **Minor** (`v0.X.0`) -- reserved for behavior-changing updates, such as prompt edits that alter distilled output, hook semantics changes, or user-visible CLI behavior changes.
- **Patch** (`v0.0.X`) -- default for everything else: refactors, bug fixes, docs, tests, internal plumbing, dependency bumps.

When in doubt, bump patch.
