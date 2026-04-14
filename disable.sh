#!/usr/bin/env bash
# lore/disable.sh — remove lore hooks + skill from the current project
#
# Usage: ~/.lore/disable.sh
#
# Run from the root of a git repository. This script:
#   1. Removes the lore hook from .claude/settings.json
#   2. Removes the /lore skill symlink
#   3. Removes the git notes display config
#
# Does NOT delete the lore/transcripts branch or existing git notes.
# Those are your data — delete them manually if you want.

set -euo pipefail

# Verify we're in a git repo
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: not inside a git repository" >&2
    exit 1
fi

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

echo "Disabling lore in $(basename "$repo_root")..."

# ---------------------------------------------------------------------------
# 1. Remove hook from .claude/settings.json
# ---------------------------------------------------------------------------

settings_file=".claude/settings.json"

if [[ -f "$settings_file" ]]; then
    if jq -e '.hooks.PostToolUse' "$settings_file" >/dev/null 2>&1; then
        tmp=$(mktemp)
        jq '
            .hooks.PostToolUse = [
                .hooks.PostToolUse[]
                | select(.hooks | all(.command != "~/.lore/scripts/lore-hook.sh"))
            ] |
            if .hooks.PostToolUse == [] then del(.hooks.PostToolUse) else . end |
            if .hooks == {} then del(.hooks) else . end
        ' "$settings_file" >"$tmp"
        mv -f "$tmp" "$settings_file"
        echo "  Hook removed from $settings_file"
    else
        echo "  No lore hook found in $settings_file"
    fi
else
    echo "  No $settings_file found"
fi

# ---------------------------------------------------------------------------
# 2. Remove skill symlink
# ---------------------------------------------------------------------------

for skills_dir in ".claude/skills" ".agents/skills"; do
    skill_target="$skills_dir/lore"
    if [[ -e "$skill_target" ]] || [[ -L "$skill_target" ]]; then
        rm -rf "$skill_target"
        echo "  Skill removed: $skill_target"
    fi
done

# ---------------------------------------------------------------------------
# 3. Remove git notes display config
# ---------------------------------------------------------------------------

if git config --get-all notes.displayRef 2>/dev/null | grep -q "refs/notes/lore"; then
    git config --unset notes.displayRef refs/notes/lore 2>/dev/null || true
    echo "  Git notes display config removed"
fi

echo ""
echo "lore disabled. Existing notes and transcripts are preserved."
echo ""
echo "To delete all lore data:"
echo "  git notes --ref=lore prune"
echo "  git branch -D lore/transcripts"
