# lore

Capture *why* code was written, not just what changed.

lore is a Claude Code hook that distills structured decision reasoning from agent sessions and stores it as git notes alongside your commits. Full session transcripts are preserved on a separate branch for deep investigation.

**Progressive disclosure:**
- `git log` -- clean history, no noise
- `git log --notes=lore` -- structured reasoning per commit
- `git show lore/transcripts:transcripts/<session>.jsonl` -- full transcript

## How it works

```
Claude Code session
  |
  +- You work, agent writes code
  |
  +- git commit
  |   +- PostToolUse hook fires
  |       +- Capture: full transcript -> lore/transcripts branch
  |       +- Distill: transcript + diff -> claude CLI -> git note
  |
  +- git log --notes=lore
      +- Structured reasoning: decisions with rationale,
         metadata
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vicyap/lore/main/install.sh | sh
```

Or from source: `go install github.com/vicyap/lore/cmd/lore@latest`

## Enable in a project

```bash
cd your-project
lore init
```

This adds a PostToolUse hook to `.claude/settings.json`, creates the orphan branch, configures git notes display, and adds `+refs/notes/*:refs/notes/*` to `remote.origin.fetch` so future `git fetch`/`git pull` brings lore notes down automatically.

## Disable

```bash
cd your-project
lore disable
```

Existing notes and transcripts are preserved.

## Usage

Once enabled, lore runs automatically on every `git commit` made during a Claude Code session. No action needed.

### CLI commands

```bash
lore status     # check if lore is enabled in this repo
lore browse     # interactive browser
lore export     # export as JSONL to stdout
lore export --format md --output notes.md
lore pull       # fetch notes + transcripts from origin (useful on fresh clones)
```

### Using git directly

```bash
git log --notes=lore                              # view notes in log
git notes --ref=lore show <hash>                  # specific commit's note
git push origin refs/notes/lore lore/transcripts  # push to remote
```

## What gets captured

Each commit gets a structured note:

```markdown
## Decisions
- Moved to event-driven notifications over polling — polling misses rapid sequential status changes and adds latency
- Used order_status_changed event rather than inline notification in action handler — keeps action and notification decoupled across the architectural boundary
- Each status change triggers exactly one notification — notification service must not import from request flow

## Metadata
- version: v0.5.0
- confidence: high
- transcript-ref: 7e9f2a1b3c4d
- branch: feature/order-notifications
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LORE_MODEL` | `opus` | Claude model for distillation |
| `LORE_DEBUG` | (unset) | Set to `1` for debug logging |

## Dependencies

- `git`
- `claude` CLI (Claude Code)

## How it stores data

- **Decision notes**: `refs/notes/lore` -- git notes attached to commits. Not visible in `git log` by default; opt-in with `--notes=lore`.
- **Transcripts**: `lore/transcripts` orphan branch -- one JSONL file per session. Written via git plumbing (no checkout needed).

Neither pollutes your commit history or working tree.

## GitHub Actions

Use the reusable workflow to automatically push lore data on merge:

```yaml
on:
  push:
    branches: [main]

jobs:
  push-lore:
    uses: vicyap/lore/.github/workflows/push-lore.yml@main
```

## Versioning

Semver, with a narrow definition of "minor":

- **Minor** (`v0.X.0`) -- behavior-changing updates: prompt edits that alter distilled output, hook semantics changes, or user-visible CLI behavior changes.
- **Patch** (`v0.0.X`) -- default for everything else: refactors, bug fixes, docs, tests, internal plumbing, dependency bumps.

## References

- Stetsenko, I. (2026). *Lore: Repurposing Git Commit Messages as a Structured Knowledge Protocol for AI Coding Agents*. [arXiv:2603.15566](https://arxiv.org/abs/2603.15566)
- [Entire CLI](https://github.com/entireio/cli) -- Git-integrated AI session capture

## License

MIT
