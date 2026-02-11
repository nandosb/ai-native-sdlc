# Phase: PLANNING

This phase produces a PERT — an ordered list of atomic tasks with dependencies, ready for tracking in Linear.

## Objective

Decompose the Scoping Document into implementable tasks that can be assigned to coder agents, with clear dependencies for topological execution.

## Steps

### 1. Read the Scoping Document

**Mode A — Notion:**
```
mcp_notion.read_page(url: state.artifacts.scoping_doc)
```

**Mode B — Local fallback:**
```
Read("artifacts/scoping-doc.md")
```

### 2. Prepare repo metadata

Build the repos array for the task-decomposer:
```json
{
  "repos": [
    { "name": "api-gateway", "language": "go", "team": "Backend" },
    { "name": "shared-events", "language": "go", "team": "Platform" }
  ]
}
```

### 3. Spawn task-decomposer

```
Task(
  subagent_type: "task-decomposer",
  prompt: "Decompose this design into tasks.\n\nScoping Document:\n{scoping_doc_text}\n\nRepositories:\n{repos_json}\n\nRun ID: {run_id}",
  description: "Decompose tasks for {run_id}",
  model: "opus"
)
```

The agent returns:
- Markdown PERT document
- JSON block with structured task data

### 4. Parse the JSON task block

Extract the JSON block from the agent's response. It should be fenced with ````json` ... `````.

Parse to get the `tasks` array:
```json
{
  "tasks": [
    {
      "id": "shared-events#1",
      "repo": "shared-events",
      "team": "Platform",
      "title": "Define WebhookTriggered event",
      "description": "...",
      "blocked_by": [],
      "size": "S"
    }
  ]
}
```

**Important**: If the JSON block is missing or malformed, ask the task-decomposer to regenerate with proper format.

### 5. Store the PERT

**Mode A — Notion:**
Create a child page under the PRD:
```
mcp_notion.create_page(
  parent: state.prd_url,
  title: "2. PERT",
  content: pert_markdown
)
```

**Mode B — Local fallback:**
Write to `artifacts/pert.md` and `artifacts/pert-tasks.json`.

### 6. Update state and pause

```json
{
  "phase": "PLANNING",
  "phase_status": "awaiting_approval",
  "artifacts": {
    "scoping_doc": "...",
    "pert": "https://notion.so/..."
  }
}
```

Store the parsed tasks temporarily (in state or local file) for the TRACKING phase.

### 7. Display to user

```
PLANNING phase complete.

PERT: {url or file path}
Tasks: {N} tasks across {M} repositories

Task breakdown:
  {repo-a}: {count} tasks
  {repo-b}: {count} tasks

Review the PERT and run /sdlc-approve to continue to TRACKING.
```

## Transition

PLANNING requires an **approval gate**. The user must run `/sdlc-approve` to advance to TRACKING.

## Error Handling

- If JSON parsing fails, attempt to extract tasks from the markdown structure
- If task-decomposer produces circular dependencies, flag to user
- If Notion is unavailable, use local files

## Idempotency

- If `artifacts.pert` already exists → overwrite
- On resume with `phase_status: awaiting_approval` → show link and wait
