package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreatePR creates a pull request from the current branch using gh CLI.
func CreatePR(worktreePath, title, body, baseBranch string) (string, error) {
	if baseBranch == "" {
		baseBranch = "main"
	}

	args := []string{"pr", "create",
		"--title", title,
		"--body", body,
		"--base", baseBranch,
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create PR: %s", strings.TrimSpace(string(output)))
	}

	prURL := strings.TrimSpace(string(output))
	return prURL, nil
}

// PushBranch pushes the current branch to origin.
func PushBranch(worktreePath string) error {
	cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("push branch: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GetCurrentBranch returns the current branch name.
func GetCurrentBranch(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// HasChanges checks if there are uncommitted changes.
func HasChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// CommitAll stages and commits all changes if there are any uncommitted files.
// This is a safety net in case the Claude session didn't commit its work.
func CommitAll(worktreePath, message string) error {
	hasChanges, err := HasChanges(worktreePath)
	if err != nil || !hasChanges {
		return err
	}

	add := exec.Command("git", "add", "-A")
	add.Dir = worktreePath
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s", strings.TrimSpace(string(out)))
	}

	commit := exec.Command("git", "commit", "-m", message)
	commit.Dir = worktreePath
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// HasCommitsAhead returns true if the current branch has commits ahead of origin/main.
func HasCommitsAhead(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "rev-list", "--count", "origin/main..HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("rev-list: %w", err)
	}
	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// PRExists checks if a PR already exists for the given branch.
func PRExists(worktreePath string) (string, bool) {
	cmd := exec.Command("gh", "pr", "view", "--json", "url", "--jq", ".url")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	url := strings.TrimSpace(string(output))
	return url, url != ""
}
