# Agentic SDLC v2

Standalone Go CLI + React Web UI that orchestrates the software development lifecycle: from a PRD to approved PRs with tests.

## Architecture

```
Web UI (React) → HTTP/WebSocket → Go Server → Engine (State Machine)
                                                 ├── Claude CLI Wrapper
                                                 ├── Git Worktree Manager
                                                 └── External APIs (Linear, Notion, GitHub)
```

- **Go CLI** (`cmd/sdlc/`) — Cobra-based CLI with subcommands per phase
- **Engine** (`internal/engine/`) — State machine, event bus, metrics collection
- **Phases** (`internal/phase/`) — Each SDLC phase as a `PhaseRunner` implementation
- **Claude** (`internal/claude/`) — Wrapper for `claude` CLI invocation + streaming
- **Git** (`internal/git/`) — Worktree lifecycle, branch/PR operations via `gh`
- **Integrations** (`internal/integrations/`) — Linear GraphQL, Notion REST, GitHub CLI
- **Server** (`internal/server/`) — HTTP + WebSocket server with embedded React frontend
- **Web UI** (`web/`) — React 19, Vite, Tailwind CSS dashboard
- **Prompts** (`prompts/`) — Markdown templates for each agent type

## Key Principles

1. **Phases are independent.** Each subcommand (design, plan, track, execute) can run standalone with explicit inputs.
2. **Git isolation via worktrees.** Each issue gets its own worktree — the user's main tree is never touched.
3. **LLMs only where reasoning is needed.** Deterministic operations (file checks, JSON parsing, API calls) are done in Go — no agent tokens wasted.
4. **State persists in `state.json`.** Written atomically (temp + rename) before every action. Supports resume across crashes.
5. **Idempotent operations.** Issues matched by title before creation. Worktrees reused if they exist. PRs checked before creation.
6. **Real-time events.** Event bus publishes phase/agent/issue events → WebSocket → React UI.

## CLI Commands

```
sdlc init       — Validate manifest, initialize state.json
sdlc bootstrap  — Generate CLAUDE.md + ARCHITECTURE.md per repo
sdlc design     — PRD → Scoping document (standalone: --prd <file>)
sdlc plan       — Scoping doc → PERT (standalone: --scoping-doc <file>)
sdlc track      — PERT → Linear issues
sdlc execute    — Issues → PRs (parallel worktrees, --parallel <n>)
sdlc run        — Full pipeline with approval gates
sdlc approve    — Approve current gate
sdlc status     — Print current state
sdlc serve      — Start web UI on :3000
```

## State Management

- `state.json` — Current run state (phase, issues, artifacts, metrics)
- `metrics.jsonl` — Per-agent token usage (appended per invocation)
- `manifest.example.yaml` — Example manifest (copy to `manifest.yaml` to configure)

## Build

```bash
# Go CLI
go build -o sdlc ./cmd/sdlc/

# React frontend (outputs to internal/server/static/)
cd web && npm install && npm run build

# Full rebuild
cd web && npm run build && cd .. && go build -o sdlc ./cmd/sdlc/
```

## Conventions

- Agent prompts live in `prompts/*.md` with `{{variable}}` interpolation
- Language detection: `go.mod` → Go, `package.json` → TypeScript, `pyproject.toml` → Python
- PRs created via `gh` CLI from within worktrees
- Linear uses GraphQL API directly (env: `LINEAR_API_KEY`)
- Notion uses REST API directly (env: `NOTION_API_KEY`)
- Max 3 review iterations per issue before escalating to human
- Parallel execution controlled by `--parallel <n>` (default: 3)
