package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "lore",
		Short: "Capture why code was written, not just what changed",
		Long: `lore captures structured decision reasoning from AI agent sessions
and stores it as git notes alongside commits. Full session transcripts
are preserved on a separate branch for deep investigation.`,
		Version: version,
	}

	rootCmd.AddCommand(
		initCmd(),
		hookCmd(),
		showCmd(),
		statusCmd(),
		pushCmd(),
		exportCmd(),
		disableCmd(),
		browseCmd(),
		updateCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
