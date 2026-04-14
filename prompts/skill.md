# lore — View decision reasoning for commits

Use this skill when the user invokes `/lore` or asks about the reasoning behind commits.

lore captures structured decision notes (the "why" behind code changes) as git notes on `refs/notes/lore`, and full session transcripts on the `lore/transcripts` orphan branch.

## Commands

The user may invoke `/lore` with one of these subcommands. Parse the argument to determine which.

### `/lore show [N | <hash>]`

Show decision notes for recent commits.

- `/lore show` or `/lore show 5` — show notes for the last N commits (default 5)
- `/lore show <hash>` — show the note for a specific commit

**Implementation:**

```bash
# Last N commits with notes
git log -N --notes=lore --format="## %h %s%n%N%n---"

# Specific commit
git notes --ref=lore show <hash>
```

If a commit has no lore note, say so — not every commit will have one (only commits made during Claude Code sessions are annotated).

### `/lore transcript [<hash>]`

Show the full session transcript for a commit.

1. Read the commit's lore note to find the session ID (in the `## Session` line)
2. Retrieve the transcript from the orphan branch:
   ```bash
   git show lore/transcripts:transcripts/<session_id>.jsonl
   ```
3. The JSONL can be large. Show a summary: count of messages, session duration (first to last timestamp), and the first few user prompts.
4. Ask the user if they want to see the full transcript or a specific section.

### `/lore push`

Push lore notes and transcripts to the remote.

```bash
git push origin refs/notes/lore
git push origin lore/transcripts
```

Report success or failure for each.

### `/lore status`

Show whether lore is enabled in the current project.

1. Check if the PostToolUse hook referencing `lore-hook.sh` exists in `.claude/settings.json`
2. Check if the `lore/transcripts` branch exists
3. Count how many commits have lore notes:
   ```bash
   git notes --ref=lore list | wc -l
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
