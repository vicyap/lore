package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_FreshRepo(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, stderr, exitCode := runLoreWithStdin(t, dir, "n\n", "init")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	if !strings.Contains(stdout, "lore enabled") {
		t.Errorf("expected 'lore enabled' in output, got:\n%s", stdout)
	}

	// Verify settings.json has hook
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	if !strings.Contains(string(data), "lore hook") {
		t.Errorf("settings.json should contain 'lore hook', got:\n%s", data)
	}

	// Verify orphan branch exists
	out := runCmdOutput(t, dir, "git", "rev-parse", "--verify", "refs/heads/lore/transcripts")
	if strings.TrimSpace(out) == "" {
		t.Error("orphan branch should exist")
	}

	// Verify notes.displayRef
	out = runCmdOutput(t, dir, "git", "config", "--get-all", "notes.displayRef")
	if !strings.Contains(out, "refs/notes/lore") {
		t.Errorf("notes.displayRef should include refs/notes/lore, got: %s", out)
	}
}

func TestInit_Idempotent(t *testing.T) {
	dir := setupTestRepo(t)

	runLoreWithStdin(t, dir, "n\n", "init")
	stdout, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "already installed") {
		t.Errorf("expected 'already installed' in output, got:\n%s", stdout)
	}

	// Verify only one hook entry
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	count := strings.Count(string(data), "lore hook")
	if count != 1 {
		t.Errorf("expected exactly 1 lore hook entry, found %d", count)
	}
}

func TestInit_PreservesExistingSettings(t *testing.T) {
	dir := setupTestRepo(t)

	// Create existing settings
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	existing := `{"permissions":{"allow":["Bash(git *)"]},"hooks":{"PreToolUse":[{"matcher":"Read","hooks":[{"type":"command","command":"echo read"}]}]}}`
	os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(existing), 0o644)

	runLoreWithStdin(t, dir, "n\n", "init")

	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	if _, ok := settings["permissions"]; !ok {
		t.Error("existing 'permissions' key was lost")
	}

	var hooks map[string]json.RawMessage
	json.Unmarshal(settings["hooks"], &hooks)
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("existing 'PreToolUse' hook was lost")
	}
	if _, ok := hooks["PostToolUse"]; !ok {
		t.Error("PostToolUse should have been added")
	}
}

func TestInit_NotGitRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo

	_, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for non-git directory")
	}
}

func TestInit_ConfiguresFetchRefspec(t *testing.T) {
	dir := setupTestRepo(t)
	bareOrigin := t.TempDir()
	runCmd(t, bareOrigin, "git", "init", "--bare")
	runCmd(t, dir, "git", "remote", "add", "origin", bareOrigin)

	_, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	out := runCmdOutput(t, dir, "git", "config", "--get-all", "remote.origin.fetch")
	if !strings.Contains(out, "+refs/notes/*:refs/notes/*") {
		t.Errorf("remote.origin.fetch should include +refs/notes/*:refs/notes/*, got:\n%s", out)
	}

	// Second init must be idempotent — no duplicate refspec entry.
	runLoreWithStdin(t, dir, "n\n", "init")
	out = runCmdOutput(t, dir, "git", "config", "--get-all", "remote.origin.fetch")
	if count := strings.Count(out, "+refs/notes/*:refs/notes/*"); count != 1 {
		t.Errorf("expected exactly 1 notes refspec entry, found %d:\n%s", count, out)
	}
}

func TestInit_SkipsFetchRefspecWithoutOrigin(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "No origin remote") {
		t.Errorf("expected 'No origin remote' message when no origin configured, got:\n%s", stdout)
	}
}

func TestInit_SkipsWorkflowPromptWhenInstalled(t *testing.T) {
	dir := setupTestRepo(t)

	// Pre-install the workflow so the prompt should be skipped.
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "lore.yml"), []byte("name: lore\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Deliberately pass NO stdin — if the prompt fires and tries to read,
	// we'd get an empty answer (and defaultYes=true would overwrite the file).
	stdout, _, exitCode := runLore(t, dir, "init")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "workflow already installed") {
		t.Errorf("expected 'workflow already installed' in output, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "Install GitHub Actions workflow?") {
		t.Errorf("prompt should have been skipped, but question was printed:\n%s", stdout)
	}

	// The pre-existing placeholder content must be preserved.
	content := readFile(t, filepath.Join(dir, ".github", "workflows", "lore.yml"))
	if strings.TrimSpace(content) != "name: lore" {
		t.Errorf("existing workflow content should be preserved, got:\n%s", content)
	}
}

func TestInit_ReplacesOldHook(t *testing.T) {
	dir := setupTestRepo(t)

	// Install old-style hook
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	oldSettings := `{"hooks":{"PostToolUse":[{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(*git commit*)","command":"~/.lore/scripts/lore-hook.sh","timeout":120}]}]}}`
	os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(oldSettings), 0o644)

	stdout, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "Replaced old hook") {
		t.Errorf("expected 'Replaced old hook' in output, got:\n%s", stdout)
	}

	// Verify old hook gone, new hook present
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	content := string(data)
	if strings.Contains(content, "lore-hook.sh") {
		t.Error("old hook should be replaced")
	}
	if !strings.Contains(content, "lore hook") {
		t.Error("new hook should be installed")
	}
}
