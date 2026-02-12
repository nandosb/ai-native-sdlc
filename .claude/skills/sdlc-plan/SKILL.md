---
name: sdlc-plan
description: Decompose scoping document into a PERT task graph with dependencies
agent: task-decomposer
context: fork
---

# SDLC Plan — scoping document to PERT

## Input → Output

```
Input:  .sdlc/artifacts/scoping-doc.md + .sdlc/state.json (repos)
Output: .sdlc/artifacts/pert.md, updated .sdlc/state.json
Agent:  task-decomposer
```

## Step 0: Validate

1. Read `.sdlc/state.json`. Missing → **STOP**: "Run `/sdlc-bootstrap` first."
2. Check `state.artifacts.scoping_doc`. Empty → **STOP**: "Run `/sdlc-design` first."
3. Read the file at that path. Missing → **STOP**: "Scoping doc file missing. Run `/sdlc-design`."
4. Check `state.repos`. Empty → **STOP**: "Run `/sdlc-bootstrap` first."

All four checks must pass.

## Step 1: Read inputs

- Read the scoping document.
- Read the repos list from state (name, path, team, language).

## Step 2: Generate PERT

Using the task-decomposer agent's persona, approach, and output format — produce the PERT (markdown + JSON block) from the scoping document and repo info.

## Step 3: Save

Write the full PERT (markdown + JSON) to `.sdlc/artifacts/pert.md`.

## Step 4: Update state

- `phase`: `"planning"`, `phase_status`: `"completed"`
- `artifacts.pert`: `".sdlc/artifacts/pert.md"`
- `updated_at`: ISO timestamp
- Preserve all other fields.
