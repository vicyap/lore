#!/usr/bin/env bash
# pith/scripts/lib.sh — shared utilities for pith hooks
# Sourced by other scripts, not executed directly.

set -euo pipefail

PITH_DIR="${PITH_DIR:-$HOME/.pith}"
PITH_BRANCH="pith/transcripts"
PITH_NOTES_REF="pith"

# ---------------------------------------------------------------------------
# JSON helpers (require jq)
# ---------------------------------------------------------------------------

pith_parse_field() {
    local json="$1" field="$2"
    echo "$json" | jq -r "$field // empty"
}

# ---------------------------------------------------------------------------
# Git helpers
# ---------------------------------------------------------------------------

pith_get_commit_hash() {
    git rev-parse HEAD
}

pith_get_diff() {
    local commit_hash="$1"
    # Handle first commit (no parent)
    if git rev-parse --verify "${commit_hash}^" >/dev/null 2>&1; then
        git diff "${commit_hash}^..${commit_hash}"
    else
        git show "$commit_hash" --format="" --diff-filter=ACMR
    fi
}

pith_get_commit_subject() {
    local commit_hash="$1"
    git log -1 --format="%s" "$commit_hash"
}

pith_get_branch_name() {
    git symbolic-ref --short HEAD 2>/dev/null || echo "detached"
}

# ---------------------------------------------------------------------------
# Git notes
# ---------------------------------------------------------------------------

pith_write_note() {
    local commit_hash="$1" content_file="$2"
    git notes --ref="$PITH_NOTES_REF" add -f --file="$content_file" "$commit_hash"
}

pith_read_note() {
    local commit_hash="$1"
    git notes --ref="$PITH_NOTES_REF" show "$commit_hash" 2>/dev/null || true
}

pith_has_note() {
    local commit_hash="$1"
    git notes --ref="$PITH_NOTES_REF" show "$commit_hash" >/dev/null 2>&1
}

# ---------------------------------------------------------------------------
# Orphan branch operations (git plumbing — no checkout needed)
# ---------------------------------------------------------------------------

pith_orphan_exists() {
    git rev-parse --verify "refs/heads/$PITH_BRANCH" >/dev/null 2>&1
}

pith_orphan_init() {
    if pith_orphan_exists; then
        return 0
    fi

    # Create an empty tree and initial commit
    local empty_tree
    empty_tree=$(git hash-object -t tree /dev/null)
    local init_commit
    init_commit=$(echo "Initialize pith transcripts" \
        | GIT_AUTHOR_NAME="pith" \
            GIT_AUTHOR_EMAIL="pith@local" \
            GIT_COMMITTER_NAME="pith" \
            GIT_COMMITTER_EMAIL="pith@local" \
            git commit-tree "$empty_tree")
    git update-ref "refs/heads/$PITH_BRANCH" "$init_commit"
}

pith_orphan_write_file() {
    local filepath="$1" source_file="$2" commit_message="$3"

    pith_orphan_init

    # Create blob from source file
    local blob_hash
    blob_hash=$(git hash-object -w "$source_file")

    # Get current tree (if branch has commits)
    local parent_commit current_tree
    parent_commit=$(git rev-parse "refs/heads/$PITH_BRANCH")
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
        | GIT_AUTHOR_NAME="pith" \
            GIT_AUTHOR_EMAIL="pith@local" \
            GIT_COMMITTER_NAME="pith" \
            GIT_COMMITTER_EMAIL="pith@local" \
            git commit-tree "$new_tree" -p "$parent_commit")

    # Update branch ref
    git update-ref "refs/heads/$PITH_BRANCH" "$new_commit"
}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

pith_log() {
    local level="$1"
    shift
    if [[ "${PITH_DEBUG:-}" == "1" ]] || [[ "$level" != "debug" ]]; then
        echo "[pith:$level] $*" >&2
    fi
}

pith_info() { pith_log "info" "$@"; }
pith_debug() { pith_log "debug" "$@"; }
pith_error() { pith_log "error" "$@"; }
