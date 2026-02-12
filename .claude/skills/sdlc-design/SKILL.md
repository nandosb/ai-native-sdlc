---
name: sdlc-design
description: Analyze PRD + repo context and produce a scoping document
agent: solution-designer
context: fork
---

# SDLC Design — PRD to scoping document

## Input → Output

```
Input:  manifest.yaml (PRD URL) + CLAUDE.md / ARCHITECTURE.md per repo
Output: .sdlc/artifacts/scoping-doc.md, updated .sdlc/state.json
Agent:  solution-designer
```

## Step 0: Validate

1. Read `manifest.yaml`. Missing → **STOP**: "Run `/sdlc-init` first."
2. Check `prd` field. Empty → **STOP**: "No PRD in manifest. Run `/sdlc-init`."
3. Read `.sdlc/state.json`. Missing → **STOP**: "Run `/sdlc-bootstrap` first."
4. Check `state.repos`. Empty → **STOP**: "No repos in state. Run `/sdlc-bootstrap`."
5. For each repo, check `<path>/CLAUDE.md` exists. None have it → **STOP**: "Run `/sdlc-bootstrap` first."
6. **Integration check**: If `prd` is a Notion URL → call `notion-search` with query `"test"`. Fails → **STOP**: "Notion MCP not available. Add the Notion MCP server to Claude Code settings."

All six checks must pass.

## Step 1: Fetch PRD

- Notion URL → use Notion MCP to fetch page content.
- Local path → read file. Missing → **STOP**.

## Step 2: Read repo context

For each repo in state: read `CLAUDE.md` and `docs/ARCHITECTURE.md` (if exists).

## Step 3: Generate scoping document

Using the solution-designer agent's persona, format, and quality criteria — produce the scoping document from the PRD content (Step 1) and repo context (Step 2).

## Step 4: Save

Write to `.sdlc/artifacts/scoping-doc.md`.

## Step 5: Update state

- `phase`: `"design"`, `phase_status`: `"completed"`
- `artifacts.scoping_doc`: `".sdlc/artifacts/scoping-doc.md"`
- `updated_at`: ISO timestamp
- Preserve all other fields.
