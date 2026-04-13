#!/usr/bin/env bash
# pith/scripts/capture.sh — capture transcript to orphan branch
#
# Usage: capture.sh <transcript_path> <session_id> <commit_hash>
#
# Writes the full transcript JSONL to the pith/transcripts orphan branch
# as transcripts/{session_id}.jsonl. Uses git plumbing commands — no
# checkout, no stash, no disruption to the working tree.

set -euo pipefail

PITH_DIR="${PITH_DIR:-$HOME/.pith}"
# shellcheck source=lib.sh
source "$PITH_DIR/scripts/lib.sh"

transcript_path="$1"
session_id="$2"
commit_hash="$3"

if [[ ! -f "$transcript_path" ]]; then
    pith_error "Transcript not found: $transcript_path"
    exit 1
fi

pith_debug "Capturing transcript for session=$session_id commit=$commit_hash"

pith_orphan_write_file \
    "transcripts/${session_id}.jsonl" \
    "$transcript_path" \
    "transcript for ${commit_hash:0:12} (session ${session_id:0:8})"

pith_debug "Transcript captured to $PITH_BRANCH"
