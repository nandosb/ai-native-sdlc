# Phase: EXECUTING

This is the main implementation phase. It loops through issues in topological order, spawning coder and reviewer agents for each.

## Objective

For each task (Linear issue), implement the code, review it, and get it to a state where a human can review and merge the PR.

## Execution Loop

```
while there are issues with status != "done":
  1. Find next ready issue (topological sort)
  2. Prepare coder context
  3. Pull latest code
  4. Spawn coder → get PR
  5. Spawn quality-reviewer → get verdict
  6. If REQUEST_CHANGES → spawn feedback-writer → re-spawn coder (max 3x)
  7. If APPROVE → mark awaiting_human, notify user
  8. Wait for human action (merge or feedback)
  9. On merge → mark done, unblock dependents, continue loop
  10. On human feedback → spawn coder with feedback, go to step 5
```

## Detailed Steps

### 1. Find next ready issue

Apply topological sort:
- Find issues where `status` is `ready` (not blocked, not done, not in progress)
- Prefer issues in repos that already have context loaded (minimize context switches)
- If multiple ready issues exist, pick the first one by their order in the PERT

### 2. Prepare coder context

Read the repo's `CLAUDE.md`:
```
Read({repo_path}/CLAUDE.md)
```

Build the coder prompt with:
- `repo_path` from manifest
- `task_title` and `task_description` from the issue
- `claude_md` contents
- `language` from bootstrap detection
- `branch_base`: usually `main`

### 3. Pull latest code

```bash
cd {repo_path}
git checkout main
git pull origin main
```

**Important**: This is deterministic — do it directly, no agent needed.

### 4. Spawn coder

Update issue status to `implementing`:
```json
{ "status": "implementing" }
```

```
Task(
  subagent_type: "coder",
  prompt: "Implement this task:\n\nTitle: {title}\nDescription: {description}\n\nRepo: {repo_path}\nLanguage: {language}\n\nCLAUDE.md:\n{claude_md}\n\nBranch from: main",
  description: "Implement {issue_id}: {title}"
)
```

Parse the result to get `pr_url` and `pr_number`. Update state:
```json
{
  "status": "reviewing",
  "pr": "org/repo#N",
  "pr_number": N
}
```

### 5. Spawn quality-reviewer

```
Task(
  subagent_type: "quality-reviewer",
  prompt: "Review PR #{pr_number} in {repo_path}.\n\nTask: {title}\nDescription: {description}\n\nCLAUDE.md:\n{claude_md}\nLanguage: {language}",
  description: "Review PR for {issue_id}",
  model: "opus"
)
```

### 6. Handle review verdict

**If APPROVE:**
```json
{ "status": "awaiting_human" }
```
Notify the user:
```
PR ready for human review: {pr_url}
Merge when ready, then run /sdlc-resume.
```

**If REQUEST_CHANGES (iteration < 3):**

Increment `review_iterations`. Spawn feedback-writer:
```
Task(
  subagent_type: "feedback-writer",
  prompt: "Post review feedback on PR #{pr_number} in {repo_path}.\n\nVerdict: REQUEST_CHANGES\nSummary: {summary}\nObservations: {observations_json}",
  description: "Post feedback on {issue_id}"
)
```

Then re-spawn coder with feedback:
```
Task(
  subagent_type: "coder",
  prompt: "Address review feedback for task:\n\nTitle: {title}\nFeedback: {observations}\n\nRepo: {repo_path}\nBranch: feat/{branch_name}",
  description: "Iterate on {issue_id}: address review feedback"
)
```

Go back to step 5 (quality-reviewer again).

**If REQUEST_CHANGES (iteration >= 3):**
Mark as `awaiting_human` and escalate:
```
Agent review loop reached max iterations (3) for {issue_id}.
PR: {pr_url}
Last feedback: {summary}

Please review manually and either merge or provide guidance.
Run /sdlc-resume after taking action.
```

### 7. Pause for human review

After setting `awaiting_human`, the loop pauses. The user will:
- Review and merge the PR → run `/sdlc-resume`
- Request changes → comments appear on PR → run `/sdlc-resume`

### 8. After merge (triggered by /sdlc-resume)

When `session-resumer` detects a merge:
1. Mark issue as `done`
2. Update Linear issue status to Done
3. Check all `blocked_by` references — if all blockers are `done`, set dependent issues to `ready`
4. Continue the loop with the next ready issue

### 9. Completion

When all issues have `status: done`:
```json
{
  "phase": "COMPLETED",
  "phase_status": "completed"
}
```

Display final summary:
```
All {N} issues implemented and merged!

Summary:
  Issues: {N} completed
  PRs: {N} merged
  Agent review iterations: {N}
  Human review iterations: {N}
  Duration: {time}

Run /sdlc-status for detailed metrics.
```

## State Updates

Write `state.json` BEFORE every action:
- Before spawning coder → `implementing`
- Before spawning reviewer → `reviewing`
- Before posting feedback → still `reviewing`
- After approve → `awaiting_human`
- After merge detected → `done`

## Error Handling

- If coder fails → retry once. If still fails → mark issue as `failed`, skip, continue with next ready issue
- If reviewer fails → skip review, mark as `awaiting_human` (let human review)
- If feedback-writer fails → log warning, continue with coder iteration anyway (coder gets the raw observations)
- If git operations fail (conflict, etc.) → report to user, pause

## Idempotency

- Before creating branch: check if it exists
- Before creating PR: check if one exists for the branch
- Before spawning coder on resume: check if PR already has the changes
- On re-entry to EXECUTING: resume from the first non-done issue in topological order
