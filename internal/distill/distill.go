package distill

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
	"github.com/vicyap/lore/internal/transcript"
	"github.com/vicyap/lore/prompts"
)

// Run performs the full distillation pipeline:
// window transcript, get diff, call claude CLI, write git note.
func Run(cfg config.Config, transcriptPath, sessionID, commitHash string) error {
	// Get the diff
	diffContent, err := git.GetDiff(commitHash)
	if err != nil {
		return fmt.Errorf("get diff: %w", err)
	}

	// Truncate diff if too large
	if len(diffContent) > cfg.MaxDiffChars {
		diffContent = diffContent[:cfg.MaxDiffChars] +
			fmt.Sprintf("\n...(diff truncated at %d chars)...", cfg.MaxDiffChars)
	}

	// Extract transcript window
	entries, err := transcript.ParseJSONL(transcriptPath)
	if err != nil {
		return fmt.Errorf("parse transcript: %w", err)
	}
	transcriptWindow := transcript.ExtractWindow(entries, cfg.MaxTranscriptChars)

	// Get metadata
	branchName := git.GetBranchName()
	commitSubject, _ := git.GetCommitSubject(commitHash)

	// Build prompt input
	promptInput := fmt.Sprintf(`## Commit
%s %s
Branch: %s
Session: %s

## Diff
`+"```diff\n%s\n```"+`

## Transcript (agent session leading to this commit)
%s`,
		commitHash[:12], commitSubject, branchName, sessionID,
		diffContent, transcriptWindow,
	)

	// Write distill prompt to temp file
	promptFile, err := os.CreateTemp("", "lore-prompt-*.md")
	if err != nil {
		return fmt.Errorf("create prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.Write(prompts.DistillPrompt()); err != nil {
		promptFile.Close()
		return fmt.Errorf("write prompt file: %w", err)
	}
	promptFile.Close()

	// Call claude CLI
	distilled, err := runClaude(cfg.Model, promptFile.Name(), promptInput)
	if err != nil {
		// Write fallback note
		distilled = fmt.Sprintf(`## Intent
(distillation failed — claude CLI error)

## Confidence
low

## Session
%s | %s`, sessionID, branchName)
	}

	// Skip writing if output is empty (git notes rejects empty content)
	if strings.TrimSpace(distilled) == "" {
		return fmt.Errorf("distillation produced empty output")
	}

	// Write git note
	return git.WriteNote(cfg.NotesRef, commitHash, distilled)
}

// BuildPromptInput constructs the prompt input for distillation.
// Exported for testing.
func BuildPromptInput(commitHash, commitSubject, branchName, sessionID, diffContent, transcriptWindow string) string {
	return fmt.Sprintf(`## Commit
%s %s
Branch: %s
Session: %s

## Diff
`+"```diff\n%s\n```"+`

## Transcript (agent session leading to this commit)
%s`,
		commitHash, commitSubject, branchName, sessionID,
		diffContent, transcriptWindow,
	)
}

func runClaude(model, systemPromptFile, input string) (string, error) {
	cmd := exec.Command("claude", "-p",
		"--model", model,
		"--system-prompt-file", systemPromptFile,
		"Distill the decision reasoning for this commit.",
	)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude CLI: %w", err)
	}
	return string(out), nil
}
