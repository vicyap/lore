package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}

	// Create initial commit
	dummyFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "README.md"},
		{"git", "commit", "-m", "initial commit"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}

	return dir
}

func TestOrphanInit(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	branch := "lore/test-transcripts"

	if OrphanExists(branch) {
		t.Fatal("branch should not exist yet")
	}

	if err := OrphanInit(branch); err != nil {
		t.Fatalf("OrphanInit: %v", err)
	}

	if !OrphanExists(branch) {
		t.Fatal("branch should exist after init")
	}

	// Idempotent
	if err := OrphanInit(branch); err != nil {
		t.Fatalf("OrphanInit (idempotent): %v", err)
	}
}

func TestOrphanWriteAndReadFile(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	branch := "lore/test-transcripts"

	// Write a file
	sourceFile := filepath.Join(dir, "test-content.txt")
	if err := os.WriteFile(sourceFile, []byte("hello from lore"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := OrphanWriteFile(branch, "transcripts/sess-001.jsonl", sourceFile, "test commit"); err != nil {
		t.Fatalf("OrphanWriteFile: %v", err)
	}

	// Read it back
	content, err := OrphanReadFile(branch, "transcripts/sess-001.jsonl")
	if err != nil {
		t.Fatalf("OrphanReadFile: %v", err)
	}
	if content != "hello from lore" {
		t.Errorf("expected 'hello from lore', got %q", content)
	}

	// Working tree should not be affected
	files, err := filepath.Glob(filepath.Join(dir, "transcripts", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("orphan write should not create files in working tree, found: %v", files)
	}
}

func TestOrphanWriteMultipleFiles(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	branch := "lore/test-transcripts"

	for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
		sourceFile := filepath.Join(dir, name)
		if err := os.WriteFile(sourceFile, []byte("content of "+name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := OrphanWriteFile(branch, "data/"+name, sourceFile, "add "+name); err != nil {
			t.Fatalf("OrphanWriteFile %s: %v", name, err)
		}
	}

	// List files
	files, err := OrphanListFiles(branch)
	if err != nil {
		t.Fatalf("OrphanListFiles: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}

	// Verify each file
	for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
		content, err := OrphanReadFile(branch, "data/"+name)
		if err != nil {
			t.Fatalf("OrphanReadFile %s: %v", name, err)
		}
		expected := "content of " + name
		if content != expected {
			t.Errorf("file %s: expected %q, got %q", name, expected, content)
		}
	}
}

func TestOrphanOverwriteFile(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	branch := "lore/test-transcripts"

	// Write initial content
	sourceFile := filepath.Join(dir, "content.txt")
	if err := os.WriteFile(sourceFile, []byte("version 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := OrphanWriteFile(branch, "data/file.txt", sourceFile, "v1"); err != nil {
		t.Fatal(err)
	}

	// Overwrite with new content
	if err := os.WriteFile(sourceFile, []byte("version 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := OrphanWriteFile(branch, "data/file.txt", sourceFile, "v2"); err != nil {
		t.Fatal(err)
	}

	// Should read the latest version
	content, err := OrphanReadFile(branch, "data/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if content != "version 2" {
		t.Errorf("expected 'version 2', got %q", content)
	}
}
