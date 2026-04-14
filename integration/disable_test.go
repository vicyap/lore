package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDisable_RemovesHook(t *testing.T) {
	dir := setupTestRepo(t)
	runLoreWithStdin(t, dir, "n\n", "init")

	stdout, _, exitCode := runLore(t, dir, "disable")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "Hook removed") {
		t.Errorf("expected 'Hook removed' in output, got:\n%s", stdout)
	}

	// Verify hook is gone
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if strings.Contains(string(data), "lore hook") {
		t.Error("hook should be removed from settings.json")
	}

	// Verify notes.displayRef is removed
	cmd := exec.Command("git", "config", "--get-all", "notes.displayRef")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if strings.Contains(string(out), "refs/notes/lore") {
		t.Error("notes.displayRef should be removed")
	}
}

func TestDisable_PreservesData(t *testing.T) {
	dir := setupTestRepo(t)
	runLoreWithStdin(t, dir, "n\n", "init")

	// Create some data via hook
	writeFile(t, dir, "feature.go", "package main\n\nfunc feature() {}\n")
	runCmd(t, dir, "git", "add", "feature.go")
	runCmd(t, dir, "git", "commit", "-m", "add feature")

	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)
	payload := buildHookPayload("sess-preserve", transcriptPath, dir, `git commit -m "add feature"`)
	runLoreWithStdin(t, dir, payload, "hook")

	// Now disable
	runLore(t, dir, "disable")

	// Notes should still be readable
	assertNoteExists(t, dir, commitHash)

	// Orphan branch should still exist
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/lore/transcripts")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Error("orphan branch should still exist after disable")
	}
}

func TestDisable_PreservesOtherHooks(t *testing.T) {
	dir := setupTestRepo(t)

	// Install lore
	runLoreWithStdin(t, dir, "n\n", "init")

	// Add another hook manually
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	// Inject another PostToolUse entry
	modified := strings.Replace(string(data), `"lore hook"`, `"lore hook"}]}},{"matcher":"Read","hooks":[{"type":"command","command":"echo read","timeout":10`, 1)
	os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(modified), 0o644)

	runLore(t, dir, "disable")

	// The other hook should survive
	data, _ = os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if !strings.Contains(string(data), "echo read") {
		t.Error("other hooks should be preserved after disable")
	}
}

func TestDisable_ThenReInit(t *testing.T) {
	dir := setupTestRepo(t)

	// Init, create data, disable, re-init
	runLoreWithStdin(t, dir, "n\n", "init")

	writeFile(t, dir, "f.go", "package main\n")
	runCmd(t, dir, "git", "add", "f.go")
	runCmd(t, dir, "git", "commit", "-m", "add f")

	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	payload := buildHookPayload("sess-reinit", transcriptPath, dir, `git commit -m "add f"`)
	runLoreWithStdin(t, dir, payload, "hook")

	runLore(t, dir, "disable")
	runLoreWithStdin(t, dir, "n\n", "init")

	// Verify hook is back
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if !strings.Contains(string(data), "lore hook") {
		t.Error("hook should be reinstalled after re-init")
	}

	// Old data should still be there
	commitHash := getHeadHash(t, dir)
	assertNoteExists(t, dir, commitHash)
}
