package integration

import (
	"strings"
	"testing"
)

func TestInstalled_BinaryOnPath(t *testing.T) {
	stdout, _, exitCode := runLore(t, t.TempDir(), "--version")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if stdout == "" {
		t.Error("expected version output")
	}
}

func TestInstalled_FullCycle(t *testing.T) {
	dir := setupTestRepo(t)

	// Init
	stdout, _, exitCode := runLoreWithStdin(t, dir, "n\n", "init")
	if exitCode != 0 {
		t.Fatalf("init failed: exit %d, output: %s", exitCode, stdout)
	}

	// Make a commit
	writeFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	runCmd(t, dir, "git", "add", "main.go")
	runCmd(t, dir, "git", "commit", "-m", "add main.go")

	// Run hook
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)
	payload := buildHookPayload("sess-full-cycle", transcriptPath, dir, `git commit -m "add main.go"`)
	_, stderr, exitCode := runLoreWithStdin(t, dir, payload, "hook")
	if exitCode != 0 {
		t.Fatalf("hook failed: exit %d, stderr: %s", exitCode, stderr)
	}

	// Show should have the note
	stdout, _, exitCode = runLore(t, dir, "show", commitHash)
	if exitCode != 0 {
		t.Fatalf("show failed: exit %d", exitCode)
	}
	if !strings.Contains(stdout, "## Decisions") {
		t.Errorf("expected note in show output, got:\n%s", stdout)
	}

	// Status should show note count
	stdout, _, exitCode = runLore(t, dir, "status")
	if exitCode != 0 {
		t.Fatalf("status failed: exit %d", exitCode)
	}
	if !strings.Contains(stdout, "Notes:") {
		t.Errorf("expected note count in status, got:\n%s", stdout)
	}

	// Export should work
	stdout, _, exitCode = runLore(t, dir, "export", "--format", "json")
	if exitCode != 0 {
		t.Fatalf("export failed: exit %d", exitCode)
	}
	if !strings.Contains(stdout, "Decisions") {
		t.Errorf("expected note content in export, got:\n%s", stdout)
	}

	// Disable should preserve data
	stdout, _, exitCode = runLore(t, dir, "disable")
	if exitCode != 0 {
		t.Fatalf("disable failed: exit %d", exitCode)
	}

	// Note should still be readable after disable
	assertNoteExists(t, dir, commitHash)
}
