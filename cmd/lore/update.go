package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update lore to the latest version",
		RunE:  runUpdate,
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}
	installDir := filepath.Dir(self)

	fmt.Printf("Updating lore (current: %s)...\n", version)

	sh := exec.Command("sh", "-c",
		fmt.Sprintf("curl -fsSL https://raw.githubusercontent.com/vicyap/lore/main/install.sh | LORE_INSTALL=%s sh", installDir))
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr
	if err := sh.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}
