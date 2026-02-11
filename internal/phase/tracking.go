package phase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/claude"
	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/integrations"
	"github.com/yalochat/agentic-sdlc/internal/prompts"
)

// Tracking creates Linear issues from a PERT document.
type Tracking struct{}

func (t *Tracking) Name() engine.Phase { return engine.PhaseTracking }

// PERTTask represents a task extracted from the PERT document.
type PERTTask struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Repo        string   `json:"repo"`
	DependsOn   []string `json:"depends_on"`
	Estimate    string   `json:"estimate"`
}

func (t *Tracking) Run(eng *engine.Engine, params map[string]string) error {
	pertPath := eng.State.Artifacts[engine.ArtifactPERT]
	if params != nil && params[engine.ArtifactPERT] != "" {
		pertPath = params[engine.ArtifactPERT]
	}
	if pertPath == "" {
		return fmt.Errorf("no PERT document available (run planning phase first)")
	}

	pertContent, err := os.ReadFile(pertPath)
	if err != nil {
		return fmt.Errorf("read PERT: %w", err)
	}

	// Extract structured tasks from PERT
	tasks, err := parsePERTTasks(string(pertContent))
	if err != nil {
		return fmt.Errorf("parse PERT tasks: %w", err)
	}

	fmt.Printf("Found %d tasks in PERT\n", len(tasks))

	// Determine team
	team := ""
	if params != nil {
		team = params["team"]
	}
	if team == "" && len(eng.State.Repos) > 0 {
		team = eng.State.Repos[0].Team
	}

	// Create Linear issues — API-first, Claude MCP fallback
	linearClient := integrations.NewLinearClient()

	if linearClient.IsConfigured() {
		// API path: create issues one by one via GraphQL
		for _, task := range tasks {
			if _, exists := eng.State.Issues[task.ID]; exists {
				fmt.Printf("  Issue %s already tracked, skipping\n", task.ID)
				continue
			}

			fmt.Printf("  Creating issue: %s\n", task.Title)

			issueState := engine.IssueState{
				ID:        task.ID,
				Title:     task.Title,
				Repo:      task.Repo,
				DependsOn: task.DependsOn,
			}
			if len(task.DependsOn) > 0 {
				issueState.Status = engine.IssueBlocked
			} else {
				issueState.Status = engine.IssueReady
			}

			taskTeam := team
			for _, r := range eng.State.Repos {
				if r.Name == task.Repo {
					taskTeam = r.Team
					break
				}
			}

			linearID, err := linearClient.CreateIssue(integrations.LinearIssue{
				Title:       task.Title,
				Description: task.Description,
				Team:        taskTeam,
				Estimate:    task.Estimate,
			})
			if err != nil {
				fmt.Printf("    Warning: failed to create Linear issue: %v\n", err)
			} else {
				issueState.LinearID = linearID
				fmt.Printf("    Created Linear issue: %s\n", linearID)
			}

			eng.SaveIssue(issueState)
			eng.Events.Publish(engine.Event{
				Type: engine.EventIssueStatus,
				Data: map[string]interface{}{
					"issue_id": task.ID,
					"status":   issueState.Status,
					"title":    task.Title,
				},
			})
		}

		// Create blocking relationships via API
		for _, task := range tasks {
			issue := eng.State.Issues[task.ID]
			for _, depID := range task.DependsOn {
				if dep, ok := eng.State.Issues[depID]; ok && dep.LinearID != "" && issue.LinearID != "" {
					linearClient.CreateRelation(dep.LinearID, issue.LinearID, "blocks")
				}
			}
		}
	} else {
		// MCP fallback: use Claude with Linear MCP tools
		if params != nil && params["_non_interactive"] == "true" {
			return fmt.Errorf(
				"LINEAR_API_KEY is not set. " +
					"Configure LINEAR_API_KEY in your environment to create Linear issues from the web UI")
		}
		fmt.Println("LINEAR_API_KEY not set, using Claude MCP fallback for Linear")

		// Filter out already-tracked tasks
		var newTasks []PERTTask
		for _, task := range tasks {
			if _, exists := eng.State.Issues[task.ID]; exists {
				fmt.Printf("  Issue %s already tracked, skipping\n", task.ID)
				continue
			}
			newTasks = append(newTasks, task)
		}

		if len(newTasks) > 0 {
			linearMapping, err := createIssuesViaClaude(eng, newTasks, team)
			if err != nil {
				fmt.Printf("  Warning: Claude MCP Linear fallback failed: %v\n", err)
			}

			// Save issue states
			for _, task := range newTasks {
				issueState := engine.IssueState{
					ID:        task.ID,
					Title:     task.Title,
					Repo:      task.Repo,
					DependsOn: task.DependsOn,
				}
				if len(task.DependsOn) > 0 {
					issueState.Status = engine.IssueBlocked
				} else {
					issueState.Status = engine.IssueReady
				}
				if id, ok := linearMapping[task.ID]; ok {
					issueState.LinearID = id
					fmt.Printf("  Created Linear issue: %s → %s\n", task.ID, id)
				}
				eng.SaveIssue(issueState)
				eng.Events.Publish(engine.Event{
					Type: engine.EventIssueStatus,
					Data: map[string]interface{}{
						"issue_id": task.ID,
						"status":   issueState.Status,
						"title":    task.Title,
					},
				})
			}
		}
	}

	fmt.Printf("Created %d issues\n", len(tasks))
	return nil
}

// parsePERTTasks extracts tasks from PERT markdown content.
// Looks for a JSON block first, falls back to markdown parsing.
func parsePERTTasks(content string) ([]PERTTask, error) {
	// Try JSON extraction first
	jsonData, err := claude.ExtractJSON(content)
	if err == nil && jsonData != nil {
		var tasks []PERTTask
		if err := json.Unmarshal(jsonData, &tasks); err == nil {
			return tasks, nil
		}
		// Maybe wrapped in an object
		var wrapper struct {
			Tasks []PERTTask `json:"tasks"`
		}
		if err := json.Unmarshal(jsonData, &wrapper); err == nil && len(wrapper.Tasks) > 0 {
			return wrapper.Tasks, nil
		}
	}

	// Fallback: parse markdown table or list
	return parsePERTMarkdown(content)
}

// createIssuesViaClaude invokes Claude with Linear MCP tools to create issues
// when LINEAR_API_KEY is not available. Returns a mapping of task ID → Linear identifier.
func createIssuesViaClaude(eng *engine.Engine, tasks []PERTTask, team string) (map[string]string, error) {
	tasksJSON, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal tasks: %w", err)
	}

	prompt := prompts.LinearIssueCreator(string(tasksJSON), team)

	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentSpawned,
		Data: map[string]string{"agent": "linear-issue-creator"},
	})

	start := time.Now()
	result, err := claude.Run(context.Background(), claude.RunConfig{
		Prompt:       prompt,
		Model:        "sonnet",
		AllowedTools: []string{"mcp__plugin_linear_linear__*"},
	}, eng.Events, "")
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("claude linear creator failed: %w", err)
	}

	eng.Metrics.Record(engine.MetricsEntry{
		Timestamp: time.Now(),
		Agent:     "linear-issue-creator",
		Model:     "sonnet",
		TokensIn:  result.TokensIn,
		TokensOut: result.TokensOut,
		Duration:  elapsed.Milliseconds(),
		Phase:     engine.PhaseTracking,
	})

	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentCompleted,
		Data: map[string]string{"agent": "linear-issue-creator"},
	})

	// Parse the JSON mapping from Claude's output
	mapping := make(map[string]string)
	jsonData, err := claude.ExtractJSON(result.Output)
	if err == nil && jsonData != nil {
		if err := json.Unmarshal(jsonData, &mapping); err != nil {
			fmt.Printf("  Warning: could not parse Linear ID mapping: %v\n", err)
		}
	} else {
		fmt.Println("  Warning: no JSON mapping found in Claude response")
	}

	return mapping, nil
}

// parsePERTMarkdown parses tasks from markdown format.
func parsePERTMarkdown(content string) ([]PERTTask, error) {
	var tasks []PERTTask
	lines := strings.Split(content, "\n")
	taskNum := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			continue
		}
		// Simple heuristic: lines starting with - or * that look like tasks
		text := strings.TrimLeft(line, "-* ")
		if text == "" {
			continue
		}

		tasks = append(tasks, PERTTask{
			ID:    fmt.Sprintf("TASK-%03d", taskNum),
			Title: text,
		})
		taskNum++
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in PERT document")
	}

	return tasks, nil
}
