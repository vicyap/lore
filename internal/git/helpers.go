package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetCommitHash returns the current HEAD commit hash.
func GetCommitHash() (string, error) {
	return runGit("rev-parse", "HEAD")
}

// GetDiff returns the diff for a given commit.
func GetDiff(commitHash string) (string, error) {
	// Check if parent exists
	_, err := runGit("rev-parse", "--verify", commitHash+"^")
	if err != nil {
		// First commit — show the whole thing
		return runGit("show", commitHash, "--format=", "--diff-filter=ACMR")
	}
	return runGit("diff", commitHash+"^.."+commitHash)
}

// GetCommitSubject returns the subject line of a commit.
func GetCommitSubject(commitHash string) (string, error) {
	return runGit("log", "-1", "--format=%s", commitHash)
}

// GetBranchName returns the current branch name, or "detached" if HEAD is detached.
func GetBranchName() string {
	name, err := runGit("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "detached"
	}
	return name
}

// IsInsideWorkTree returns true if the current directory is inside a git work tree.
func IsInsideWorkTree() bool {
	_, err := runGit("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// GetRepoRoot returns the root directory of the current git repository.
func GetRepoRoot() (string, error) {
	return runGit("rev-parse", "--show-toplevel")
}

// GetCommitsWithNotes returns the last N commits that have lore notes.
func GetCommitsWithNotes(notesRef string, count int) (string, error) {
	return runGit("log", fmt.Sprintf("-%d", count), fmt.Sprintf("--notes=%s", notesRef), "--format=## %h %s%n%N%n---")
}

// runGit executes a git command and returns trimmed stdout.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
