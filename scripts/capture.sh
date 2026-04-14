#!/usr/bin/env bash
# lore/scripts/capture.sh — capture transcript to orphan branch
#
# Usage: capture.sh <transcript_path> <session_id> <commit_hash>
#
# Writes the full transcript JSONL to the lore/transcripts orphan branch
# as transcripts/{session_id}.jsonl. Uses git plumbing commands — no
# checkout, no stash, no disruption to the working tree.

set -euo pipefail

LORE_DIR="${LORE_DIR:-$HOME/.lore}"
# shellcheck source=lib.sh
source "$LORE_DIR/scripts/lib.sh"

transcript_path="$1"
session_id="$2"
commit_hash="$3"

if [[ ! -f "$transcript_path" ]]; then
    lore_error "Transcript not found: $transcript_path"
    exit 1
fi

lore_debug "Capturing transcript for session=$session_id commit=$commit_hash"

lore_orphan_write_file \
    "transcripts/${session_id}.jsonl" \
    "$transcript_path" \
    "transcript for ${commit_hash:0:12} (session ${session_id:0:8})"

lore_debug "Transcript captured to $LORE_BRANCH"
