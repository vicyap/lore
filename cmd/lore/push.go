package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
)

func pushCmd() *cobra.Command {
	var remote string

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push lore notes and transcripts to the remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(remote)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "origin", "git remote to push to")
	return cmd
}

func runPush(remote string) error {
	cfg := config.Load()

	fmt.Printf("Pushing lore data to %s...\n", remote)

	// Push notes
	notesCmd := exec.Command("git", "push", remote, "refs/notes/"+cfg.NotesRef)
	if out, err := notesCmd.CombinedOutput(); err != nil {
		fmt.Printf("  Notes: failed (%s)\n", trimOutput(out))
	} else {
		fmt.Println("  Notes: pushed")
	}

	// Push transcripts
	transcriptsCmd := exec.Command("git", "push", remote, cfg.Branch)
	if out, err := transcriptsCmd.CombinedOutput(); err != nil {
		fmt.Printf("  Transcripts: failed (%s)\n", trimOutput(out))
	} else {
		fmt.Println("  Transcripts: pushed")
	}

	return nil
}

func trimOutput(out []byte) string {
	s := string(out)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
