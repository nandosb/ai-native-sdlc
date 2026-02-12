---
name: sdlc-init
description: Configure the SDLC pipeline — PRD URL and repos → manifest.yaml
allowed-tools: Read, Write, Glob, Bash, AskUserQuestion
---

# SDLC Init — configure the pipeline

## Input → Output

```
Input:  user answers (interactive)
Output: manifest.yaml, .sdlc/state.json
```

## Step 0: Validate

1. If `manifest.yaml` exists, read it and show the current config.
2. Ask: "Update existing manifest or start fresh?"

## Step 1: Gather PRD

Ask for the **PRD source** (Notion URL or local file path).
- Notion URL → confirm it looks like a valid Notion URL.
- Local path → confirm the file exists on disk. If not, ask again.

## Step 2: Gather repos

For each repo ask:

| Field | Validation |
|-------|------------|
| `name` | Non-empty, no spaces |
| `path` | Directory must exist on disk |
| `team` | Non-empty |

After each: "Add another repo? (y/n)"

## Step 3: Write manifest.yaml

```yaml
prd: <value>
repos:
  - name: <value>
    path: <value>
    team: <value>
```

## Step 4: Initialize .sdlc/

Create `.sdlc/`, `.sdlc/artifacts/`. Write `.sdlc/state.json`:
```json
{
  "run_id": "",
  "prd": "<PRD from manifest>",
  "phase": "",
  "phase_status": "",
  "repos": [],
  "artifacts": {},
  "issues": [],
  "updated_at": "<ISO timestamp>"
}
```

## Step 5: Confirm

Show saved manifest. Tell user: "Next → `/sdlc-bootstrap` or `/sdlc`"
