package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/phase"
	"github.com/yalochat/agentic-sdlc/internal/store"
	"gopkg.in/yaml.v3"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"run_id":       eng.State.RunID,
		"phase":        eng.State.Phase,
		"phase_status": eng.State.PhaseStatus,
		"prd_url":      eng.State.PrdURL,
		"repos":        eng.State.Repos,
		"issue_count":  len(eng.State.Issues),
		"artifacts":    eng.State.Artifacts,
		"updated_at":   eng.State.UpdatedAt,
	}

	writeJSON(w, resp)
}

func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	// Group issues by status for kanban view
	grouped := map[engine.IssueStatus][]engine.IssueState{}
	for _, iss := range eng.State.Issues {
		grouped[iss.Status] = append(grouped[iss.Status], iss)
	}

	resp := map[string]interface{}{
		"issues":  eng.State.Issues,
		"grouped": grouped,
	}

	writeJSON(w, resp)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	writeJSON(w, eng.State.Metrics)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	var req struct {
		Action  string `json:"action"`  // "approve" or "reject"
		Comment string `json:"comment"` // optional feedback
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Action == "approve" {
		if err := eng.Approve(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"status": "approved"})
	} else {
		writeJSON(w, map[string]string{"status": "rejected", "comment": req.Comment})
	}
}

func (s *Server) handlePhases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	phases := []map[string]interface{}{
		{"name": "bootstrap", "order": 1},
		{"name": "design", "order": 2, "gate": true},
		{"name": "planning", "order": 3, "gate": true},
		{"name": "tracking", "order": 4},
		{"name": "executing", "order": 5},
	}

	// Try to use real execution records from the store
	var latestByPhase map[string]*engine.ExecutionRecord
	if s.store != nil {
		records, err := s.store.ListExecutions(eng.State.RunID)
		if err == nil && len(records) > 0 {
			latestByPhase = make(map[string]*engine.ExecutionRecord)
			for i := range records {
				rec := &records[i]
				if rec.Phase == "pipeline" {
					continue
				}
				existing, ok := latestByPhase[rec.Phase]
				if !ok || rec.CreatedAt.After(existing.CreatedAt) {
					latestByPhase[rec.Phase] = rec
				}
			}
		}
	}

	if latestByPhase != nil {
		// Use real execution data
		for i := range phases {
			name := phases[i]["name"].(string)
			if rec, ok := latestByPhase[name]; ok {
				phases[i]["status"] = string(rec.Status)
				phases[i]["execution_id"] = rec.ID
				phases[i]["tokens_in"] = rec.TokensIn
				phases[i]["tokens_out"] = rec.TokensOut
				phases[i]["started_at"] = rec.CreatedAt
				phases[i]["updated_at"] = rec.UpdatedAt
				if name == string(eng.State.Phase) {
					phases[i]["current"] = true
				}
			} else {
				phases[i]["status"] = "pending"
			}
		}
	} else {
		// Fallback: ordinal inference for old runs with no execution records
		currentPhase := string(eng.State.Phase)
		for i := range phases {
			name := phases[i]["name"].(string)
			if name == currentPhase {
				phases[i]["status"] = string(eng.State.PhaseStatus)
				phases[i]["current"] = true
			} else {
				order := phases[i]["order"].(int)
				currentOrder := phaseOrder(currentPhase)
				if order < currentOrder {
					phases[i]["status"] = "completed"
				} else {
					phases[i]["status"] = "pending"
				}
			}
		}
	}

	writeJSON(w, phases)
}

func phaseOrder(phase string) int {
	order := map[string]int{
		"init":      0,
		"bootstrap": 1,
		"design":    2,
		"planning":  3,
		"tracking":  4,
		"executing": 5,
		"completed": 6,
	}
	if o, ok := order[phase]; ok {
		return o
	}
	return -1
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	const manifestPath = "manifest.yaml"

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			eng := s.activeEngine()
			if eng != nil {
				writeJSON(w, map[string]interface{}{
					"prd":   eng.State.PrdURL,
					"repos": eng.State.Repos,
				})
			} else {
				writeJSON(w, map[string]interface{}{
					"prd":   "",
					"repos": []interface{}{},
				})
			}
			return
		}
		var m engine.Manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			http.Error(w, "invalid manifest yaml", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{
			"prd":   m.PRD,
			"repos": m.Repos,
		})

	case http.MethodPost:
		var req struct {
			PRD   string              `json:"prd"`
			Repos []engine.RepoConfig `json:"repos"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.PRD == "" {
			http.Error(w, "prd is required", http.StatusBadRequest)
			return
		}
		if len(req.Repos) == 0 {
			http.Error(w, "at least one repo is required", http.StatusBadRequest)
			return
		}
		for i, repo := range req.Repos {
			if repo.Name == "" || repo.Path == "" || repo.Team == "" {
				http.Error(w, fmt.Sprintf("repo %d: name, path, and team are required", i), http.StatusBadRequest)
				return
			}
		}

		m := engine.Manifest{PRD: req.PRD, Repos: req.Repos}
		yamlData, err := yaml.Marshal(m)
		if err != nil {
			http.Error(w, "failed to marshal manifest", http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(manifestPath, yamlData, 0644); err != nil {
			http.Error(w, "failed to write manifest", http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var runs []store.RunSummary

	if s.store != nil {
		var err error
		runs, err = s.store.ListRuns()
		if err != nil {
			http.Error(w, "failed to list runs: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Include the active engine's run if it's not already in the DB results.
	// This handles the case where the server starts with an empty DB and the
	// placeholder engine (RunID="empty") hasn't been persisted yet.
	eng := s.activeEngine()
	if eng != nil {
		found := false
		for _, r := range runs {
			if r.ID == eng.State.RunID {
				found = true
				break
			}
		}
		if !found {
			runs = append([]store.RunSummary{{
				ID:        eng.State.RunID,
				Phase:     string(eng.State.Phase),
				Status:    string(eng.State.PhaseStatus),
				PrdURL:    eng.State.PrdURL,
				IssueCount: len(eng.State.Issues),
				CreatedAt: eng.State.UpdatedAt,
				UpdatedAt: eng.State.UpdatedAt,
			}}, runs...)
		}
	}

	if runs == nil {
		runs = []store.RunSummary{}
	}
	writeJSON(w, runs)
}

func (s *Server) handleSelectRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RunID == "" {
		http.Error(w, "run_id is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	// Check if engine already loaded
	if _, ok := s.engines[req.RunID]; ok {
		s.activeRunID = req.RunID
		s.mu.Unlock()
		writeJSON(w, map[string]string{"status": "ok", "run_id": req.RunID})
		return
	}
	s.mu.Unlock()

	// Load from store
	eng, err := engine.Load(s.store, req.RunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	phase.RegisterAll(eng)

	s.mu.Lock()
	s.engines[req.RunID] = &engineEntry{engine: eng}
	s.activeRunID = req.RunID
	s.mu.Unlock()

	s.wsHub.AddEventBus(req.RunID, eng.Events)

	writeJSON(w, map[string]string{"status": "ok", "run_id": req.RunID})
}

func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PRD   string              `json:"prd"`
		Repos []engine.RepoConfig `json:"repos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.PRD == "" {
		http.Error(w, "prd is required", http.StatusBadRequest)
		return
	}
	if len(req.Repos) == 0 {
		http.Error(w, "at least one repo is required", http.StatusBadRequest)
		return
	}
	for i, repo := range req.Repos {
		if repo.Name == "" || repo.Path == "" || repo.Team == "" {
			http.Error(w, fmt.Sprintf("repo %d: name, path, and team are required", i), http.StatusBadRequest)
			return
		}
	}

	// Write manifest.yaml
	m := engine.Manifest{PRD: req.PRD, Repos: req.Repos}
	yamlData, err := yaml.Marshal(m)
	if err != nil {
		http.Error(w, "failed to marshal manifest", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile("manifest.yaml", yamlData, 0644); err != nil {
		http.Error(w, "failed to write manifest", http.StatusInternalServerError)
		return
	}

	// Create new engine from manifest
	eng, err := engine.New("manifest.yaml", s.store)
	if err != nil {
		http.Error(w, "init failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Register phases and add to engine map
	phase.RegisterAll(eng)

	runID := eng.State.RunID
	s.mu.Lock()
	s.engines[runID] = &engineEntry{engine: eng}
	s.activeRunID = runID
	s.mu.Unlock()

	s.wsHub.AddEventBus(runID, eng.Events)

	writeJSON(w, map[string]string{"status": "ok", "run_id": runID})
}

func (s *Server) handleRunPhase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract phase name from /api/phases/{name}/run
	path := strings.TrimPrefix(r.URL.Path, "/api/phases/")
	path = strings.TrimSuffix(path, "/run")
	phaseName := path

	validPhases := map[string]bool{
		"bootstrap": true,
		"design":    true,
		"planning":  true,
		"tracking":  true,
		"executing": true,
	}
	if !validPhases[phaseName] {
		http.Error(w, "unknown phase: "+phaseName, http.StatusBadRequest)
		return
	}

	entry := s.activeEntry()
	if entry == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	if !entry.running.CompareAndSwap(false, true) {
		writeJSONStatus(w, http.StatusConflict, map[string]string{"error": "phase already running"})
		return
	}

	var req struct {
		Params map[string]string `json:"params"`
	}
	// Body is optional
	json.NewDecoder(r.Body).Decode(&req)
	if req.Params == nil {
		req.Params = map[string]string{}
	}
	req.Params["_non_interactive"] = "true"

	go func() {
		defer entry.running.Store(false)
		entry.engine.RunPhase(phaseName, req.Params)
	}()

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "started", "phase": phaseName})
}

func (s *Server) handleRunAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entry := s.activeEntry()
	if entry == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	if !entry.running.CompareAndSwap(false, true) {
		writeJSONStatus(w, http.StatusConflict, map[string]string{"error": "phase already running"})
		return
	}

	go func() {
		defer entry.running.Store(false)
		entry.engine.RunAll()
	}()

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "started"})
}

func (s *Server) handleArtifactConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	phaseArtifact := make(map[string]string, len(engine.PhaseArtifact))
	for phase, key := range engine.PhaseArtifact {
		phaseArtifact[string(phase)] = key
	}

	writeJSON(w, map[string]interface{}{
		"artifacts":      engine.Artifacts,
		"phase_artifact": phaseArtifact,
	})
}

func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng := s.activeEngine()
	if eng == nil {
		http.Error(w, "no active run", http.StatusNotFound)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/api/artifacts/")
	if key == "" {
		http.Error(w, "artifact key required", http.StatusBadRequest)
		return
	}

	filePath, ok := eng.State.Artifacts[key]
	if !ok || filePath == "" {
		http.Error(w, "artifact not found: "+key, http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "failed to read artifact: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"key":     key,
		"path":    filePath,
		"content": string(content),
	})
}

// IntegrationHealth represents the connection status of a single integration.
type IntegrationHealth struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Detail    string `json:"detail,omitempty"`
	Mode      string `json:"mode,omitempty"` // "api", "mcp", or ""
	CheckedAt string `json:"checked_at"`
}

func (s *Server) handleHealthIntegrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	now := time.Now().Format(time.RFC3339)
	results := []IntegrationHealth{
		checkClaude(now),
		checkGitHub(now),
		checkLinear(now),
		checkNotion(now),
	}

	writeJSON(w, results)
}

func checkClaude(ts string) IntegrationHealth {
	h := IntegrationHealth{Name: "claude", CheckedAt: ts}
	out, err := exec.Command("claude", "--version").CombinedOutput()
	if err != nil {
		h.Detail = "claude CLI not found"
		return h
	}
	h.OK = true
	h.Detail = strings.TrimSpace(string(out))
	return h
}

func checkGitHub(ts string) IntegrationHealth {
	h := IntegrationHealth{Name: "github", CheckedAt: ts}
	out, err := exec.Command("gh", "auth", "status").CombinedOutput()
	if err != nil {
		h.Detail = strings.TrimSpace(string(out))
		if h.Detail == "" {
			h.Detail = "gh CLI not found or not authenticated"
		}
		return h
	}
	h.OK = true
	// Extract account info from output
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Logged in") || strings.Contains(line, "account") {
			h.Detail = line
			break
		}
	}
	if h.Detail == "" {
		h.Detail = "authenticated"
	}
	return h
}

func checkLinear(ts string) IntegrationHealth {
	h := IntegrationHealth{Name: "linear", CheckedAt: ts}
	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey != "" {
		h.OK = true
		h.Mode = "api"
		h.Detail = "LINEAR_API_KEY set"
		return h
	}
	// Check if Claude has Linear MCP tools by checking claude mcp list
	out, err := exec.Command("claude", "mcp", "list").CombinedOutput()
	if err == nil && strings.Contains(string(out), "linear") {
		h.OK = true
		h.Mode = "mcp"
		h.Detail = "via Claude MCP"
		return h
	}
	h.Detail = "no LINEAR_API_KEY and no MCP connection"
	return h
}

func checkNotion(ts string) IntegrationHealth {
	h := IntegrationHealth{Name: "notion", CheckedAt: ts}
	apiKey := os.Getenv("NOTION_API_KEY")
	if apiKey != "" {
		h.OK = true
		h.Mode = "api"
		h.Detail = "NOTION_API_KEY set"
		return h
	}
	// Check if Claude has Notion MCP tools
	out, err := exec.Command("claude", "mcp", "list").CombinedOutput()
	if err == nil && strings.Contains(string(out), "notion") {
		h.OK = true
		h.Mode = "mcp"
		h.Detail = "via Claude MCP"
		return h
	}
	h.Detail = "no NOTION_API_KEY and no MCP connection"
	return h
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONStatus(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
