# Agentic SDLC

Claude Code skills + subagents that orchestrate the full software development lifecycle — from a PRD to approved pull requests with tests.

```
PRD → Scoping Doc → PERT → Linear Issues → Worktree PRs → Human Review
```

## How It Works

Two layers using official Claude Code patterns:

- **Skills** (`.claude/skills/*/SKILL.md`) — Slash commands that orchestrate: validate inputs, delegate to agents, save outputs, update state. YAML frontmatter for tool restrictions, agent delegation, context isolation.
- **Subagents** (`.claude/agents/*/AGENT.md`) — Specialized AI workers with isolated context windows, restricted tools, and model selection. Each has a focused expertise.

```
Skill                     Agent                     I/O
─────                     ─────                     ───
/sdlc-init                (interactive)             → manifest.yaml
/sdlc-bootstrap     →    doc-generator              → CLAUDE.md, ARCHITECTURE.md
/sdlc-design        →    solution-designer           → scoping-doc.md
/sdlc-plan          →    task-decomposer             → pert.md
/sdlc-track         →    linear-issue-creator        → Linear issues
/sdlc-execute       →    coder + quality-reviewer    → worktrees + PRs
/sdlc-preflight           (read-only)                → integration check report
/sdlc-status              (read-only)                → formatted summary
/sdlc                     (orchestrator)             → chains all above
```

Each skill validates its required input exists before running. Missing prerequisites → STOP with "run X first".

### Pipeline Phases

| Phase | Input | Output | Agent | Model |
|-------|-------|--------|-------|-------|
| **init** | User input | `manifest.yaml` | — | — |
| **bootstrap** | `manifest.yaml` | `CLAUDE.md`, `ARCHITECTURE.md` | doc-generator | sonnet |
| **design** | PRD + repo docs | `scoping-doc.md` | solution-designer | opus |
| **plan** | `scoping-doc.md` | `pert.md` | task-decomposer | opus |
| **track** | `pert.md` (JSON) | Linear issues | linear-issue-creator | sonnet |
| **execute** | Issues (ready) | Worktrees → PRs | coder + quality-reviewer | sonnet |

### Git Worktree Isolation

Each issue gets its own worktree. Your main branch is never touched.

```
.sdlc/worktrees/
  api-gateway/
    add-endpoint/       ← isolated worktree
    add-auth/           ← isolated worktree
  web-app/
    add-booking-ui/     ← isolated worktree
```

## Prerequisites

- **Claude Code** installed and authenticated
- **GitHub CLI** (`gh`) installed and authenticated
- **Linear MCP** configured in Claude Code
- **Notion MCP** configured in Claude Code (for Notion PRDs)

## Quick Start

```bash
# Full pipeline (resumes from last checkpoint)
/sdlc

# Or step by step
/sdlc-init         # Configure PRD + repos → manifest.yaml
/sdlc-bootstrap    # manifest.yaml → CLAUDE.md + ARCHITECTURE.md
/sdlc-design       # PRD + repo docs → scoping-doc.md
/sdlc-plan         # scoping-doc.md → pert.md
/sdlc-track        # pert.md → Linear issues
/sdlc-execute      # issues → worktrees → PRs
/sdlc-preflight    # Verify integrations
/sdlc-status       # Check state at any time
```

## Configuration

### manifest.yaml

```yaml
prd: https://notion.so/org/my-prd-page    # or ./local-prd.md
repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend
  - name: web-app
    path: ../web-app
    team: Frontend
```

Create interactively with `/sdlc-init` or copy from `manifest.example.yaml`.

## Project Structure

```
.claude/
  settings.json                   Permissions + hooks
  skills/                         Slash commands (orchestration)
    sdlc/SKILL.md                   Full pipeline orchestrator
    sdlc-init/SKILL.md              Manifest configuration
    sdlc-bootstrap/SKILL.md         Repo orientation docs
    sdlc-design/SKILL.md            PRD → scoping document
    sdlc-plan/SKILL.md              Scoping doc → PERT
    sdlc-track/SKILL.md             PERT → Linear issues
    sdlc-execute/SKILL.md           Issues → PRs
    sdlc-preflight/SKILL.md         Integration verification
    sdlc-status/SKILL.md            Status reporter
  agents/                         Subagents (expertise)
    doc-generator/AGENT.md          Generates CLAUDE.md + ARCHITECTURE.md
    solution-designer/AGENT.md      PRD analysis → scoping document
    task-decomposer/AGENT.md        Scoping doc → PERT task graph
    linear-issue-creator/AGENT.md   Creates Linear issues with deps
    coder/AGENT.md                  Implements code in worktrees
    quality-reviewer/AGENT.md       Reviews code (read-only)
  rules/                          Path-scoped conventions
    state-management.md             .sdlc/ state rules
    git-worktrees.md                Worktree and branch rules
    linear-conventions.md           Linear issue rules

manifest.yaml                    Project configuration
.sdlc/                           Pipeline state + artifacts + worktrees
  state.json                       Phase, repos, artifacts, issues
  artifacts/                       scoping-doc.md, pert.md
  worktrees/                       Git worktrees per issue
```

## Design Decisions

1. **Official Claude Code patterns** — Skills with YAML frontmatter, subagents with tool restrictions, path-scoped rules, lifecycle hooks.
2. **Skills orchestrate, agents execute** — Skills validate and save state. Agents provide isolated expertise with `context: fork`.
3. **Strict input validation** — Each skill checks prerequisites. Missing → STOP with "run X first".
4. **Model selection per agent** — Opus for design/planning (complex reasoning), Sonnet for coding/tracking (fast execution).
5. **Read-only reviewer** — Quality-reviewer agent has `disallowedTools: Write, Edit` — it can only read and judge.
6. **Path-scoped rules** — Conventions activate automatically when relevant files are touched.

## License

Internal use — Yalochat.
