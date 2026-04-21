package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
	"github.com/vicyap/lore/internal/settings"
	"github.com/vicyap/lore/prompts"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap lore in the current repository",
		Long: `Initialize lore in the current git repository. This:
  1. Adds a PostToolUse hook to .claude/settings.json
  2. Creates the lore/transcripts orphan branch
  3. Configures git to display lore notes in git log`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	if !git.IsInsideWorkTree() {
		return fmt.Errorf("not inside a git repository")
	}

	// Verify claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: claude CLI not found on PATH (needed for distillation)")
	}

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("get repo root: %w", err)
	}

	fmt.Printf("Enabling lore in %s...\n", filepath.Base(repoRoot))

	// Check for old hook and offer to replace
	hasOld, _ := settings.HasOldHook(repoRoot)
	if hasOld {
		fmt.Println("  Detected old bash-based lore hook")
		if err := settings.ReplaceOldHook(repoRoot); err != nil {
			return fmt.Errorf("replace old hook: %w", err)
		}
		fmt.Println("  Replaced old hook with lore CLI hook")
	} else {
		installed, err := settings.InstallHook(repoRoot)
		if err != nil {
			return fmt.Errorf("install hook: %w", err)
		}
		if installed {
			fmt.Println("  Hook added to .claude/settings.json")
		} else {
			fmt.Println("  Hook already installed in .claude/settings.json")
		}
	}

	// Initialize orphan branch
	cfg := config.Load()
	if git.OrphanExists(cfg.Branch) {
		fmt.Printf("  Orphan branch %s already exists\n", cfg.Branch)
	} else {
		if err := git.OrphanInit(cfg.Branch); err != nil {
			return fmt.Errorf("init orphan branch: %w", err)
		}
		fmt.Printf("  Created orphan branch %s\n", cfg.Branch)
	}

	// Configure git notes display
	configureNotesDisplay(cfg.NotesRef)

	// Configure remote.origin.fetch so future clones/fetches pull notes down
	configureNotesFetchRefspec()

	// Ask about GitHub Actions workflow — skip the prompt if already installed
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "lore.yml")
	if _, err := os.Stat(workflowPath); err == nil {
		fmt.Println("  GitHub Actions workflow already installed at .github/workflows/lore.yml")
	} else if promptYesNo("Install GitHub Actions workflow? (pushes notes on merge, handles squash merges)", true) {
		if err := installWorkflow(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: workflow install failed: %v\n", err)
		} else {
			fmt.Println("  Workflow installed at .github/workflows/lore.yml")
		}
	}

	fmt.Println()
	fmt.Println("lore enabled. Decision notes will be captured on every commit.")
	fmt.Println()
	fmt.Println("View notes:       git log --notes=lore")
	fmt.Println("View transcripts: git log lore/transcripts")
	fmt.Println("Interactive:      lore browse")
	return nil
}

func configureNotesDisplay(notesRef string) {
	existing, _ := runGitConfig("--get-all", "notes.displayRef")
	if strings.Contains(existing, "refs/notes/"+notesRef) {
		fmt.Println("  Git notes display already configured")
		return
	}
	cmd := exec.Command("git", "config", "--add", "notes.displayRef", "refs/notes/"+notesRef)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: could not configure notes display: %v\n", err)
		return
	}
	fmt.Println("  Configured git to display lore notes")
}

func installWorkflow(repoRoot string) error {
	workflowDir := filepath.Join(repoRoot, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return err
	}
	workflowPath := filepath.Join(workflowDir, "lore.yml")
	return os.WriteFile(workflowPath, prompts.WorkflowTemplate(), 0o644)
}

// configureNotesFetchRefspec adds refs/notes/* to remote.origin.fetch so that
// `git fetch` / `git pull` pulls lore notes down on fresh clones. Idempotent.
func configureNotesFetchRefspec() {
	// Skip if no origin remote exists (freshly `git init`'d repo, no upstream).
	if _, err := runGitConfig("--get", "remote.origin.url"); err != nil {
		fmt.Println("  No origin remote — skipping fetch refspec config")
		return
	}

	const refspec = "+refs/notes/*:refs/notes/*"
	existing, _ := runGitConfig("--get-all", "remote.origin.fetch")
	if strings.Contains(existing, refspec) {
		fmt.Println("  Git notes fetch refspec already configured")
		return
	}

	cmd := exec.Command("git", "config", "--add", "remote.origin.fetch", refspec)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: could not configure remote.origin.fetch: %v\n", err)
		return
	}
	fmt.Println("  Configured remote.origin.fetch for refs/notes/*")
}

func runGitConfig(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"config"}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func promptYesNo(question string, defaultYes bool) bool {
	suffix := " [Y/n] "
	if !defaultYes {
		suffix = " [y/N] "
	}

	fmt.Print(question + suffix)

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" {
		return defaultYes
	}
	return answer == "y" || answer == "yes"
}
