package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/distill"
	"github.com/vicyap/lore/internal/git"
)

func hookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook",
		Short:  "PostToolUse hook entry point (called by Claude Code)",
		Hidden: true,
		RunE:   runHook,
	}
}

// hookInput matches the JSON structure from Claude Code's PostToolUse hook.
type hookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	ToolInput      struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func runHook(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Read JSON from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		logError("failed to read stdin: %v", err)
		return nil // non-fatal
	}

	var hook hookInput
	if err := json.Unmarshal(input, &hook); err != nil {
		logError("failed to parse hook input: %v", err)
		return nil
	}

	// Defense in depth — verify this is a git commit
	if !strings.Contains(hook.ToolInput.Command, "git commit") {
		return nil
	}

	if hook.SessionID == "" || hook.TranscriptPath == "" || hook.CWD == "" {
		logError("missing required fields in hook input")
		return nil
	}

	if err := os.Chdir(hook.CWD); err != nil {
		logError("chdir to %s: %v", hook.CWD, err)
		return nil
	}

	commitHash, err := git.GetCommitHash()
	if err != nil {
		logError("get commit hash: %v", err)
		return nil
	}

	logInfo("Processing commit %s (session %s)", commitHash[:12], hook.SessionID[:min(8, len(hook.SessionID))])

	// Step 1: Capture transcript to orphan branch
	transcriptCommit, err := captureTranscript(cfg, hook.TranscriptPath, hook.SessionID, commitHash)
	if err != nil {
		logError("transcript capture failed (non-fatal): %v", err)
	}

	// Step 2: Distill reasoning into git note
	if err := distill.Run(cfg, hook.TranscriptPath, commitHash, transcriptCommit, version); err != nil {
		logError("distillation failed (non-fatal): %v", err)
	}

	logInfo("Done: commit %s", commitHash[:12])
	return nil
}

func captureTranscript(cfg config.Config, transcriptPath, sessionID, commitHash string) (string, error) {
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("transcript not found: %s", transcriptPath)
	}

	filepath := fmt.Sprintf("transcripts/%s.jsonl", sessionID)
	message := fmt.Sprintf("transcript for %s (session %s)", commitHash[:12], sessionID[:min(8, len(sessionID))])

	return git.OrphanWriteFile(cfg.Branch, filepath, transcriptPath, message)
}

func logInfo(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[lore:info] "+format+"\n", args...)
}

func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[lore:error] "+format+"\n", args...)
}
