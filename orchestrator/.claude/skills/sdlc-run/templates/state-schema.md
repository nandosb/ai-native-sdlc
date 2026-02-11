# state.json Schema

This document defines the structure of `state.json`, the persistent state file for an Agentic SDLC run.

## Schema

```json
{
  "run_id": "string — unique identifier, format: YYYY-MM-DD-{feature-slug}",
  "prd_url": "string — Notion URL of the PRD",
  "phase": "BOOTSTRAP | DESIGN | PLANNING | TRACKING | EXECUTING | COMPLETED",
  "phase_status": "in_progress | awaiting_approval | awaiting_human | completed",

  "bootstrap": {
    "languages": {
      "{repo_name}": "go | typescript | python"
    },
    "docs_generated": {
      "{repo_name}": "org/repo#N (PR URL or number)"
    }
  },

  "artifacts": {
    "scoping_doc": "string | null — Notion URL of the Scoping Document",
    "pert": "string | null — Notion URL of the PERT"
  },

  "issues": {
    "{LINEAR_ID}": {
      "repo": "string — repo name from manifest",
      "title": "string — issue title",
      "status": "blocked | ready | implementing | reviewing | awaiting_human | done | failed",
      "pr": "string | null — org/repo#N",
      "pr_number": "number | null",
      "blocked_by": ["string — LINEAR_IDs"],
      "review_iterations": "number — count of agent review cycles",
      "human_iterations": "number — count of human review cycles"
    }
  },

  "updated_at": "string — ISO-8601 timestamp",

  "metrics": {
    "started_at": "string — ISO-8601 timestamp",
    "phases": {
      "{PHASE_NAME}": {
        "started_at": "string",
        "ended_at": "string | null",
        "duration_s": "number | null"
      }
    },
    "agent_invocations": [
      {
        "agent": "string — agent name",
        "model": "string — sonnet | opus",
        "repo": "string | null — repo name if applicable",
        "issue": "string | null — LINEAR_ID if applicable",
        "started_at": "string",
        "duration_s": "number",
        "tokens": {
          "input": "number",
          "output": "number"
        }
      }
    ],
    "totals": {
      "duration_s": "number",
      "agent_spawns": "number",
      "review_iterations": {
        "agent": "number",
        "human": "number"
      },
      "issues_created": "number",
      "prs_created": "number",
      "prs_merged": "number"
    }
  }
}
```

## Phase Transitions

```
BOOTSTRAP → DESIGN → PLANNING → TRACKING → EXECUTING → COMPLETED
                ↑         ↑
          (approval) (approval)
```

## Issue Status Transitions

```
blocked → ready → implementing → reviewing → awaiting_human → done
                       ↑              |
                       └──────────────┘  (REQUEST_CHANGES → re-implement)
```

## Rules

1. `state.json` MUST be written BEFORE each action (write-ahead)
2. `updated_at` MUST be refreshed on every write
3. `phase_status` is always relative to the current `phase`
4. Issue `blocked_by` references other Linear IDs in the `issues` map
5. An issue transitions from `blocked` to `ready` when ALL its `blocked_by` issues have status `done`
