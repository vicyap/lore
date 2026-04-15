package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// OrphanExists returns true if the orphan branch exists.
func OrphanExists(branch string) bool {
	_, err := runGit("rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// OrphanInit creates the orphan branch with an empty initial commit.
func OrphanInit(branch string) error {
	if OrphanExists(branch) {
		return nil
	}

	emptyTree, err := runGit("hash-object", "-t", "tree", "/dev/null")
	if err != nil {
		return fmt.Errorf("hash empty tree: %w", err)
	}

	initCommit, err := runGitWithEnv(
		[]string{
			"GIT_AUTHOR_NAME=lore",
			"GIT_AUTHOR_EMAIL=lore@local",
			"GIT_COMMITTER_NAME=lore",
			"GIT_COMMITTER_EMAIL=lore@local",
		},
		"commit-tree", emptyTree, "-m", "Initialize lore transcripts",
	)
	if err != nil {
		return fmt.Errorf("create init commit: %w", err)
	}

	_, err = runGit("update-ref", "refs/heads/"+branch, initCommit)
	return err
}

// OrphanWriteFile writes a file to the orphan branch using git plumbing.
// No checkout or working tree disruption. Returns the new commit hash.
func OrphanWriteFile(branch, filepath, sourceFile, commitMessage string) (string, error) {
	if err := OrphanInit(branch); err != nil {
		return "", err
	}

	blobHash, err := runGit("hash-object", "-w", sourceFile)
	if err != nil {
		return "", fmt.Errorf("hash blob: %w", err)
	}

	parentCommit, err := runGit("rev-parse", "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("get parent: %w", err)
	}

	currentTree, err := runGit("rev-parse", parentCommit+"^{tree}")
	if err != nil {
		return "", fmt.Errorf("get tree: %w", err)
	}

	tempIndex, err := os.CreateTemp("", "lore-index-*")
	if err != nil {
		return "", fmt.Errorf("create temp index: %w", err)
	}
	tempIndexPath := tempIndex.Name()
	tempIndex.Close()
	os.Remove(tempIndexPath) // git needs it to not exist initially
	defer os.Remove(tempIndexPath)

	// Read existing tree into temp index
	if err := runGitWithIndex(tempIndexPath, "read-tree", currentTree); err != nil {
		return "", fmt.Errorf("read-tree: %w", err)
	}

	// Add/update file in temp index
	cacheInfo := fmt.Sprintf("100644,%s,%s", blobHash, filepath)
	if err := runGitWithIndex(tempIndexPath, "update-index", "--add", "--cacheinfo", cacheInfo); err != nil {
		return "", fmt.Errorf("update-index: %w", err)
	}

	// Write the new tree
	newTree, err := runGitWithIndexOutput(tempIndexPath, "write-tree")
	if err != nil {
		return "", fmt.Errorf("write-tree: %w", err)
	}

	// Create commit
	newCommit, err := runGitWithEnv(
		[]string{
			"GIT_AUTHOR_NAME=lore",
			"GIT_AUTHOR_EMAIL=lore@local",
			"GIT_COMMITTER_NAME=lore",
			"GIT_COMMITTER_EMAIL=lore@local",
		},
		"commit-tree", newTree, "-p", parentCommit, "-m", commitMessage,
	)
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}

	_, err = runGit("update-ref", "refs/heads/"+branch, newCommit)
	return newCommit, err
}

// OrphanReadFile reads a file from the orphan branch.
func OrphanReadFile(branch, filepath string) (string, error) {
	return runGit("show", "refs/heads/"+branch+":"+filepath)
}

// OrphanListFiles lists files on the orphan branch.
func OrphanListFiles(branch string) ([]string, error) {
	output, err := runGit("ls-tree", "-r", "--name-only", "refs/heads/"+branch)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}

// runGitWithEnv runs a git command with extra environment variables.
func runGitWithEnv(env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitWithIndex runs a git command with a custom GIT_INDEX_FILE.
func runGitWithIndex(indexPath string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), string(out), err)
	}
	return nil
}

// runGitWithIndexOutput runs a git command with a custom GIT_INDEX_FILE and returns stdout.
func runGitWithIndexOutput(indexPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexPath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
