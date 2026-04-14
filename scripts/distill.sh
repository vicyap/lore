#!/usr/bin/env bash
# lore/scripts/distill.sh — distill reasoning into a git note
#
# Usage: distill.sh <transcript_path> <session_id> <commit_hash>
#
# Extracts the relevant transcript window, gets the commit diff, sends
# both to claude CLI for distillation, and writes the structured output
# as a git note on refs/notes/lore.

set -euo pipefail

LORE_DIR="${LORE_DIR:-$HOME/.lore}"
# shellcheck source=lib.sh
source "$LORE_DIR/scripts/lib.sh"

transcript_path="$1"
session_id="$2"
commit_hash="$3"

LORE_MODEL="${LORE_MODEL:-sonnet}"
LORE_MAX_DIFF_CHARS="${LORE_MAX_DIFF_CHARS:-20000}"
LORE_MAX_TRANSCRIPT_CHARS="${LORE_MAX_TRANSCRIPT_CHARS:-50000}"

# Get the diff for this commit
diff_content=$(lore_get_diff "$commit_hash")

# Truncate diff if too large
if [[ ${#diff_content} -gt $LORE_MAX_DIFF_CHARS ]]; then
    diff_content="${diff_content:0:$LORE_MAX_DIFF_CHARS}
...(diff truncated at ${LORE_MAX_DIFF_CHARS} chars)..."
fi

# Extract the relevant transcript window
transcript_window=$(python3 "$LORE_DIR/scripts/extract_window.py" \
    "$transcript_path" --max-chars "$LORE_MAX_TRANSCRIPT_CHARS")

# Get metadata for the session line
branch_name=$(lore_get_branch_name)
commit_subject=$(lore_get_commit_subject "$commit_hash")

# Build the prompt input
prompt_input="## Commit
${commit_hash:0:12} ${commit_subject}
Branch: ${branch_name}
Session: ${session_id}

## Diff
\`\`\`diff
${diff_content}
\`\`\`

## Transcript (agent session leading to this commit)
${transcript_window}"

lore_debug "Distilling reasoning for commit=$commit_hash (model=$LORE_MODEL)"

# Call claude CLI for distillation
# -p: print mode (non-interactive, stdout only)
distilled=$(echo "$prompt_input" | claude -p --model "$LORE_MODEL" \
    --system-prompt-file "$LORE_DIR/prompts/distill.md" \
    "Distill the decision reasoning for this commit." 2>/dev/null) || {
    lore_error "Claude CLI distillation failed"
    # Write a fallback note so we at least record the session reference
    distilled="## Intent
(distillation failed — claude CLI error)

## Confidence
low

## Session
${session_id} | ${branch_name}"
}

# Write the distilled note
note_file=$(mktemp)
echo "$distilled" >"$note_file"
lore_write_note "$commit_hash" "$note_file"
rm -f "$note_file"

lore_debug "Decision note written for commit=$commit_hash"
