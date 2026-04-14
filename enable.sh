#!/usr/bin/env bash
# lore/enable.sh — install lore hooks + skill into the current project
#
# Usage: ~/.lore/enable.sh
#
# Run from the root of a git repository. This script:
#   1. Merges the PostToolUse hook into .claude/settings.json
#   2. Symlinks the /lore skill into the project
#   3. Initializes the lore/transcripts orphan branch
#   4. Configures git to display lore notes in git log

set -euo pipefail

LORE_DIR="${LORE_DIR:-$HOME/.lore}"

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

echo "Enabling lore in $(basename "$repo_root")..."

# ---------------------------------------------------------------------------
# 1. Merge hook into .claude/settings.json
# ---------------------------------------------------------------------------

settings_file=".claude/settings.json"
mkdir -p .claude

lore_hook='{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "if": "Bash(git commit*)",
      "command": "~/.lore/scripts/lore-hook.sh",
      "timeout": 120,
      "statusMessage": "lore: distilling reasoning..."
    }
  ]
}'

if [[ -f "$settings_file" ]]; then
    # Check if lore hook is already installed
    if jq -e '.hooks.PostToolUse[]? | select(.hooks[]?.command == "~/.lore/scripts/lore-hook.sh")' "$settings_file" >/dev/null 2>&1; then
        echo "  Hook already installed in $settings_file"
    else
        # Merge: add to existing PostToolUse array, or create it
        tmp=$(mktemp)
        jq --argjson hook "$lore_hook" '
            .hooks //= {} |
            .hooks.PostToolUse //= [] |
            .hooks.PostToolUse += [$hook]
        ' "$settings_file" >"$tmp"
        mv -f "$tmp" "$settings_file"
        echo "  Hook added to $settings_file"
    fi
else
    # Create new settings file with just the hook
    jq -n --argjson hook "$lore_hook" '{
        hooks: {
            PostToolUse: [$hook]
        }
    }' >"$settings_file"
    echo "  Created $settings_file with lore hook"
fi

# ---------------------------------------------------------------------------
# 2. Symlink the /lore skill
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

skill_target="$skills_dir/lore"
if [[ -e "$skill_target" ]] || [[ -L "$skill_target" ]]; then
    echo "  Skill already linked at $skill_target"
else
    # Create directory for multi-file skill
    mkdir -p "$skill_target"
    ln -sf "$LORE_DIR/skill/lore.md" "$skill_target/lore.md"
    echo "  Skill linked: $skill_target -> $LORE_DIR/skill/lore.md"
fi

# ---------------------------------------------------------------------------
# 3. Initialize orphan branch
# ---------------------------------------------------------------------------

# shellcheck source=scripts/lib.sh
source "$LORE_DIR/scripts/lib.sh"

if lore_orphan_exists; then
    echo "  Orphan branch $LORE_BRANCH already exists"
else
    lore_orphan_init
    echo "  Created orphan branch $LORE_BRANCH"
fi

# ---------------------------------------------------------------------------
# 4. Configure git notes display
# ---------------------------------------------------------------------------

# Add lore notes to default display (so git log --notes shows them)
existing_refs=$(git config --get-all notes.displayRef 2>/dev/null || true)
if echo "$existing_refs" | grep -q "refs/notes/lore"; then
    echo "  Git notes display already configured"
else
    git config --add notes.displayRef refs/notes/lore
    echo "  Configured git to display lore notes"
fi

echo ""
echo "lore enabled. Decision notes will be captured on every commit."
echo ""
echo "View notes:       git log --notes=lore"
echo "View transcripts: git log lore/transcripts"
echo "Interactive:      /lore show (in Claude Code)"
