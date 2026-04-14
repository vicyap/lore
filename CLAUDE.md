# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

lore is a Claude Code PostToolUse hook that captures structured decision reasoning ("Decision Shadow") from agent sessions and stores it as git notes alongside commits. Full session transcripts are preserved on an orphan branch. It runs automatically — no user action needed beyond enabling it in a project.

## Development commands

```bash
# Lint shell scripts
shellcheck scripts/*.sh enable.sh disable.sh

# Format shell scripts
shfmt -w -i 4 -bn -ci scripts/*.sh enable.sh disable.sh

# Test the hook manually (pipe hook JSON to stdin)
echo '{"session_id":"test","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","tool_input":{"command":"git commit -m test"}}' | bash scripts/lore-hook.sh

# Test extract_window.py
python3 scripts/extract_window.py /path/to/transcript.jsonl --max-chars 50000
```

No build step, no test suite, no package manager. The project is pure bash + one Python script.

## Architecture

The hook fires on `git commit` during a Claude Code session and runs two steps sequentially:

1. **Capture** (`capture.sh`): writes the full JSONL transcript to a `lore/transcripts` orphan branch using git plumbing (no checkout, no working tree disruption).
2. **Distill** (`distill.sh`): windows the transcript via `extract_window.py`, combines it with the commit diff, pipes both to `claude -p --bare` with the system prompt in `prompts/distill.md`, and writes the output as a git note on `refs/notes/lore`.

Key design decisions:
- All orphan branch writes use git plumbing (`hash-object`, `read-tree`, `update-index`, `write-tree`, `commit-tree`, `update-ref`) to avoid touching the working tree or index.
- `extract_window.py` windows the transcript between the previous commit's tool call and the current one, keeping only the relevant deliberation. It strips verbose tool outputs, keeping just tool names and key arguments.
- Hook errors are non-fatal — both capture and distill failures are caught and logged but don't block the agent.
- `distill.sh` calls `claude -p --bare` to prevent hook recursion and skip project context.

## File roles

- `enable.sh` / `disable.sh` — per-project install/uninstall (hook in `.claude/settings.json`, skill symlink, orphan branch init, git notes display config)
- `scripts/lib.sh` — shared helpers sourced by all scripts (git notes, orphan branch plumbing, logging)
- `scripts/lore-hook.sh` — PostToolUse hook entry point, reads JSON from stdin
- `scripts/capture.sh` / `scripts/distill.sh` — the two pipeline stages
- `scripts/extract_window.py` — transcript windowing and condensation
- `prompts/distill.md` — system prompt for reasoning extraction (defines the output schema: Intent, Constraints, Rejected Alternatives, Directives, Confidence, Session)
- `skill/lore.md` — Claude Code `/lore` skill definition (show, transcript, push, status subcommands)

## Configuration

Environment variables: `LORE_DIR` (install path), `LORE_MODEL` (distillation model, default `sonnet`), `LORE_MAX_DIFF_CHARS` (20000), `LORE_MAX_TRANSCRIPT_CHARS` (50000), `LORE_DEBUG` (set to `1` for debug logging).

## Conventions

- Shell scripts use `set -euo pipefail`. All scripts are formatted with `shfmt -i 4 -bn -ci` and pass `shellcheck`.
- Git operations that touch the orphan branch must use plumbing commands only — never `git checkout` or `git stash`.
- The `lore_` prefix namespaces all shared functions in `lib.sh`.
