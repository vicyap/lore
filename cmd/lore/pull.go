package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
)

func pullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Fetch lore notes and transcripts from origin",
		Long: `Pull lore data from the origin remote. Fetches:
  - refs/notes/* (decision notes)
  - the lore/transcripts orphan branch

Useful on fresh clones, since git clone does not fetch notes by default.`,
		RunE: runPull,
	}
}

func runPull(cmd *cobra.Command, args []string) error {
	if !git.IsInsideWorkTree() {
		return fmt.Errorf("not inside a git repository")
	}

	if _, err := runGitConfig("--get", "remote.origin.url"); err != nil {
		return fmt.Errorf("no origin remote configured")
	}

	cfg := config.Load()

	fmt.Println("Pulling lore data from origin...")

	if err := gitFetch("refs/notes/*:refs/notes/*"); err != nil {
		return fmt.Errorf("fetch notes: %w", err)
	}
	fmt.Printf("  Fetched %d notes (refs/notes/%s)\n", git.CountNotes(cfg.NotesRef), cfg.NotesRef)

	branchRefspec := "refs/heads/" + cfg.Branch + ":refs/heads/" + cfg.Branch
	if err := gitFetch(branchRefspec); err != nil {
		// Best-effort: branch may not exist on origin yet.
		fmt.Fprintf(os.Stderr, "  warning: could not fetch %s: %v\n", cfg.Branch, err)
	} else if files, err := git.OrphanListFiles(cfg.Branch); err == nil {
		fmt.Printf("  Fetched %d transcripts (%s)\n", len(files), cfg.Branch)
	}

	return nil
}

func gitFetch(refspec string) error {
	cmd := exec.Command("git", "fetch", "origin", refspec)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}
