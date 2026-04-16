package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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

	// Respect an explicit LORE_VERSION override; otherwise resolve it
	// ourselves via the /releases/latest redirect so we never touch
	// api.github.com (which has a 60/hour unauthenticated quota).
	target := os.Getenv("LORE_VERSION")
	if target == "" {
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()
		target, err = latestRelease(ctx, latestReleaseURL)
		if err != nil {
			return fmt.Errorf("determine latest version: %w", err)
		}
	}

	if version != "dev" && target == version {
		fmt.Printf("lore %s is already the latest release.\n", version)
		return nil
	}

	fmt.Printf("Updating lore %s → %s...\n", version, target)

	sh := exec.Command("sh", "-c",
		"curl -fsSL https://raw.githubusercontent.com/vicyap/lore/main/install.sh | sh")
	sh.Env = append(os.Environ(),
		"LORE_INSTALL="+installDir,
		"LORE_VERSION="+target,
	)
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr
	if err := sh.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}
