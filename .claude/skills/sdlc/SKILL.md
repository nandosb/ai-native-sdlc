---
name: sdlc
description: Run the full SDLC pipeline or resume from last checkpoint
skills:
  - sdlc-preflight
  - sdlc-init
  - sdlc-bootstrap
  - sdlc-design
  - sdlc-plan
  - sdlc-track
  - sdlc-execute
  - sdlc-status
---

# SDLC Orchestrator — full pipeline

## Input → Output

```
Input:  manifest.yaml + .sdlc/state.json
Output: Runs each phase sequentially, confirming between phases
```

## Pipeline

```
/sdlc-init → /sdlc-preflight → /sdlc-bootstrap → /sdlc-design → /sdlc-plan → /sdlc-track → /sdlc-execute
```

## Step 0: Entry point

1. Read `manifest.yaml`. Missing → run init inline (ask user for PRD + repos, write manifest, create `.sdlc/`).
2. Read `.sdlc/state.json` if it exists.

## Step 0.5: Preflight check

Run `/sdlc-preflight`. If FAIL → **STOP**. Show fix instructions from the preflight report.

## Step 1: Determine resume point

| `phase` | `phase_status` | Next |
|---------|----------------|------|
| _(empty)_ | — | bootstrap |
| `bootstrap` | `completed` | design |
| `design` | `completed` | plan |
| `planning` | `completed` | track |
| `tracking` | `completed` | execute |
| `executing` | `completed` | done |
| _any_ | `in_progress` / `failed` | retry that phase |

Show: "Pipeline: `<phase>` (`<status>`). Next: `<next>`". Ask confirmation.

## Step 2: Run phases

For each phase, follow its skill's instructions. The skill delegates to its agent.

## Step 3: Between phases

After each phase:
1. Show summary of what was produced.
2. Ask: "Continue to `<next>`? (y/n)"
3. Yes → re-read `.sdlc/state.json` (fresh), proceed.
4. No → stop. User resumes later with `/sdlc`.

## Step 4: Done

```
Pipeline complete!
  Issues: N (done: X, failed: Y)
  PRs:
    - <url> — <title>
```

## Rules

- Confirm before each phase.
- Re-read state.json before each phase.
- Phase fails → ask retry or skip.
