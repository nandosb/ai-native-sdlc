---
name: sdlc-status
description: Shows the current state and metrics of the Agentic SDLC run
allowed-tools: Read, Bash
---

# /sdlc-status

Display the current state, progress, and cost metrics of the active SDLC run.

## Instructions

### 1. Read state.json

Read `state.json` from the project root. If it doesn't exist, report:
```
No active SDLC run found. Run /sdlc-run to start a new run.
```

### 2. Display run overview

```
Run: {run_id}
Phase: {phase} ({phase_status})
Started: {metrics.started_at}
Duration: {calculated from started_at to now}
```

### 3. Display phase progress

For each completed phase, show timing:
```
Phases:
  BOOTSTRAP  ✓  {duration_s}s
  DESIGN     ✓  {duration_s}s
  PLANNING   ✓  {duration_s}s
  TRACKING   ✓  {duration_s}s
  EXECUTING  ⏳  in progress
```

### 4. Display issue status (if TRACKING or EXECUTING)

Count issues by status and show a summary table:

```
Issues ({total}):
  ✓ done:            {count}
  ⏳ implementing:    {count}
  🔍 reviewing:       {count}
  👤 awaiting_human:  {count}
  🚫 blocked:         {count}
  📋 ready:           {count}

{issue_id}: {title} ({repo}) — {status}
{issue_id}: {title} ({repo}) — {status}
...
```

### 5. Read metrics.jsonl (if exists)

Read `metrics.jsonl` to get per-agent token usage. Aggregate by agent type:

```
Agent Metrics:
  {agent_type}  ×{count}  {total_duration}s  {total_tokens}K tokens  ~${estimated_cost}
  ...
```

Use pricing:
- Opus: $15/M input, $75/M output
- Sonnet: $3/M input, $15/M output

### 6. Display totals

```
Totals:
  Agent spawns:        {metrics.totals.agent_spawns}
  Review iterations:   {metrics.totals.review_iterations.agent} agent, {metrics.totals.review_iterations.human} human
  Issues created:      {metrics.totals.issues_created}
  PRs created:         {metrics.totals.prs_created}
  PRs merged:          {metrics.totals.prs_merged}
  Estimated cost:      ~${total}
```

### 7. If metrics.jsonl doesn't exist

Show only the information available from state.json (phases, issues, totals) and note:
```
Note: Token metrics not available. Ensure .claude/hooks/track-agent-metrics.sh is configured.
```

## Error Handling

- If state.json is malformed, report the parse error
- If metrics.jsonl has malformed lines, skip them and count errors
- Always show whatever information IS available, even if some sources are missing
