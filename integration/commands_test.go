package integration

import (
	"strings"
	"testing"
)

func TestShow_RecentDefault(t *testing.T) {
	dir := setupRepoWithNotes(t, 3)

	stdout, _, exitCode := runLore(t, dir, "show")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// Should contain notes
	if !strings.Contains(stdout, "## Decisions") {
		t.Errorf("expected notes in output, got:\n%s", stdout)
	}
}

func TestShow_WithCount(t *testing.T) {
	dir := setupRepoWithNotes(t, 5)

	stdout, _, exitCode := runLore(t, dir, "show", "2")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// Count how many note sections appear
	noteCount := strings.Count(stdout, "## Decisions")
	if noteCount > 2 {
		t.Errorf("expected at most 2 notes, got %d", noteCount)
	}
}

func TestShow_SpecificCommit(t *testing.T) {
	dir := setupRepoWithNotes(t, 1)
	commitHash := getHeadHash(t, dir)

	stdout, _, exitCode := runLore(t, dir, "show", commitHash)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "## Decisions") {
		t.Errorf("expected note for specific commit, got:\n%s", stdout)
	}
}

func TestShow_NoNotes(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, exitCode := runLore(t, dir, "show")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// When no notes exist, git log --notes=lore still shows commits but with empty note sections.
	// The output should not contain any "## Decisions" sections (which only appear in real notes).
	if strings.Contains(stdout, "## Decisions") {
		t.Errorf("expected no note content, got:\n%s", stdout)
	}
}

func TestStatus_Enabled(t *testing.T) {
	dir := setupTestRepo(t)
	runLoreWithStdin(t, dir, "n\n", "init")

	stdout, _, exitCode := runLore(t, dir, "status")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "yes") {
		t.Errorf("expected 'yes' for enabled status, got:\n%s", stdout)
	}
}

func TestExport_JSON(t *testing.T) {
	dir := setupRepoWithNotes(t, 2)

	stdout, _, exitCode := runLore(t, dir, "export", "--format", "json")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// Should be valid JSONL
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 {
		t.Fatal("expected JSONL output")
	}
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "{") {
			t.Errorf("expected JSON line, got: %s", line)
		}
	}
}

func TestExport_Markdown(t *testing.T) {
	dir := setupRepoWithNotes(t, 2)

	stdout, _, exitCode := runLore(t, dir, "export", "--format", "md")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "# Lore Decision Notes") {
		t.Errorf("expected markdown header, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "---") {
		t.Errorf("expected separators in markdown output")
	}
}
