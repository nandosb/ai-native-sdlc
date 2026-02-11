package phase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/claude"
	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/integrations"
	"github.com/yalochat/agentic-sdlc/internal/prompts"
)

// Design generates a scoping document from a PRD.
type Design struct{}

func (d *Design) Name() engine.Phase { return engine.PhaseDesign }

func (d *Design) Run(eng *engine.Engine, params map[string]string) error {
	// Build repo context summary
	repoSummary := buildRepoSummary(eng.State.Repos)

	// Resolve the PRD source and build the appropriate prompt + tools
	var prompt string
	allowedTools := []string{"Read"}

	notionURL := resolveNotionURL(eng.State.PrdURL, params)
	notion := integrations.NewNotionClient()

	if notionURL != "" && notion.IsConfigured() {
		// Notion URL + API key → pre-fetch PRD content via Notion API
		fmt.Println("Pre-fetching PRD via Notion API...")
		prdContent, err := notion.ReadPage(notionURL)
		if err != nil {
			return fmt.Errorf("fetch PRD from Notion: %w", err)
		}
		prompt = prompts.SolutionDesigner(prdContent, repoSummary)
	} else if notionURL != "" {
		// Notion URL but no API key → fall back to MCP tools
		if params != nil && params["_non_interactive"] == "true" {
			return fmt.Errorf(
				"PRD is a Notion URL but NOTION_API_KEY is not set. " +
					"Configure NOTION_API_KEY in your environment to read Notion pages from the web UI")
		}
		fmt.Println("PRD is a Notion URL, will use Notion MCP tools (NOTION_API_KEY not set)")
		prompt = prompts.SolutionDesignerFromNotion(notionURL, repoSummary)
		allowedTools = append(allowedTools, "mcp__plugin_Notion_notion__*")
	} else {
		// Local file or generic URL
		prdContent, err := loadDocument(eng.State.PrdURL, params)
		if err != nil {
			return fmt.Errorf("load PRD: %w", err)
		}
		prompt = prompts.SolutionDesigner(prdContent, repoSummary)
	}

	fmt.Println("Running solution designer...")
	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentSpawned,
		Data: map[string]string{"agent": "solution-designer"},
	})

	start := time.Now()
	result, err := claude.Run(context.Background(), claude.RunConfig{
		Prompt:       prompt,
		Model:        "opus",
		AllowedTools: allowedTools,
	}, eng.Events, "")
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("solution designer failed: %w", err)
	}

	eng.Metrics.Record(engine.MetricsEntry{
		Timestamp: time.Now(),
		Agent:     "solution-designer",
		Model:     "opus",
		TokensIn:  result.TokensIn,
		TokensOut: result.TokensOut,
		Duration:  elapsed.Milliseconds(),
		Phase:     engine.PhaseDesign,
	})

	eng.Events.Publish(engine.Event{
		Type: engine.EventAgentCompleted,
		Data: map[string]string{"agent": "solution-designer"},
	})

	// Save scoping document
	output := params["output"]
	if output == "" {
		if err := ensureArtifactsDir(); err != nil {
			return fmt.Errorf("create artifacts dir: %w", err)
		}
		output = filepath.Join(artifactsDir, eng.State.RunID+"-scoping-doc.md")
	}
	if err := os.WriteFile(output, []byte(result.Output), 0644); err != nil {
		return fmt.Errorf("write scoping doc: %w", err)
	}
	eng.SaveArtifact(engine.ArtifactScopingDoc, output)
	fmt.Printf("Scoping document written to %s\n", output)

	// Notion writeback — non-fatal
	if notionURL != "" && notion.IsConfigured() {
		notionTitle := engine.Artifacts[engine.ArtifactScopingDoc].NotionTitle
		pageURL, err := notion.CreatePage(notionURL, notionTitle, result.Output)
		if err != nil {
			fmt.Printf("Warning: failed to write scoping doc to Notion: %v\n", err)
		} else {
			eng.SaveArtifact(engine.ArtifactScopingDocNotion, pageURL)
			fmt.Printf("Scoping document published to Notion: %s\n", pageURL)
		}
	}

	return nil
}

// isNotionURL returns true if the given string looks like a Notion URL.
func isNotionURL(s string) bool {
	return strings.Contains(s, "notion.so") || strings.Contains(s, "notion.site")
}

// resolveNotionURL checks params and engine state for a Notion URL.
// Returns the URL if found, empty string otherwise.
func resolveNotionURL(prdURL string, params map[string]string) string {
	if params != nil && params["prd"] != "" {
		if isNotionURL(params["prd"]) {
			return params["prd"]
		}
		return ""
	}
	if isNotionURL(prdURL) {
		return prdURL
	}
	return ""
}

func loadDocument(url string, params map[string]string) (string, error) {
	// Check params for explicit path override
	if params != nil {
		if prd := params["prd"]; prd != "" {
			url = prd
		}
	}

	if url == "" {
		return "", fmt.Errorf("no PRD URL or path specified")
	}

	// If it's a local file, read it
	if data, err := os.ReadFile(url); err == nil {
		return string(data), nil
	}

	// Non-Notion URLs: return the URL as-is for the prompt
	return fmt.Sprintf("PRD URL: %s\n(Content should be fetched from the URL above)", url), nil
}

func buildRepoSummary(repos []engine.RepoConfig) string {
	summary := "Repositories:\n"
	for _, r := range repos {
		summary += fmt.Sprintf("- %s (%s) at %s, team: %s\n", r.Name, r.Language, r.Path, r.Team)
	}
	return summary
}
