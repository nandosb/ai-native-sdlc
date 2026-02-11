package engine

import (
	"sync"
	"time"
)

// MetricsEntry is a single metrics log entry.
type MetricsEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
	Model     string    `json:"model"`
	TokensIn  int64     `json:"tokens_in"`
	TokensOut int64     `json:"tokens_out"`
	Cost      float64   `json:"cost"`
	Duration  int64     `json:"duration_ms"`
	IssueID   string    `json:"issue_id,omitempty"`
	Phase     Phase     `json:"phase"`
}

// MetricRecorder is the interface the MetricsCollector uses to persist entries.
type MetricRecorder interface {
	RecordMetric(runID string, entry MetricsEntry) error
	RecordPhaseTiming(runID string, phase Phase, durationMs int64) error
}

// MetricsCollector collects and persists agent metrics.
type MetricsCollector struct {
	mu       sync.Mutex
	store    MetricRecorder
	runID    string
	state    *MetricsState
	bus      *EventBus
}

// NewMetricsCollector creates a collector backed by a store.
func NewMetricsCollector(st MetricRecorder, runID string, state *MetricsState, bus *EventBus) *MetricsCollector {
	return &MetricsCollector{
		store: st,
		runID: runID,
		state: state,
		bus:   bus,
	}
}

// SetRunID updates the run ID (used when switching active runs).
func (mc *MetricsCollector) SetRunID(runID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.runID = runID
}

// Record logs a single agent invocation.
func (mc *MetricsCollector) Record(entry MetricsEntry) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update in-memory aggregate
	mc.state.TokensIn += entry.TokensIn
	mc.state.TokensOut += entry.TokensOut
	mc.state.TotalCost += entry.Cost

	usage := mc.state.ByAgent[entry.Agent]
	usage.TokensIn += entry.TokensIn
	usage.TokensOut += entry.TokensOut
	usage.Cost += entry.Cost
	usage.Calls++
	mc.state.ByAgent[entry.Agent] = usage

	// Persist to store
	if mc.store != nil {
		mc.store.RecordMetric(mc.runID, entry)
	}

	// Publish metrics event
	if mc.bus != nil {
		mc.bus.Publish(Event{
			Type: EventMetricsUpdated,
			Data: entry,
		})
	}

	return nil
}

// RecordPhaseTiming records how long a phase took.
func (mc *MetricsCollector) RecordPhaseTiming(phase Phase, durationMs int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.state.PhaseTimings[string(phase)] = durationMs

	if mc.store != nil {
		mc.store.RecordPhaseTiming(mc.runID, phase, durationMs)
	}
}
