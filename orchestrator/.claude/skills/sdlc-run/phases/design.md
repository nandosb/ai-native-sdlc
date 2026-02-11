# Phase: DESIGN

This phase produces a Scoping Document — the technical design for the feature described in the PRD.

## Objective

Transform the PRD into an actionable technical design that maps features to repositories with specific implementation details.

## Steps

### 1. Read the PRD

Read the PRD content. Two modes:

**Mode A — Notion MCP available:**
Use the Notion MCP to read the PRD page at the URL from `manifest.yaml`:
```
mcp_notion.read_page(url: state.prd_url)
```

**Mode B — Notion MCP not available (fallback):**
Ask the user to paste the PRD content directly. Store it as a local file `artifacts/prd.md`.

### 2. Prepare repo context

For each repo, read its `CLAUDE.md` and create a brief summary (under 500 words per repo). This reduces context sent to the solution-designer.

```
Read({repo_path}/CLAUDE.md)
```

Build the context object:
```json
{
  "repos": [
    { "name": "api-gateway", "language": "go", "claude_md_summary": "..." },
    { "name": "shared-events", "language": "go", "claude_md_summary": "..." }
  ]
}
```

### 3. Spawn solution-designer

```
Task(
  subagent_type: "solution-designer",
  prompt: "Design the technical solution.\n\nPRD:\n{prd_text}\n\nRepositories:\n{repos_context}\n\nRun ID: {run_id}",
  description: "Design solution for {run_id}",
  model: "opus"
)
```

The agent returns the Scoping Document as markdown.

### 4. Store the Scoping Document

**Mode A — Notion MCP available:**
Create a child page under the PRD in Notion:
```
mcp_notion.create_page(
  parent: state.prd_url,
  title: "1. Scoping Document",
  content: scoping_doc_markdown
)
```
Store the URL in `state.json` under `artifacts.scoping_doc`.

**Mode B — Fallback:**
Write to `artifacts/scoping-doc.md` locally.

### 5. Update state and pause

```json
{
  "phase": "DESIGN",
  "phase_status": "awaiting_approval",
  "artifacts": { "scoping_doc": "https://notion.so/..." }
}
```

### 6. Display to user

```
DESIGN phase complete.

Scoping Document: {url or file path}

Review the design and run /sdlc-approve to continue to PLANNING.
```

## Transition

DESIGN requires an **approval gate**. The user must run `/sdlc-approve` to advance to PLANNING.

## Error Handling

- If Notion MCP is unavailable, fall back to local file storage
- If solution-designer fails, save partial output and report to user
- If PRD is empty or unreadable, ask user for clarification

## Idempotency

- If `artifacts.scoping_doc` already has a URL → overwrite the Notion page (or local file) instead of creating a duplicate
- On resume with `phase_status: awaiting_approval` → just show the link and wait for approve
