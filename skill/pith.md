# pith — View decision reasoning for commits

Use this skill when the user invokes `/pith` or asks about the reasoning behind commits.

pith captures structured decision notes (the "why" behind code changes) as git notes on `refs/notes/pith`, and full session transcripts on the `pith/transcripts` orphan branch.

## Commands

The user may invoke `/pith` with one of these subcommands. Parse the argument to determine which.

### `/pith show [N | <hash>]`

Show decision notes for recent commits.

- `/pith show` or `/pith show 5` — show notes for the last N commits (default 5)
- `/pith show <hash>` — show the note for a specific commit

**Implementation:**

```bash
# Last N commits with notes
git log -N --notes=pith --format="## %h %s%n%N%n---"

# Specific commit
git notes --ref=pith show <hash>
```

If a commit has no pith note, say so — not every commit will have one (only commits made during Claude Code sessions are annotated).

### `/pith transcript [<hash>]`

Show the full session transcript for a commit.

1. Read the commit's pith note to find the session ID (in the `## Session` line)
2. Retrieve the transcript from the orphan branch:
   ```bash
   git show pith/transcripts:transcripts/<session_id>.jsonl
   ```
3. The JSONL can be large. Show a summary: count of messages, session duration (first to last timestamp), and the first few user prompts.
4. Ask the user if they want to see the full transcript or a specific section.

### `/pith push`

Push pith notes and transcripts to the remote.

```bash
git push origin refs/notes/pith
git push origin pith/transcripts
```

Report success or failure for each.

### `/pith status`

Show whether pith is enabled in the current project.

1. Check if the PostToolUse hook referencing `pith-hook.sh` exists in `.claude/settings.json`
2. Check if the `pith/transcripts` branch exists
3. Count how many commits have pith notes:
   ```bash
   git notes --ref=pith list | wc -l
   ```
4. Report the status.

## Display Format

When showing decision notes, display them as-is (they're already markdown). Add the commit hash and subject as a header:

```
### abc123def012 — fix medication change Slack notification

## Intent
...

## Constraints
...
```

When showing multiple notes, separate them with `---`.
