package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
	"github.com/vicyap/lore/internal/settings"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show lore status in the current repository",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	if !git.IsInsideWorkTree() {
		return fmt.Errorf("not inside a git repository")
	}

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	hookInstalled, _ := settings.HasHook(repoRoot)
	if hookInstalled {
		fmt.Println("lore is enabled in this repository")
	} else {
		fmt.Println("lore is NOT enabled in this repository")
	}
	fmt.Println()

	printStatus("PostToolUse hook", hookInstalled)

	// Old hook?
	hasOld, _ := settings.HasOldHook(repoRoot)
	if hasOld {
		fmt.Println("  (old bash hook detected — run 'lore init' to upgrade)")
	}

	// Orphan branch?
	branchExists := git.OrphanExists(cfg.Branch)
	printStatus("Orphan branch ("+cfg.Branch+")", branchExists)

	// Count notes
	noteCount := git.CountNotes(cfg.NotesRef)
	fmt.Printf("  Notes:                %d\n", noteCount)

	// Count transcript files
	if branchExists {
		files, err := git.OrphanListFiles(cfg.Branch)
		if err == nil {
			fmt.Printf("  Transcript files:     %d\n", len(files))
		}
	}

	return nil
}

func printStatus(label string, ok bool) {
	status := "no"
	if ok {
		status = "yes"
	}
	fmt.Printf("  %-24s%s\n", label+":", status)
}
