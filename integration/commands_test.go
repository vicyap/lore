package integration

import (
	"strings"
	"testing"
)

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
