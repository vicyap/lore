package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
)

func mergeNotesCmd() *cobra.Command {
	var prNumber int

	cmd := &cobra.Command{
		Use:   "merge-notes [--pr N]",
		Short: "Aggregate branch notes onto a squash-merged commit",
		Long: `After a squash merge, the original branch commits' lore notes are
orphaned because the squash commit has a new hash. This command finds
the notes from the original PR commits and writes a combined note on
the current HEAD.

Used automatically by the lore GitHub Actions workflow on PR merge.
Can also be run manually after a local squash merge.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMergeNotes(prNumber)
		},
	}

	cmd.Flags().IntVar(&prNumber, "pr", 0, "PR number to aggregate notes from (auto-detects if omitted)")
	return cmd
}

func runMergeNotes(prNumber int) error {
	cfg := config.Load()

	if !git.IsInsideWorkTree() {
		return fmt.Errorf("not inside a git repository")
	}

	headHash, err := git.GetCommitHash()
	if err != nil {
		return fmt.Errorf("get HEAD: %w", err)
	}

	// Already has a note — skip
	if git.HasNote(cfg.NotesRef, headHash) {
		fmt.Println("HEAD already has a lore note, skipping.")
		return nil
	}

	// Find the PR
	if prNumber == 0 {
		detected, err := detectPRForCommit(headHash)
		if err != nil {
			fmt.Printf("No PR found for HEAD commit (gh: %v). Nothing to merge.\n", err)
			return nil
		}
		if detected == 0 {
			fmt.Println("No PR found for HEAD commit. Nothing to merge.")
			return nil
		}
		prNumber = detected
	}

	fmt.Printf("Aggregating notes from PR #%d onto %s...\n", prNumber, headHash[:12])

	// Get PR commit hashes
	commits, err := getPRCommits(prNumber)
	if err != nil {
		return fmt.Errorf("get PR commits: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println("No commits found in PR.")
		return nil
	}

	// Collect notes from those commits
	var notes []aggregatedNote
	for _, hash := range commits {
		note, err := git.ReadNote(cfg.NotesRef, hash)
		if err != nil || note == "" {
			continue
		}
		subject, _ := git.GetCommitSubject(hash)
		notes = append(notes, aggregatedNote{
			commitHash: hash,
			subject:    subject,
			note:       note,
		})
	}

	if len(notes) == 0 {
		fmt.Println("No lore notes found on PR commits.")
		return nil
	}

	// Build aggregated note
	combined := buildAggregatedNote(notes, prNumber)

	// Write to HEAD
	if err := git.WriteNote(cfg.NotesRef, headHash, combined); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	fmt.Printf("Wrote aggregated note from %d commit(s) onto %s\n", len(notes), headHash[:12])
	return nil
}

type aggregatedNote struct {
	commitHash string
	subject    string
	note       string
}

func buildAggregatedNote(notes []aggregatedNote, prNumber int) string {
	if len(notes) == 1 {
		// Single note — use it directly, just add provenance
		return fmt.Sprintf("%s\n\n## Provenance\nSquash merge of PR #%d (1 commit: %s)",
			strings.TrimSpace(notes[0].note), prNumber, notes[0].commitHash[:12])
	}

	var buf strings.Builder

	// Extract and deduplicate sections across all notes
	allIntents := collectSection(notes, "## Intent")
	allConstraints := collectSection(notes, "## Constraints")
	allRejected := collectSection(notes, "## Rejected Alternatives")
	allDirectives := collectSection(notes, "## Directives")
	allSessions := collectSection(notes, "## Session")

	if len(allIntents) > 0 {
		buf.WriteString("## Intent\n")
		buf.WriteString(strings.Join(allIntents, "\n"))
		buf.WriteString("\n\n")
	}

	if len(allConstraints) > 0 {
		buf.WriteString("## Constraints\n")
		buf.WriteString(strings.Join(dedup(allConstraints), "\n"))
		buf.WriteString("\n\n")
	}

	if len(allRejected) > 0 {
		buf.WriteString("## Rejected Alternatives\n")
		buf.WriteString(strings.Join(dedup(allRejected), "\n"))
		buf.WriteString("\n\n")
	}

	if len(allDirectives) > 0 {
		buf.WriteString("## Directives\n")
		buf.WriteString(strings.Join(dedup(allDirectives), "\n"))
		buf.WriteString("\n\n")
	}

	buf.WriteString("## Confidence\nmedium (aggregated from multiple commits)\n\n")

	if len(allSessions) > 0 {
		buf.WriteString("## Session\n")
		buf.WriteString(strings.Join(dedup(allSessions), "\n"))
		buf.WriteString("\n\n")
	}

	// Provenance
	buf.WriteString(fmt.Sprintf("## Provenance\nSquash merge of PR #%d (%d commits):\n", prNumber, len(notes)))
	for _, note := range notes {
		buf.WriteString(fmt.Sprintf("- %s %s\n", note.commitHash[:12], note.subject))
	}

	return strings.TrimSpace(buf.String())
}

func collectSection(notes []aggregatedNote, header string) []string {
	var lines []string
	for _, note := range notes {
		content := extractSection(note.note, header)
		if content != "" {
			lines = append(lines, content)
		}
	}
	return lines
}

func extractSection(note, header string) string {
	idx := strings.Index(note, header)
	if idx == -1 {
		return ""
	}
	content := note[idx+len(header):]
	content = strings.TrimLeft(content, "\n")

	// Find the next ## header
	nextHeader := strings.Index(content, "\n## ")
	if nextHeader != -1 {
		content = content[:nextHeader]
	}
	return strings.TrimSpace(content)
}

func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func detectPRForCommit(commitHash string) (int, error) {
	// Use gh to find the PR that created this commit
	out, err := exec.Command("gh", "pr", "list",
		"--search", commitHash,
		"--state", "merged",
		"--json", "number",
		"--limit", "1",
	).Output()
	if err != nil {
		return 0, err
	}

	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		return 0, err
	}
	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].Number, nil
}

func getPRCommits(prNumber int) ([]string, error) {
	out, err := exec.Command("gh", "pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--json", "commits",
	).Output()
	if err != nil {
		return nil, err
	}

	var pr struct {
		Commits []struct {
			OID string `json:"oid"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, err
	}

	var hashes []string
	for _, commit := range pr.Commits {
		hashes = append(hashes, commit.OID)
	}
	return hashes, nil
}
