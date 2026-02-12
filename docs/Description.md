# Agentic SDLC

Extension de Claude Code que orquesta el ciclo de desarrollo de software: desde un PRD hasta PRs aprobados con tests.

No es un CLI custom. Vive dentro del ecosistema de Claude Code: **skills** (slash commands como `/sdlc`) son el punto de entrada del usuario, y **agents** (sub-agentes con contexto aislado) hacen el trabajo pesado.

---

## 1. Terminologia de Claude Code

El sistema usa cuatro primitivas de Claude Code:

| Primitiva | Ubicacion | Proposito |
|-----------|-----------|-----------|
| **Skill** | `.claude/skills/<name>/SKILL.md` | Slash command (`/name`) que el usuario invoca. Orquesta: valida inputs, delega a agent, guarda outputs, actualiza estado. YAML frontmatter para restricciones. |
| **Agent** | `.claude/agents/<name>/AGENT.md` | Sub-agente con contexto aislado via `context: fork`. Tiene su propio modelo, tools permitidas y system prompt. NO se invoca como `/algo`. |
| **Rule** | `.claude/rules/<name>.md` | Convencion path-scoped que se activa automaticamente cuando se tocan archivos relevantes. YAML frontmatter con `paths:`. |
| **Hook** | `.claude/settings.json` | Evento de lifecycle (PreToolUse, Stop) que ejecuta un comando shell. |

Ejemplo concreto:

- `/sdlc-design` → **Skill** (el usuario lo invoca). Vive en `.claude/skills/sdlc-design/SKILL.md`
- `solution-designer` → **Agent** (el skill lo delega via `agent:` en frontmatter). Vive en `.claude/agents/solution-designer/AGENT.md`
- `state-management` → **Rule** que se activa cuando se toca `.sdlc/**` o skills. Vive en `.claude/rules/state-management.md`

---

## 2. Vision

Dado un **PRD** (Notion o archivo local) como entrada, el sistema orquesta agentes especializados que:

1. Preparan repos sin documentacion (generan CLAUDE.md + ARCHITECTURE.md)
2. Disenan la solucion tecnica → Scoping Doc
3. Crean un plan de ejecucion con dependencias → PERT
4. Registran las tareas en Linear con relaciones de bloqueo
5. Implementan cada tarea (codigo + tests) en un PR independiente
6. Revisan cada PR automaticamente (max 3 iteraciones)
7. Piden review humano antes de continuar

Agnostico al lenguaje. Soporta PRDs que involucran multiples repositorios.

---

## 3. Arquitectura

### 3.1 Estructura de archivos

```
.claude/
  settings.json                     Permisos, hooks
  settings.local.json               Overrides personales (gitignored)

  skills/                           Orchestration (slash commands)
    sdlc/SKILL.md                     /sdlc — pipeline completo
    sdlc-init/SKILL.md                /sdlc-init — configura manifest.yaml
    sdlc-bootstrap/SKILL.md           /sdlc-bootstrap — docs de orientacion
    sdlc-design/SKILL.md              /sdlc-design — PRD → scoping doc
    sdlc-plan/SKILL.md                /sdlc-plan — scoping doc → PERT
    sdlc-track/SKILL.md               /sdlc-track — PERT → Linear issues
    sdlc-execute/SKILL.md             /sdlc-execute — issues → PRs
    sdlc-status/SKILL.md              /sdlc-status — estado read-only

  agents/                           Subagents (expertise aislada)
    doc-generator/AGENT.md            Genera CLAUDE.md + ARCHITECTURE.md
    solution-designer/AGENT.md        PRD → scoping document
    task-decomposer/AGENT.md          Scoping doc → PERT tasks
    linear-issue-creator/AGENT.md     Crea issues en Linear con deps
    coder/AGENT.md                    Implementa codigo en worktrees
    quality-reviewer/AGENT.md         Revisa codigo (read-only, sin edits)

  rules/                            Convenciones path-scoped
    state-management.md               Reglas de lectura/escritura de state.json
    git-worktrees.md                  Convenciones de worktrees y branches
    linear-conventions.md             Reglas de creacion de issues en Linear

manifest.yaml                      Configuracion del proyecto (PRD + repos)
.sdlc/                             Estado del pipeline (runtime, gitignored)
  state.json                         Fase, repos, artefactos, issues
  artifacts/                         scoping-doc.md, pert.md
  worktrees/                         Git worktrees por issue
```

### 3.2 Como funciona

1. El usuario abre Claude Code en el directorio del orquestador
2. Invoca `/sdlc` (pipeline completo) o un skill individual (`/sdlc-design`)
3. El skill valida que sus inputs existen (manifest.yaml, scoping-doc.md, etc.)
4. Si faltan → STOP con "run X first"
5. Si existen → delega al agent correspondiente via `agent:` + `context: fork`
6. El agent trabaja en contexto aislado con tools restringidas
7. El skill guarda los outputs y actualiza `.sdlc/state.json`

### 3.3 Skills orquestan, agents ejecutan

**Principio**: un LLM solo se usa donde se necesita razonamiento, creatividad, o comprension de lenguaje natural. Operaciones deterministicas las hace el skill directamente.

| Operacion | Quien la ejecuta | Por que |
|-----------|-------------------|---------|
| Verificar si existe CLAUDE.md | Skill (Glob) | Deterministico |
| Leer/escribir state.json | Skill (Read/Write) | Deterministico |
| Crear issues en Linear | Agent (linear-issue-creator) | Requiere interpretar PERT + crear relaciones |
| Generar CLAUDE.md para un repo | Agent (doc-generator) | Requiere analisis del codebase |
| Disenar solucion tecnica | Agent (solution-designer) | Requiere razonamiento profundo |
| Descomponer en tareas | Agent (task-decomposer) | Requiere razonamiento sobre dependencias |
| Implementar codigo + tests | Agent (coder) | Requiere creatividad y comprension de dominio |
| Revisar calidad del PR | Agent (quality-reviewer) | Requiere juicio sobre calidad y seguridad |

---

## 4. Agents (6 sub-agentes)

Cada agent es un archivo `AGENT.md` en `.claude/agents/<name>/` con YAML frontmatter. Son delegados por skills via el campo `agent:` en el frontmatter del skill. **No se invocan como `/slash-command`** — eso son skills.

### 4.1 Definiciones

| Agent | Modelo | Tools | Proposito |
|-------|--------|-------|-----------|
| **doc-generator** | sonnet | Read, Glob, Grep, Write, Bash | Genera CLAUDE.md + ARCHITECTURE.md analizando el codebase |
| **solution-designer** | opus | Read, Glob, Grep, Write | Disena solucion tecnica a partir de PRD + contexto de repos |
| **task-decomposer** | opus | Read, Write | Descompone scoping doc en tareas atomicas con dependencias (PERT) |
| **linear-issue-creator** | sonnet | Read, Write | Crea issues en Linear en orden topologico con relaciones de bloqueo |
| **coder** | sonnet | Read, Write, Edit, Glob, Grep, Bash | Implementa codigo + tests en worktrees, crea PRs via `gh` |
| **quality-reviewer** | sonnet | Read, Glob, Grep, Bash | Revisa PRs (read-only: `disallowedTools: Write, Edit`) |

### 4.2 Modelo por agent

| Modelo | Agents | Cuando se usa |
|--------|--------|---------------|
| **Opus** | solution-designer, task-decomposer | Razonamiento profundo: diseno, descomposicion |
| **Sonnet** | doc-generator, coder, linear-issue-creator, quality-reviewer | Ejecucion rapida: codear, crear issues, revisar |

---

## 5. Skills (slash commands)

El usuario interactua con el sistema via skills:

| Skill | Agent | Input → Output |
|-------|-------|----------------|
| `/sdlc-init` | (interactivo) | User input → `manifest.yaml` |
| `/sdlc-bootstrap` | doc-generator | `manifest.yaml` → CLAUDE.md, ARCHITECTURE.md |
| `/sdlc-design` | solution-designer | PRD + repo docs → `scoping-doc.md` |
| `/sdlc-plan` | task-decomposer | `scoping-doc.md` → `pert.md` |
| `/sdlc-track` | linear-issue-creator | `pert.md` → Linear issues |
| `/sdlc-execute` | coder + quality-reviewer | Issues → worktrees → PRs |
| `/sdlc-status` | (read-only) | → resumen formateado |
| `/sdlc` | (orquestador) | encadena todos los anteriores |

Cada skill valida sus inputs antes de ejecutar. Si faltan → STOP con "run X first".

---

## 6. Flujo y fases

```
/sdlc
 │
 ▼
INIT (interactivo)
 Pregunta PRD URL + repos → escribe manifest.yaml + state.json
 │
 ▼
BOOTSTRAP
 Por cada repo en manifest: verifica CLAUDE.md
 Si falta → delega a doc-generator (Sonnet)
 │
 ▼
DESIGN
 Lee PRD (Notion MCP o archivo local)
 Lee CLAUDE.md de cada repo
 Delega a solution-designer (Opus)
 → .sdlc/artifacts/scoping-doc.md
 │
 ▼
PLAN
 Lee scoping-doc.md
 Delega a task-decomposer (Opus)
 → .sdlc/artifacts/pert.md (markdown + JSON)
 │
 ▼
TRACK
 Parsea JSON del PERT
 Delega a linear-issue-creator (Sonnet)
 → Issues en Linear con relaciones de bloqueo
 │
 ▼
EXECUTE (loop por cada issue en orden topologico)
 │
 ├─ Setup: git fetch + worktree en .sdlc/worktrees/<repo>/<slug>/
 ├─ Delega a coder (Sonnet): implementa + tests + commit
 ├─ Delega a quality-reviewer (Sonnet): revisa (max 3 iteraciones)
 ├─ Push + PR via gh
 └─ Actualiza Linear + desbloquea dependientes
```

---

## 7. Estado

### state.json

`.sdlc/state.json` es la unica fuente de verdad del pipeline. Reglas (en `.claude/rules/state-management.md`):

- Siempre leer fresco antes de modificar
- Siempre poner `updated_at` en cada escritura
- Nunca borrar campos — solo actualizar o agregar
- `phase_status`: `""`, `"in_progress"`, `"completed"`, `"failed"`
- `issues[].status`: `"ready"`, `"blocked"`, `"in_progress"`, `"done"`, `"failed"`
- Cuando un issue se completa, verificar dependientes y desbloquear

---

## 8. Git Worktree Isolation

Cada issue se implementa en su propio worktree. El main tree del usuario nunca se toca.

Reglas (en `.claude/rules/git-worktrees.md`):

- Worktrees en `.sdlc/worktrees/<repo-name>/<slug>/`
- Branches: `feat/<slug>`
- Siempre `git fetch origin` antes de crear
- Reusar worktree si ya existe
- Commits referencian Linear ID
- Push con `-u origin`
- PRs via `gh pr create`

---

## 9. Hooks y permisos

Configurados en `.claude/settings.json`:

**Permisos**:
- Allow: Read, Glob, Grep, Bash(git/gh/ls/mkdir/cat)
- Deny: rm -rf /, git push --force, Read(.env)

**Hooks**:
- `PreToolUse[Write]`: recuerda poner `updated_at` al escribir state.json
- `Stop`: muestra fase actual del pipeline al terminar

---

## 10. Soporte multi-repo

### manifest.yaml

```yaml
prd: https://notion.so/org/prd-mi-feature
repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend
  - name: web-app
    path: ../web-app
    team: Frontend
```

### Deteccion de lenguaje

El skill detecta lenguaje automaticamente:
- `go.mod` → Go
- `package.json` → TypeScript/JavaScript
- `pyproject.toml` → Python

### Cross-repo

El task-decomposer produce dependencias cross-repo. El orquestador ejecuta issues en orden topologico y desbloquea dependientes al completar.

---

## 11. Decisiones de diseno

1. **Extension de Claude Code, no CLI custom** — Sin codigo custom para mantener. Todo son archivos .md con YAML frontmatter.
2. **Skills orquestan, agents ejecutan** — Skills validan y guardan estado. Agents proveen expertise aislada.
3. **Validacion estricta de inputs** — Cada skill verifica prerequisitos. Missing → STOP.
4. **Modelo por agent** — Opus para diseno/planificacion (razonamiento profundo), Sonnet para ejecucion (rapido).
5. **Reviewer read-only** — quality-reviewer tiene `disallowedTools: Write, Edit`.
6. **Path-scoped rules** — Convenciones se activan automaticamente.
7. **Hub-and-spoke plano** — No hay anidamiento de agents. Skills delegan directamente.
8. **6 agents totales** — Solo donde se necesita razonamiento. Operaciones deterministicas van en el skill.
