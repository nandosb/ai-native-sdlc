---
name: sdlc-preflight
description: Verify all integrations needed for the pipeline are available
allowed-tools: Read, Glob, Bash
---

# SDLC Preflight — verify integrations

## Input → Output

```
Input:  manifest.yaml + .sdlc/state.json (optional)
Output: Pass/fail report per integration. No state changes.
```

## Purpose

Run before starting the pipeline (or any phase) to catch missing integrations early.
Can be called standalone (`/sdlc-preflight`) or invoked by `/sdlc` at startup.

## Step 0: Read context

1. Read `manifest.yaml`. Missing → **STOP**: "Run `/sdlc-init` first."
2. Read `.sdlc/state.json` if it exists (to know which phase comes next).
3. Determine `next_phase` from state (same logic as `/sdlc` resume table).
   No state → next is `bootstrap`.

## Step 1: Check core tools

These are always required:

| Check | Command | Pass condition |
|-------|---------|----------------|
| **Git** | `git --version` | Exit 0, version printed |
| **GitHub CLI** | `gh --version` | Exit 0, version printed |
| **GitHub auth** | `gh auth status` | Exit 0, shows authenticated user |

Any failure → record as `FAIL`.

## Step 2: Check repo access

For each repo in `manifest.yaml`:

| Check | Command | Pass condition |
|-------|---------|----------------|
| Path exists | `test -d <path>` | Exit 0 |
| Is git repo | `git -C <path> rev-parse --git-dir` | Exit 0 |
| Has remote | `git -C <path> remote get-url origin` | Exit 0, URL printed |

Any failure → record as `FAIL` with repo name.

## Step 3: Check MCP integrations

### Notion MCP (required if PRD is a Notion URL)

Check: Call `notion-search` with query `"test"` (via Notion MCP tool).
- Responds → `PASS`
- Errors or tool not available → `FAIL`

Skip if PRD field is a local file path (not a Notion URL).

### Linear MCP (required from tracking phase onward)

Check: Call `list_teams` (via Linear MCP tool).
- Responds with team list → `PASS`
- Errors or tool not available → `FAIL`

Skip if `next_phase` is `bootstrap`, `design`, or `plan` (not yet needed).

## Step 4: Determine phase-specific requirements

| Next phase | Required checks |
|------------|----------------|
| `bootstrap` | Git, repo access |
| `design` | Git, repo access, Notion (if PRD is URL) |
| `plan` | Git, repo access |
| `track` | Git, repo access, Linear MCP |
| `execute` | Git, repo access, GitHub CLI + auth, Linear MCP |

Mark checks as `REQUIRED` or `OPTIONAL` based on next phase.

## Step 5: Report

```
Preflight Check
===============
Next phase: <next_phase>

Core tools:
  ✓ git          2.x.x
  ✓ gh           2.x.x
  ✓ gh auth      authenticated as <user>

Repos:
  ✓ booking-app  /path/to/repo (git remote: origin)
  ✗ payments     /path/missing — directory not found

Integrations:
  ✓ Notion MCP   connected
  ✗ Linear MCP   not available — install Linear MCP server

Result: PASS | FAIL
```

## Step 6: Decide

- All REQUIRED checks pass → `PASS`. Print report and return.
- Any REQUIRED check fails → `FAIL`. Print report with fix instructions:

| Failed check | Fix instruction |
|--------------|----------------|
| git | Install git |
| gh | Install GitHub CLI: `brew install gh` |
| gh auth | Run `gh auth login` |
| Repo path | Check path in `manifest.yaml` — run `/sdlc-init` to fix |
| Repo not git | Initialize repo or check path |
| Notion MCP | Add Notion MCP server to Claude Code settings |
| Linear MCP | Add Linear MCP server to Claude Code settings |

**STOP** on FAIL. Do not proceed to any phase.

## Rules

- **Read-only.** No state changes, no file modifications.
- **Always show full report** even on PASS (so user sees what was checked).
- **Phase-aware.** Only require integrations needed for the next phase.
- **Fail fast.** Run all checks, then report all failures at once (not one at a time).
