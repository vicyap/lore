#!/usr/bin/env bash
# pith/scripts/distill.sh — distill reasoning into a git note
#
# Usage: distill.sh <transcript_path> <session_id> <commit_hash>
#
# Extracts the relevant transcript window, gets the commit diff, sends
# both to claude CLI for distillation, and writes the structured output
# as a git note on refs/notes/pith.

set -euo pipefail

PITH_DIR="${PITH_DIR:-$HOME/.pith}"
# shellcheck source=lib.sh
source "$PITH_DIR/scripts/lib.sh"

transcript_path="$1"
session_id="$2"
commit_hash="$3"

PITH_MODEL="${PITH_MODEL:-sonnet}"
PITH_MAX_DIFF_CHARS="${PITH_MAX_DIFF_CHARS:-20000}"
PITH_MAX_TRANSCRIPT_CHARS="${PITH_MAX_TRANSCRIPT_CHARS:-50000}"

# Get the diff for this commit
diff_content=$(pith_get_diff "$commit_hash")

# Truncate diff if too large
if [[ ${#diff_content} -gt $PITH_MAX_DIFF_CHARS ]]; then
    diff_content="${diff_content:0:$PITH_MAX_DIFF_CHARS}
...(diff truncated at ${PITH_MAX_DIFF_CHARS} chars)..."
fi

# Extract the relevant transcript window
transcript_window=$(python3 "$PITH_DIR/scripts/extract_window.py" \
    "$transcript_path" --max-chars "$PITH_MAX_TRANSCRIPT_CHARS")

# Get metadata for the session line
branch_name=$(pith_get_branch_name)
commit_subject=$(pith_get_commit_subject "$commit_hash")

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

pith_debug "Distilling reasoning for commit=$commit_hash (model=$PITH_MODEL)"

# Call claude CLI for distillation
# --bare: skip hooks (prevents recursion), skip CLAUDE.md, skip plugins
# -p: print mode (non-interactive, stdout only)
distilled=$(echo "$prompt_input" | claude -p --bare --model "$PITH_MODEL" \
    --system-prompt-file "$PITH_DIR/prompts/distill.md" \
    "Distill the decision reasoning for this commit." 2>/dev/null) || {
    pith_error "Claude CLI distillation failed"
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
pith_write_note "$commit_hash" "$note_file"
rm -f "$note_file"

pith_debug "Decision note written for commit=$commit_hash"
