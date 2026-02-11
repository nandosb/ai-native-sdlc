---
name: sdlc-run
description: Starts the Agentic SDLC flow from a manifest.yaml
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, Task
argument-hint: "[manifest.yaml]"
---

# /sdlc-run

Orchestrate the full software development lifecycle: from PRD to merged PRs with tests.

You are the **orchestrator**. You manage state, spawn agents for reasoning-heavy tasks, and execute deterministic operations directly. Agents NEVER nest — you spawn all of them.

## Initialization

### 1. Load manifest

Read `manifest.yaml` (or the path provided as argument) from the project root:

```yaml
prd: https://notion.so/...
repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend
    language: go         # optional
    skills: []           # optional
```

Validate:
- `prd` field exists and is a valid URL (or note for fallback)
- `repos` is a non-empty array
- Each repo has `name`, `path`, and `team`
- Each repo `path` resolves to an existing directory (check with Glob/Bash)

If validation fails, report specific errors and stop.

### 2. Check for existing state

Read `state.json`. If it exists and has a `run_id`:
```
An existing run was found: {run_id} (phase: {phase})
Use /sdlc-resume to continue this run, or delete state.json to start fresh.
```
Stop and let the user decide.

### 3. Initialize state.json

Generate `run_id` from today's date and a slug derived from the PRD URL or manifest filename:

```json
{
  "run_id": "YYYY-MM-DD-{feature-slug}",
  "prd_url": "{from manifest}",
  "phase": "BOOTSTRAP",
  "phase_status": "in_progress",
  "bootstrap": { "languages": {}, "docs_generated": {} },
  "artifacts": { "scoping_doc": null, "pert": null },
  "issues": {},
  "updated_at": "{now ISO-8601}",
  "metrics": {
    "started_at": "{now ISO-8601}",
    "phases": {
      "BOOTSTRAP": { "started_at": "{now}", "ended_at": null, "duration_s": null }
    },
    "agent_invocations": [],
    "totals": {
      "duration_s": 0,
      "agent_spawns": 0,
      "review_iterations": { "agent": 0, "human": 0 },
      "issues_created": 0,
      "prs_created": 0,
      "prs_merged": 0
    }
  }
}
```

Write this to `state.json`.

### 4. Display start message

```
Starting Agentic SDLC run: {run_id}
PRD: {prd_url}
Repos: {list of repo names with languages if known}

Phase 1/6: BOOTSTRAP
```

## Phase Router

The run progresses through phases sequentially. Each phase has detailed instructions in a supporting file.

| Phase | File | Approval Gate | Description |
|---|---|---|---|
| BOOTSTRAP | `phases/bootstrap.md` | No | Detect languages, generate missing CLAUDE.md |
| DESIGN | `phases/design.md` | **Yes** | Read PRD, spawn solution-designer, produce Scoping Doc |
| PLANNING | `phases/planning.md` | **Yes** | Spawn task-decomposer, produce PERT |
| TRACKING | `phases/tracking.md` | No | Create Linear issues with dependencies |
| EXECUTING | `phases/executing.md` | No (per-issue human gates) | Implement, review, iterate per issue |
| COMPLETED | — | — | All done |

### How to execute a phase:

1. Read the phase file: `Read("phases/{phase_name}.md")`
2. Follow its instructions step by step
3. Update `state.json` after each significant action (write-ahead)
4. When the phase completes:
   - If it has an approval gate → set `phase_status: "awaiting_approval"`, display message, STOP
   - If no gate → advance to next phase automatically

### Phase transitions:

```
BOOTSTRAP (auto) → DESIGN (gate) → /sdlc-approve → PLANNING (gate) → /sdlc-approve → TRACKING (auto) → EXECUTING (per-issue) → COMPLETED
```

## State Management Rules

1. **Write-ahead**: Always update `state.json` BEFORE performing an action
2. **Refresh `updated_at`**: On every write to state.json
3. **Record metrics**: Update phase timings and totals after each operation
4. **Agent tracking**: After each Task spawn, record in `metrics.agent_invocations`:
   ```json
   {
     "agent": "{agent_name}",
     "model": "{model}",
     "repo": "{repo or null}",
     "issue": "{issue_id or null}",
     "started_at": "{before spawn}",
     "duration_s": "{calculated after}",
     "tokens": { "input": 0, "output": 0 }
   }
   ```
   Token counts come from `metrics.jsonl` (populated by the hook). Set to 0 initially.

## Agent Spawning Rules

When spawning agents via Task:

1. **Use the correct model**: Opus for solution-designer, task-decomposer, quality-reviewer. Sonnet for all others.
2. **Include full context**: Each agent gets everything it needs in the prompt — they have no access to state.json or manifest.yaml.
3. **Parse structured output**: Agents return JSON. Parse it to update state.
4. **Handle failures**: If an agent returns an error or malformed output, log the error in state and either retry once or escalate to the user.
5. **Increment counters**: `metrics.totals.agent_spawns++` after each spawn.

## Language Detection

Do this directly — no agent needed:

```
For each repo in manifest:
  if manifest has language field → use it
  elif Glob("{repo_path}/go.mod") matches → "go"
  elif Glob("{repo_path}/package.json") matches → "typescript"
  elif Glob("{repo_path}/pyproject.toml") or Glob("{repo_path}/setup.py") matches → "python"
  else → ask user
```

## MCP Operations

The orchestrator calls MCP servers directly for deterministic operations:

- **Notion MCP**: Read PRD, create Scoping Doc page, create PERT page
- **Linear MCP**: Create issues, set blockers, update status

Agents do NOT call MCP servers (except `session-resumer` which has Linear in foreground).

If an MCP server is unavailable, fall back:
- **Notion unavailable**: Ask user to paste PRD text. Store artifacts as local files in `artifacts/`.
- **Linear unavailable**: Report error. TRACKING and EXECUTING phases require Linear.

## Idempotency

All operations must be idempotent (safe to re-run):

| Operation | How |
|---|---|
| Generate CLAUDE.md | doc-generator checks existence first |
| Create Scoping Doc in Notion | Search by title under PRD, overwrite if exists |
| Create Linear issues | Search by title in team before creating |
| Create git branch | Check with `git branch --list` |
| Create PR | Check with `gh pr list --head {branch}` |

## Error Recovery

- On any error, write current state to `state.json` before stopping
- Include error details in the state for debugging
- The user can always run `/sdlc-resume` to pick up from the last good state
- The user can always run `/sdlc-status` to see where things stand

## State Schema Reference

See `templates/state-schema.md` for the full state.json schema with field descriptions and valid transitions.
