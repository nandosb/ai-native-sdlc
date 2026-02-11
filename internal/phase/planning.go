package phase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/claude"
	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/prompts"
)

// Planning generates a PERT from a scoping document.
type Planning struct{}

func (p *Planning) Name() engine.Phase { return engine.PhasePlanning }

func (p *Planning) Run(eng *engine.Engine, params map[string]string) error {
	// Load scoping document
	scopingDocPath := eng.State.Artifacts[engine.ArtifactScopingDoc]
	if params != nil && params[engine.ArtifactScopingDoc] != "" {
		scopingDocPath = params[engine.ArtifactScopingDoc]
	}
	if scopingDocPath == "" {
		return fmt.Errorf("no scoping document available (run design phase first)")
	}

	scopingContent, err := os.ReadFile(scopingDocPath)
	if err != nil {
		return fmt.Errorf("read scoping doc: %w", err)
	}

	repoSummary := buildRepoSummary(eng.State.Repos)
	prompt := prompts.TaskDecomposer(string(scopingContent), repoSummary)

	fmt.Println("Running task decomposer...")
	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentSpawned,
		Data: map[string]string{"agent": "task-decomposer"},
	})

	start := time.Now()
	result, err := claude.Run(context.Background(), claude.RunConfig{
		Prompt:       prompt,
		Model:        "opus",
		AllowedTools: []string{"Read"},
	}, eng.Events, "")
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("task decomposer failed: %w", err)
	}

	eng.Metrics.Record(engine.MetricsEntry{
		Timestamp: time.Now(),
		Agent:     "task-decomposer",
		Model:     "opus",
		TokensIn:  result.TokensIn,
		TokensOut: result.TokensOut,
		Duration:  elapsed.Milliseconds(),
		Phase:     engine.PhasePlanning,
	})

	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentCompleted,
		Data: map[string]string{"agent": "task-decomposer"},
	})

	// Save PERT document
	output := ""
	if params != nil && params["output"] != "" {
		output = params["output"]
	}
	if output == "" {
		if err := ensureArtifactsDir(); err != nil {
			return fmt.Errorf("create artifacts dir: %w", err)
		}
		output = filepath.Join(artifactsDir, eng.State.RunID+"-pert.md")
	}
	if err := os.WriteFile(output, []byte(result.Output), 0644); err != nil {
		return fmt.Errorf("write PERT: %w", err)
	}
	eng.SaveArtifact(engine.ArtifactPERT, output)
	fmt.Printf("PERT document written to %s\n", output)

	return nil
}
