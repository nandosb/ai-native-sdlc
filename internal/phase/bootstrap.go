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

// Bootstrap generates CLAUDE.md and ARCHITECTURE.md per repo.
type Bootstrap struct{}

func (b *Bootstrap) Name() engine.Phase { return engine.PhaseBootstrap }

func (b *Bootstrap) Run(eng *engine.Engine, params map[string]string) error {
	targetRepo := ""
	if params != nil {
		targetRepo = params["repo"]
	}

	for _, repo := range eng.State.Repos {
		if targetRepo != "" && repo.Name != targetRepo {
			continue
		}

		fmt.Printf("Bootstrapping %s...\n", repo.Name)

		absPath, err := filepath.Abs(repo.Path)
		if err != nil {
			return fmt.Errorf("resolve path for %s: %w", repo.Name, err)
		}

		repoState := eng.State.Bootstrap[repo.Name]

		// Detect language if not specified
		lang := repo.Language
		if lang == "" {
			lang = detectLanguage(absPath)
		}

		// Generate CLAUDE.md if not present
		if !repoState.ClaudeMD {
			claudePath := filepath.Join(absPath, "CLAUDE.md")
			if _, err := os.Stat(claudePath); os.IsNotExist(err) {
				fmt.Printf("  Generating CLAUDE.md for %s (%s)...\n", repo.Name, lang)
				prompt := prompts.DocGenerator(repo.Name, lang, "CLAUDE.md")

				eng.Events.Publish(engine.Event{
					Type: engine.EventAgentSpawned,
					Data: map[string]string{"agent": "doc-generator", "repo": repo.Name, "target": "CLAUDE.md"},
				})

				start := time.Now()
				result, err := claude.Run(context.Background(), claude.RunConfig{
					Prompt:       prompt,
					CWD:          absPath,
					Model:        "sonnet",
					AllowedTools: []string{"Read", "Write", "Glob", "Grep", "Bash"},
				}, eng.Events, "")
				elapsed := time.Since(start)

				if err != nil {
					return fmt.Errorf("generate CLAUDE.md for %s: %w", repo.Name, err)
				}

				eng.Metrics.Record(engine.MetricsEntry{
					Timestamp: time.Now(),
					Agent:     "doc-generator",
					Model:     "sonnet",
					TokensIn:  result.TokensIn,
					TokensOut: result.TokensOut,
					Duration:  elapsed.Milliseconds(),
					Phase:     engine.PhaseBootstrap,
				})

				eng.Events.Publish(engine.Event{
					Type: engine.EventAgentCompleted,
					Data: map[string]string{"agent": "doc-generator", "repo": repo.Name, "target": "CLAUDE.md"},
				})
			}
			repoState.ClaudeMD = true
		}

		// Generate ARCHITECTURE.md if not present
		if !repoState.ArchitectureMD {
			archPath := filepath.Join(absPath, "ARCHITECTURE.md")
			if _, err := os.Stat(archPath); os.IsNotExist(err) {
				fmt.Printf("  Generating ARCHITECTURE.md for %s...\n", repo.Name)
				prompt := prompts.DocGenerator(repo.Name, lang, "ARCHITECTURE.md")

				start := time.Now()
				result, err := claude.Run(context.Background(), claude.RunConfig{
					Prompt:       prompt,
					CWD:          absPath,
					Model:        "sonnet",
					AllowedTools: []string{"Read", "Write", "Glob", "Grep", "Bash"},
				}, eng.Events, "")
				elapsed := time.Since(start)

				if err != nil {
					return fmt.Errorf("generate ARCHITECTURE.md for %s: %w", repo.Name, err)
				}

				eng.Metrics.Record(engine.MetricsEntry{
					Timestamp: time.Now(),
					Agent:     "doc-generator",
					Model:     "sonnet",
					TokensIn:  result.TokensIn,
					TokensOut: result.TokensOut,
					Duration:  elapsed.Milliseconds(),
					Phase:     engine.PhaseBootstrap,
				})
			}
			repoState.ArchitectureMD = true
		}

		eng.SaveBootstrapState(repo.Name, repoState)
	}

	return nil
}

// detectLanguage checks for common project files to determine the language.
func detectLanguage(repoPath string) string {
	checks := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"package.json", "typescript"},
		{"pyproject.toml", "python"},
		{"Cargo.toml", "rust"},
		{"pom.xml", "java"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(repoPath, c.file)); err == nil {
			return c.lang
		}
	}
	return "unknown"
}
