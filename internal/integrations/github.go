package integrations

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubClient wraps the gh CLI for GitHub operations.
type GitHubClient struct{}

// NewGitHubClient creates a new GitHub client.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{}
}

// PRInfo contains information about a pull request.
type PRInfo struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
}

// CreatePR creates a pull request from the current branch.
func (gh *GitHubClient) CreatePR(cwd, title, body, base string) (*PRInfo, error) {
	if base == "" {
		base = "main"
	}

	cmd := exec.Command("gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--base", base,
		"--json", "url,number,state,title",
	)
	cmd.Dir = cwd

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh pr create: %s", strings.TrimSpace(string(output)))
	}

	var info PRInfo
	if err := json.Unmarshal(output, &info); err != nil {
		// gh pr create might not return JSON, just the URL
		info.URL = strings.TrimSpace(string(output))
	}

	return &info, nil
}

// ViewPR checks if a PR exists for the current branch.
func (gh *GitHubClient) ViewPR(cwd string) (*PRInfo, error) {
	cmd := exec.Command("gh", "pr", "view", "--json", "url,number,state,title")
	cmd.Dir = cwd

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("no PR found")
	}

	var info PRInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("parse PR info: %w", err)
	}

	return &info, nil
}

// MergePR merges the current PR.
func (gh *GitHubClient) MergePR(cwd string, squash bool) error {
	args := []string{"pr", "merge", "--delete-branch"}
	if squash {
		args = append(args, "--squash")
	} else {
		args = append(args, "--merge")
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = cwd

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr merge: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// IsAuthenticated checks if gh CLI is authenticated.
func (gh *GitHubClient) IsAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}
