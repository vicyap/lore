package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/vicyap/lore/internal/git"
)

// NoteEntry represents a single exported note.
type NoteEntry struct {
	Commit  string `json:"commit"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Note    string `json:"note"`
}

// AsJSON writes all notes as JSONL to the writer.
func AsJSON(notesRef string, writer io.Writer) error {
	entries, err := collectNotes(notesRef)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(writer)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("encode entry: %w", err)
		}
	}
	return nil
}

// AsMarkdown writes all notes as a Markdown document to the writer.
func AsMarkdown(notesRef string, writer io.Writer) error {
	entries, err := collectNotes(notesRef)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(writer, "No lore notes found.")
		return nil
	}

	fmt.Fprintln(writer, "# Lore Decision Notes")
	fmt.Fprintln(writer)

	for _, entry := range entries {
		fmt.Fprintf(writer, "### %s -- %s\n\n", entry.Commit, entry.Subject)
		fmt.Fprintf(writer, "*%s, %s*\n\n", entry.Author, entry.Date)
		fmt.Fprintln(writer, entry.Note)
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "---")
		fmt.Fprintln(writer)
	}
	return nil
}

func collectNotes(notesRef string) ([]NoteEntry, error) {
	pairs, err := git.ListNotes(notesRef)
	if err != nil {
		return nil, err
	}

	var entries []NoteEntry
	for _, pair := range pairs {
		commitHash := pair[1]

		note, err := git.ReadNote(notesRef, commitHash)
		if err != nil || note == "" {
			continue
		}

		subject := getCommitField(commitHash, "%s")
		author := getCommitField(commitHash, "%an")
		date := getCommitField(commitHash, "%ai")

		entries = append(entries, NoteEntry{
			Commit:  commitHash[:min(12, len(commitHash))],
			Subject: subject,
			Author:  author,
			Date:    date,
			Note:    note,
		})
	}
	return entries, nil
}

func getCommitField(commitHash, format string) string {
	cmd := exec.Command("git", "log", "-1", "--format="+format, commitHash)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
