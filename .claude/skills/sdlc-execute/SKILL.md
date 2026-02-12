---
name: sdlc-execute
description: Implement each ready issue — worktree, code, tests, PR
skills:
  - sdlc-status
---

# SDLC Execute — issues to PRs

## Input → Output

```
Input:  .sdlc/state.json (issues with status "ready", repos with paths)
Output: Git worktrees with code, PRs via gh, updated .sdlc/state.json
Agents: coder, quality-reviewer
```

## Step 0: Validate

1. Read `.sdlc/state.json`. Missing → **STOP**: "Run `/sdlc-bootstrap` first."
2. Check `state.issues`. Empty → **STOP**: "Run `/sdlc-track` first."
3. Filter `status == "ready"`. None ready → **STOP**: "No ready issues."
4. Check `state.repos`. Empty → **STOP**: "Run `/sdlc-bootstrap` first."
5. For each repo referenced by a ready issue, verify `path` exists. Missing → **STOP**, list them.
6. **Integration checks**:
   - Run `gh auth status`. Fails → **STOP**: "GitHub CLI not authenticated. Run `gh auth login`."
   - For each repo referenced by a ready issue, run `git -C <path> rev-parse --git-dir`. Fails → **STOP**: "Repo `<name>` is not a valid git repository."

All seven checks must pass.

## Step 1: Compute execution waves

Group all "ready" issues into **waves** for parallel execution.

### 1a: Build the wave schedule

```
wave = 0
remaining = all issues (ready + blocked)
while remaining has issues:
  wave_issues = issues in remaining whose depends_on are ALL resolved (done/failed) or empty
  if wave_issues is empty → break (all remaining are stuck)
  assign wave_issues to wave N
  mark them as "will be done" for next iteration
  wave += 1
```

### 1b: Display the execution plan

```
Execution plan:
  Wave 0 (parallel): repo#1 "Add data model", repo#2 "Add config"
  Wave 1 (parallel): repo#3 "Add endpoints" (blocked by #1)
  Wave 2 (serial):   repo#4 "Add UI" (blocked by #3)
```

Ask: "Proceed with this execution plan? (y/n)"

## Step 2: Execute each wave

For each wave, in order:

### If wave has 1 issue → execute serial (same as before)

Follow Step 3 (single issue) below.

### If wave has 2+ issues → execute in parallel

1. **Set up all worktrees first** (serial — git operations on same repo need sequencing):
   For each issue in the wave:
   ```bash
   cd <repo_path>
   git fetch origin
   git worktree add .sdlc/worktrees/<repo>/<slug> -b feat/<slug> origin/main
   ```
   Slug = title → lowercase, hyphens, no special chars. If worktree exists → reuse.

2. **Launch parallel subagents** using the Task tool:
   For each issue in the wave, launch a Task subagent with `subagent_type: "general-purpose"`. Each subagent receives:
   - The coder agent prompt (from `.claude/agents/coder/AGENT.md`)
   - The quality-reviewer agent prompt (from `.claude/agents/quality-reviewer/AGENT.md`)
   - The issue description, acceptance criteria, linear_id
   - The worktree path (all work happens here)
   - The repo's CLAUDE.md and ARCHITECTURE.md content
   - Instruction: implement → self-review (max 3 iterations) → commit → push → create PR → return result JSON

   Each subagent returns a result:
   ```json
   {
     "issue_id": "repo#1",
     "status": "done" | "failed",
     "branch": "feat/<slug>",
     "pr_url": "https://github.com/.../pull/N",
     "iterations": 2,
     "error": null | "description of failure"
   }
   ```

   **Launch ALL subagents for the wave simultaneously** (multiple Task calls in one message). Do NOT wait for one to finish before starting the next.

3. **Collect results and update state** (serial — after all subagents return):
   For each result:
   - Update `state.issues[].status`, `branch`, `pr_url`, `iterations`
   - Re-read `state.json` fresh before each write (avoid stale overwrites)
   - Unblock dependents: if all `depends_on` are "done" → flip to "ready"
   - `updated_at`: ISO timestamp

4. **Show wave summary**:
   ```
   Wave 0 complete:
     ✓ repo#1  "Add data model"  → github.com/.../pull/1  (1 iter)
     ✓ repo#2  "Add config"      → github.com/.../pull/2  (2 iter)
   ```

### Move to next wave

Re-read `state.json`. Compute newly "ready" issues. If any → continue to next wave. If none → done.

## Step 3: Single issue execution (used by serial path)

### 3a: Set up worktree

```bash
cd <repo_path>
git fetch origin
git worktree add .sdlc/worktrees/<repo>/<slug> -b feat/<slug> origin/main
```
Slug = title → lowercase, hyphens, no special chars. If worktree exists → reuse.

### 3b: Read context

Read `<repo_path>/CLAUDE.md` and `docs/ARCHITECTURE.md`. Read issue description from state (or fetch from Linear via `linear_id`).

### 3c: Implement

Read `.claude/agents/coder/AGENT.md`. Adopt that agent's persona. Implement the task following the agent's approach and quality standards.

### 3d: Self-review (max 3 iterations)

Read `.claude/agents/quality-reviewer/AGENT.md`. Adopt that agent's persona. Review the changes:
1. Run test suite (from CLAUDE.md).
2. `git diff` to review.
3. If CHANGES REQUESTED → fix and loop.
4. If APPROVED → break.
After 3 failed iterations → set status to `"failed"`, move on.

### 3e: Commit + push

```bash
git add -A
git commit -m "feat: <description>\n\nImplements <linear_id>"
git push -u origin feat/<slug>
```

### 3f: Create PR

```bash
gh pr create --title "feat: <title>" --body "Implements <linear_id>..."
```

### 3g: Update state

- This issue: `status → "done"`, `branch`, `pr_url`, `iterations`
- Unblock: if all `depends_on` for another issue are `"done"` → flip to `"ready"`
- `updated_at`: ISO timestamp

## Step 4: Finalize

- `phase`: `"executing"`, `phase_status`: `"completed"`

```
Execute complete:
  Wave 0: repo#1 ✓, repo#2 ✓
  Wave 1: repo#3 ✓
  Wave 2: repo#4 ✗ (failed after 3 iterations)

  Total: 3 done, 1 failed
  PRs:
    - github.com/.../pull/1 — Add data model
    - github.com/.../pull/2 — Add config
    - github.com/.../pull/3 — Add endpoints
```

## Rules

- **Worktree setup is always serial** — git operations on the same repo must not race.
- **Implementation is parallel within a wave** — each subagent works in its own worktree.
- **State updates are always serial** — re-read state.json before each write.
- **Waves are sequential** — wave N+1 only starts after wave N completes and dependents are unblocked.
- **Failed issues do NOT block the wave** — other issues in the same wave continue independently.
- **Failed issues DO block dependents** — if a downstream issue depends on a failed one, it stays "blocked".
