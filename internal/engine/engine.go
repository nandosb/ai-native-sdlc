package engine

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Manifest represents the top-level manifest.yaml.
type Manifest struct {
	PRD   string       `yaml:"prd"`
	Repos []RepoConfig `yaml:"repos"`
}

// PhaseRunner executes a single phase of the pipeline.
type PhaseRunner interface {
	Name() Phase
	Run(eng *Engine, params map[string]string) error
}

// RunStore is the subset of store.Store that Engine needs (avoids import cycle).
type RunStore interface {
	MetricRecorder
	CreateRun(state *State) error
	LoadRun(runID string) (*State, error)
	LatestRun() (*State, error)
	SaveRunMeta(runID string, phase Phase, status PhaseStatus) error
	SaveRepos(runID string, repos []RepoConfig) error
	SaveBootstrap(runID string, repo string, rs RepoState) error
	SaveArtifact(runID string, key, value string) error
	SaveIssue(runID string, issue IssueState) error
	SaveIssues(runID string, issues map[string]IssueState) error

	// Execution persistence
	CreateExecution(rec ExecutionRecord) error
	UpdateExecutionStatus(id string, status ExecutionStatus, errorMsg string) error
	UpdateExecutionTokens(id string, tokensIn, tokensOut int64) error
	ListExecutions(runID string) ([]ExecutionRecord, error)
	LatestExecution(runID string, phase string) (*ExecutionRecord, error)
}

// Engine is the main orchestrator.
type Engine struct {
	State    *State
	Events   *EventBus
	Metrics  *MetricsCollector
	Store    RunStore
	phases   map[Phase]PhaseRunner
	parallel int
}

// New creates a new engine from a manifest file, initializing fresh state.
func New(manifestPath string, st RunStore) (*Engine, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if len(m.Repos) == 0 {
		return nil, fmt.Errorf("manifest must define at least one repo")
	}

	runID := uuid.New().String()[:8]
	state := NewState(runID, m.PRD, m.Repos)

	eng := &Engine{
		State:    state,
		Events:   NewEventBus(),
		Store:    st,
		phases:   make(map[Phase]PhaseRunner),
		parallel: 3,
	}
	eng.Metrics = NewMetricsCollector(st, runID, &eng.State.Metrics, eng.Events)

	if st != nil {
		if err := st.CreateRun(state); err != nil {
			return nil, fmt.Errorf("create run in store: %w", err)
		}
	}

	return eng, nil
}

// Load reads an existing run from the store.
func Load(st RunStore, runID string) (*Engine, error) {
	var state *State
	var err error
	if runID != "" {
		state, err = st.LoadRun(runID)
	} else {
		state, err = st.LatestRun()
	}
	if err != nil {
		return nil, err
	}
	eng := &Engine{
		State:    state,
		Events:   NewEventBus(),
		Store:    st,
		phases:   make(map[Phase]PhaseRunner),
		parallel: 3,
	}
	eng.Metrics = NewMetricsCollector(st, state.RunID, &eng.State.Metrics, eng.Events)
	return eng, nil
}

// NewEmpty creates an engine for the serve command, loading the latest run from the store if available.
func NewEmpty(st RunStore) *Engine {
	if st != nil {
		if eng, err := Load(st, ""); err == nil {
			return eng
		}
	}
	state := NewState("empty", "", nil)
	eng := &Engine{
		State:    state,
		Events:   NewEventBus(),
		Store:    st,
		phases:   make(map[Phase]PhaseRunner),
		parallel: 3,
	}
	eng.Metrics = NewMetricsCollector(st, "empty", &eng.State.Metrics, eng.Events)
	return eng
}

// RegisterPhase adds a phase runner to the engine.
func (e *Engine) RegisterPhase(p PhaseRunner) {
	e.phases[p.Name()] = p
}

// SetParallel sets the max concurrent goroutines for execution.
func (e *Engine) SetParallel(n int) {
	if n > 0 {
		e.parallel = n
	}
}

// Parallel returns the max concurrent goroutines.
func (e *Engine) Parallel() int {
	return e.parallel
}

// RunPhase executes a single named phase.
func (e *Engine) RunPhase(name string, params map[string]string) error {
	phase := Phase(name)
	runner, ok := e.phases[phase]
	if !ok {
		return fmt.Errorf("unknown phase: %s", name)
	}

	// Create execution record
	execID := uuid.New().String()[:12]
	parentID := ""
	if params != nil {
		parentID = params["_pipeline_exec_id"]
	}
	now := time.Now()
	if e.Store != nil {
		e.Store.CreateExecution(ExecutionRecord{
			ID:        execID,
			RunID:     e.State.RunID,
			ParentID:  parentID,
			Type:      ExecTypePhase,
			Phase:     name,
			Status:    ExecRunning,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	e.State.Phase = phase
	e.State.PhaseStatus = StatusRunning
	e.saveState()

	e.Events.Publish(Event{
		Type: EventPhaseStarted,
		Data: map[string]string{"phase": name, "execution_id": execID},
	})

	start := time.Now()
	err := runner.Run(e, params)
	elapsed := time.Since(start).Milliseconds()

	e.Metrics.RecordPhaseTiming(phase, elapsed)

	if err != nil {
		e.State.PhaseStatus = StatusFailed
		e.saveState()
		if e.Store != nil {
			e.Store.UpdateExecutionStatus(execID, ExecFailed, err.Error())
		}
		e.Events.Publish(Event{
			Type: EventError,
			Data: map[string]string{"phase": name, "error": err.Error(), "execution_id": execID},
		})
		return err
	}

	e.State.PhaseStatus = StatusCompleted
	e.saveState()
	if e.Store != nil {
		e.Store.UpdateExecutionStatus(execID, ExecCompleted, "")
	}

	e.Events.Publish(Event{
		Type: EventPhaseCompleted,
		Data: map[string]string{"phase": name, "duration_ms": fmt.Sprintf("%d", elapsed), "execution_id": execID},
	})

	return nil
}

// RunAll executes the full pipeline with approval gates.
func (e *Engine) RunAll() error {
	// Create parent pipeline execution
	pipelineExecID := uuid.New().String()[:12]
	now := time.Now()
	if e.Store != nil {
		e.Store.CreateExecution(ExecutionRecord{
			ID:        pipelineExecID,
			RunID:     e.State.RunID,
			Type:      ExecTypePhase,
			Phase:     "pipeline",
			Status:    ExecRunning,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	phases := []struct {
		name string
		gate bool
	}{
		{"bootstrap", false},
		{"design", true},
		{"planning", true},
		{"tracking", false},
		{"executing", false},
	}

	for _, p := range phases {
		fmt.Printf("\n=== Phase: %s ===\n", p.name)
		params := map[string]string{"_pipeline_exec_id": pipelineExecID}
		if err := e.RunPhase(p.name, params); err != nil {
			if e.Store != nil {
				e.Store.UpdateExecutionStatus(pipelineExecID, ExecFailed, err.Error())
			}
			return fmt.Errorf("phase %s failed: %w", p.name, err)
		}

		if p.gate {
			fmt.Printf("\n⏸  Approval gate reached after %s phase.\n", p.name)
			fmt.Println("   Run `sdlc approve` to continue, or review artifacts first.")
			e.State.PhaseStatus = StatusGate
			e.saveState()
			e.Events.Publish(Event{
				Type: EventPhaseGate,
				Data: map[string]string{"phase": p.name},
			})
			// Pipeline paused at gate — keep execution running
			return nil
		}
	}

	e.State.Phase = PhaseCompleted
	e.State.PhaseStatus = StatusCompleted
	e.saveState()
	if e.Store != nil {
		e.Store.UpdateExecutionStatus(pipelineExecID, ExecCompleted, "")
	}
	fmt.Println("\n=== Pipeline completed ===")
	return nil
}

// Approve approves the current gate and continues the pipeline.
func (e *Engine) Approve() error {
	if e.State.PhaseStatus != StatusGate {
		return fmt.Errorf("no pending approval gate (current: %s/%s)", e.State.Phase, e.State.PhaseStatus)
	}

	fmt.Printf("Approved gate at phase: %s\n", e.State.Phase)
	e.State.PhaseStatus = StatusCompleted
	e.saveState()

	// Determine next phase and continue
	next := e.nextPhase()
	if next == "" {
		fmt.Println("Pipeline completed.")
		return nil
	}

	fmt.Printf("Continuing to phase: %s\n", next)
	return e.runFromPhase(next)
}

// PrintStatus outputs the current state to stdout.
func (e *Engine) PrintStatus() {
	fmt.Printf("Run ID:       %s\n", e.State.RunID)
	fmt.Printf("Phase:        %s\n", e.State.Phase)
	fmt.Printf("Phase Status: %s\n", e.State.PhaseStatus)
	fmt.Printf("PRD:          %s\n", e.State.PrdURL)
	fmt.Printf("Repos:        %d\n", len(e.State.Repos))
	fmt.Printf("Issues:       %d\n", len(e.State.Issues))
	fmt.Printf("Updated:      %s\n", e.State.UpdatedAt.Format(time.RFC3339))

	if len(e.State.Issues) > 0 {
		counts := map[IssueStatus]int{}
		for _, iss := range e.State.Issues {
			counts[iss.Status]++
		}
		fmt.Println("\nIssue Status:")
		for status, count := range counts {
			fmt.Printf("  %-16s %d\n", status, count)
		}
	}

	fmt.Printf("\nMetrics:\n")
	fmt.Printf("  Tokens In:  %d\n", e.State.Metrics.TokensIn)
	fmt.Printf("  Tokens Out: %d\n", e.State.Metrics.TokensOut)
	fmt.Printf("  Total Cost: $%.4f\n", e.State.Metrics.TotalCost)
}

// SaveIssue updates a single issue in memory and persists it to the store.
func (e *Engine) SaveIssue(issue IssueState) {
	e.State.Issues[issue.ID] = issue
	if e.Store != nil {
		e.Store.SaveIssue(e.State.RunID, issue)
	}
}

// SaveArtifact stores an artifact key-value in memory and persists it.
func (e *Engine) SaveArtifact(key, value string) {
	e.State.Artifacts[key] = value
	if e.Store != nil {
		e.Store.SaveArtifact(e.State.RunID, key, value)
	}
}

// SaveBootstrapState updates bootstrap state for a repo in memory and persists it.
func (e *Engine) SaveBootstrapState(repo string, rs RepoState) {
	e.State.Bootstrap[repo] = rs
	if e.Store != nil {
		e.Store.SaveBootstrap(e.State.RunID, repo, rs)
	}
}

// SetActiveRun loads a different run from the store and switches to it.
func (e *Engine) SetActiveRun(runID string) error {
	if e.Store == nil {
		return fmt.Errorf("no store configured")
	}
	state, err := e.Store.LoadRun(runID)
	if err != nil {
		return fmt.Errorf("load run %s: %w", runID, err)
	}
	e.State = state
	e.Metrics.SetRunID(runID)
	e.Events.Publish(Event{
		Type: EventPhaseStarted,
		Data: map[string]string{"phase": "switched", "run_id": runID},
	})
	return nil
}

func (e *Engine) saveState() {
	if e.Store != nil {
		e.Store.SaveRunMeta(e.State.RunID, e.State.Phase, e.State.PhaseStatus)
	}
}

func (e *Engine) nextPhase() Phase {
	order := []Phase{PhaseBootstrap, PhaseDesign, PhasePlanning, PhaseTracking, PhaseExecuting}
	for i, p := range order {
		if p == e.State.Phase && i+1 < len(order) {
			return order[i+1]
		}
	}
	return ""
}

// ReloadFromManifest reads the manifest file, updates state with its contents,
// generates a new RunID, resets the phase, and publishes a refresh event.
func (e *Engine) ReloadFromManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	if len(m.Repos) == 0 {
		return fmt.Errorf("manifest must define at least one repo")
	}

	newState := NewState(uuid.New().String()[:8], m.PRD, m.Repos)
	if e.Store != nil {
		if err := e.Store.CreateRun(newState); err != nil {
			return fmt.Errorf("create run in store: %w", err)
		}
	}

	e.State = newState
	e.Metrics.SetRunID(e.State.RunID)

	e.Events.Publish(Event{
		Type: EventPhaseStarted,
		Data: map[string]string{"phase": "init"},
	})

	return nil
}

func (e *Engine) runFromPhase(phase Phase) error {
	// Create pipeline-continuation execution
	pipelineExecID := uuid.New().String()[:12]
	now := time.Now()
	if e.Store != nil {
		e.Store.CreateExecution(ExecutionRecord{
			ID:        pipelineExecID,
			RunID:     e.State.RunID,
			Type:      ExecTypePhase,
			Phase:     "pipeline",
			Status:    ExecRunning,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	order := []struct {
		name Phase
		gate bool
	}{
		{PhaseBootstrap, false},
		{PhaseDesign, true},
		{PhasePlanning, true},
		{PhaseTracking, false},
		{PhaseExecuting, false},
	}

	started := false
	for _, p := range order {
		if p.name == phase {
			started = true
		}
		if !started {
			continue
		}

		fmt.Printf("\n=== Phase: %s ===\n", p.name)
		params := map[string]string{"_pipeline_exec_id": pipelineExecID}
		if err := e.RunPhase(string(p.name), params); err != nil {
			if e.Store != nil {
				e.Store.UpdateExecutionStatus(pipelineExecID, ExecFailed, err.Error())
			}
			return fmt.Errorf("phase %s failed: %w", p.name, err)
		}

		if p.gate {
			fmt.Printf("\n⏸  Approval gate reached after %s phase.\n", p.name)
			fmt.Println("   Run `sdlc approve` to continue.")
			e.State.PhaseStatus = StatusGate
			e.saveState()
			return nil
		}
	}

	e.State.Phase = PhaseCompleted
	e.State.PhaseStatus = StatusCompleted
	e.saveState()
	if e.Store != nil {
		e.Store.UpdateExecutionStatus(pipelineExecID, ExecCompleted, "")
	}
	fmt.Println("\n=== Pipeline completed ===")
	return nil
}
