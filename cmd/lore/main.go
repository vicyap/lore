package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:   "lore",
		Short: "Capture why code was written, not just what changed",
		Long: `lore captures structured decision reasoning from AI agent sessions
and stores it as git notes alongside commits. Full session transcripts
are preserved on a separate branch for deep investigation.`,
		Version: version,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		initCmd(),
		disableCmd(),
		browseCmd(),
		statusCmd(),
		exportCmd(),
		squashCmd(),
		updateCmd(),
		versionCmd(),
		hookCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
