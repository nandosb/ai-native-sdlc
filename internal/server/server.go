package server

import (
	"embed"
	"io/fs"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/store"
)

//go:embed static
var staticFS embed.FS

// engineEntry wraps an engine with a per-engine running lock.
type engineEntry struct {
	engine  *engine.Engine
	running atomic.Bool
}

// Server serves the web UI and API.
type Server struct {
	mu          sync.RWMutex
	engines     map[string]*engineEntry // runID -> entry
	activeRunID string                  // currently selected run
	store       store.Store
	port        int
	mux         *http.ServeMux
	wsHub       *WSHub
	execMgr     *engine.ExecutionManager
}

// New creates a new server.
func New(eng *engine.Engine, st store.Store, port int) *Server {
	s := &Server{
		engines: make(map[string]*engineEntry),
		store:   st,
		port:    port,
		mux:     http.NewServeMux(),
		wsHub:   NewWSHub(),
		execMgr: engine.NewExecutionManager(eng.Events),
	}

	// Register initial engine
	runID := eng.State.RunID
	s.engines[runID] = &engineEntry{engine: eng}
	s.activeRunID = runID
	s.wsHub.AddEventBus(runID, eng.Events)

	s.registerRoutes()
	return s
}

// activeEngine returns the engine for the currently selected run, or nil.
func (s *Server) activeEngine() *engine.Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry := s.engines[s.activeRunID]
	if entry == nil {
		return nil
	}
	return entry.engine
}

// activeEntry returns the engineEntry for the currently selected run, or nil.
func (s *Server) activeEntry() *engineEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.engines[s.activeRunID]
}

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/issues", s.handleIssues)
	s.mux.HandleFunc("/api/metrics", s.handleMetrics)
	s.mux.HandleFunc("/api/approve", s.handleApprove)
	s.mux.HandleFunc("/api/phases", s.handlePhases)
	s.mux.HandleFunc("/api/manifest", s.handleManifest)
	s.mux.HandleFunc("/api/runs", s.handleRuns)
	s.mux.HandleFunc("/api/runs/select", s.handleSelectRun)
	s.mux.HandleFunc("/api/runs/", s.handleRunAction)
	s.mux.HandleFunc("/api/init", s.handleInit)
	s.mux.HandleFunc("/api/phases/", s.handleRunPhase)
	s.mux.HandleFunc("/api/run", s.handleRunAll)
	s.mux.HandleFunc("/api/health/integrations", s.handleHealthIntegrations)
	s.mux.HandleFunc("/api/artifacts/config", s.handleArtifactConfig)
	s.mux.HandleFunc("/api/artifacts/", s.handleArtifact)

	// Execution endpoints (Runner tab)
	s.mux.HandleFunc("/api/executions", s.handleExecutions)
	s.mux.HandleFunc("/api/executions/", s.handleExecAction)

	// WebSocket
	s.mux.HandleFunc("/ws/events", s.wsHub.HandleWebSocket)

	// Static files (embedded React build)
	staticContent, err := fs.Sub(staticFS, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticContent))
		s.mux.Handle("/", spaHandler(fileServer))
	}
}

// Start begins serving HTTP.
func (s *Server) Start(addr string) error {
	go s.wsHub.Run()
	return http.ListenAndServe(addr, corsMiddleware(s.mux))
}

// spaHandler wraps a file server to serve index.html for unknown routes (SPA routing).
func spaHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the file first
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
