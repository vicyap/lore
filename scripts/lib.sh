#!/usr/bin/env bash
# lore/scripts/lib.sh — shared utilities for lore hooks
# Sourced by other scripts, not executed directly.

set -euo pipefail

LORE_DIR="${LORE_DIR:-$HOME/.lore}"
LORE_BRANCH="lore/transcripts"
LORE_NOTES_REF="lore"

# ---------------------------------------------------------------------------
# JSON helpers (require jq)
# ---------------------------------------------------------------------------

lore_parse_field() {
    local json="$1" field="$2"
    echo "$json" | jq -r "$field // empty"
}

# ---------------------------------------------------------------------------
# Git helpers
# ---------------------------------------------------------------------------

lore_get_commit_hash() {
    git rev-parse HEAD
}

lore_get_diff() {
    local commit_hash="$1"
    # Handle first commit (no parent)
    if git rev-parse --verify "${commit_hash}^" >/dev/null 2>&1; then
        git diff "${commit_hash}^..${commit_hash}"
    else
        git show "$commit_hash" --format="" --diff-filter=ACMR
    fi
}

lore_get_commit_subject() {
    local commit_hash="$1"
    git log -1 --format="%s" "$commit_hash"
}

lore_get_branch_name() {
    git symbolic-ref --short HEAD 2>/dev/null || echo "detached"
}

# ---------------------------------------------------------------------------
# Git notes
# ---------------------------------------------------------------------------

lore_write_note() {
    local commit_hash="$1" content_file="$2"
    git notes --ref="$LORE_NOTES_REF" add -f --file="$content_file" "$commit_hash"
}

lore_read_note() {
    local commit_hash="$1"
    git notes --ref="$LORE_NOTES_REF" show "$commit_hash" 2>/dev/null || true
}

lore_has_note() {
    local commit_hash="$1"
    git notes --ref="$LORE_NOTES_REF" show "$commit_hash" >/dev/null 2>&1
}

# ---------------------------------------------------------------------------
# Orphan branch operations (git plumbing — no checkout needed)
# ---------------------------------------------------------------------------

lore_orphan_exists() {
    git rev-parse --verify "refs/heads/$LORE_BRANCH" >/dev/null 2>&1
}

lore_orphan_init() {
    if lore_orphan_exists; then
        return 0
    fi

    # Create an empty tree and initial commit
    local empty_tree
    empty_tree=$(git hash-object -t tree /dev/null)
    local init_commit
    init_commit=$(echo "Initialize lore transcripts" \
        | GIT_AUTHOR_NAME="lore" \
            GIT_AUTHOR_EMAIL="lore@local" \
            GIT_COMMITTER_NAME="lore" \
            GIT_COMMITTER_EMAIL="lore@local" \
            git commit-tree "$empty_tree")
    git update-ref "refs/heads/$LORE_BRANCH" "$init_commit"
}

lore_orphan_write_file() {
    local filepath="$1" source_file="$2" commit_message="$3"

    lore_orphan_init

    # Create blob from source file
    local blob_hash
    blob_hash=$(git hash-object -w "$source_file")

    # Get current tree (if branch has commits)
    local parent_commit current_tree
    parent_commit=$(git rev-parse "refs/heads/$LORE_BRANCH")
    current_tree=$(git rev-parse "${parent_commit}^{tree}")

    # Build new tree: read existing tree, replace/add our file
    # We use git-read-tree + git-update-index via a temp index
    local temp_index
    temp_index=$(mktemp)
    rm -f "$temp_index" # git needs it to not exist initially

    # Read existing tree into temp index
    GIT_INDEX_FILE="$temp_index" git read-tree "$current_tree"

    # Add/update our file in the temp index
    GIT_INDEX_FILE="$temp_index" git update-index --add --cacheinfo "100644,$blob_hash,$filepath"

    # Write the new tree
    local new_tree
    new_tree=$(GIT_INDEX_FILE="$temp_index" git write-tree)

    # Clean up temp index
    rm -f "$temp_index"

    # Create commit with parent
    local new_commit
    new_commit=$(echo "$commit_message" \
        | GIT_AUTHOR_NAME="lore" \
            GIT_AUTHOR_EMAIL="lore@local" \
            GIT_COMMITTER_NAME="lore" \
            GIT_COMMITTER_EMAIL="lore@local" \
            git commit-tree "$new_tree" -p "$parent_commit")

    # Update branch ref
    git update-ref "refs/heads/$LORE_BRANCH" "$new_commit"
}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

lore_log() {
    local level="$1"
    shift
    if [[ "${LORE_DEBUG:-}" == "1" ]] || [[ "$level" != "debug" ]]; then
        echo "[lore:$level] $*" >&2
    fi
}

lore_info() { lore_log "info" "$@"; }
lore_debug() { lore_log "debug" "$@"; }
lore_error() { lore_log "error" "$@"; }
