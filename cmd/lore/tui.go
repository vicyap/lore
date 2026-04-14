package main

import (
	"github.com/spf13/cobra"
	"github.com/vicyap/lore/internal/tui"
)

func browseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Interactive browser for decision notes",
		Long: `Launch an interactive terminal UI to browse lore decision notes.

Navigate commits, view full notes with markdown rendering, and search
across all captured decision reasoning.

  j/k or arrows: navigate
  enter: view note detail
  /: search
  q: quit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunWithFallback()
		},
	}
}
