package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHook_SimpleCommit(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)

	payload := buildHookPayload("sess-001", transcriptPath, dir, `git commit -am "fix nil pointer"`)
	stdout, stderr, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	// Note should exist on HEAD
	assertNoteExists(t, dir, commitHash)
	assertNoteContains(t, dir, commitHash, "## Intent")

	// Transcript should be on orphan branch
	assertOrphanFileExists(t, dir, "lore/transcripts", "transcripts/sess-001.jsonl")

	// Fake claude should have been called with correct args
	logContent := readFile(t, logFile)
	if !strings.Contains(logContent, "--model sonnet") {
		t.Errorf("expected --model sonnet in claude log, got:\n%s", logContent)
	}
	if !strings.Contains(logContent, "## Commit") {
		t.Errorf("expected ## Commit in claude stdin, got:\n%s", logContent)
	}
	if !strings.Contains(logContent, "## Diff") {
		t.Errorf("expected ## Diff in claude stdin, got:\n%s", logContent)
	}
}

func TestHook_MultiCommitSession(t *testing.T) {
	dir := setupRepoWithCommits(t, 3)
	transcriptPath := copyTranscript(t, "transcript_multi.jsonl")
	commitHash := getHeadHash(t, dir)

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)

	payload := buildHookPayload("sess-002", transcriptPath, dir, `git commit -am "wire Slack notifications"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	assertNoteExists(t, dir, commitHash)

	// The windowed transcript should NOT contain first commit content
	logContent := readFile(t, logFile)
	if strings.Contains(logContent, "event type definitions") {
		t.Error("windowed transcript should not contain first commit's content")
	}
	// But should contain the last window's content
	if !strings.Contains(logContent, "Slack notification") {
		t.Error("windowed transcript should contain last window's content")
	}
}

func TestHook_FirstCommit(t *testing.T) {
	dir := setupTestRepo(t) // only has the initial commit
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)

	payload := buildHookPayload("sess-003", transcriptPath, dir, `git commit -m "initial"`)
	_, stderr, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d, stderr: %s", exitCode, stderr)
	}

	assertNoteExists(t, dir, commitHash)
}

func TestHook_NonCommitCommand(t *testing.T) {
	dir := setupTestRepo(t)
	commitHash := getHeadHash(t, dir)

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)

	payload := buildHookPayload("sess-004", "/dev/null", dir, "git status")
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// No note should be written
	assertNoNote(t, dir, commitHash)

	// Fake claude should NOT have been called
	if _, err := os.Stat(logFile); err == nil {
		t.Error("claude shim should not have been called for non-commit command")
	}
}

func TestHook_MalformedJSON(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, exitCode := runLoreWithStdin(t, dir, "{ not valid json", "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (non-fatal), got %d", exitCode)
	}
	if !strings.Contains(stderr, "failed to parse") {
		t.Errorf("expected parse error in stderr, got: %s", stderr)
	}
}

func TestHook_MissingFields(t *testing.T) {
	dir := setupTestRepo(t)

	payload := buildHookPayload("", "/dev/null", dir, "git commit -m test")
	_, stderr, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (non-fatal), got %d", exitCode)
	}
	if !strings.Contains(stderr, "missing required fields") {
		t.Errorf("expected missing fields error in stderr, got: %s", stderr)
	}
}

func TestHook_TranscriptNotFound(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)

	payload := buildHookPayload("sess-005", "/nonexistent/transcript.jsonl", dir, `git commit -m "test"`)
	_, stderr, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (non-fatal), got %d", exitCode)
	}
	// Should log error about transcript
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "failed") {
		t.Errorf("expected error about transcript in stderr, got: %s", stderr)
	}
}

func TestHook_ClaudeError(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)

	t.Setenv("FAKECLAUDE_EXIT_CODE", "1")

	payload := buildHookPayload("sess-006", transcriptPath, dir, `git commit -m "test"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (non-fatal), got %d", exitCode)
	}

	// Fallback note should be written
	assertNoteContains(t, dir, commitHash, "distillation failed")
}

func TestHook_ClaudeEmptyOutput(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")

	t.Setenv("FAKECLAUDE_EMPTY", "1")

	payload := buildHookPayload("sess-007", transcriptPath, dir, `git commit -m "test"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	// Should still exit cleanly (non-fatal)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// git notes doesn't accept empty content, so no note is written.
	// The transcript should still be captured on the orphan branch.
	assertOrphanFileExists(t, dir, "lore/transcripts", "transcripts/sess-007.jsonl")
}

func TestHook_DetachedHEAD(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)

	// Detach HEAD
	runCmd(t, dir, "git", "checkout", "--detach")

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)

	payload := buildHookPayload("sess-008", transcriptPath, dir, `git commit -m "test"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	assertNoteExists(t, dir, commitHash)

	logContent := readFile(t, logFile)
	if !strings.Contains(logContent, "Branch: detached") {
		t.Errorf("expected 'Branch: detached' in prompt, got:\n%s", logContent)
	}
}

func TestHook_CustomModel(t *testing.T) {
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)
	t.Setenv("LORE_MODEL", "haiku")

	payload := buildHookPayload("sess-009", transcriptPath, dir, `git commit -m "test"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	logContent := readFile(t, logFile)
	if !strings.Contains(logContent, "--model haiku") {
		t.Errorf("expected --model haiku in claude log, got:\n%s", logContent)
	}
}

func TestHook_LargeDiff(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a large file to generate a big diff
	bigContent := strings.Repeat("// line of code\n", 2000)
	writeFile(t, dir, "big.go", "package main\n\n"+bigContent)
	runCmd(t, dir, "git", "add", "big.go")
	runCmd(t, dir, "git", "commit", "-m", "add big file")

	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)
	t.Setenv("LORE_MAX_DIFF_CHARS", "1000")

	payload := buildHookPayload("sess-010", transcriptPath, dir, `git commit -m "big"`)
	_, _, exitCode := runLoreWithStdin(t, dir, payload, "hook")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	logContent := readFile(t, logFile)
	if !strings.Contains(logContent, "diff truncated at 1000 chars") {
		t.Errorf("expected diff truncation message in prompt, got:\n%s", logContent[:min(500, len(logContent))])
	}
}

func TestHook_InstructionInStdinNotArgs(t *testing.T) {
	// Regression test: the distill instruction must be in stdin (combined with
	// the prompt input), not as a positional CLI arg. When passed as a separate
	// arg, the model treats stdin as conversation context and echoes transcript
	// content into the note output.
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")

	logFile := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("FAKECLAUDE_LOG", logFile)

	payload := buildHookPayload("sess-regression", transcriptPath, dir, `git commit -m "test"`)
	runLoreWithStdin(t, dir, payload, "hook")

	logContent := readFile(t, logFile)

	// The instruction should appear in STDIN, not in ARGS
	if !strings.Contains(logContent, "STDIN:\n") {
		t.Fatal("expected STDIN section in log")
	}

	stdinStart := strings.Index(logContent, "STDIN:\n") + len("STDIN:\n")
	stdinContent := logContent[stdinStart:]

	if !strings.Contains(stdinContent, "Distill the decision reasoning") {
		t.Error("distill instruction should be in stdin, not as a CLI arg")
	}

	argsLine := strings.SplitN(logContent, "\n", 2)[0]
	if strings.Contains(argsLine, "Distill") {
		t.Error("distill instruction should NOT be in CLI args")
	}
}

func TestHook_NoteStartsWithSchema(t *testing.T) {
	// Regression test: the written note must start with "## Intent" (the first
	// section of the distill schema). If the model echoes transcript content
	// before the schema, this test catches it.
	dir := setupRepoWithCommits(t, 1)
	transcriptPath := copyTranscript(t, "transcript_simple.jsonl")
	commitHash := getHeadHash(t, dir)

	payload := buildHookPayload("sess-schema", transcriptPath, dir, `git commit -m "test"`)
	runLoreWithStdin(t, dir, payload, "hook")

	note := runCmdOutput(t, dir, "git", "notes", "--ref=lore", "show", commitHash)
	note = strings.TrimSpace(note)

	if !strings.HasPrefix(note, "## Intent") {
		t.Errorf("note should start with '## Intent', got:\n%s", note[:min(200, len(note))])
	}

	// Note should not contain transcript markers
	for _, marker := range []string{"**User:**", "**Assistant:**", "[Tool:"} {
		if strings.Contains(note, marker) {
			t.Errorf("note should not contain transcript marker %q", marker)
		}
	}
}
