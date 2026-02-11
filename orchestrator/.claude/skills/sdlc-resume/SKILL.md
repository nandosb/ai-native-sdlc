---
name: sdlc-resume
description: Resumes an interrupted SDLC run by analyzing current state
allowed-tools: Read, Write, Bash, Task
---

# /sdlc-resume

Resume an interrupted SDLC run. Compares persisted state with actual GitHub/Linear state to determine what to do next.

## Instructions

### 1. Read state.json

Read `state.json` from the project root. If it doesn't exist:
```
No active SDLC run found. Run /sdlc-run to start a new run.
```

### 2. Read manifest.yaml

Load the manifest to get repo paths and configuration.

### 3. Spawn session-resumer

The session-resumer agent analyzes the real state of GitHub and Linear against state.json.

```
Task(
  subagent_type: "session-resumer",
  prompt: "Analyze the current state and determine next action.\n\nState path: state.json\n\nRepos:\n{repos_json}",
  description: "Analyze state for resume"
)
```

**Important**: session-resumer runs in **foreground** because it uses the Linear MCP.

### 4. Process the resumer's instructions

The session-resumer returns a JSON instruction set with:
- `action`: what to do next
- `updates`: state changes to apply
- `message`: human-readable summary

### 5. Apply state updates

For each update in the instruction set:
1. Update `state.json` with the new status
2. Update Linear issues if needed (via Linear MCP)

### 6. Execute the recommended action

| Action | What to do |
|---|---|
| `continue_executing` | Read `phases/executing.md`, continue loop from next ready issue |
| `re_execute_design` | Read `phases/design.md`, re-run design phase |
| `show_approval_gate` | Display artifact link, wait for /sdlc-approve |
| `create_missing_issues` | Read `phases/tracking.md`, create missing issues |
| `spawn_coder` | Extract human feedback from PR, spawn coder with it |
| `wait_for_human` | Inform user PR is still awaiting review |
| `completed` | Display completion summary |

### 7. Display status

Show the resumer's message and current state:

```
Resume analysis:
{resumer.message}

Current state: {phase} / {phase_status}
{additional context based on action}
```

## Handling specific scenarios

### PR was merged
```
PR {pr_url} was merged.
Updated {issue_id} → done
Unblocked: {unblocked_issues}
Continuing with next issue: {next_issue_id}
```

### PR has human feedback
```
PR {pr_url} received human feedback.
Spawning coder to address comments...
```
Extract comments via:
```bash
gh pr view {pr_number} --repo {org/repo} --json reviews,comments
```

### All done
```
All issues completed!
Run /sdlc-status for the full summary.
```

## Error Handling

- If session-resumer fails → fall back to basic state.json analysis (show phase and status)
- If state.json is from an old/incompatible version → report and suggest starting fresh
- If repos have moved or are inaccessible → report which repos can't be reached

## Idempotency

- Safe to run multiple times — session-resumer always checks real state
- State updates are based on actual GitHub/Linear state, not assumptions
- Running /sdlc-resume when nothing has changed simply shows current status
