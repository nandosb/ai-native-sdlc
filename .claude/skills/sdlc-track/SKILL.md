---
name: sdlc-track
description: Create Linear issues from PERT task list with blocking relationships
agent: linear-issue-creator
context: fork
---

# SDLC Track — PERT to Linear issues

## Input → Output

```
Input:  .sdlc/artifacts/pert.md (JSON task block) + .sdlc/state.json (repos)
Output: Linear issues with blocking relationships, updated .sdlc/state.json
Agent:  linear-issue-creator
```

## Step 0: Validate

1. Read `.sdlc/state.json`. Missing → **STOP**: "Run `/sdlc-bootstrap` first."
2. Check `state.artifacts.pert`. Empty → **STOP**: "Run `/sdlc-plan` first."
3. Read the file at that path. Missing → **STOP**: "PERT file missing. Run `/sdlc-plan`."
4. Parse the fenced JSON block at the end. No JSON or empty tasks → **STOP**: "PERT has no JSON task block. Run `/sdlc-plan`."
5. Check `state.repos`. Empty → **STOP**: "Run `/sdlc-bootstrap` first."
6. **Integration check**: Call `list_teams` via Linear MCP. Fails → **STOP**: "Linear MCP not available. Add the Linear MCP server to Claude Code settings."

All six checks must pass.

## Step 1: Read inputs

- Parse the tasks JSON array from the PERT.
- Read repos from state for team info.

## Step 2: Create issues

Using the linear-issue-creator agent's approach — validate graph, create issues in topological order, wire blocking relationships.

## Step 3: Update state

- `phase`: `"tracking"`, `phase_status`: `"completed"`
- `issues`: array with `{ id, title, repo, status, linear_id, branch, pr_url, depends_on, iterations }`
  - Status: `"ready"` if no deps, `"blocked"` otherwise
- `updated_at`: ISO timestamp
- Preserve all other fields.

## Step 4: Report

```
Track complete — N issues created:
  repo#1 → TEAM-101  (ready)
  repo#2 → TEAM-102  (blocked by #1)
```
