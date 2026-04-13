#!/usr/bin/env bash
# pith/enable.sh — install pith hooks + skill into the current project
#
# Usage: ~/.pith/enable.sh
#
# Run from the root of a git repository. This script:
#   1. Merges the PostToolUse hook into .claude/settings.json
#   2. Symlinks the /pith skill into the project
#   3. Initializes the pith/transcripts orphan branch
#   4. Configures git to display pith notes in git log

set -euo pipefail

PITH_DIR="${PITH_DIR:-$HOME/.pith}"

# Verify we're in a git repo
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: not inside a git repository" >&2
    exit 1
fi

# Verify dependencies
for cmd in jq python3 claude git; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Error: $cmd is required but not found" >&2
        exit 1
    fi
done

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

echo "Enabling pith in $(basename "$repo_root")..."

# ---------------------------------------------------------------------------
# 1. Merge hook into .claude/settings.json
# ---------------------------------------------------------------------------

settings_file=".claude/settings.json"
mkdir -p .claude

pith_hook='{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "if": "Bash(git commit*)",
      "command": "~/.pith/scripts/pith-hook.sh",
      "timeout": 120,
      "statusMessage": "pith: distilling reasoning..."
    }
  ]
}'

if [[ -f "$settings_file" ]]; then
    # Check if pith hook is already installed
    if jq -e '.hooks.PostToolUse[]? | select(.hooks[]?.command == "~/.pith/scripts/pith-hook.sh")' "$settings_file" >/dev/null 2>&1; then
        echo "  Hook already installed in $settings_file"
    else
        # Merge: add to existing PostToolUse array, or create it
        tmp=$(mktemp)
        jq --argjson hook "$pith_hook" '
            .hooks //= {} |
            .hooks.PostToolUse //= [] |
            .hooks.PostToolUse += [$hook]
        ' "$settings_file" >"$tmp"
        mv -f "$tmp" "$settings_file"
        echo "  Hook added to $settings_file"
    fi
else
    # Create new settings file with just the hook
    jq -n --argjson hook "$pith_hook" '{
        hooks: {
            PostToolUse: [$hook]
        }
    }' >"$settings_file"
    echo "  Created $settings_file with pith hook"
fi

# ---------------------------------------------------------------------------
# 2. Symlink the /pith skill
# ---------------------------------------------------------------------------

# Detect skills directory: prefer .claude/skills if it exists or is a symlink
if [[ -d ".claude/skills" ]] || [[ -L ".claude/skills" ]]; then
    skills_dir=".claude/skills"
elif [[ -d ".agents/skills" ]]; then
    skills_dir=".agents/skills"
else
    skills_dir=".claude/skills"
    mkdir -p "$skills_dir"
fi

skill_target="$skills_dir/pith"
if [[ -e "$skill_target" ]] || [[ -L "$skill_target" ]]; then
    echo "  Skill already linked at $skill_target"
else
    # Create directory for multi-file skill
    mkdir -p "$skill_target"
    ln -sf "$PITH_DIR/skill/pith.md" "$skill_target/pith.md"
    echo "  Skill linked: $skill_target -> $PITH_DIR/skill/pith.md"
fi

# ---------------------------------------------------------------------------
# 3. Initialize orphan branch
# ---------------------------------------------------------------------------

# shellcheck source=scripts/lib.sh
source "$PITH_DIR/scripts/lib.sh"

if pith_orphan_exists; then
    echo "  Orphan branch $PITH_BRANCH already exists"
else
    pith_orphan_init
    echo "  Created orphan branch $PITH_BRANCH"
fi

# ---------------------------------------------------------------------------
# 4. Configure git notes display
# ---------------------------------------------------------------------------

# Add pith notes to default display (so git log --notes shows them)
existing_refs=$(git config --get-all notes.displayRef 2>/dev/null || true)
if echo "$existing_refs" | grep -q "refs/notes/pith"; then
    echo "  Git notes display already configured"
else
    git config --add notes.displayRef refs/notes/pith
    echo "  Configured git to display pith notes"
fi

echo ""
echo "pith enabled. Decision notes will be captured on every commit."
echo ""
echo "View notes:       git log --notes=pith"
echo "View transcripts: git log pith/transcripts"
echo "Interactive:      /pith show (in Claude Code)"
