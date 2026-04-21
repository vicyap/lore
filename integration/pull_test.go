package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPull_FetchesNotesAndTranscripts(t *testing.T) {
	// Bare origin: the "server" that the pusher pushes to and the puller pulls from.
	bareOrigin := t.TempDir()
	runCmd(t, bareOrigin, "git", "init", "--bare")

	// Pusher: create a normal repo, publish a commit, a note, and the transcripts branch.
	pusher := setupTestRepo(t)
	runCmd(t, pusher, "git", "remote", "add", "origin", bareOrigin)
	runCmd(t, pusher, "git", "branch", "-M", "main")
	runCmd(t, pusher, "git", "push", "-u", "origin", "main")

	// Add a lore note on HEAD.
	note := "## Decisions\n- Seeded note for pull test\n\n## Metadata\n- version: dev"
	runCmd(t, pusher, "git", "notes", "--ref=lore", "add", "-m", note)
	runCmd(t, pusher, "git", "push", "origin", "refs/notes/lore")

	// Create an orphan lore/transcripts branch with a transcript file, then push it.
	runCmd(t, pusher, "git", "checkout", "--orphan", "lore/transcripts")
	runCmd(t, pusher, "git", "rm", "-rf", ".")
	writeFile(t, pusher, "t/session.jsonl", "{}\n")
	runCmd(t, pusher, "git", "add", "t/session.jsonl")
	runCmd(t, pusher, "git", "commit", "-m", "add transcript")
	runCmd(t, pusher, "git", "push", "origin", "lore/transcripts")

	// Puller: fresh clone. A vanilla `git clone` does not bring notes or the orphan branch.
	puller := t.TempDir()
	cloneDest := filepath.Join(puller, "clone")
	runCmd(t, puller, "git", "clone", bareOrigin, cloneDest)

	// Sanity: confirm the clone is indeed missing notes before we pull.
	if out := runCmdOutput(t, cloneDest, "git", "notes", "--ref=lore", "list"); strings.TrimSpace(out) != "" {
		t.Fatalf("fresh clone unexpectedly has lore notes: %s", out)
	}

	// The command under test.
	stdout, stderr, exitCode := runLore(t, cloneDest, "pull")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	// Notes should now be present.
	if out := runCmdOutput(t, cloneDest, "git", "notes", "--ref=lore", "list"); strings.TrimSpace(out) == "" {
		t.Errorf("expected lore notes after pull, got none")
	}
	if !strings.Contains(stdout, "Fetched 1 notes") {
		t.Errorf("expected 'Fetched 1 notes' in output, got:\n%s", stdout)
	}

	// The orphan branch should be present locally.
	assertOrphanFileExists(t, cloneDest, "lore/transcripts", "t/session.jsonl")
	if !strings.Contains(stdout, "Fetched 1 transcripts") {
		t.Errorf("expected 'Fetched 1 transcripts' in output, got:\n%s", stdout)
	}
}

func TestPull_NoOriginFails(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, stderr, exitCode := runLore(t, dir, "pull")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit without origin, got 0\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "no origin remote") {
		t.Errorf("expected 'no origin remote' in output, got:\n%s", combined)
	}
}

func TestPull_NotGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, _, exitCode := runLore(t, dir, "pull")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit outside git repo")
	}
}
