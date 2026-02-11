package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yalochat/agentic-sdlc/internal/engine"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// taggedEvent wraps an engine event with a run ID.
type taggedEvent struct {
	RunID string
	Event engine.Event
}

// busEntry tracks a subscribed event bus and its cancel func.
type busEntry struct {
	bus    *engine.EventBus
	ch     chan engine.Event
	cancel context.CancelFunc
}

// WSHub manages WebSocket connections and broadcasts events from multiple event buses.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
	eventCh chan taggedEvent
	buses   map[string]*busEntry // runID -> entry
}

// NewWSHub creates a new WebSocket hub (no bus required at construction).
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*websocket.Conn]bool),
		eventCh: make(chan taggedEvent, 256),
		buses:   make(map[string]*busEntry),
	}
}

// AddEventBus subscribes to an event bus and tags its events with runID.
func (h *WSHub) AddEventBus(runID string, bus *engine.EventBus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If already subscribed, remove the old one first
	if existing, ok := h.buses[runID]; ok {
		existing.cancel()
		bus.Unsubscribe(existing.ch)
		delete(h.buses, runID)
	}

	ch := bus.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())

	h.buses[runID] = &busEntry{bus: bus, ch: ch, cancel: cancel}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				h.eventCh <- taggedEvent{RunID: runID, Event: evt}
			}
		}
	}()
}

// RemoveEventBus unsubscribes from the event bus for a given runID.
func (h *WSHub) RemoveEventBus(runID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if entry, ok := h.buses[runID]; ok {
		entry.cancel()
		entry.bus.Unsubscribe(entry.ch)
		delete(h.buses, runID)
	}
}

// Run starts the hub's event broadcast loop.
func (h *WSHub) Run() {
	for te := range h.eventCh {
		// Marshal the event with run_id at top level
		raw, err := json.Marshal(te.Event)
		if err != nil {
			continue
		}

		// Inject run_id into the JSON
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		obj["run_id"] = te.RunID

		data, err := json.Marshal(obj)
		if err != nil {
			continue
		}

		h.mu.RLock()
		for conn := range h.clients {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				conn.Close()
				delete(h.clients, conn)
			}
		}
		h.mu.RUnlock()
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket.
func (h *WSHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Keep connection alive, read messages (for future commands)
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
