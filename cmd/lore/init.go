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
  3. Configures git to display lore notes in git log
  4. Optionally installs the /lore skill for Claude Code`,
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

	// Ask about skill installation
	if promptYesNo("Install /lore skill for Claude Code?", true) {
		if err := installSkill(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: skill install failed: %v\n", err)
		} else {
			fmt.Println("  Skill installed")
		}
	}

	fmt.Println()
	fmt.Println("lore enabled. Decision notes will be captured on every commit.")
	fmt.Println()
	fmt.Println("View notes:       git log --notes=lore")
	fmt.Println("View transcripts: git log lore/transcripts")
	fmt.Println("Interactive:      lore show (or lore browse)")
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

func installSkill(repoRoot string) error {
	// Detect skills directory
	skillsDir := filepath.Join(repoRoot, ".claude", "skills")
	if info, err := os.Stat(filepath.Join(repoRoot, ".agents", "skills")); err == nil && info.IsDir() {
		skillsDir = filepath.Join(repoRoot, ".agents", "skills")
	}

	skillDir := filepath.Join(skillsDir, "lore")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}

	skillContent, err := prompts.SkillDefinition()
	if err != nil {
		return fmt.Errorf("read embedded skill: %w", err)
	}

	return os.WriteFile(filepath.Join(skillDir, "lore.md"), skillContent, 0o644)
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
