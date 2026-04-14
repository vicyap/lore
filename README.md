# lore

Capture *why* code was written, not just what changed.

lore is a Claude Code hook that distills structured decision reasoning from agent sessions and stores it as git notes alongside your commits. Full session transcripts are preserved on a separate branch for deep investigation.

Inspired by:
- [Lore: Repurposing Git Commit Messages as a Structured Knowledge Protocol for AI Coding Agents](https://arxiv.org/abs/2603.15566) (Stetsenko, 2026) — introduces the "Decision Shadow" concept and the idea of encoding constraints, rejected alternatives, and directives alongside commits
- [Entire CLI](https://github.com/entireio/cli) — a Git-integrated tool that captures AI agent session transcripts on a separate branch, keeping your main history clean

**Progressive disclosure:**
- `git log` — clean history, no noise
- `git log --notes=lore` — structured reasoning per commit
- `git show lore/transcripts:transcripts/<session>.jsonl` — full transcript

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
      +- Structured reasoning: intent, constraints,
         rejected alternatives, directives
```

## Install

```bash
git clone https://github.com/vicyap/lore ~/.lore
```

## Enable in a project

```bash
cd your-project
~/.lore/enable.sh
```

This adds a PostToolUse hook to `.claude/settings.json` and links the `/lore` skill.

## Disable

```bash
cd your-project
~/.lore/disable.sh
```

Existing notes and transcripts are preserved.

## Usage

Once enabled, lore runs automatically on every `git commit` made during a Claude Code session. No action needed.

### View decision notes

```bash
# Recent commits with reasoning
git log --notes=lore

# Specific commit
git notes --ref=lore show <hash>
```

### Interactive (in Claude Code)

```
/lore show          # last 5 commits with notes
/lore show abc123   # specific commit
/lore transcript    # full session transcript
/lore push          # push notes + transcripts to remote
/lore status        # check if lore is enabled
```

### Push to remote

```bash
git push origin refs/notes/lore
git push origin lore/transcripts
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
| `LORE_DIR` | `~/.lore` | lore installation directory |
| `LORE_MODEL` | `sonnet` | Claude model for distillation |
| `LORE_MAX_DIFF_CHARS` | `20000` | Max diff size sent to distillation |
| `LORE_MAX_TRANSCRIPT_CHARS` | `50000` | Max transcript window size |
| `LORE_DEBUG` | (unset) | Set to `1` for debug logging |

## Dependencies

- `git`
- `jq`
- `python3`
- `claude` CLI (Claude Code)

## How it stores data

- **Decision notes**: `refs/notes/lore` — git notes attached to commits. Not visible in `git log` by default; opt-in with `--notes=lore`.
- **Transcripts**: `lore/transcripts` orphan branch — one JSONL file per session. Written via git plumbing (no checkout needed).

Neither pollutes your commit history or working tree.

## References

- Stetsenko, I. (2026). *Lore: Repurposing Git Commit Messages as a Structured Knowledge Protocol for AI Coding Agents*. [arXiv:2603.15566](https://arxiv.org/abs/2603.15566)
- [Entire CLI](https://github.com/entireio/cli) — Git-integrated AI session capture

## License

MIT
