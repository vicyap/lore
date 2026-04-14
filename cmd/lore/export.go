package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/config"
	loreExport "github.com/vicyap/lore/internal/export"
)

func exportCmd() *cobra.Command {
	var format string
	var output string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all lore notes",
		Long: `Export all lore decision notes in the repository.

  lore export                    Export as JSONL to stdout
  lore export --format md        Export as Markdown
  lore export --output notes.md  Write to file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(format, output)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "output format: json or md")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file (default: stdout)")
	return cmd
}

func runExport(format, outputPath string) error {
	cfg := config.Load()

	var writer *os.File
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}

	switch format {
	case "json":
		return loreExport.AsJSON(cfg.NotesRef, writer)
	case "md":
		return loreExport.AsMarkdown(cfg.NotesRef, writer)
	default:
		return fmt.Errorf("unknown format %q (use 'json' or 'md')", format)
	}
}
