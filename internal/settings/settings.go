package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const hookCommand = "lore hook"

// HookEntry matches the Claude Code PostToolUse hook structure.
type HookEntry struct {
	Matcher string     `json:"matcher"`
	Hooks   []HookItem `json:"hooks"`
}

// HookItem is a single hook within a HookEntry.
type HookItem struct {
	Type          string `json:"type"`
	If            string `json:"if,omitempty"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// Settings represents .claude/settings.json.
type Settings struct {
	Hooks map[string][]HookEntry `json:"hooks,omitempty"`
	// Preserve unknown fields
	Extra map[string]json.RawMessage `json:"-"`
}

// loreHookEntry returns the standard lore hook entry.
func loreHookEntry() HookEntry {
	return HookEntry{
		Matcher: "Bash",
		Hooks: []HookItem{
			{
				Type:          "command",
				If:            "Bash(*git commit*)",
				Command:       hookCommand,
				Timeout:       120,
				StatusMessage: "lore: distilling reasoning...",
			},
		},
	}
}

// SettingsPath returns the path to .claude/settings.json relative to the repo root.
func SettingsPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".claude", "settings.json")
}

// Read reads and parses .claude/settings.json.
func Read(repoRoot string) (map[string]json.RawMessage, error) {
	path := SettingsPath(repoRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return raw, nil
}

// Write writes settings back to .claude/settings.json with indentation.
func Write(repoRoot string, settings map[string]json.RawMessage) error {
	path := SettingsPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// HasHook checks if the lore hook is already installed.
func HasHook(repoRoot string) (bool, error) {
	settings, err := Read(repoRoot)
	if err != nil {
		return false, err
	}
	return hasLoreHook(settings), nil
}

// InstallHook adds the lore PostToolUse hook to settings.json.
// Returns true if newly installed, false if already present.
func InstallHook(repoRoot string) (bool, error) {
	settings, err := Read(repoRoot)
	if err != nil {
		return false, err
	}

	if hasLoreHook(settings) {
		return false, nil
	}

	// Get or create hooks object
	var hooks map[string]json.RawMessage
	if raw, ok := settings["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			hooks = make(map[string]json.RawMessage)
		}
	} else {
		hooks = make(map[string]json.RawMessage)
	}

	// Get or create PostToolUse array
	var postToolUse []HookEntry
	if raw, ok := hooks["PostToolUse"]; ok {
		_ = json.Unmarshal(raw, &postToolUse)
	}

	postToolUse = append(postToolUse, loreHookEntry())

	postToolUseJSON, _ := json.Marshal(postToolUse)
	hooks["PostToolUse"] = postToolUseJSON

	hooksJSON, _ := json.Marshal(hooks)
	settings["hooks"] = hooksJSON

	return true, Write(repoRoot, settings)
}

// RemoveHook removes the lore hook from settings.json.
// Returns true if removed, false if not found.
func RemoveHook(repoRoot string) (bool, error) {
	settings, err := Read(repoRoot)
	if err != nil {
		return false, err
	}

	if !hasLoreHook(settings) {
		return false, nil
	}

	var hooks map[string]json.RawMessage
	raw, ok := settings["hooks"]
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return false, nil
	}

	rawPTU, ok := hooks["PostToolUse"]
	if !ok {
		return false, nil
	}

	var postToolUse []HookEntry
	if err := json.Unmarshal(rawPTU, &postToolUse); err != nil {
		return false, nil
	}

	// Filter out lore hooks
	var filtered []HookEntry
	for _, entry := range postToolUse {
		if !isLoreHookEntry(entry) {
			filtered = append(filtered, entry)
		}
	}

	if len(filtered) == 0 {
		delete(hooks, "PostToolUse")
	} else {
		filteredJSON, _ := json.Marshal(filtered)
		hooks["PostToolUse"] = filteredJSON
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		hooksJSON, _ := json.Marshal(hooks)
		settings["hooks"] = hooksJSON
	}

	return true, Write(repoRoot, settings)
}

// HasOldHook checks if the old bash-based lore hook is installed.
func HasOldHook(repoRoot string) (bool, error) {
	settings, err := Read(repoRoot)
	if err != nil {
		return false, err
	}
	return hasOldLoreHook(settings), nil
}

// ReplaceOldHook replaces the old bash hook with the new CLI hook.
func ReplaceOldHook(repoRoot string) error {
	settings, err := Read(repoRoot)
	if err != nil {
		return err
	}

	var hooks map[string]json.RawMessage
	raw, ok := settings["hooks"]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return nil
	}

	rawPTU, ok := hooks["PostToolUse"]
	if !ok {
		return nil
	}

	var postToolUse []HookEntry
	if err := json.Unmarshal(rawPTU, &postToolUse); err != nil {
		return nil
	}

	// Replace old hooks with new
	var updated []HookEntry
	replaced := false
	for _, entry := range postToolUse {
		if isOldLoreHookEntry(entry) {
			if !replaced {
				updated = append(updated, loreHookEntry())
				replaced = true
			}
			// Skip duplicate old entries
		} else {
			updated = append(updated, entry)
		}
	}

	updatedJSON, _ := json.Marshal(updated)
	hooks["PostToolUse"] = updatedJSON
	hooksJSON, _ := json.Marshal(hooks)
	settings["hooks"] = hooksJSON

	return Write(repoRoot, settings)
}

func hasLoreHook(settings map[string]json.RawMessage) bool {
	raw, ok := settings["hooks"]
	if !ok {
		return false
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return false
	}

	rawPTU, ok := hooks["PostToolUse"]
	if !ok {
		return false
	}

	var postToolUse []HookEntry
	if err := json.Unmarshal(rawPTU, &postToolUse); err != nil {
		return false
	}

	for _, entry := range postToolUse {
		if isLoreHookEntry(entry) {
			return true
		}
	}
	return false
}

func hasOldLoreHook(settings map[string]json.RawMessage) bool {
	raw, ok := settings["hooks"]
	if !ok {
		return false
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return false
	}

	rawPTU, ok := hooks["PostToolUse"]
	if !ok {
		return false
	}

	var postToolUse []HookEntry
	if err := json.Unmarshal(rawPTU, &postToolUse); err != nil {
		return false
	}

	for _, entry := range postToolUse {
		if isOldLoreHookEntry(entry) {
			return true
		}
	}
	return false
}

func isLoreHookEntry(entry HookEntry) bool {
	for _, hook := range entry.Hooks {
		if hook.Command == hookCommand {
			return true
		}
	}
	return false
}

func isOldLoreHookEntry(entry HookEntry) bool {
	for _, hook := range entry.Hooks {
		if hook.Command == "~/.lore/scripts/lore-hook.sh" {
			return true
		}
	}
	return false
}
