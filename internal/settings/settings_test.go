package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestInstallHook_Fresh(t *testing.T) {
	dir := setupTestDir(t)

	installed, err := InstallHook(dir)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !installed {
		t.Fatal("expected true for fresh install")
	}

	// Verify the file was created
	data, err := os.ReadFile(SettingsPath(dir))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	has, _ := HasHook(dir)
	if !has {
		t.Fatal("HasHook should return true after install")
	}
}

func TestInstallHook_Idempotent(t *testing.T) {
	dir := setupTestDir(t)

	InstallHook(dir)
	installed, err := InstallHook(dir)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if installed {
		t.Fatal("expected false for idempotent install")
	}
}

func TestInstallHook_PreservesExisting(t *testing.T) {
	dir := setupTestDir(t)

	// Create settings with existing data
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	existing := `{"permissions":{"allow":["Bash(git *)"]}, "hooks":{"PreToolUse":[{"matcher":"Read","hooks":[{"type":"command","command":"echo read"}]}]}}`
	os.WriteFile(SettingsPath(dir), []byte(existing), 0o644)

	installed, err := InstallHook(dir)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !installed {
		t.Fatal("expected true")
	}

	// Verify existing data is preserved
	data, _ := os.ReadFile(SettingsPath(dir))
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	if _, ok := settings["permissions"]; !ok {
		t.Error("existing 'permissions' key was lost")
	}

	// Verify PreToolUse is preserved
	var hooks map[string]json.RawMessage
	json.Unmarshal(settings["hooks"], &hooks)
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("existing 'PreToolUse' hook was lost")
	}
}

func TestRemoveHook(t *testing.T) {
	dir := setupTestDir(t)

	InstallHook(dir)

	removed, err := RemoveHook(dir)
	if err != nil {
		t.Fatalf("RemoveHook: %v", err)
	}
	if !removed {
		t.Fatal("expected true for removal")
	}

	has, _ := HasHook(dir)
	if has {
		t.Fatal("HasHook should return false after removal")
	}
}

func TestRemoveHook_NotInstalled(t *testing.T) {
	dir := setupTestDir(t)

	removed, err := RemoveHook(dir)
	if err != nil {
		t.Fatalf("RemoveHook: %v", err)
	}
	if removed {
		t.Fatal("expected false when hook not installed")
	}
}

func TestRemoveHook_PreservesOtherHooks(t *testing.T) {
	dir := setupTestDir(t)

	// Install lore hook plus another hook
	InstallHook(dir)

	// Add another PostToolUse hook manually
	settings, _ := Read(dir)
	var hooks map[string]json.RawMessage
	json.Unmarshal(settings["hooks"], &hooks)

	var postToolUse []HookEntry
	json.Unmarshal(hooks["PostToolUse"], &postToolUse)
	postToolUse = append(postToolUse, HookEntry{
		Matcher: "Read",
		Hooks:   []HookItem{{Type: "command", Command: "echo read"}},
	})
	updatedJSON, _ := json.Marshal(postToolUse)
	hooks["PostToolUse"] = updatedJSON
	hooksJSON, _ := json.Marshal(hooks)
	settings["hooks"] = hooksJSON
	Write(dir, settings)

	// Remove lore hook
	RemoveHook(dir)

	// The other hook should still be there
	settings, _ = Read(dir)
	json.Unmarshal(settings["hooks"], &hooks)
	if _, ok := hooks["PostToolUse"]; !ok {
		t.Fatal("PostToolUse should still exist with other hooks")
	}
}

func TestHasOldHook(t *testing.T) {
	dir := setupTestDir(t)

	// Install old-style hook
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	oldSettings := `{"hooks":{"PostToolUse":[{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(*git commit*)","command":"~/.lore/scripts/lore-hook.sh","timeout":120}]}]}}`
	os.WriteFile(SettingsPath(dir), []byte(oldSettings), 0o644)

	has, err := HasOldHook(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("should detect old hook")
	}
}

func TestReplaceOldHook(t *testing.T) {
	dir := setupTestDir(t)

	// Install old-style hook
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	oldSettings := `{"hooks":{"PostToolUse":[{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(*git commit*)","command":"~/.lore/scripts/lore-hook.sh","timeout":120}]}]}}`
	os.WriteFile(SettingsPath(dir), []byte(oldSettings), 0o644)

	if err := ReplaceOldHook(dir); err != nil {
		t.Fatal(err)
	}

	// Old hook should be gone
	has, _ := HasOldHook(dir)
	if has {
		t.Fatal("old hook should be replaced")
	}

	// New hook should be present
	has, _ = HasHook(dir)
	if has == false {
		t.Fatal("new hook should be installed")
	}
}
