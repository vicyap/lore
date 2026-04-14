package git

import (
	"os"
	"testing"
)

func TestWriteAndReadNote(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatalf("GetCommitHash: %v", err)
	}

	notesRef := "lore-test"
	content := "## Intent\nFix login bug\n\n## Confidence\nhigh"

	if err := WriteNote(notesRef, commitHash, content); err != nil {
		t.Fatalf("WriteNote: %v", err)
	}

	got, err := ReadNote(notesRef, commitHash)
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if got != content {
		t.Errorf("ReadNote:\ngot:  %q\nwant: %q", got, content)
	}
}

func TestHasNote(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatal(err)
	}

	notesRef := "lore-test"

	if HasNote(notesRef, commitHash) {
		t.Fatal("should not have note before writing")
	}

	if err := WriteNote(notesRef, commitHash, "test note"); err != nil {
		t.Fatal(err)
	}

	if !HasNote(notesRef, commitHash) {
		t.Fatal("should have note after writing")
	}
}

func TestReadNote_NoNote(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatal(err)
	}

	note, err := ReadNote("lore-test", commitHash)
	if err != nil {
		t.Fatalf("ReadNote should not error for missing note: %v", err)
	}
	if note != "" {
		t.Errorf("expected empty string for missing note, got %q", note)
	}
}

func TestListNotes(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatal(err)
	}

	notesRef := "lore-test"

	// No notes initially
	pairs, err := ListNotes(notesRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(pairs))
	}

	// Add a note
	if err := WriteNote(notesRef, commitHash, "test"); err != nil {
		t.Fatal(err)
	}

	pairs, err = ListNotes(notesRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 note, got %d", len(pairs))
	}
	if pairs[0][1] != commitHash {
		t.Errorf("expected commit hash %q, got %q", commitHash, pairs[0][1])
	}
}

func TestCountNotes(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatal(err)
	}

	notesRef := "lore-test"

	if count := CountNotes(notesRef); count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	WriteNote(notesRef, commitHash, "test")

	if count := CountNotes(notesRef); count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestWriteNote_Overwrite(t *testing.T) {
	dir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	commitHash, err := GetCommitHash()
	if err != nil {
		t.Fatal(err)
	}

	notesRef := "lore-test"

	WriteNote(notesRef, commitHash, "version 1")
	WriteNote(notesRef, commitHash, "version 2")

	got, _ := ReadNote(notesRef, commitHash)
	if got != "version 2" {
		t.Errorf("expected 'version 2', got %q", got)
	}
}
