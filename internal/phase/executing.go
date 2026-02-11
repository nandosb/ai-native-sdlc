package phase

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/claude"
	"github.com/yalochat/agentic-sdlc/internal/engine"
	gitops "github.com/yalochat/agentic-sdlc/internal/git"
	"github.com/yalochat/agentic-sdlc/internal/prompts"
)

const maxReviewIterations = 3

// Executing runs issues through worktrees with parallel execution.
type Executing struct{}

func (ex *Executing) Name() engine.Phase { return engine.PhaseExecuting }

func (ex *Executing) Run(eng *engine.Engine, params map[string]string) error {
	// Check for single issue mode
	singleIssue := ""
	if params != nil {
		singleIssue = params["issue"]
	}

	// Build dependency graph and batch by level
	batches := topologicalBatch(eng.State.Issues, singleIssue)
	if len(batches) == 0 {
		fmt.Println("No issues ready for execution")
		return nil
	}

	wm := gitops.NewWorktreeManager(".")

	for batchIdx, batch := range batches {
		fmt.Printf("\n--- Batch %d/%d (%d issues) ---\n", batchIdx+1, len(batches), len(batch))

		var wg sync.WaitGroup
		sem := make(chan struct{}, eng.Parallel())
		errCh := make(chan error, len(batch))

		for _, issueID := range batch {
			issue := eng.State.Issues[issueID]
			if issue.Status != engine.IssueReady {
				continue
			}

			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(iss engine.IssueState) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				if err := executeIssue(eng, wm, iss); err != nil {
					fmt.Printf("  ERROR [%s]: %v\n", iss.ID, err)
					errCh <- fmt.Errorf("issue %s: %w", iss.ID, err)
				}
			}(issue)
		}

		wg.Wait()
		close(errCh)

		// Collect errors (continue on individual failures)
		for err := range errCh {
			fmt.Printf("  Warning: %v\n", err)
		}

		// Update blocked issues that may now be ready
		updateBlockedIssues(eng)
	}

	return nil
}

// executeIssue runs a single issue through the coder → reviewer → PR pipeline.
func executeIssue(eng *engine.Engine, wm *gitops.WorktreeManager, issue engine.IssueState) error {
	// Find repo config
	var repoConfig engine.RepoConfig
	for _, r := range eng.State.Repos {
		if r.Name == issue.Repo {
			repoConfig = r
			break
		}
	}
	if repoConfig.Path == "" && len(eng.State.Repos) > 0 {
		repoConfig = eng.State.Repos[0]
	}

	slug := slugify(issue.ID + "-" + issue.Title)
	issue.Branch = "feat/" + slug

	// Update status to implementing
	issue.Status = engine.IssueImplementing
	updateIssue(eng, issue)

	// Create worktree
	fmt.Printf("  [%s] Creating worktree...\n", issue.ID)
	wtPath, err := wm.Create(repoConfig.Path, slug)
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	issue.Worktree = wtPath
	updateIssue(eng, issue)

	// Run coder
	fmt.Printf("  [%s] Running coder agent...\n", issue.ID)
	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentSpawned,
		Data: map[string]string{"agent": "coder", "issue": issue.ID},
	})

	coderPrompt := prompts.Coder(issue.Title, issue.ID, repoConfig.Language)
	start := time.Now()
	coderResult, err := claude.Run(context.Background(), claude.RunConfig{
		Prompt:       coderPrompt,
		CWD:          wtPath,
		Model:        "sonnet",
		AllowedTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"},
	}, eng.Events, issue.ID)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("coder failed: %w", err)
	}

	eng.Metrics.Record(engine.MetricsEntry{
		Timestamp: time.Now(),
		Agent:     "coder",
		Model:     "sonnet",
		TokensIn:  coderResult.TokensIn,
		TokensOut: coderResult.TokensOut,
		Duration:  elapsed.Milliseconds(),
		IssueID:   issue.ID,
		Phase:     engine.PhaseExecuting,
	})

	// Review loop
	for iteration := 1; iteration <= maxReviewIterations; iteration++ {
		issue.Iterations = iteration
		issue.Status = engine.IssueReviewing
		updateIssue(eng, issue)

		fmt.Printf("  [%s] Review iteration %d/%d...\n", issue.ID, iteration, maxReviewIterations)

		eng.Events.Publish(engine.Event{
			Type: engine.EventAgentSpawned,
			Data: map[string]string{"agent": "quality-reviewer", "issue": issue.ID},
		})

		reviewPrompt := prompts.QualityReviewer(issue.Title, repoConfig.Language)
		start = time.Now()
		reviewResult, err := claude.Run(context.Background(), claude.RunConfig{
			Prompt:       reviewPrompt,
			CWD:          wtPath,
			Model:        "opus",
			AllowedTools: []string{"Read", "Glob", "Grep", "Bash"},
		}, eng.Events, issue.ID)
		elapsed = time.Since(start)

		if err != nil {
			return fmt.Errorf("reviewer failed: %w", err)
		}

		eng.Metrics.Record(engine.MetricsEntry{
			Timestamp: time.Now(),
			Agent:     "quality-reviewer",
			Model:     "opus",
			TokensIn:  reviewResult.TokensIn,
			TokensOut: reviewResult.TokensOut,
			Duration:  elapsed.Milliseconds(),
			IssueID:   issue.ID,
			Phase:     engine.PhaseExecuting,
		})

		// Check if review passed
		if isApproved(reviewResult.Output) {
			fmt.Printf("  [%s] Review approved!\n", issue.ID)
			break
		}

		if iteration == maxReviewIterations {
			fmt.Printf("  [%s] Max review iterations reached, escalating to human\n", issue.ID)
			break
		}

		// Apply feedback
		fmt.Printf("  [%s] Applying reviewer feedback...\n", issue.ID)
		feedbackPrompt := prompts.FeedbackWriter(reviewResult.Output)
		start = time.Now()
		fbResult, err := claude.Run(context.Background(), claude.RunConfig{
			Prompt:       feedbackPrompt,
			CWD:          wtPath,
			Model:        "sonnet",
			AllowedTools: []string{"Bash"},
		}, eng.Events, issue.ID)
		elapsed = time.Since(start)

		if err != nil {
			fmt.Printf("  [%s] Warning: feedback application failed: %v\n", issue.ID, err)
			break
		}

		eng.Metrics.Record(engine.MetricsEntry{
			Timestamp: time.Now(),
			Agent:     "feedback-writer",
			Model:     "sonnet",
			TokensIn:  fbResult.TokensIn,
			TokensOut: fbResult.TokensOut,
			Duration:  elapsed.Milliseconds(),
			IssueID:   issue.ID,
			Phase:     engine.PhaseExecuting,
		})
	}

	// Push branch and create PR
	fmt.Printf("  [%s] Pushing branch and creating PR...\n", issue.ID)
	if err := gitops.PushBranch(wtPath); err != nil {
		return fmt.Errorf("push branch: %w", err)
	}

	prURL, exists := gitops.PRExists(wtPath)
	if !exists {
		prURL, err = gitops.CreatePR(wtPath, issue.Title, fmt.Sprintf("Resolves %s\n\nGenerated by Agentic SDLC", issue.ID), "main")
		if err != nil {
			return fmt.Errorf("create PR: %w", err)
		}
	}
	issue.PRURL = prURL
	issue.Status = engine.IssueAwaitingHuman
	updateIssue(eng, issue)

	fmt.Printf("  [%s] PR created: %s\n", issue.ID, prURL)
	return nil
}

// isApproved checks if a review output indicates approval.
func isApproved(output string) bool {
	lower := fmt.Sprintf("%s", output)
	approvalSignals := []string{"approved", "lgtm", "looks good", "no issues found", "all checks pass"}
	for _, signal := range approvalSignals {
		if containsCI(lower, signal) {
			return true
		}
	}
	return false
}

func containsCI(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findCI(s, substr))
}

func findCI(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func updateIssue(eng *engine.Engine, issue engine.IssueState) {
	eng.SaveIssue(issue)
	eng.Events.Publish(engine.Event{
		Type: engine.EventIssueStatus,
		Data: map[string]interface{}{
			"issue_id": issue.ID,
			"status":   issue.Status,
		},
	})
}

// ComputeBatches returns issues grouped into topological batches for parallel execution.
func ComputeBatches(issues map[string]engine.IssueState, singleIssue string) [][]string {
	return topologicalBatch(issues, singleIssue)
}

// topologicalBatch returns issues grouped by dependency level.
func topologicalBatch(issues map[string]engine.IssueState, singleIssue string) [][]string {
	if singleIssue != "" {
		if _, ok := issues[singleIssue]; ok {
			return [][]string{{singleIssue}}
		}
		return nil
	}

	// Build in-degree map
	inDegree := map[string]int{}
	dependents := map[string][]string{} // dep → [issues depending on it]

	for id, iss := range issues {
		if iss.Status == engine.IssueDone || iss.Status == engine.IssueAwaitingHuman {
			continue
		}
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range iss.DependsOn {
			depIss, exists := issues[dep]
			if exists && depIss.Status != engine.IssueDone {
				inDegree[id]++
				dependents[dep] = append(dependents[dep], id)
			}
		}
	}

	var batches [][]string
	for len(inDegree) > 0 {
		// Collect all nodes with in-degree 0
		var batch []string
		for id, deg := range inDegree {
			if deg == 0 {
				batch = append(batch, id)
			}
		}
		if len(batch) == 0 {
			// Cycle detected — add remaining
			for id := range inDegree {
				batch = append(batch, id)
			}
			sort.Strings(batch)
			batches = append(batches, batch)
			break
		}

		sort.Strings(batch)
		batches = append(batches, batch)

		// Remove batch from graph
		for _, id := range batch {
			delete(inDegree, id)
			for _, dep := range dependents[id] {
				if _, ok := inDegree[dep]; ok {
					inDegree[dep]--
				}
			}
		}
	}

	return batches
}

// updateBlockedIssues transitions blocked issues to ready if deps are done.
func updateBlockedIssues(eng *engine.Engine) {
	for id, issue := range eng.State.Issues {
		if issue.Status != engine.IssueBlocked {
			continue
		}
		allDone := true
		for _, dep := range issue.DependsOn {
			if depIss, ok := eng.State.Issues[dep]; ok {
				if depIss.Status != engine.IssueDone && depIss.Status != engine.IssueAwaitingHuman {
					allDone = false
					break
				}
			}
		}
		if allDone {
			issue.Status = engine.IssueReady
			eng.SaveIssue(issue)
			eng.Events.Publish(engine.Event{
				Type: engine.EventIssueStatus,
				Data: map[string]interface{}{
					"issue_id": id,
					"status":   engine.IssueReady,
				},
			})
		}
	}
}

func slugify(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32)
		} else if c == ' ' || c == '_' || c == '/' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	if len(result) > 60 {
		result = result[:60]
	}
	return string(result)
}
