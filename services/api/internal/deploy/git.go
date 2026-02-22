package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CloneRepo performs a shallow git clone of the given repo URL and branch
// into the specified target directory.
func CloneRepo(ctx context.Context, repoURL, branch, targetDir string) error {
	if repoURL == "" {
		return fmt.Errorf("repo URL is required")
	}
	if branch == "" {
		branch = "main"
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create clone directory: %w", err)
	}

	// Shallow clone with depth 1 (faster, less disk)
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1",
		"--branch", branch,
		"--single-branch",
		repoURL,
		targetDir,
	)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // Never prompt for credentials
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s — %w", string(output), err)
	}

	return nil
}

// GetLatestCommitSHA returns the HEAD commit SHA of the cloned repo.
func GetLatestCommitSHA(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	sha := string(output)
	// Trim trailing newline
	if len(sha) > 0 && sha[len(sha)-1] == '\n' {
		sha = sha[:len(sha)-1]
	}

	return sha, nil
}

// CloneAndDetect clones the repo and detects the framework.
// Returns the clone directory path, detected framework, and commit SHA.
func CloneAndDetect(ctx context.Context, repoURL, branch, baseDir string) (cloneDir string, sha string, err error) {
	// Create a unique clone directory
	cloneDir = filepath.Join(baseDir, "clone")
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create base dir: %w", err)
	}

	if err := CloneRepo(ctx, repoURL, branch, cloneDir); err != nil {
		return "", "", err
	}

	sha, err = GetLatestCommitSHA(ctx, cloneDir)
	if err != nil {
		return cloneDir, "", err
	}

	return cloneDir, sha, nil
}

// CleanupClone removes the cloned repository directory.
func CleanupClone(cloneDir string) error {
	return os.RemoveAll(cloneDir)
}
