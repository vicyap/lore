package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
	"github.com/vicyap/lore/internal/settings"
)

func disableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Remove lore hooks from the current repository",
		Long: `Disable lore in the current repository by removing the PostToolUse hook.
Existing notes and transcripts are preserved.`,
		RunE: runDisable,
	}
}

func runDisable(cmd *cobra.Command, args []string) error {
	if !git.IsInsideWorkTree() {
		return fmt.Errorf("not inside a git repository")
	}

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	cfg := config.Load()

	fmt.Printf("Disabling lore in %s...\n", repoRoot)

	// Remove hook
	removed, err := settings.RemoveHook(repoRoot)
	if err != nil {
		return fmt.Errorf("remove hook: %w", err)
	}
	if removed {
		fmt.Println("  Hook removed from .claude/settings.json")
	} else {
		fmt.Println("  No lore hook found in .claude/settings.json")
	}

	// Remove notes display config
	removeNotesDisplay(cfg.NotesRef)

	fmt.Println()
	fmt.Println("lore disabled. Existing notes and transcripts are preserved.")
	fmt.Println()
	fmt.Println("To delete all lore data:")
	fmt.Printf("  git notes --ref=%s prune\n", cfg.NotesRef)
	fmt.Printf("  git branch -D %s\n", cfg.Branch)
	return nil
}

func removeNotesDisplay(notesRef string) {
	ref := "refs/notes/" + notesRef
	cmd := exec.Command("git", "config", "--unset", "notes.displayRef", ref)
	if err := cmd.Run(); err != nil {
		// May not be set — that's fine
		return
	}
	fmt.Println("  Git notes display config removed")
}
