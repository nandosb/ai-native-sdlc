package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExecutionType distinguishes phase-level vs issue-level executions.
type ExecutionType string

const (
	ExecTypePhase ExecutionType = "phase"
	ExecTypeIssue ExecutionType = "issue"
)

// ExecutionStatus tracks the lifecycle of an execution.
type ExecutionStatus string

const (
	ExecRunning      ExecutionStatus = "running"
	ExecWaitingInput ExecutionStatus = "waiting_input"
	ExecCompleted    ExecutionStatus = "completed"
	ExecFailed       ExecutionStatus = "failed"
	ExecCancelled    ExecutionStatus = "cancelled"
)

// Message is a single chat message within an execution session.
type Message struct {
	Role      string    `json:"role"`      // "system", "assistant", "user"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Execution represents a single Claude session tied to a phase or issue.
type Execution struct {
	ID        string            `json:"id"`
	RunID     string            `json:"run_id"`
	Type      ExecutionType     `json:"type"`
	Phase     string            `json:"phase"`
	IssueID   string            `json:"issue_id,omitempty"`
	Status    ExecutionStatus   `json:"status"`
	SessionID string            `json:"session_id"`
	Messages  []Message         `json:"messages"`
	Params    map[string]string `json:"params"`
	ParentID  string            `json:"parent_id,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	TokensIn  int64             `json:"tokens_in"`
	TokensOut int64             `json:"tokens_out"`
}

// ExecutionRecord is the persistence-oriented representation (no Messages, no cancel funcs).
type ExecutionRecord struct {
	ID           string          `json:"id"`
	RunID        string          `json:"run_id"`
	ParentID     string          `json:"parent_id,omitempty"`
	Type         ExecutionType   `json:"type"`
	Phase        string          `json:"phase"`
	IssueID      string          `json:"issue_id,omitempty"`
	Status       ExecutionStatus `json:"status"`
	SessionID    string          `json:"session_id,omitempty"`
	TokensIn     int64           `json:"tokens_in"`
	TokensOut    int64           `json:"tokens_out"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ToRecord converts an in-memory Execution to a persistence-oriented ExecutionRecord.
func (e *Execution) ToRecord() ExecutionRecord {
	return ExecutionRecord{
		ID:        e.ID,
		RunID:     e.RunID,
		ParentID:  e.ParentID,
		Type:      e.Type,
		Phase:     e.Phase,
		IssueID:   e.IssueID,
		Status:    e.Status,
		SessionID: e.SessionID,
		TokensIn:  e.TokensIn,
		TokensOut: e.TokensOut,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// ExecutionManager manages active executions with thread-safe access.
type ExecutionManager struct {
	mu         sync.RWMutex
	executions map[string]*Execution
	cancel     map[string]context.CancelFunc
	bus        *EventBus
	maxKept    int
}

// NewExecutionManager creates a manager that publishes events to the given bus.
func NewExecutionManager(bus *EventBus) *ExecutionManager {
	return &ExecutionManager{
		executions: make(map[string]*Execution),
		cancel:     make(map[string]context.CancelFunc),
		bus:        bus,
		maxKept:    100,
	}
}

// Create adds a new execution and publishes an EventExecStarted event.
func (em *ExecutionManager) Create(exec *Execution) {
	em.mu.Lock()
	defer em.mu.Unlock()

	exec.CreatedAt = time.Now()
	exec.UpdatedAt = exec.CreatedAt
	if exec.Messages == nil {
		exec.Messages = []Message{}
	}
	if exec.Params == nil {
		exec.Params = make(map[string]string)
	}
	em.executions[exec.ID] = exec
	em.evictOldest()

	if em.bus != nil {
		em.bus.Publish(Event{
			Type: EventExecStarted,
			Data: map[string]interface{}{
				"execution_id": exec.ID,
				"run_id":       exec.RunID,
				"type":         exec.Type,
				"phase":        exec.Phase,
				"issue_id":     exec.IssueID,
			},
		})
	}
}

// Get returns a copy of the execution, or nil if not found.
func (em *ExecutionManager) Get(id string) *Execution {
	em.mu.RLock()
	defer em.mu.RUnlock()
	exec, ok := em.executions[id]
	if !ok {
		return nil
	}
	cp := *exec
	cp.Messages = make([]Message, len(exec.Messages))
	copy(cp.Messages, exec.Messages)
	return &cp
}

// List returns executions filtered by runID (empty means all).
func (em *ExecutionManager) List(runID string) []*Execution {
	em.mu.RLock()
	defer em.mu.RUnlock()
	var result []*Execution
	for _, exec := range em.executions {
		if runID == "" || exec.RunID == runID {
			cp := *exec
			cp.Messages = make([]Message, len(exec.Messages))
			copy(cp.Messages, exec.Messages)
			result = append(result, &cp)
		}
	}
	return result
}

// AppendMessage adds a message to the execution's chat history.
func (em *ExecutionManager) AppendMessage(id string, msg Message) {
	em.mu.Lock()
	defer em.mu.Unlock()
	exec, ok := em.executions[id]
	if !ok {
		return
	}
	msg.Timestamp = time.Now()
	exec.Messages = append(exec.Messages, msg)
	exec.UpdatedAt = time.Now()
}

// UpdateLastAssistant replaces the content of the last assistant message,
// or appends a new one if there is none. Used for streaming text updates.
func (em *ExecutionManager) UpdateLastAssistant(id string, content string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	exec, ok := em.executions[id]
	if !ok {
		return
	}
	// Find last assistant message and update it
	for i := len(exec.Messages) - 1; i >= 0; i-- {
		if exec.Messages[i].Role == "assistant" {
			exec.Messages[i].Content = content
			exec.Messages[i].Timestamp = time.Now()
			exec.UpdatedAt = time.Now()
			return
		}
	}
	// No assistant message yet, append one
	exec.Messages = append(exec.Messages, Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	})
	exec.UpdatedAt = time.Now()
}

// UpdateStatus changes the execution status and publishes the corresponding event.
func (em *ExecutionManager) UpdateStatus(id string, status ExecutionStatus) {
	em.mu.Lock()
	exec, ok := em.executions[id]
	if !ok {
		em.mu.Unlock()
		return
	}
	exec.Status = status
	exec.UpdatedAt = time.Now()
	runID := exec.RunID
	em.mu.Unlock()

	if em.bus == nil {
		return
	}

	var evtType EventType
	switch status {
	case ExecRunning:
		evtType = EventExecStarted
	case ExecWaitingInput:
		evtType = EventExecWaiting
	case ExecCompleted:
		evtType = EventExecCompleted
	case ExecFailed:
		evtType = EventExecFailed
	case ExecCancelled:
		evtType = EventExecCancelled
	default:
		return
	}

	em.bus.Publish(Event{
		Type: evtType,
		Data: map[string]interface{}{
			"execution_id": id,
			"run_id":       runID,
			"status":       status,
		},
	})
}

// UpdateTokens adds token counts to the execution totals.
func (em *ExecutionManager) UpdateTokens(id string, tokensIn, tokensOut int64) {
	em.mu.Lock()
	defer em.mu.Unlock()
	exec, ok := em.executions[id]
	if !ok {
		return
	}
	exec.TokensIn += tokensIn
	exec.TokensOut += tokensOut
	exec.UpdatedAt = time.Now()
}

// SetCancel stores the cancel function for an execution.
func (em *ExecutionManager) SetCancel(id string, fn context.CancelFunc) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.cancel[id] = fn
}

// Cancel stops a running execution by calling its cancel function.
func (em *ExecutionManager) Cancel(id string) error {
	em.mu.Lock()
	fn, ok := em.cancel[id]
	exec := em.executions[id]
	em.mu.Unlock()

	if exec == nil {
		return fmt.Errorf("execution not found: %s", id)
	}
	if !ok || fn == nil {
		return fmt.Errorf("no cancel function for execution: %s", id)
	}

	fn()
	em.UpdateStatus(id, ExecCancelled)
	return nil
}

// evictOldest removes the oldest completed executions when over the limit.
// Must be called with mu held.
func (em *ExecutionManager) evictOldest() {
	if len(em.executions) <= em.maxKept {
		return
	}
	var oldestID string
	var oldestTime time.Time
	for id, exec := range em.executions {
		if exec.Status != ExecRunning && exec.Status != ExecWaitingInput {
			if oldestID == "" || exec.CreatedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = exec.CreatedAt
			}
		}
	}
	if oldestID != "" {
		delete(em.executions, oldestID)
		delete(em.cancel, oldestID)
	}
}
