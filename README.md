# Agentic SDLC

Go server + React Web UI that orchestrates the full software development lifecycle with AI — from a PRD to approved pull requests with tests.

```
PRD → Scoping Doc → PERT → Linear Issues → Worktree PRs → Human Review
```

## How It Works

The server exposes a REST API and WebSocket for real-time events. Each phase invokes Claude via the `claude` CLI for reasoning tasks, while deterministic work (git, APIs, file I/O) runs directly in Go. Phase execution is asynchronous — endpoints return `202 Accepted` and push progress over WebSocket.

```
┌─────────────────────────────────────────────────┐
│              Web UI (React + Tailwind)           │
│  Dashboard │ Issues │ Logs │ Metrics │ Approvals │
└──────────────────┬──────────────────────────────┘
                   │ WebSocket + REST
┌──────────────────▼──────────────────────────────┐
│              Go HTTP Server (:3000)              │
│  /api/init  /api/phases/*/run  /api/run  /ws     │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│              Engine (State Machine)               │
│  Phases │ Event Bus │ State │ Metrics             │
└───┬──────────┬──────────┬───────────────────────┘
    │          │          │
┌───▼───┐ ┌───▼────┐ ┌───▼────┐
│Claude │ │  Git   │ │External│
│  CLI  │ │Worktree│ │  APIs  │
└───────┘ └────────┘ └────────┘
```

### Pipeline Phases

| Phase | Input | Output | Agent |
|-------|-------|--------|-------|
| **bootstrap** | Repo path | `CLAUDE.md`, `ARCHITECTURE.md` | doc-generator (sonnet) |
| **design** | PRD | Scoping document | solution-designer (opus) |
| **planning** | Scoping doc | PERT (tasks + deps) | task-decomposer (opus) |
| **tracking** | PERT | Linear issues | _(deterministic)_ |
| **executing** | Issues | Pull requests | coder + quality-reviewer (sonnet/opus) |

Approval gates pause the pipeline after **design** and **planning** for human review.

### Agent Prompts

Six specialized agent prompts live in `prompts/`:

| Prompt | Role |
|--------|------|
| `doc-generator.md` | Generates `CLAUDE.md` and `ARCHITECTURE.md` for a repo |
| `solution-designer.md` | Analyzes a PRD and produces a scoping document |
| `task-decomposer.md` | Breaks a scoping document into a PERT task graph |
| `coder.md` | Implements a task inside a git worktree |
| `quality-reviewer.md` | Reviews code changes and requests fixes |
| `feedback-writer.md` | Writes structured feedback for review iterations |

All templates use `{{variable}}` interpolation and are embedded at compile time.

### Git Worktree Isolation

Each issue gets its own worktree. Your main branch is never touched.

```
.sdlc/worktrees/
  api-gateway/
    LIN-101-add-endpoint/     ← isolated worktree
    LIN-102-add-auth/         ← isolated worktree
  notification-worker/
    LIN-103-add-handler/      ← isolated worktree
```

Issues are topologically sorted by dependencies and executed in parallel batches.

## Prerequisites

- **Go** 1.24+
- **Node.js** 18+ (for building the frontend)
- **Claude CLI** (`claude`) installed and authenticated
- **GitHub CLI** (`gh`) installed and authenticated
- _(optional)_ `LINEAR_API_KEY` env var for Linear integration
- _(optional)_ `NOTION_API_KEY` env var for Notion PRD reading

## Quick Start

```bash
# 1. Clone and configure
cp .env.example .env
cp manifest.example.yaml manifest.yaml
# Edit .env with your API keys and manifest.yaml with your repos/PRD

# 2. Build everything (frontend + backend)
make all

# 3. Start the server
make server
# or: ./sdlc

# 4. Open the Web UI
open http://localhost:3000

# 5. Create a run via the API
curl -X POST http://localhost:3000/api/init \
  -H 'Content-Type: application/json' \
  -d '{
    "prd": "./my-prd.md",
    "repos": [
      { "name": "my-service", "path": "../my-service", "team": "My-Team", "language": "go" }
    ]
  }'

# 6. Run a single phase
curl -X POST http://localhost:3000/api/phases/bootstrap/run

# 7. Or run the full pipeline
curl -X POST http://localhost:3000/api/run
```

## REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/status` | Current run state |
| `GET` | `/api/issues` | Issues grouped by status |
| `GET` | `/api/metrics` | Token/cost metrics |
| `GET` | `/api/phases` | Phase pipeline info |
| `GET` | `/api/manifest` | Read manifest |
| `POST` | `/api/manifest` | Save manifest to disk |
| `POST` | `/api/init` | Create new run from manifest body |
| `POST` | `/api/phases/{name}/run` | Start a phase (async, returns 202) |
| `POST` | `/api/run` | Start full pipeline (async, returns 202) |
| `POST` | `/api/approve` | Approve current gate |
| `GET` | `/api/runs` | List all runs |
| `POST` | `/api/runs/select` | Switch active run |
| `GET` | `/api/health/integrations` | Check Linear/Notion connectivity |
| `GET` | `/api/artifacts/config` | Artifact configuration |
| `GET` | `/api/artifacts/{key}` | Download a generated artifact |
| `GET` | `/api/executions` | List parallel executions |
| `POST` | `/api/executions/` | Manage execution actions |
| `WS` | `/ws/events` | Real-time events |

### `POST /api/init`

Creates a new run. Writes the manifest to disk and initializes the engine.

```json
{
  "prd": "./requirements.md",
  "repos": [
    { "name": "api-gateway", "path": "../api-gateway", "team": "Backend" }
  ]
}
```

Returns: `{ "status": "ok", "run_id": "a1b2c3d4" }`

### `POST /api/phases/{name}/run`

Starts a single phase asynchronously. Valid names: `bootstrap`, `design`, `planning`, `tracking`, `executing`.

Optional body:
```json
{ "params": { "repo": "api-gateway" } }
```

Returns `202`: `{ "status": "started", "phase": "bootstrap" }`

Returns `409` if a phase is already running.

### `POST /api/run`

Starts the full pipeline asynchronously. When it hits an approval gate, the `EventPhaseGate` event fires over WebSocket.

Returns `202`: `{ "status": "started" }`

### `POST /api/approve`

```json
{ "action": "approve" }
```

## Web UI

Open `http://localhost:3000` after starting the server.

| Page | Description |
|------|-------------|
| **Dashboard** | Phase progress bar, active agents, quick stats, recent events |
| **Runs** | Run list, run selector, new execution form |
| **Issues** | Kanban board: Blocked → Ready → Implementing → Reviewing → Awaiting Human → Done |
| **Logs** | Real-time agent output, filterable by agent type and issue |
| **Metrics** | Token usage, cost breakdown by agent, phase timing bars |
| **Artifacts** | View generated documents (scoping doc, PERT) |
| **Manifest Editor** | Edit the manifest from the browser |
| **Approval Gate** | Modal with artifact reference, approve/reject with comments |

Events stream in real-time via WebSocket.

## Configuration

### Environment Variables

```bash
# .env (see .env.example)
PORT=3000            # HTTP server port
DB_PATH=sdlc.db      # Path to SQLite database
LINEAR_API_KEY=       # Enables Linear issue creation in the tracking phase
NOTION_API_KEY=       # Enables reading PRDs from Notion pages
```

### manifest.example.yaml

Copy to `manifest.yaml` before use, or create via `POST /api/init`:

```bash
cp manifest.example.yaml manifest.yaml
```

```yaml
prd: https://notion.so/org/my-prd-page    # or ./local-prd.md

repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend
    language: go          # optional, auto-detected from go.mod/package.json/etc.

  - name: web-app
    path: ../web-app
    team: Frontend
```

## Project Structure

```
cmd/sdlc/                  Server entrypoint
internal/
  engine/                  State machine, event bus, execution manager, metrics
  phase/                   Phase implementations (bootstrap, design, planning, tracking, executing)
  claude/                  Claude CLI wrapper + output parser
  git/                     Worktree manager + gh CLI operations
  integrations/            Linear (GraphQL), Notion (REST), GitHub (CLI)
  prompts/                 Embedded prompt templates with {{variable}} interpolation
  server/                  HTTP + WebSocket server, CORS, SPA routing
  store/                   SQLite persistence (runs, issues, artifacts, metrics)
prompts/                   Agent prompt templates (markdown)
web/                       React 19 frontend (Vite + TypeScript + Tailwind)
  src/components/          Dashboard, IssueBoard, AgentLogs, ArtifactViewer, ApprovalGate
  src/hooks/               useWebSocket, useRuns, useExecution
  src/lib/                 HTTP client utilities
docs/                      Project documentation (Description, User Guide)
```

## Persistence

State is persisted in SQLite (`sdlc.db` by default, WAL mode):

| Table | Content |
|-------|---------|
| `runs` | Run metadata (ID, PRD URL, current phase, status) |
| `repos` | Repositories per run |
| `bootstrap` | Generated CLAUDE.md / ARCHITECTURE.md per repo |
| `artifacts` | Key-value artifact storage (scoping doc, PERT) |
| `issues` | Issues with status, Linear ID, branch, worktree path, PR URL |
| `metrics_entries` | Per-agent token usage and cost |
| `phase_timings` | Duration per phase |

Legacy `state.json` files are automatically migrated on first startup.

## Build

### Using Make

```bash
make all        # Build frontend + backend
make build      # Go binary only (CGO_ENABLED=0)
make web        # Frontend only (npm install + build)
make server     # Build and start
make run        # Build everything and start
make dev        # Frontend dev server with hot reload
make vet        # Go static analysis
make tsc        # TypeScript type check
make tidy       # go mod tidy
make clean      # Remove build artifacts
```

### Manual

```bash
# Go binary
CGO_ENABLED=0 go build -o sdlc ./cmd/sdlc/

# React frontend (outputs to web/dist/, copied to internal/server/static/)
cd web && npm install && npm run build

# Full rebuild
cd web && npm run build && cd .. && go build -o sdlc ./cmd/sdlc/
```

## Development

```bash
# Backend — rebuild on changes
go build -o sdlc ./cmd/sdlc/ && ./sdlc

# Frontend — dev server with hot reload (proxies API to :3000)
cd web && npm run dev

# Type check
cd web && npx tsc -b

# Go lint
go vet ./...
```

## How the Execution Phase Works

1. Issues are topologically sorted by dependency
2. Batched by dependency level (issues with no unmet deps run together)
3. Each issue in a batch runs in a goroutine (limited by concurrency setting):
   - Create git worktree from `origin/main`
   - Invoke Claude (coder agent) to implement the task
   - Invoke Claude (quality-reviewer agent) to review changes
   - If changes requested → invoke feedback-writer agent → re-review (max 3 iterations)
   - Push branch and create PR via `gh`
4. After a batch completes, blocked issues are re-evaluated
5. Next batch starts

## License

Internal use — Yalochat.
