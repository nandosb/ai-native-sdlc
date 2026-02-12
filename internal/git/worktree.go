package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const worktreeBase = ".sdlc/worktrees"

// WorktreeManager manages git worktrees for isolated development.
type WorktreeManager struct {
	rootDir string
}

// NewWorktreeManager creates a manager rooted at the given directory.
func NewWorktreeManager(rootDir string) *WorktreeManager {
	return &WorktreeManager{rootDir: rootDir}
}

// Create creates a new worktree for the given repo and issue.
// Returns the absolute path to the worktree.
func (wm *WorktreeManager) Create(repoPath, issueSlug string) (string, error) {
	repoName := filepath.Base(repoPath)
	wtPath := filepath.Join(wm.rootDir, worktreeBase, repoName, issueSlug)

	// Check if worktree already exists and is valid
	if _, err := os.Stat(wtPath); err == nil {
		// Verify it's a real git worktree (has .git file)
		if _, err := os.Stat(filepath.Join(wtPath, ".git")); err == nil {
			absWT, _ := filepath.Abs(wtPath)
			return absWT, nil
		}
		// Directory exists but isn't a valid worktree — remove orphaned dir
		os.RemoveAll(wtPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(wtPath), 0755); err != nil {
		return "", fmt.Errorf("create worktree dir: %w", err)
	}

	branch := "feat/" + issueSlug

	// Create worktree with new branch from origin/main
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	// Prune stale worktree entries (cleans up orphaned metadata)
	prune := exec.Command("git", "-C", absRepo, "worktree", "prune")
	prune.Run()

	// Fetch latest
	fetch := exec.Command("git", "-C", absRepo, "fetch", "origin", "main")
	fetch.Stderr = os.Stderr
	fetch.Run() // Ignore error, may not have remote

	// Create worktree
	cmd := exec.Command("git", "-C", absRepo, "worktree", "add", wtPath, "-b", branch, "origin/main")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Branch might already exist, try without -b
		cmd2 := exec.Command("git", "-C", absRepo, "worktree", "add", wtPath, branch)
		if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("create worktree: %s\n%s", string(output), string(output2))
		}
	} else {
		_ = output
	}

	absWT, _ := filepath.Abs(wtPath)
	return absWT, nil
}

// Remove removes a worktree and optionally deletes its branch.
func (wm *WorktreeManager) Remove(repoPath, issueSlug string, deleteBranch bool) error {
	repoName := filepath.Base(repoPath)
	wtPath := filepath.Join(wm.rootDir, worktreeBase, repoName, issueSlug)

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	// Remove worktree
	cmd := exec.Command("git", "-C", absRepo, "worktree", "remove", wtPath, "--force")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove worktree: %s", string(output))
	}

	// Optionally delete branch
	if deleteBranch {
		branch := "feat/" + issueSlug
		cmd := exec.Command("git", "-C", absRepo, "branch", "-d", branch)
		cmd.CombinedOutput() // Ignore error
	}

	return nil
}

// List returns all active worktrees for a repo.
func (wm *WorktreeManager) List(repoPath string) ([]string, error) {
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}

	cmd := exec.Command("git", "-C", absRepo, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	var worktrees []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if strings.Contains(path, worktreeBase) {
				worktrees = append(worktrees, path)
			}
		}
	}

	return worktrees, nil
}

// WorktreePath returns the expected path for a worktree.
func (wm *WorktreeManager) WorktreePath(repoPath, issueSlug string) string {
	repoName := filepath.Base(repoPath)
	return filepath.Join(wm.rootDir, worktreeBase, repoName, issueSlug)
}
