# pith

Capture *why* code was written, not just what changed.

pith is a Claude Code hook that distills structured decision reasoning from agent sessions and stores it as git notes alongside your commits. Full session transcripts are preserved on a separate branch for deep investigation.

**Progressive disclosure:**
- `git log` — clean history, no noise
- `git log --notes=pith` — structured reasoning per commit
- `git show pith/transcripts:transcripts/<session>.jsonl` — full transcript

## How it works

```
Claude Code session
  │
  ├─ You work, agent writes code
  │
  ├─ git commit
  │   └─ PostToolUse hook fires
  │       ├─ Capture: full transcript → pith/transcripts branch
  │       └─ Distill: transcript + diff → claude CLI → git note
  │
  └─ git log --notes=pith
      └─ Structured reasoning: intent, constraints,
         rejected alternatives, directives
```

## Install

```bash
git clone https://github.com/vicyap/pith ~/.pith
```

## Enable in a project

```bash
cd your-project
~/.pith/enable.sh
```

This adds a PostToolUse hook to `.claude/settings.json` and links the `/pith` skill.

## Disable

```bash
cd your-project
~/.pith/disable.sh
```

Existing notes and transcripts are preserved.

## Usage

Once enabled, pith runs automatically on every `git commit` made during a Claude Code session. No action needed.

### View decision notes

```bash
# Recent commits with reasoning
git log --notes=pith

# Specific commit
git notes --ref=pith show <hash>
```

### Interactive (in Claude Code)

```
/pith show          # last 5 commits with notes
/pith show abc123   # specific commit
/pith transcript    # full session transcript
/pith push          # push notes + transcripts to remote
/pith status        # check if pith is enabled
```

### Push to remote

```bash
git push origin refs/notes/pith
git push origin pith/transcripts
```

## What gets captured

Each commit gets a structured note:

```markdown
## Intent
Refactor Slack notification to fire on medication-change events
instead of request-status-change events.

## Constraints
- Slack service must not import from request flow (architectural boundary)
- Notification must fire exactly once per medication change

## Rejected Alternatives
- Polling approach — miss rapid sequential changes, adds latency
- Inline notification in action handler — couples action to notification

## Directives
- If adding new Slack notifications, follow event-driven pattern
- The medication_changed event shape is canonical

## Confidence
high

## Session
abc123-def456 | victor/USE-42-slack-fix
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PITH_DIR` | `~/.pith` | pith installation directory |
| `PITH_MODEL` | `sonnet` | Claude model for distillation |
| `PITH_MAX_DIFF_CHARS` | `20000` | Max diff size sent to distillation |
| `PITH_MAX_TRANSCRIPT_CHARS` | `50000` | Max transcript window size |
| `PITH_DEBUG` | (unset) | Set to `1` for debug logging |

## Dependencies

- `git`
- `jq`
- `python3`
- `claude` CLI (Claude Code)

## How it stores data

- **Decision notes**: `refs/notes/pith` — git notes attached to commits. Not visible in `git log` by default; opt-in with `--notes=pith`.
- **Transcripts**: `pith/transcripts` orphan branch — one JSONL file per session. Written via git plumbing (no checkout needed).

Neither pollutes your commit history or working tree.

## License

MIT
