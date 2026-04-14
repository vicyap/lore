#!/usr/bin/env bash
# lore/scripts/lore-hook.sh — PostToolUse hook entry point
#
# Called by Claude Code's PostToolUse hook when a Bash tool call matches
# "git commit*". Reads hook JSON from stdin, orchestrates transcript
# capture and reasoning distillation.
#
# Hook configuration (in .claude/settings.json):
#   {
#     "hooks": {
#       "PostToolUse": [{
#         "matcher": "Bash",
#         "hooks": [{
#           "type": "command",
#           "if": "Bash(*git commit*)",
#           "command": "~/.lore/scripts/lore-hook.sh",
#           "timeout": 120,
#           "statusMessage": "lore: distilling reasoning..."
#         }]
#       }]
#     }
#   }

set -euo pipefail

LORE_DIR="${LORE_DIR:-$HOME/.lore}"
# shellcheck source=lib.sh
source "$LORE_DIR/scripts/lib.sh"

# Read hook JSON from stdin
input=$(cat)

session_id=$(lore_parse_field "$input" '.session_id')
transcript_path=$(lore_parse_field "$input" '.transcript_path')
cwd=$(lore_parse_field "$input" '.cwd')
tool_command=$(lore_parse_field "$input" '.tool_input.command')

# Defense in depth — the "if" filter should already handle this,
# but verify we're looking at a git commit
if [[ ! "$tool_command" =~ git\ commit ]]; then
    exit 0
fi

# Verify we have the required fields
if [[ -z "$session_id" || -z "$transcript_path" || -z "$cwd" ]]; then
    lore_error "Missing required fields in hook input"
    exit 0 # Exit cleanly so we don't block the agent
fi

cd "$cwd"

# Get the commit that was just made
commit_hash=$(lore_get_commit_hash)

lore_info "Processing commit ${commit_hash:0:12} (session ${session_id:0:8})"

# Step 1: Capture transcript to orphan branch
"$LORE_DIR/scripts/capture.sh" "$transcript_path" "$session_id" "$commit_hash" || {
    lore_error "Transcript capture failed (non-fatal)"
}

# Step 2: Distill reasoning into git note
"$LORE_DIR/scripts/distill.sh" "$transcript_path" "$session_id" "$commit_hash" || {
    lore_error "Distillation failed (non-fatal)"
}

lore_info "Done: commit ${commit_hash:0:12}"
