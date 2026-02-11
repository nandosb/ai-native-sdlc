package store

import (
	"time"

	"github.com/yalochat/agentic-sdlc/internal/engine"
)

// RunSummary is a lightweight representation for listing runs.
type RunSummary struct {
	ID         string    `json:"id"`
	Phase      string    `json:"phase"`
	Status     string    `json:"phase_status"`
	PrdURL     string    `json:"prd_url"`
	IssueCount int       `json:"issue_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Store defines the persistence interface for SDLC state.
type Store interface {
	// Run lifecycle
	CreateRun(state *engine.State) error
	LoadRun(runID string) (*engine.State, error)
	LatestRun() (*engine.State, error)
	ListRuns() ([]RunSummary, error)
	DeleteRun(runID string) error

	// Partial saves
	SaveRunMeta(runID string, phase engine.Phase, status engine.PhaseStatus) error
	SaveRepos(runID string, repos []engine.RepoConfig) error
	SaveBootstrap(runID string, repo string, rs engine.RepoState) error
	SaveArtifact(runID string, key, value string) error
	SaveIssue(runID string, issue engine.IssueState) error
	SaveIssues(runID string, issues map[string]engine.IssueState) error

	// Metrics
	RecordMetric(runID string, entry engine.MetricsEntry) error
	RecordPhaseTiming(runID string, phase engine.Phase, durationMs int64) error
	LoadMetricsAggregate(runID string) (engine.MetricsState, error)

	// Executions
	CreateExecution(rec engine.ExecutionRecord) error
	UpdateExecutionStatus(id string, status engine.ExecutionStatus, errorMsg string) error
	UpdateExecutionTokens(id string, tokensIn, tokensOut int64) error
	ListExecutions(runID string) ([]engine.ExecutionRecord, error)
	LatestExecution(runID string, phase string) (*engine.ExecutionRecord, error)

	// Migration helper
	ImportState(state *engine.State) error

	Close() error
}
