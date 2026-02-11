# Phase: TRACKING

This phase creates Linear issues from the PERT tasks and configures their dependency relationships.

## Objective

Create one Linear issue per PERT task, set up blocking relationships, and prepare the issues map in `state.json` for the EXECUTING phase.

## Steps

### 1. Load PERT tasks

Read the parsed tasks from:
- `artifacts/pert-tasks.json` (local), OR
- The tasks stored in state from the PLANNING phase

### 2. Create issues in Linear

For each task, create a Linear issue using the Linear MCP:

```
mcp_linear.create_issue(
  team: task.team,
  title: task.title,
  description: task.description,
  priority: 3  // Normal
)
```

**Important**: Before creating, search for existing issues with the same title in the same team to ensure idempotency:
```
mcp_linear.list_issues(
  team: task.team,
  query: task.title
)
```
If found → use existing issue, don't create duplicate.

Map the PERT task ID (e.g., `shared-events#1`) to the Linear issue ID (e.g., `LIN-101`).

### 3. Configure blocking relationships

For each task with `blocked_by` entries:

1. Resolve the PERT IDs to Linear IDs using the mapping from step 2
2. Set the blocking relationship:
```
mcp_linear.update_issue(
  id: linear_issue_id,
  blockedBy: [resolved_linear_ids]
)
```

### 4. Build the issues map in state.json

```json
{
  "issues": {
    "LIN-101": {
      "repo": "shared-events",
      "title": "Define WebhookTriggered event",
      "status": "ready",
      "pr": null,
      "pr_number": null,
      "blocked_by": [],
      "review_iterations": 0,
      "human_iterations": 0
    },
    "LIN-102": {
      "repo": "api-gateway",
      "title": "Add POST /webhooks endpoint",
      "status": "blocked",
      "pr": null,
      "pr_number": null,
      "blocked_by": ["LIN-101"],
      "review_iterations": 0,
      "human_iterations": 0
    }
  }
}
```

Issues with no `blocked_by` entries start as `"ready"`. Issues with dependencies start as `"blocked"`.

### 5. Update state

```json
{
  "phase": "TRACKING",
  "phase_status": "completed",
  "issues": { ... }
}
```

### 6. Display to user

```
TRACKING phase complete. Issues created in Linear:

  LIN-101: Define WebhookTriggered event (shared-events) — ready
  LIN-102: Add POST /webhooks endpoint (api-gateway) — blocked by LIN-101
  ...

Total: {N} issues across {M} teams

Advancing to EXECUTING phase...
```

## Transition

TRACKING completes automatically → advance to EXECUTING. No approval gate.

## Error Handling

- If Linear MCP is unavailable, report error and stop
- If issue creation fails for one task, continue with others and report partial failures
- If a blocking reference can't be resolved (PERT ID not found), log warning and skip that dependency

## Idempotency

- Search by title before creating → prevents duplicates
- On resume: check which issues already exist in Linear, create only missing ones
- Re-running TRACKING with existing issues updates the state map without creating duplicates

## Topological Sort Note

The orchestrator determines execution order in the EXECUTING phase using this algorithm:
1. Find all issues where `status != "done"` AND all `blocked_by` issues have `status == "done"`
2. These are the "ready" issues — execute them in order (first by repo to minimize context switches)
3. After an issue completes → re-evaluate blocked issues to find newly unblocked ones
