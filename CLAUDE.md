# Agentic SDLC

Claude Code skills + subagents that orchestrate the full software development lifecycle: from a PRD to approved PRs with tests.

## .claude/ structure

```
.claude/
  settings.json                     Permissions, hooks
  settings.local.json               Personal overrides (gitignored)

  skills/                           Orchestration (slash commands)
    sdlc/SKILL.md                     /sdlc — full pipeline orchestrator
    sdlc-init/SKILL.md                /sdlc-init — configure manifest.yaml
    sdlc-bootstrap/SKILL.md           /sdlc-bootstrap — repo orientation docs
    sdlc-design/SKILL.md              /sdlc-design — PRD → scoping doc
    sdlc-plan/SKILL.md                /sdlc-plan — scoping doc → PERT
    sdlc-track/SKILL.md               /sdlc-track — PERT → Linear issues
    sdlc-execute/SKILL.md             /sdlc-execute — issues → PRs
    sdlc-preflight/SKILL.md            /sdlc-preflight — verify integrations
    sdlc-status/SKILL.md              /sdlc-status — read-only status

  agents/                           Subagents (isolated expertise)
    doc-generator/AGENT.md            Generates CLAUDE.md + ARCHITECTURE.md
    solution-designer/AGENT.md        PRD → scoping document
    task-decomposer/AGENT.md          Scoping doc → PERT tasks
    linear-issue-creator/AGENT.md     Creates Linear issues with deps
    coder/AGENT.md                    Implements code in worktrees
    quality-reviewer/AGENT.md         Reviews code (read-only, no edits)

  rules/                            Modular conventions
    state-management.md               state.json read/write rules
    git-worktrees.md                  Worktree and branch conventions
    linear-conventions.md             Linear issue creation rules
```

## How it works

**Skills** are slash commands that orchestrate: validate inputs → delegate to agent → save outputs → update state. Each skill declares its agent via `agent:` frontmatter and runs it in an isolated context via `context: fork`.

**Agents** are subagents with YAML frontmatter: tool restrictions, model selection, and a system prompt defining their expertise.

**Rules** are path-scoped conventions that apply automatically when relevant files are touched.

**Hooks** in `settings.json` fire on lifecycle events (pre-tool, post-stop).

## Usage

```bash
/sdlc              # Full pipeline (resumes from checkpoint)
/sdlc-init         # Configure PRD + repos → manifest.yaml
/sdlc-bootstrap    # manifest.yaml → CLAUDE.md + ARCHITECTURE.md
/sdlc-design       # PRD + repo docs → scoping-doc.md
/sdlc-plan         # scoping-doc.md → pert.md
/sdlc-track        # pert.md → Linear issues
/sdlc-execute      # issues → worktrees → PRs
/sdlc-preflight    # Verify integrations before running
/sdlc-status       # Read-only status
```

## State

- `manifest.yaml` — PRD URL, repos list
- `.sdlc/state.json` — Phase, repos, artifacts, issues
- `.sdlc/artifacts/` — scoping-doc.md, pert.md

## Conventions

- Language detection: `go.mod` → Go, `package.json` → TypeScript, `pyproject.toml` → Python
- PRs via `gh` CLI from worktrees
- Linear/Notion via MCP tools
- Max 3 review iterations per issue
