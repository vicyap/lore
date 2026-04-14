package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
)

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [N | hash]",
		Short: "Show decision notes for recent commits",
		Long: `Show lore decision notes attached to commits.

  lore show        Show notes for the last 5 commits
  lore show 10     Show notes for the last 10 commits
  lore show abc123 Show the note for a specific commit`,
		Args: cobra.MaximumNArgs(1),
		RunE: runShow,
	}
}

func runShow(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	if len(args) == 0 {
		return showRecent(cfg, 5)
	}

	arg := args[0]

	// Try as a number first
	if count, err := strconv.Atoi(arg); err == nil && count > 0 {
		return showRecent(cfg, count)
	}

	// Otherwise treat as a commit hash
	return showCommit(cfg, arg)
}

func showRecent(cfg config.Config, count int) error {
	output, err := git.GetCommitsWithNotes(cfg.NotesRef, count)
	if err != nil {
		return fmt.Errorf("get commits: %w", err)
	}
	if output == "" {
		fmt.Println("No lore notes found in recent commits.")
		fmt.Println("Notes are created when you make commits during Claude Code sessions with lore enabled.")
		return nil
	}
	fmt.Println(output)
	return nil
}

func showCommit(cfg config.Config, commitHash string) error {
	note, err := git.ReadNote(cfg.NotesRef, commitHash)
	if err != nil {
		return fmt.Errorf("read note: %w", err)
	}
	if note == "" {
		fmt.Printf("No lore note found for commit %s\n", commitHash)
		return nil
	}
	fmt.Println(note)
	return nil
}
