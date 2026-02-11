# Agentic SDLC

Claude Code extension that orchestrates the software development lifecycle: from a PRD to approved PRs with tests.

## Architecture

This is NOT a custom CLI. It lives inside Claude Code's ecosystem:

- **Skills** (`.claude/skills/`) — slash commands the user invokes (`/sdlc-run`, `/sdlc-approve`, `/sdlc-status`, `/sdlc-resume`)
- **Agents** (`.claude/agents/`) — sub-agents spawned via `Task` tool. Never invoked directly by the user.
- **Hooks** (`.claude/hooks/`) — scripts triggered by Claude Code lifecycle events (e.g., `SubagentStop`)

The main Claude Code session is the **orchestrator**. It has access to all tools and MCPs.

## Key Principles

1. **Agents don't nest.** Hub-and-spoke: the orchestrator spawns all agents directly.
2. **LLMs only where reasoning is needed.** Deterministic operations (file checks, JSON parsing, MCP CRUD, CLI commands) are done by the orchestrator directly — no agent tokens wasted.
3. **State persists in `state.json`.** Written BEFORE each action to support resume across sessions.
4. **Idempotent operations.** doc-generator checks if CLAUDE.md exists. Linear issues are matched by title before creation. Branches and PRs are checked before creation.
5. **MCP servers** (Linear, Notion) are called by the orchestrator for structured CRUD. Only `session-resumer` uses MCP directly (foreground, for reasoning about state).

## State Management

- `state.json` — current run state (phase, issues, artifacts, metrics)
- `metrics.jsonl` — per-agent token usage (appended by hook)
- `manifest.yaml` — repo configuration and PRD reference

## File Layout

```
CLAUDE.md              ← this file
manifest.yaml          ← repos + PRD config
state.json             ← runtime state (generated)
metrics.jsonl          ← token metrics (generated)
.claude/
  settings.json        ← hooks configuration
  hooks/               ← lifecycle scripts
  skills/              ← slash commands (/sdlc-run, etc.)
  agents/              ← sub-agent definitions (7 agents)
docs/
  Description.md       ← full design document
```

## Conventions

- Agent instructions are written in **English**
- Skills reference phase files in `skills/sdlc-run/phases/` for detailed per-phase instructions
- The orchestrator detects repo language by checking `go.mod`, `package.json`, `pyproject.toml`
- PRs are created via `gh` CLI (not GitHub MCP)
- Linear and Notion operations use their respective MCP servers
- Max 3 agent review iterations before escalating to human
