package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// promptsDir is the directory containing prompt template files.
var promptsDir = "prompts"

// SetPromptsDir overrides the default prompts directory.
func SetPromptsDir(dir string) {
	promptsDir = dir
}

// DocGenerator returns the prompt for generating documentation files.
func DocGenerator(repoName, language, targetFile string) string {
	tmpl := loadTemplate("doc-generator.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"repo_name":   repoName,
			"language":    language,
			"target_file": targetFile,
		})
	}

	return fmt.Sprintf(`You are a documentation generator for the %s repository (language: %s).

Your task is to explore this codebase and generate a comprehensive %s file.

Instructions:
1. Read the project structure (use Glob to find key files)
2. Understand the architecture, key patterns, and conventions
3. Write a clear, concise %s that helps developers understand and work with this codebase

For CLAUDE.md: Focus on conventions, coding patterns, important file paths, build/test commands, and anything an AI assistant needs to know to work effectively in this repo.

For ARCHITECTURE.md: Focus on system design, component relationships, data flow, key abstractions, and how the pieces fit together.

Write the file directly using the Write tool.`, repoName, language, targetFile, targetFile)
}

// SolutionDesigner returns the prompt for generating a scoping document.
func SolutionDesigner(prdContent, repoSummary string) string {
	tmpl := loadTemplate("solution-designer.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"prd_content":  prdContent,
			"repo_summary": repoSummary,
		})
	}

	return fmt.Sprintf(`You are a senior solution architect. Your task is to create a detailed scoping document from the following PRD.

## PRD Content
%s

## Available Repositories
%s

## Instructions

Create a comprehensive scoping document that includes:
1. **Summary** — What we're building and why
2. **Technical Approach** — How each repo will be modified
3. **API Changes** — New or modified endpoints/interfaces
4. **Data Model Changes** — Schema modifications
5. **Integration Points** — How components interact
6. **Risk Assessment** — Technical risks and mitigations
7. **Out of Scope** — What this does NOT include

Output the scoping document in markdown format.`, prdContent, repoSummary)
}

// TaskDecomposer returns the prompt for generating a PERT from a scoping doc.
func TaskDecomposer(scopingContent, repoSummary string) string {
	tmpl := loadTemplate("task-decomposer.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"scoping_content": scopingContent,
			"repo_summary":    repoSummary,
		})
	}

	return fmt.Sprintf(`You are a technical project planner. Decompose the following scoping document into implementable tasks with dependencies.

## Scoping Document
%s

## Repositories
%s

## Instructions

Create a PERT (task dependency graph) as a JSON array with this structure:

` + "```json" + `
[
  {
    "id": "TASK-001",
    "title": "Short task title",
    "description": "Detailed description of what to implement",
    "repo": "repo-name",
    "depends_on": [],
    "estimate": "S/M/L"
  }
]
` + "```" + `

Rules:
- Each task should be implementable in a single PR
- Tasks should be small enough for one developer session
- Dependencies must form a DAG (no cycles)
- Include test tasks for each feature task
- Use "S" (< 1 hour), "M" (1-3 hours), "L" (3-8 hours) estimates`, scopingContent, repoSummary)
}

// TaskDecomposerFromNotion returns a prompt that instructs Claude to first
// fetch a scoping document from Notion via MCP tools, then decompose it.
func TaskDecomposerFromNotion(notionURL, repoSummary string) string {
	preamble := fmt.Sprintf(`IMPORTANT: The scoping document is stored in Notion. Before doing anything else, use your Notion MCP tools to read the full content of this page:

%s

Read the page and all its sub-pages/blocks to get the complete scoping document content. Then proceed with the instructions below using that content as the scoping document.

`, notionURL)

	base := TaskDecomposer("[Scoping document content fetched from Notion — see instructions above]", repoSummary)
	return preamble + base
}

// Coder returns the prompt for implementing an issue.
func Coder(issueTitle, issueID, language, description string) string {
	tmpl := loadTemplate("coder.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"issue_title": issueTitle,
			"issue_id":    issueID,
			"language":    language,
			"description": description,
		})
	}

	descBlock := ""
	if description != "" {
		descBlock = "\n## Description\n" + description + "\n"
	}

	return fmt.Sprintf(`You are an expert %s developer. Implement the following task:

## Task: %s (ID: %s)
%s
## Instructions

1. Read the existing CLAUDE.md and ARCHITECTURE.md to understand project conventions
2. Implement the required changes
3. Write tests for your changes
4. Run existing tests to ensure nothing breaks
5. Commit your changes with a clear message referencing %s

Follow existing code patterns and conventions. Write clean, well-tested code.
Do not modify files outside the scope of this task.`, language, issueTitle, issueID, descBlock, issueID)
}

// CoderFromLinear returns a prompt that instructs Claude to first fetch the
// issue description from Linear via MCP tools, then proceed with implementation.
func CoderFromLinear(issueTitle, issueID, linearID, language string) string {
	preamble := fmt.Sprintf(`IMPORTANT: The full issue details are stored in Linear. Before doing anything else, use your Linear MCP tools to fetch the complete description of issue %s.

Read the issue description and acceptance criteria, then proceed with the instructions below using that context.

`, linearID)

	base := Coder(issueTitle, issueID, language, "[Issue description fetched from Linear — see instructions above]")
	return preamble + base
}

// QualityReviewer returns the prompt for reviewing code changes.
func QualityReviewer(issueTitle, language string) string {
	tmpl := loadTemplate("quality-reviewer.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"issue_title": issueTitle,
			"language":    language,
		})
	}

	return fmt.Sprintf(`You are a senior %s code reviewer. Review the changes made for: %s

## Instructions

1. Run the test suite and verify all tests pass
2. Review code changes for:
   - Correctness — Does the implementation match the task requirements?
   - Tests — Are there adequate tests? Do they cover edge cases?
   - Style — Does the code follow project conventions?
   - Security — Any potential vulnerabilities?
   - Performance — Any obvious performance issues?
3. Check for common issues:
   - Unused imports or variables
   - Missing error handling
   - Hardcoded values that should be configurable

## Output Format

If the code looks good, output: "APPROVED: [brief reason]"
If changes are needed, output: "CHANGES REQUESTED:" followed by specific, actionable feedback.`, language, issueTitle)
}

// FeedbackWriter returns the prompt for applying review feedback.
func FeedbackWriter(reviewFeedback string) string {
	tmpl := loadTemplate("feedback-writer.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"review_feedback": reviewFeedback,
		})
	}

	return fmt.Sprintf(`You are a developer applying code review feedback. The reviewer has requested the following changes:

## Review Feedback
%s

## Instructions

Apply each piece of feedback. After making changes, run the tests to verify everything passes.
Commit the fixes with a message like "address review feedback".`, reviewFeedback)
}

func loadTemplate(name string) string {
	path := filepath.Join(promptsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// SolutionDesignerFromNotion wraps SolutionDesigner with a preamble instructing
// Claude to fetch the PRD from a Notion page using MCP tools before proceeding.
func SolutionDesignerFromNotion(notionURL, repoSummary string) string {
	preamble := fmt.Sprintf(`IMPORTANT: The PRD is stored in Notion. Before doing anything else, use your Notion MCP tools to read the full content of this page:

%s

Read the page and all its sub-pages/blocks to get the complete PRD content. Then proceed with the instructions below using that content as the PRD.

`, notionURL)

	// Build the standard prompt with a placeholder PRD reference
	base := SolutionDesigner("[PRD content fetched from Notion — see instructions above]", repoSummary)
	return preamble + base
}

// LinearIssueCreator returns a prompt instructing Claude to create Linear issues
// via MCP tools from a JSON task list.
func LinearIssueCreator(tasksJSON, team string) string {
	tmpl := loadTemplate("linear-issue-creator.md")
	if tmpl != "" {
		return interpolate(tmpl, map[string]string{
			"tasks_json": tasksJSON,
			"team":       team,
		})
	}

	// Inline fallback
	return fmt.Sprintf(`You are a project tracking assistant. Create Linear issues from the following task list.

## Team
%s

## Tasks (JSON)
`+"```json"+`
%s
`+"```"+`

## Instructions

1. For each task in the JSON array, create a Linear issue on the "%s" team with:
   - Title: the task's "title"
   - Description: the task's "description"
   - Estimate (if available): map "S"→1, "M"→2, "L"→5

2. For tasks that have "depends_on" entries, set the blocking relationships:
   - The depended-on issue should block the dependent issue

3. After creating all issues, output ONLY a JSON object mapping task IDs to Linear issue identifiers. Example:
`+"```json"+`
{"TASK-001": "TEAM-123", "TASK-002": "TEAM-124"}
`+"```"+`

Do not output anything else besides the JSON mapping.`, team, tasksJSON, team)
}

func interpolate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
