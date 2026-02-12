package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/yalochat/agentic-sdlc/internal/claude"
	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/integrations"
	"github.com/yalochat/agentic-sdlc/internal/prompts"
)

// handleExecutions dispatches GET/POST for /api/executions.
func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListExecutions(w, r)
	case http.MethodPost:
		s.handleCreateExecution(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleExecAction routes /api/executions/{id}[/action] requests.
func (s *Server) handleExecAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/executions/")
	parts := strings.SplitN(path, "/", 2)
	execID := parts[0]
	if execID == "" {
		http.Error(w, "execution id required", http.StatusBadRequest)
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		s.handleGetExecution(w, r, execID)
	case action == "message" && r.Method == http.MethodPost:
		s.handleSendMessage(w, r, execID)
	case action == "approve" && r.Method == http.MethodPost:
		s.handleApproveExecution(w, r, execID)
	case action == "cancel" && r.Method == http.MethodPost:
		s.handleCancelExecution(w, r, execID)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")

	// Start with in-memory executions
	liveExecs := s.execMgr.List(runID)
	seen := make(map[string]bool, len(liveExecs))
	for _, ex := range liveExecs {
		seen[ex.ID] = true
	}

	// Merge stored records that are not already in memory
	if s.store != nil && runID != "" {
		stored, err := s.store.ListExecutions(runID)
		if err == nil {
			for _, rec := range stored {
				if seen[rec.ID] {
					continue
				}
				liveExecs = append(liveExecs, &engine.Execution{
					ID:        rec.ID,
					RunID:     rec.RunID,
					ParentID:  rec.ParentID,
					Type:      rec.Type,
					Phase:     rec.Phase,
					IssueID:   rec.IssueID,
					Status:    rec.Status,
					SessionID: rec.SessionID,
					TokensIn:  rec.TokensIn,
					TokensOut: rec.TokensOut,
					CreatedAt: rec.CreatedAt,
					UpdatedAt: rec.UpdatedAt,
				})
			}
		}
	}

	if liveExecs == nil {
		liveExecs = []*engine.Execution{}
	}
	writeJSON(w, liveExecs)
}

func (s *Server) handleCreateExecution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID   string            `json:"run_id"`
		Type    string            `json:"type"`
		Phase   string            `json:"phase"`
		IssueID string            `json:"issue_id,omitempty"`
		Params  map[string]string `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Phase == "" {
		http.Error(w, "phase is required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = "phase"
	}
	if req.Params == nil {
		req.Params = make(map[string]string)
	}

	// Use active engine's run ID if not specified
	eng := s.activeEngine()
	if req.RunID == "" && eng != nil {
		req.RunID = eng.State.RunID
	}

	executionID := uuid.New().String()
	sessionID := uuid.New().String()

	exec := &engine.Execution{
		ID:        executionID,
		RunID:     req.RunID,
		Type:      engine.ExecutionType(req.Type),
		Phase:     req.Phase,
		IssueID:   req.IssueID,
		Status:    engine.ExecRunning,
		SessionID: sessionID,
		Params:    req.Params,
	}

	s.execMgr.Create(exec)

	// Persist to store
	if s.store != nil {
		s.store.CreateExecution(exec.ToRecord())
	}

	// Build initial prompt from phase + params
	prompt := buildPrompt(req.Phase, req.Params, eng)

	// Resolve CWD
	cwd := resolveCWD(req.Phase, req.IssueID, eng)

	ctx, cancel := context.WithCancel(context.Background())
	s.execMgr.SetCancel(executionID, cancel)

	// Record the initial system message
	s.execMgr.AppendMessage(executionID, engine.Message{
		Role:    "system",
		Content: "Starting " + req.Phase + " phase...",
	})

	log.Printf("[exec] creating execution %s: phase=%s cwd=%s", executionID, req.Phase, cwd)

	go func() {
		cfg := claude.RunConfig{
			Prompt:       prompt,
			CWD:          cwd,
			Model:        "sonnet",
			SessionID:    sessionID,
			AllowedTools: toolsForPhase(req.Phase, req.Params, eng),
		}

		var bus *engine.EventBus
		if eng != nil {
			bus = eng.Events
		}

		log.Printf("[exec] %s: starting claude session", executionID)
		result, err := claude.RunSession(ctx, cfg, bus, s.execMgr, executionID)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("[exec] %s: cancelled", executionID)
				return
			}
			log.Printf("[exec] %s: error: %v", executionID, err)
			s.execMgr.AppendMessage(executionID, engine.Message{
				Role:    "system",
				Content: "Error: " + err.Error(),
			})
			s.execMgr.UpdateStatus(executionID, engine.ExecFailed)
			if s.store != nil {
				s.store.UpdateExecutionStatus(executionID, engine.ExecFailed, err.Error())
			}
			return
		}
		log.Printf("[exec] %s: completed (exit=%d, tokens=%d/%d, output=%d bytes)",
			executionID, result.ExitCode, result.TokensIn, result.TokensOut, len(result.Output))
		s.execMgr.UpdateStatus(executionID, engine.ExecWaitingInput)
		if s.store != nil {
			s.store.UpdateExecutionStatus(executionID, engine.ExecWaitingInput, "")
			s.store.UpdateExecutionTokens(executionID, result.TokensIn, result.TokensOut)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]interface{}{
		"id":         executionID,
		"session_id": sessionID,
		"status":     "running",
	})
}

func (s *Server) handleGetExecution(w http.ResponseWriter, _ *http.Request, execID string) {
	exec := s.execMgr.Get(execID)
	if exec != nil {
		writeJSON(w, exec)
		return
	}

	// Fall back to store for persisted executions not in memory
	if s.store != nil {
		rec, err := s.store.GetExecution(execID)
		if err == nil && rec != nil {
			writeJSON(w, &engine.Execution{
				ID:        rec.ID,
				RunID:     rec.RunID,
				ParentID:  rec.ParentID,
				Type:      rec.Type,
				Phase:     rec.Phase,
				IssueID:   rec.IssueID,
				Status:    rec.Status,
				SessionID: rec.SessionID,
				TokensIn:  rec.TokensIn,
				TokensOut: rec.TokensOut,
				CreatedAt: rec.CreatedAt,
				UpdatedAt: rec.UpdatedAt,
			})
			return
		}
	}

	http.Error(w, "execution not found", http.StatusNotFound)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request, execID string) {
	exec := s.execMgr.Get(execID)
	if exec == nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	// Reject if already running (concurrent send guard)
	if exec.Status == engine.ExecRunning {
		writeJSONStatus(w, http.StatusConflict, map[string]string{
			"error": "execution is already running, wait for it to finish",
		})
		return
	}

	// Reject if terminal state
	if exec.Status == engine.ExecCompleted || exec.Status == engine.ExecFailed || exec.Status == engine.ExecCancelled {
		http.Error(w, "execution is in terminal state: "+string(exec.Status), http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	// Append user message
	s.execMgr.AppendMessage(execID, engine.Message{
		Role:    "user",
		Content: req.Content,
	})

	if s.activeEngine() != nil {
		s.activeEngine().Events.Publish(engine.Event{
			Type: engine.EventExecMessage,
			Data: map[string]interface{}{
				"execution_id": execID,
				"role":         "user",
				"content":      req.Content,
			},
		})
	}

	// Update status to running
	s.execMgr.UpdateStatus(execID, engine.ExecRunning)

	// Resolve CWD
	eng := s.activeEngine()
	cwd := resolveCWD(exec.Phase, exec.IssueID, eng)

	ctx, cancel := context.WithCancel(context.Background())
	s.execMgr.SetCancel(execID, cancel)

	log.Printf("[exec] %s: sending message (%d bytes)", execID, len(req.Content))

	go func() {
		cfg := claude.RunConfig{
			Prompt:       req.Content,
			CWD:          cwd,
			Model:        "sonnet",
			SessionID:    exec.SessionID,
			Resume:       true,
			AllowedTools: toolsForPhase(exec.Phase, exec.Params, eng),
		}

		var bus *engine.EventBus
		if eng != nil {
			bus = eng.Events
		}

		result, err := claude.RunSession(ctx, cfg, bus, s.execMgr, execID)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("[exec] %s: message cancelled", execID)
				return
			}
			log.Printf("[exec] %s: message error: %v", execID, err)
			s.execMgr.AppendMessage(execID, engine.Message{
				Role:    "system",
				Content: "Error: " + err.Error(),
			})
			s.execMgr.UpdateStatus(execID, engine.ExecFailed)
			if s.store != nil {
				s.store.UpdateExecutionStatus(execID, engine.ExecFailed, err.Error())
			}
			return
		}
		log.Printf("[exec] %s: message done (exit=%d, output=%d bytes)",
			execID, result.ExitCode, len(result.Output))
		s.execMgr.UpdateStatus(execID, engine.ExecWaitingInput)
		if s.store != nil {
			s.store.UpdateExecutionStatus(execID, engine.ExecWaitingInput, "")
			s.store.UpdateExecutionTokens(execID, result.TokensIn, result.TokensOut)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "sent"})
}

func (s *Server) handleApproveExecution(w http.ResponseWriter, _ *http.Request, execID string) {
	exec := s.execMgr.Get(execID)
	if exec == nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}
	s.execMgr.UpdateStatus(execID, engine.ExecCompleted)
	if s.store != nil {
		s.store.UpdateExecutionStatus(execID, engine.ExecCompleted, "")
	}
	writeJSON(w, map[string]string{"status": "completed"})
}

func (s *Server) handleCancelExecution(w http.ResponseWriter, _ *http.Request, execID string) {
	if err := s.execMgr.Cancel(execID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.store != nil {
		s.store.UpdateExecutionStatus(execID, engine.ExecCancelled, "")
	}
	writeJSON(w, map[string]string{"status": "cancelled"})
}

// buildPrompt constructs an initial prompt for a phase execution.
func buildPrompt(phase string, params map[string]string, eng *engine.Engine) string {
	switch phase {
	case "design":
		prd := params["prd"]
		if prd == "" && eng != nil {
			prd = eng.State.PrdURL
		}
		repoSummary := ""
		if eng != nil {
			repoSummary = buildRepoSummaryServer(eng.State.Repos)
		}
		notionURL := resolveNotionURLServer(prd, params)
		if notionURL != "" {
			notion := integrations.NewNotionClient()
			if notion.IsConfigured() {
				// Pre-fetch PRD content via Notion API
				prdContent, err := notion.ReadPage(notionURL)
				if err == nil {
					return prompts.SolutionDesigner(prdContent, repoSummary)
				}
				log.Printf("[exec] failed to pre-fetch PRD from Notion: %v, falling back to MCP tools", err)
			}
			// Notion URL but no API key (or fetch failed) → let Claude use MCP tools
			return prompts.SolutionDesignerFromNotion(notionURL, repoSummary)
		}
		return prompts.SolutionDesigner(prd, repoSummary)
	case "planning":
		doc := params["scoping_doc"]
		if doc == "" && eng != nil {
			doc = eng.State.Artifacts["scoping_doc"]
		}
		repoSummary := ""
		if eng != nil {
			repoSummary = buildRepoSummaryServer(eng.State.Repos)
		}
		// Check if the scoping doc is a Notion URL
		if isNotionURLServer(doc) {
			notion := integrations.NewNotionClient()
			if notion.IsConfigured() {
				content, err := notion.ReadPage(doc)
				if err == nil {
					return prompts.TaskDecomposer(content, repoSummary)
				}
				log.Printf("[exec] failed to pre-fetch scoping doc from Notion: %v, falling back to MCP tools", err)
			}
			return prompts.TaskDecomposerFromNotion(doc, repoSummary)
		}
		// Local file — read its contents for the prompt
		if content, err := os.ReadFile(doc); err == nil {
			return prompts.TaskDecomposer(string(content), repoSummary)
		}
		return prompts.TaskDecomposer(doc, repoSummary)
	case "tracking":
		pert := params["pert"]
		if pert == "" && eng != nil {
			pert = eng.State.Artifacts["pert"]
		}
		team := params["team"]
		return "Create Linear issues from the PERT document. PERT path: " + pert + " Team: " + team
	case "executing":
		issueParam := params["issue"]
		issueID, issueTitle := parseIssueParam(issueParam)
		log.Printf("[prompt] executing: issueParam=%q → issueID=%q title=%q", issueParam, issueID, issueTitle)

		// Determine language and LinearID from engine state
		language := ""
		linearID := ""
		if eng != nil {
			if iss, ok := eng.State.Issues[issueID]; ok {
				linearID = iss.LinearID
				log.Printf("[prompt] executing: found issue %s in state, linearID=%q", issueID, linearID)
				// Find repo language
				for _, r := range eng.State.Repos {
					if r.Name == iss.Repo {
						language = r.Language
						break
					}
				}
			}
			if language == "" && len(eng.State.Repos) > 0 {
				language = eng.State.Repos[0].Language
			}
		} else if eng != nil {
			log.Printf("[prompt] executing: issue %q NOT found in state (have %d issues, keys: %v)", issueID, len(eng.State.Issues), issueKeys(eng))
		}

		// Try to pre-fetch issue description from Linear API
		if linearID != "" {
			linear := integrations.NewLinearClient()
			if linear.IsConfigured() {
				details, err := linear.GetIssueByIdentifier(linearID)
				if err == nil && details.Description != "" {
					log.Printf("[exec] pre-fetched Linear issue %s description (%d bytes)", linearID, len(details.Description))
					return prompts.Coder(issueTitle, issueID, language, details.Description)
				}
				if err != nil {
					log.Printf("[exec] failed to pre-fetch Linear issue %s: %v, falling back to MCP", linearID, err)
				}
			}
			// API key missing or fetch failed — instruct Claude to fetch via MCP tools
			return prompts.CoderFromLinear(issueTitle, issueID, linearID, language)
		}

		// No LinearID available
		return prompts.Coder(issueTitle, issueID, language, "")
	case "bootstrap":
		repo := params["repo"]
		return "Bootstrap the repository by generating CLAUDE.md and ARCHITECTURE.md. Repo: " + repo
	default:
		return "Execute the " + phase + " phase."
	}
}

// toolsForPhase returns the AllowedTools list matching what the real phase runners use.
func toolsForPhase(phase string, params map[string]string, eng *engine.Engine) []string {
	base := []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}
	switch phase {
	case "design":
		prd := ""
		if params != nil {
			prd = params["prd"]
		}
		if prd == "" && eng != nil {
			prd = eng.State.PrdURL
		}
		notionURL := resolveNotionURLServer(prd, params)
		if notionURL != "" {
			notion := integrations.NewNotionClient()
			if !notion.IsConfigured() {
				// No API key — need MCP tools to read Notion
				return append(base, "mcp__plugin_Notion_notion__*")
			}
		}
		return base
	case "planning":
		doc := ""
		if params != nil {
			doc = params["scoping_doc"]
		}
		if doc == "" && eng != nil {
			doc = eng.State.Artifacts["scoping_doc"]
		}
		if isNotionURLServer(doc) {
			notion := integrations.NewNotionClient()
			if !notion.IsConfigured() {
				return append(base, "mcp__plugin_Notion_notion__*")
			}
		}
		return base
	case "tracking":
		// Tracking needs Linear MCP tools (create issues) and Notion MCP tools (read PERT)
		return append(base, "mcp__plugin_linear_linear__*", "mcp__plugin_Notion_notion__*")
	case "executing":
		// Always grant Linear MCP tools for executing phase (matches CLI behavior)
		return append(base, "mcp__plugin_linear_linear__*")
	default:
		return base
	}
}

// resolveCWD determines the working directory for an execution.
func resolveCWD(phase, issueID string, eng *engine.Engine) string {
	if eng == nil {
		return "."
	}
	// For executing with a specific issue, use the worktree path if available
	if phase == "executing" && issueID != "" {
		if iss, ok := eng.State.Issues[issueID]; ok && iss.Worktree != "" {
			return iss.Worktree
		}
	}
	// Default to first repo path
	if len(eng.State.Repos) > 0 && eng.State.Repos[0].Path != "" {
		return eng.State.Repos[0].Path
	}
	return "."
}

// isNotionURLServer returns true if the given string looks like a Notion URL.
func isNotionURLServer(s string) bool {
	return strings.Contains(s, "notion.so") || strings.Contains(s, "notion.site")
}

// resolveNotionURLServer checks params and prdURL for a Notion URL.
func resolveNotionURLServer(prdURL string, params map[string]string) string {
	if params != nil && params["prd"] != "" {
		if isNotionURLServer(params["prd"]) {
			return params["prd"]
		}
		return ""
	}
	if isNotionURLServer(prdURL) {
		return prdURL
	}
	return ""
}

// issueKeys returns the keys of eng.State.Issues for debugging.
func issueKeys(eng *engine.Engine) []string {
	keys := make([]string, 0, len(eng.State.Issues))
	for k := range eng.State.Issues {
		keys = append(keys, k)
	}
	return keys
}

// parseIssueParam splits "TASK-001: Some title" into (id, title).
// If no colon separator is found, the whole string is treated as the ID.
func parseIssueParam(s string) (id, title string) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, ": "); idx >= 0 {
		return s[:idx], s[idx+2:]
	}
	return s, s
}

// buildRepoSummaryServer builds a text summary of configured repositories.
func buildRepoSummaryServer(repos []engine.RepoConfig) string {
	summary := "Repositories:\n"
	for _, r := range repos {
		summary += "- " + r.Name + " (" + r.Language + ") at " + r.Path + ", team: " + r.Team + "\n"
	}
	return summary
}
