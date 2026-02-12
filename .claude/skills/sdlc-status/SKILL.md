---
name: sdlc-status
description: Display current SDLC pipeline state. Read-only, no modifications.
allowed-tools: Read, Glob, Grep
---

# SDLC Status — display pipeline state

## Input → Output

```
Input:  .sdlc/state.json + manifest.yaml
Output: Formatted summary (no state changes)
```

## Step 0: Read

1. Read `manifest.yaml`. Missing → "No manifest.yaml. Run `/sdlc-init`."
2. Read `.sdlc/state.json`. Missing → "No state. Run `/sdlc-init` → `/sdlc-bootstrap`."

If either missing, show what IS available and stop.

## Step 1: Summary

```
SDLC Pipeline Status
====================
Run ID:   <run_id or "-">
PRD:      <prd from manifest>
Phase:    <phase or "not started">
Status:   <phase_status or "-">
Updated:  <updated_at or "-">

Repos:
  <name> (<language>) — <team>

Artifacts:
  scoping_doc: <path or "—">
  pert:        <path or "—">
```

## Step 2: Next action

| Current | Next |
|---------|------|
| No state | `/sdlc-init` → `/sdlc-bootstrap` |
| bootstrap done | `/sdlc-design` |
| design done | `/sdlc-plan` |
| planning done | `/sdlc-track` |
| tracking done | `/sdlc-execute` |
| executing done | Pipeline complete |

## Step 3: Issues (if any)

```
Issues: N total (ready: X, blocked: Y, done: Z, failed: W)
  STATUS  ID        LINEAR    TITLE                PR
  done    repo#1    IGN-101   Add data model       github.com/.../pull/1
  ready   repo#2    IGN-102   Add endpoints        —
  blocked repo#3    IGN-103   Add UI               —
```
