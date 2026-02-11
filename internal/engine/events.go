package engine

import (
	"sync"
	"time"
)

// EventType identifies the kind of event.
type EventType string

const (
	EventPhaseStarted      EventType = "phase.started"
	EventPhaseCompleted    EventType = "phase.completed"
	EventPhaseGate         EventType = "phase.gate"
	EventIssueStatus       EventType = "issue.status_changed"
	EventAgentSpawned      EventType = "agent.spawned"
	EventAgentOutput       EventType = "agent.output"
	EventAgentCompleted    EventType = "agent.completed"
	EventMetricsUpdated    EventType = "metrics.updated"
	EventError             EventType = "error"

	// Execution lifecycle events (Runner tab)
	EventExecStarted   EventType = "execution.started"
	EventExecOutput    EventType = "execution.output"
	EventExecMessage   EventType = "execution.message"
	EventExecCompleted EventType = "execution.completed"
	EventExecFailed    EventType = "execution.failed"
	EventExecWaiting   EventType = "execution.waiting_input"
	EventExecCancelled EventType = "execution.cancelled"
)

// Event is a single event published through the bus.
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventBus is a simple pub/sub event bus.
type EventBus struct {
	mu          sync.RWMutex
	subscribers []chan Event
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Subscribe returns a channel that receives events.
func (eb *EventBus) Subscribe() chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan Event, 100)
	eb.subscribers = append(eb.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber channel.
func (eb *EventBus) Unsubscribe(ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for i, sub := range eb.subscribers {
		if sub == ch {
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// Publish sends an event to all subscribers (non-blocking).
func (eb *EventBus) Publish(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.subscribers {
		select {
		case ch <- evt:
		default:
			// Drop if subscriber is full
		}
	}
}
