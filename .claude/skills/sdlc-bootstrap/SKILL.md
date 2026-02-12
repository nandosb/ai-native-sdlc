---
name: sdlc-bootstrap
description: Generate CLAUDE.md + ARCHITECTURE.md for each repo in manifest.yaml
agent: doc-generator
context: fork
---

# SDLC Bootstrap — repo orientation docs

## Input → Output

```
Input:  manifest.yaml
Output: CLAUDE.md + docs/ARCHITECTURE.md per repo, updated .sdlc/state.json
Agent:  doc-generator
```

## Step 0: Validate

1. Read `manifest.yaml`. Missing → **STOP**: "Run `/sdlc-init` first."
2. Parse `repos`. Empty → **STOP**: "No repos in manifest. Run `/sdlc-init`."
3. For each repo, verify `path` exists on disk. Any missing → **STOP**, list them.

## Step 1: Read or create state

Read `.sdlc/state.json`. If missing, create with empty defaults + `prd` from manifest.

## Step 2: Process each repo

For each repo:

### 2a: Detect language (if not set)
`cd` into repo path. `go.mod` → Go, `package.json` → TypeScript, `pyproject.toml` → Python, else "unknown".

### 2b: Generate CLAUDE.md
If `CLAUDE.md` does NOT exist → use the doc-generator agent's approach to generate it at repo root.
If exists → skip.

### 2c: Generate ARCHITECTURE.md
If `docs/ARCHITECTURE.md` does NOT exist → use the doc-generator agent's approach to generate it.
If exists → skip.

**NEVER overwrite existing files.**

## Step 3: Update state

- `run_id`: generate 8 hex chars if empty
- `prd`: from manifest
- `phase`: `"bootstrap"`, `phase_status`: `"completed"`
- `repos`: array with `{ name, path, team, language }`
- Preserve `artifacts` and `issues`
- `updated_at`: ISO timestamp

## Step 4: Report

```
Bootstrap complete:
  booking-app (TypeScript)
    ✓ CLAUDE.md — generated
    ✓ docs/ARCHITECTURE.md — generated
```
