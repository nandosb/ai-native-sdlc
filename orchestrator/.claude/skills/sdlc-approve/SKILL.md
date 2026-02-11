---
name: sdlc-approve
description: Approves the current approval gate in the active SDLC run
allowed-tools: Read, Write, Bash, Task
---

# /sdlc-approve

Approve the current gate and advance the SDLC run to the next phase.

## Instructions

### 1. Read state.json

Read `state.json` from the project root. If it doesn't exist:
```
No active SDLC run found. Run /sdlc-run to start a new run.
```

### 2. Validate current state is an approval gate

Check that `phase_status` is `awaiting_approval`. Valid gates:

| Phase | What was produced | Next phase |
|---|---|---|
| DESIGN | Scoping Document | PLANNING |
| PLANNING | PERT (task plan) | TRACKING |

If `phase_status` is NOT `awaiting_approval`:
```
No approval gate pending.
Current state: {phase} / {phase_status}

Approval gates occur after DESIGN and PLANNING phases.
```

### 3. Advance to next phase

**If approving DESIGN → advance to PLANNING:**

1. Update state:
```json
{
  "phase": "PLANNING",
  "phase_status": "in_progress"
}
```

2. Read the phase instructions from `phases/planning.md`
3. Execute the PLANNING phase (spawn task-decomposer, store PERT)
4. The PLANNING phase will set its own approval gate when complete

**If approving PLANNING → advance to TRACKING:**

1. Update state:
```json
{
  "phase": "TRACKING",
  "phase_status": "in_progress"
}
```

2. Read the phase instructions from `phases/tracking.md`
3. Execute the TRACKING phase (create Linear issues)
4. TRACKING completes automatically and advances to EXECUTING
5. Read `phases/executing.md` and begin execution

### 4. Record phase timing

Update `metrics.phases` for the completed phase:
```json
{
  "ended_at": "{now}",
  "duration_s": "{calculated}"
}
```

Start timing for the new phase:
```json
{
  "started_at": "{now}"
}
```

### 5. Display confirmation

```
Approved: {phase} phase
Advancing to: {next_phase}

{next_phase specific message}
```

## Error Handling

- If state.json doesn't exist → clear error message
- If not at an approval gate → explain current state
- If the next phase fails to start → preserve state, report error
