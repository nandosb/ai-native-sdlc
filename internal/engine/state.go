package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Phase represents the current pipeline phase.
type Phase string

const (
	PhaseInit      Phase = "init"
	PhaseBootstrap Phase = "bootstrap"
	PhaseDesign    Phase = "design"
	PhasePlanning  Phase = "planning"
	PhaseTracking  Phase = "tracking"
	PhaseExecuting Phase = "executing"
	PhaseCompleted Phase = "completed"
)

// PhaseStatus represents the status within a phase.
type PhaseStatus string

const (
	StatusPending   PhaseStatus = "pending"
	StatusRunning   PhaseStatus = "running"
	StatusGate      PhaseStatus = "gate"
	StatusCompleted PhaseStatus = "completed"
	StatusFailed    PhaseStatus = "failed"
)

// IssueStatus represents the status of an individual issue.
type IssueStatus string

const (
	IssueBlocked       IssueStatus = "blocked"
	IssueReady         IssueStatus = "ready"
	IssueImplementing  IssueStatus = "implementing"
	IssueReviewing     IssueStatus = "reviewing"
	IssueAwaitingHuman IssueStatus = "awaiting_human"
	IssueDone          IssueStatus = "done"
)

// State is the full runtime state persisted to the store.
type State struct {
	RunID       string                `json:"run_id"`
	PrdURL      string                `json:"prd_url"`
	Phase       Phase                 `json:"phase"`
	PhaseStatus PhaseStatus           `json:"phase_status"`
	Repos       []RepoConfig          `json:"repos"`
	Bootstrap   map[string]RepoState  `json:"bootstrap"`
	Artifacts   map[string]string     `json:"artifacts"`
	Issues      map[string]IssueState `json:"issues"`
	Metrics     MetricsState          `json:"metrics"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// RepoConfig holds configuration for a single repo.
type RepoConfig struct {
	Name     string `json:"name" yaml:"name"`
	Path     string `json:"path" yaml:"path"`
	Team     string `json:"team" yaml:"team"`
	Language string `json:"language,omitempty" yaml:"language"`
}

// RepoState tracks bootstrap status for a repo.
type RepoState struct {
	ClaudeMD       bool `json:"claude_md"`
	ArchitectureMD bool `json:"architecture_md"`
}

// IssueState tracks the status of a single issue in the pipeline.
type IssueState struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Repo       string      `json:"repo"`
	Status     IssueStatus `json:"status"`
	LinearID   string      `json:"linear_id,omitempty"`
	Branch     string      `json:"branch,omitempty"`
	Worktree   string      `json:"worktree,omitempty"`
	PRURL      string      `json:"pr_url,omitempty"`
	DependsOn  []string    `json:"depends_on,omitempty"`
	Iterations int         `json:"iterations"`
}

// MetricsState holds aggregate metrics.
type MetricsState struct {
	TokensIn     int64            `json:"tokens_in"`
	TokensOut    int64            `json:"tokens_out"`
	TotalCost    float64          `json:"total_cost"`
	ByAgent      map[string]Usage `json:"by_agent"`
	PhaseTimings map[string]int64 `json:"phase_timings"`
}

// Usage tracks token usage for a single agent type.
type Usage struct {
	TokensIn  int64   `json:"tokens_in"`
	TokensOut int64   `json:"tokens_out"`
	Cost      float64 `json:"cost"`
	Calls     int     `json:"calls"`
}

// NewState creates a fresh state with the given run ID.
func NewState(runID, prdURL string, repos []RepoConfig) *State {
	return &State{
		RunID:       runID,
		PrdURL:      prdURL,
		Phase:       PhaseInit,
		PhaseStatus: StatusCompleted,
		Repos:       repos,
		Bootstrap:   make(map[string]RepoState),
		Artifacts:   make(map[string]string),
		Issues:      make(map[string]IssueState),
		Metrics: MetricsState{
			ByAgent:      make(map[string]Usage),
			PhaseTimings: make(map[string]int64),
		},
		UpdatedAt: time.Now(),
	}
}

// LoadStateFromFile reads and parses a legacy state.json file (for migration).
func LoadStateFromFile(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	return &s, nil
}
