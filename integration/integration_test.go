package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	loreBinary      string
	fakeClaudeBinary string
	originalPath    string
	testBinDir      string
)

func TestMain(m *testing.M) {
	// Create a temp dir for built binaries
	var err error
	testBinDir, err = os.MkdirTemp("", "lore-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(testBinDir)

	// Get project root (one level up from integration/)
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get project root: %v\n", err)
		os.Exit(1)
	}

	// Build lore binary
	loreBinary = filepath.Join(testBinDir, "lore")
	cmd := exec.Command("go", "build", "-o", loreBinary, "./cmd/lore")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build lore: %s\n%v\n", out, err)
		os.Exit(1)
	}

	// Build fake claude shim
	fakeClaudeBinary = filepath.Join(testBinDir, "claude")
	cmd = exec.Command("go", "build", "-o", fakeClaudeBinary, "./integration/testdata/fakeclaude")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fakeclaude: %s\n%v\n", out, err)
		os.Exit(1)
	}

	// Prepend test bin dir to PATH
	originalPath = os.Getenv("PATH")
	os.Setenv("PATH", testBinDir+":"+originalPath)

	// Set deterministic git identity
	os.Setenv("GIT_AUTHOR_NAME", "Test Author")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test Author")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")

	code := m.Run()

	os.Setenv("PATH", originalPath)
	os.Exit(code)
}

// setupTestRepo creates a temporary git repo with an initial commit.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test Author")

	writeFile(t, dir, "README.md", "# test\n")
	runCmd(t, dir, "git", "add", "README.md")
	runCmd(t, dir, "git", "commit", "-m", "initial commit")

	return dir
}

// setupRepoWithCommits creates a repo with n additional commits beyond the initial.
func setupRepoWithCommits(t *testing.T, count int) string {
	t.Helper()
	dir := setupTestRepo(t)

	for idx := range count {
		filename := fmt.Sprintf("file_%d.go", idx)
		content := fmt.Sprintf("package main\n\nfunc f%d() {}\n", idx)
		writeFile(t, dir, filename, content)
		runCmd(t, dir, "git", "add", filename)
		runCmd(t, dir, "git", "commit", "-m", fmt.Sprintf("add %s", filename))
	}

	return dir
}

// setupRepoWithNotes creates a repo with pre-seeded lore notes.
func setupRepoWithNotes(t *testing.T, count int) string {
	t.Helper()
	dir := setupRepoWithCommits(t, count)

	// Get all commit hashes
	out := runCmdOutput(t, dir, "git", "log", "--format=%H", "--reverse")
	hashes := strings.Split(strings.TrimSpace(out), "\n")

	for idx, hash := range hashes {
		note := fmt.Sprintf("## Intent\nTest note %d\n\n## Confidence\nhigh\n\n## Session\nsess-%d | main", idx, idx)
		runCmd(t, dir, "git", "notes", "--ref=lore", "add", "-m", note, hash)
	}

	return dir
}

// runLore executes the lore binary and returns stdout, stderr, and exit code.
func runLore(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()
	return runLoreWithStdin(t, dir, "", args...)
}

// runLoreWithStdin executes the lore binary with stdin and returns stdout, stderr, exit code.
func runLoreWithStdin(t *testing.T, dir, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(loreBinary, args...)
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run lore: %v", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// buildHookPayload creates a hook JSON string.
func buildHookPayload(sessionID, transcriptPath, cwd, command string) string {
	payload := map[string]any{
		"session_id":      sessionID,
		"transcript_path": transcriptPath,
		"cwd":             cwd,
		"tool_input": map[string]string{
			"command": command,
		},
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

// copyTranscript copies a fixture JSONL to a temp location and returns the path.
func copyTranscript(t *testing.T, fixtureName string) string {
	t.Helper()
	// Resolve fixture path relative to this test file
	src := filepath.Join("..", "testdata", fixtureName)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}

	dst := filepath.Join(t.TempDir(), fixtureName)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return dst
}

// getHeadHash returns the HEAD commit hash.
func getHeadHash(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(runCmdOutput(t, dir, "git", "rev-parse", "HEAD"))
}

// assertNoteExists checks that a lore note exists for the given commit.
func assertNoteExists(t *testing.T, dir, commitHash string) {
	t.Helper()
	cmd := exec.Command("git", "notes", "--ref=lore", "show", commitHash)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("expected note on %s, got error: %s: %v", commitHash[:8], out, err)
	}
}

// assertNoteContains checks that a lore note exists and contains the given substring.
func assertNoteContains(t *testing.T, dir, commitHash, substr string) {
	t.Helper()
	cmd := exec.Command("git", "notes", "--ref=lore", "show", commitHash)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("read note for %s: %v", commitHash[:8], err)
	}
	if !strings.Contains(string(out), substr) {
		t.Errorf("note for %s should contain %q, got:\n%s", commitHash[:8], substr, out)
	}
}

// assertNoNote checks that no lore note exists for the given commit.
func assertNoNote(t *testing.T, dir, commitHash string) {
	t.Helper()
	cmd := exec.Command("git", "notes", "--ref=lore", "show", commitHash)
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		t.Errorf("expected no note on %s, but one exists", commitHash[:8])
	}
}

// assertOrphanFileExists checks a file exists on the orphan branch.
func assertOrphanFileExists(t *testing.T, dir, branch, filepath string) {
	t.Helper()
	cmd := exec.Command("git", "show", "refs/heads/"+branch+":"+filepath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("expected file %s on branch %s: %s: %v", filepath, branch, out, err)
	}
}

// runCmd runs a command and fails the test on error.
func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}

// runCmdOutput runs a command and returns stdout.
func runCmdOutput(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return string(out)
}

// writeFile creates a file in dir with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readFile reads a file from dir.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
