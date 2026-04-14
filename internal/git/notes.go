package git

import (
	"fmt"
	"os"
	"strings"
)

// WriteNote writes content as a git note on the given commit.
func WriteNote(notesRef, commitHash, content string) error {
	tmpfile, err := os.CreateTemp("", "lore-note-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpfile.Close()

	_, err = runGit("notes", "--ref="+notesRef, "add", "-f", "--file="+tmpfile.Name(), commitHash)
	return err
}

// ReadNote reads the git note for a commit. Returns empty string if no note exists.
func ReadNote(notesRef, commitHash string) (string, error) {
	note, err := runGit("notes", "--ref="+notesRef, "show", commitHash)
	if err != nil {
		return "", nil // no note exists
	}
	return note, nil
}

// HasNote returns true if a note exists for the given commit.
func HasNote(notesRef, commitHash string) bool {
	_, err := runGit("notes", "--ref="+notesRef, "show", commitHash)
	return err == nil
}

// ListNotes returns all note/commit hash pairs for the given notes ref.
func ListNotes(notesRef string) ([][2]string, error) {
	output, err := runGit("notes", "--ref="+notesRef, "list")
	if err != nil {
		return nil, nil // no notes
	}
	if output == "" {
		return nil, nil
	}

	var pairs [][2]string
	for _, line := range strings.Split(output, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			pairs = append(pairs, [2]string{parts[0], parts[1]})
		}
	}
	return pairs, nil
}

// CountNotes returns the number of notes for the given ref.
func CountNotes(notesRef string) int {
	pairs, _ := ListNotes(notesRef)
	return len(pairs)
}
