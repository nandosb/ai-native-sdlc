# Agentic SDLC

Extensión de Claude Code que orquesta el ciclo de desarrollo de software: desde un PRD hasta PRs aprobados con tests.

No es un CLI custom. Vive dentro del ecosistema de Claude Code: **skills** (slash commands como `/sdlc-run`) son el punto de entrada del usuario, y **agents** (sub-agentes spawneados via `Task`) hacen el trabajo pesado.

---

## 1. Terminología de Claude Code

El sistema usa tres primitivas de Claude Code. Es importante no confundirlas:

| Primitiva                     | Ubicación                               | ¿Es `/slash-command`? | Propósito                                                                                                                                           |
| ----------------------------- | --------------------------------------- | --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Skill**                     | `.claude/skills/<name>/SKILL.md`        | **Sí** → `/name`      | Instrucciones que el usuario invoca como slash command. Puede incluir archivos de apoyo (templates, scripts). Reemplaza al legacy `commands/`.      |
| **Agent**                     | `.claude/agents/<name>.md`              | **No**                | Sub-agente con contexto aislado, spawneado via la tool `Task`. Tiene su propio modelo, tools permitidas y system prompt. NO se invoca como `/algo`. |
| **Skill precargado en agent** | Campo `skills:` en frontmatter de agent | No directamente       | Skills cuyo contenido completo se inyecta al system prompt del agent al spawnearlo. Sirven como conocimiento, no como commands.                     |

Ejemplo concreto en este sistema:

- `/sdlc-run` → **Skill** (el usuario lo invoca). Vive en `.claude/skills/sdlc-run/SKILL.md`
- `coder` → **Agent** (el orquestador lo spawna via `Task`). Vive en `.claude/agents/coder.md`
- `test-driven-development` → **Skill precargado** dentro del agent `coder` (campo `skills:` en frontmatter). Le inyecta conocimiento sobre TDD al agent cuando arranca

> **Nota sobre commands**: `.claude/commands/` es el formato legacy que fue merged into skills. No lo usamos. Todo va en `.claude/skills/`.

---

## 2. Visión

Dado un **PRD** en Notion como entrada, el sistema orquesta agentes especializados que:

1. Preparan repos sin documentación (generan CLAUDE.md)
2. Diseñan la solución técnica → Scoping Doc (Notion)
3. Crean un plan de ejecución ordenado por repositorio → PERT (Notion)
4. Registran las tareas en Linear con dependencias
5. Implementan cada tarea (código + tests) en un PR independiente
6. Revisan el PR (agente + humano) e iteran hasta merge

Agnóstico al lenguaje. Soporta PRDs que involucran múltiples repositorios.

---

## 3. Arquitectura

### 3.1 Modelo: extensión de Claude Code

El sistema es un conjunto de archivos en `.claude/`:

```
proyecto-orquestador/
  .claude/
    skills/                          ← slash commands del usuario
      sdlc-run/SKILL.md           ← /sdlc-run — inicia un run
      sdlc-approve/SKILL.md       ← /sdlc-approve — aprueba el gate actual
      sdlc-status/SKILL.md        ← /sdlc-status — muestra estado + métricas
      sdlc-resume/SKILL.md        ← /sdlc-resume — retoma un run interrumpido
    agents/                          ← sub-agentes (NO son slash commands)
      doc-generator.md             ← spawneado via Task: genera CLAUDE.md
      solution-designer.md         ← spawneado via Task: diseña solución
      task-decomposer.md           ← spawneado via Task: descompone en tareas
      coder.md                     ← spawneado via Task: implementa código + tests
      quality-reviewer.md          ← spawneado via Task: revisa PRs
      feedback-writer.md           ← spawneado via Task: escribe feedback
      session-resumer.md           ← spawneado via Task: reconstruye contexto
    hooks/
      track-agent-metrics.sh       ← hook SubagentStop: parsea tokens
    settings.json                  ← configura hooks
  CLAUDE.md                        ← convenciones del orquestador
  state.json                       ← estado persistido del run actual
  metrics.jsonl                    ← tokens por invocación (Tier 2)
  manifest.yaml                    ← configuración de repos y PRD
```

### 3.2 Cómo funciona

1. El usuario abre Claude Code en el directorio del orquestador
2. Invoca `/sdlc-run` — esto carga las instrucciones del skill
3. La **sesión principal de Claude Code es el orquestador** — tiene acceso a todas las tools y MCPs
4. Para tareas que requieren razonamiento profundo, el orquestador **spawna agents** via la tool `Task`
5. Los agents en **foreground** pueden usar MCPs (via `mcpServers` en frontmatter); los de **background** no
6. Para operaciones determinísticas (verificar archivos, parsear JSON, llamar MCPs, correr comandos), el orquestador las ejecuta directamente — **sin gastar tokens en agents**
7. El estado se persiste en `state.json` para soportar resume entre sesiones

### 3.3 Qué es un agente vs qué es código

Principio: **un LLM solo se usa donde se necesita razonamiento, creatividad, o comprensión de lenguaje natural.** Operaciones determinísticas son código (Bash, tools nativas).

| Operación | Quién la ejecuta | Por qué |
|---|---|---|
| Verificar si existe CLAUDE.md | Orquestador (Glob) | Determinístico: check de archivo |
| Detectar lenguaje del repo | Orquestador (Read + parse) | Determinístico: leer go.mod, package.json, etc. |
| Leer/escribir state.json | Orquestador (Read/Write) | Determinístico: serialización |
| Correr `gh pr view` para detectar merge | Orquestador (Bash) | Determinístico: comando + parse JSON |
| Crear issues en Linear | Orquestador (Linear MCP) | Determinístico: datos estructurados del PERT |
| Configurar blockers en Linear | Orquestador (Linear MCP) | Determinístico: relaciones del PERT |
| Generar CLAUDE.md para un repo | **Sub-agente** `doc-generator` | Requiere análisis y razonamiento sobre el codebase |
| Diseñar solución técnica | **Sub-agente** `solution-designer` | Requiere razonamiento profundo sobre PRD + contexto |
| Descomponer diseño en tareas | **Sub-agente** `task-decomposer` | Requiere razonamiento sobre granularidad y dependencias |
| Implementar código + tests | **Sub-agente** `coder` | Requiere creatividad, comprensión de dominio |
| Revisar calidad del PR | **Sub-agente** `quality-reviewer` | Requiere juicio sobre calidad, seguridad, patterns |
| Escribir feedback de review | **Sub-agente** `feedback-writer` | Requiere comunicación clara y accionable |
| Reconstruir contexto en resume | **Sub-agente** `session-resumer` (foreground, con Linear MCP) | Requiere razonamiento sobre estado + decisiones |

---

## 4. Agents (7 sub-agentes)

Cada agent es un archivo `.md` en `.claude/agents/` con YAML frontmatter. Son spawneados por el orquestador (sesión principal) via la tool `Task`. **No se invocan como `/slash-command`** — eso son skills. **No hay anidamiento**: los agents no spawnan otros agents. La arquitectura es hub-and-spoke plano.

### 4.1 Definiciones

**`doc-generator.md`** — genera CLAUDE.md + ARCHITECTURE.md

```yaml
---
name: doc-generator
description: Analiza un codebase y genera CLAUDE.md y ARCHITECTURE.md
model: sonnet
tools: Read, Glob, Grep, Write
skills:
  - crafting-effective-readmes
---
```

Usa Read/Glob/Grep (tools nativas de Claude Code) para explorar el repo. **No usa Serena** — Serena es una herramienta de navegación de símbolos, no de análisis de arquitectura. Las tools nativas son suficientes para leer archivos, buscar patterns, y entender estructura.

**`solution-designer.md`** — diseña la solución técnica

```yaml
---
name: solution-designer
description: Diseña la solución técnica a partir de un PRD y contexto de repos
model: opus
tools: Read
skills:
  - architecture-patterns
  - api-design-principles
---
```

Recibe como prompt: PRD (texto) + contexto de cada repo (CLAUDE.md resumido). Produce: Scoping Doc en Markdown. El orquestador lo sube a Notion.

**`task-decomposer.md`** — descompone diseño en tareas

```yaml
---
name: task-decomposer
description: Descompone un diseño técnico en tareas atómicas implementables
model: opus
tools: Read
skills:
  - writing-plans
  - subagent-driven-development
---
```

Recibe: Scoping Doc. Produce: PERT (lista de tareas por repo con dependencias). El orquestador lo sube a Notion y crea issues en Linear.

**`coder.md`** — implementa código + tests

```yaml
---
name: coder
description: Implementa código y tests para una tarea específica en un repo
model: sonnet
tools: Read, Write, Edit, Glob, Grep, Bash
skills:
  - test-driven-development
  - verification-before-completion
  - using-git-worktrees
---
```

Recibe: descripción del issue + acceptance criteria + CLAUDE.md del repo. Trabaja directamente en el repo (el orquestador le pasa el path). Produce: commits en un branch + PR via `gh`.

> Skills por lenguaje se precargan dinámicamente: el orquestador agrega al prompt del coder los skills correspondientes al lenguaje del repo.

**`quality-reviewer.md`** — revisa PRs

```yaml
---
name: quality-reviewer
description: Revisa un PR evaluando calidad, correctitud, seguridad y tests
model: opus
tools: Read, Glob, Grep, Bash
skills:
  - code-review-excellence
  - security-review
---
```

Recibe: URL del PR + diff. Lee el código, corre tests si necesario. Produce: Approve o lista de change requests con comentarios específicos.

**`feedback-writer.md`** — escribe comentarios de review

```yaml
---
name: feedback-writer
description: Convierte observaciones de review en comentarios accionables en GitHub
model: sonnet
tools: Bash
---
```

Recibe: lista de observaciones del quality-reviewer. Produce: comentarios en el PR via `gh pr review`.

**`session-resumer.md`** — reconstruye contexto en resume

```yaml
---
name: session-resumer
description: Analiza state.json y el estado real de GitHub/Linear para determinar siguiente acción
model: sonnet
tools: Read, Bash
mcpServers:
  - linear
skills:
  - session-handoff
---
```

Recibe: state.json + acceso a `gh` (Bash) y Linear MCP (foreground). Produce: instrucciones de qué hacer next para el orquestador. Corre en **foreground** porque necesita MCP y puede requerir interacción.

### 4.2 Skills precargados por lenguaje

Estos no son slash commands — son skills de conocimiento que se inyectan al system prompt del agent via el campo `skills:` en su frontmatter. El orquestador detecta el lenguaje (leyendo go.mod, package.json, pyproject.toml directamente) y agrega los skills correspondientes al prompt del `coder` y `quality-reviewer`:

| Lenguaje | Skills | Source (skills.sh) |
|---|---|---|
| **Go** | `golang-pro`, `go-concurrency-patterns` | jeffallan/claude-skills, wshobson/agents |
| **TypeScript** | `typescript-advanced-types`, `typescript-expert` | wshobson/agents, sickn33/antigravity-awesome-skills |
| **Python** | `python-design-patterns`, `python-testing-patterns`, `python-project-structure` | wshobson/agents |

Se instalan con `npx skills add <owner/repo>`. Overrideable por repo en `manifest.yaml`.

Para lenguajes sin skills en skills.sh: crear custom con `anthropics/skills/skill-creator`.

### 4.3 Modelo por agent

| Modelo | Agents | Cuándo se usa |
|---|---|---|
| **Opus** | solution-designer, task-decomposer, quality-reviewer | Razonamiento profundo: diseño, descomposición, review |
| **Sonnet** | doc-generator, coder, feedback-writer, session-resumer | Ejecución: codear, analizar, escribir feedback |

Solo 7 agents. Opus en 3 (los que realmente necesitan razonamiento profundo), Sonnet en 4.

---

## 5. Skills (slash commands del usuario)

El usuario interactúa con el sistema via skills — archivos en `.claude/skills/` que se invocan como `/slash-commands`. Estos son el punto de entrada; los agents (sección 4) los spawna el orquestador internamente:

### `/sdlc-run`

Inicia un nuevo run. Lee `manifest.yaml`, ejecuta las fases.

```yaml
---
name: sdlc-run
description: Inicia el flujo de Agentic SDLC desde un manifest.yaml
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, Task
argument-hint: "[manifest.yaml]"
---
```

### `/sdlc-approve`

Aprueba el gate actual (diseño o plan).

```yaml
---
name: sdlc-approve
description: Aprueba el approval gate actual del run activo
allowed-tools: Read, Write, Bash, Task
---
```

### `/sdlc-status`

Muestra estado actual del run + métricas (tiempo, tokens, costo estimado).

```yaml
---
name: sdlc-status
description: Muestra estado y métricas del run de Agentic SDLC
allowed-tools: Read, Bash
---
```

### `/sdlc-resume`

Retoma un run interrumpido. Spawna `session-resumer` para analizar estado.

```yaml
---
name: sdlc-resume
description: Retoma un run interrumpido leyendo state.json
allowed-tools: Read, Write, Bash, Task
---
```

### Flujo típico

```
usuario: /sdlc-run manifest.yaml

Claude: Leyendo manifest.yaml...
        Repos: api-gateway (Go), notification-worker (Go), shared-events (Go)

        BOOTSTRAP:
        ✓ api-gateway — CLAUDE.md encontrado
        ✓ shared-events — CLAUDE.md encontrado
        ✗ notification-worker — sin CLAUDE.md, generando...
          [spawna doc-generator] → CLAUDE.md generado, PR creado: org/notification-worker#1

        DESIGN:
        [spawna solution-designer con PRD + contexto de repos]
        → Scoping Doc escrito en Notion: <url>

        ⏸ Revisa el Scoping Doc y ejecuta /sdlc-approve para continuar.

usuario: /sdlc-approve

Claude: PLANNING:
        [spawna task-decomposer con Scoping Doc]
        → PERT escrito en Notion: <url>

        ⏸ Revisa el PERT y ejecuta /sdlc-approve para continuar.

usuario: /sdlc-approve

Claude: TRACKING:
        Creando issues en Linear...
        ✓ LIN-101: Definir evento WebhookTriggered (shared-events)
        ✓ LIN-102: Agregar endpoint POST /webhooks (api-gateway)
        ...

        EXECUTING (LIN-101):
        [spawna coder en shared-events]
        → PR creado: org/shared-events#12
        [spawna quality-reviewer]
        → Approve

        PR listo para review humano: org/shared-events#12
        Merge en GitHub cuando listo. Ejecuta /sdlc-resume después del merge.

usuario: *mergea el PR en GitHub*
usuario: /sdlc-resume

Claude: [spawna session-resumer]
        → PR org/shared-events#12 fue mergeado
        Actualizando Linear LIN-101 → Done
        Desbloqueando LIN-102, LIN-104...
        Continuando con LIN-102...
```

---

## 6. Flujo y fases

### 6.1 Diagrama

```
/sdlc-run manifest.yaml
 │
 ▼
BOOTSTRAP (automático)
 El orquestador verifica CLAUDE.md en cada repo (Glob)
 Si falta → spawna doc-generator (Sonnet, tools: Read/Glob/Grep/Write)
 Detecta lenguaje por presencia de go.mod / package.json / pyproject.toml
 │
 ▼
DESIGN
 Orquestador lee PRD de Notion (Notion MCP, foreground)
 Orquestador lee CLAUDE.md de cada repo (Read)
 Spawna solution-designer (Opus) con PRD + contexto
 Orquestador sube Scoping Doc a Notion (Notion MCP)
 │
 │ ⏸ /sdlc-approve
 ▼
PLANNING
 Spawna task-decomposer (Opus) con Scoping Doc
 Orquestador sube PERT a Notion (Notion MCP)
 │
 │ ⏸ /sdlc-approve
 ▼
TRACKING
 Orquestador crea issues en Linear (Linear MCP) — sin sub-agente
 Orquestador configura blockers (Linear MCP) — sin sub-agente
 │
 ▼
EXECUTING (loop por cada issue, respetando topological sort)
 │
 ├─ Orquestador hace git pull origin main en el repo
 ├─ Spawna coder (Sonnet) con issue + CLAUDE.md del repo
 ├─ Spawna quality-reviewer (Opus) con el diff del PR
 │   ├─ Si rechaza → spawna feedback-writer (Sonnet)
 │   │              → spawna coder de nuevo (con feedback, max 3x)
 │   └─ Si aprueba → PR listo para review humano
 │
 ├─ ⏸ Esperando review humano
 │   Orquestador verifica merge via: gh pr view (Bash, polling o manual)
 │   Si humano pide cambios → spawna coder con comentarios del humano
 │   Si humano mergea → actualizar Linear (MCP), desbloquear, siguiente issue
 │
 └─ Cuando todos los issues están done → COMPLETED
```

### 6.2 MCP en agents: foreground vs background

Los agents soportan MCP servers via el campo `mcpServers` en su frontmatter. **La restricción es solo para agents en background**:

| Modo | MCP disponible | Permisos interactivos | Cuándo se usa |
|---|---|---|---|
| **Foreground** | **Sí** | Sí (pasa prompts al usuario) | Agent necesita MCP o interacción |
| **Background** | **No** | No (auto-deny) | Tareas independientes en paralelo |

**Decisión de diseño**: aunque los agents en foreground **pueden** usar MCP, mantenemos las operaciones determinísticas de MCP (crear issues, leer PRD, subir docs) en el orquestador por principio: no gastar tokens de LLM en llamadas estructuradas. La excepción es `session-resumer`, que necesita verificar estado en Linear como parte de su razonamiento.

- El orquestador hace las llamadas a **Notion MCP y Linear MCP** cuando son operaciones determinísticas (datos estructurados, CRUD)
- `session-resumer` tiene acceso a **Linear MCP** (foreground) porque necesita razonar sobre el estado real
- `coder` usa `gh` CLI via Bash (no MCP) para crear PRs — más simple y funciona en background
- El resto de agents solo usan tools nativas (Read, Write, Edit, Glob, Grep, Bash)

---

## 7. Estado y resume

### 7.1 state.json

```json
{
  "run_id": "2025-02-10-webhook-notifications",
  "prd_url": "https://notion.so/...",
  "phase": "EXECUTING",
  "phase_status": "awaiting_human",
  "bootstrap": {
    "languages": { "api-gateway": "go", "notification-worker": "go", "shared-events": "go" },
    "docs_generated": { "notification-worker": "org/notification-worker#1" }
  },
  "artifacts": {
    "scoping_doc": "https://notion.so/...",
    "pert": "https://notion.so/..."
  },
  "issues": {
    "LIN-101": { "repo": "shared-events", "status": "done", "pr": "org/shared-events#12" },
    "LIN-102": { "repo": "api-gateway", "status": "done", "pr": "org/api-gateway#45" },
    "LIN-103": { "repo": "api-gateway", "status": "awaiting_human", "pr": "org/api-gateway#47", "review_iterations": 1 },
    "LIN-104": { "repo": "notification-worker", "status": "blocked", "blocked_by": ["LIN-101"] },
    "LIN-105": { "repo": "notification-worker", "status": "blocked", "blocked_by": ["LIN-104"] }
  },
  "updated_at": "2025-02-10T14:32:00Z",
  "metrics": {
    "started_at": "2025-02-10T10:00:00Z",
    "phases": {
      "BOOTSTRAP": { "started_at": "...", "ended_at": "...", "duration_s": 45 },
      "DESIGN": { "started_at": "...", "ended_at": "...", "duration_s": 120 }
    },
    "totals": {
      "agent_spawns": 8,
      "review_iterations": { "agent": 2, "human": 1 },
      "issues_created": 5,
      "prs_created": 3,
      "prs_merged": 2
    }
  }
}
```

El orquestador escribe state.json **antes** de cada acción para soportar resume.

### 7.2 Resume

Cuando el usuario ejecuta `/sdlc-resume`:

1. El orquestador lee `state.json`
2. Spawna `session-resumer` (Sonnet, **foreground**) que analiza:
   - Estado en state.json
   - Estado real en GitHub (`gh pr view` via Bash)
   - Estado real en Linear (MCP directo — tiene `mcpServers: [linear]`)
3. `session-resumer` retorna instrucciones de qué hacer
4. El orquestador ejecuta

| Estado | Qué hace resume |
|---|---|
| `DESIGN` (sin scoping doc) | Re-ejecuta: lee PRD, spawna solution-designer |
| `DESIGN.awaiting_approval` | Muestra link al scoping doc, espera /sdlc-approve |
| `PLANNING.awaiting_approval` | Muestra link al PERT, espera /sdlc-approve |
| `TRACKING` (parcial) | Verifica issues en Linear por título, crea faltantes |
| `EXECUTING.implementing` | Verifica si branch/PR existen, continúa o re-ejecuta coder |
| `EXECUTING.awaiting_human` | Verifica merge status en GitHub, actúa según resultado |
| `COMPLETED` | Informa que ya terminó |

### 7.3 Idempotencia

| Operación | Cómo se garantiza |
|---|---|
| Generar CLAUDE.md | doc-generator verifica si ya existe antes de generar |
| Scoping Doc en Notion | Orquestador busca página con mismo título bajo PRD. Si existe, sobreescribe |
| Issues en Linear | Orquestador busca por título exacto antes de crear |
| Branch + PR | coder verifica si branch existe (`git branch --list`), si PR existe (`gh pr list`) |

---

## 8. Métricas

Cada run recopila métricas automáticamente. Tres niveles de detalle según configuración.

### 8.1 Tier 1: Determinístico (siempre disponible)

El orquestador registra timestamps y contadores directamente en `state.json`. No requiere configuración — son operaciones de Write antes y después de cada Task call.

| Métrica | Cómo se captura |
|---|---|
| Tiempo por fase | Timestamps start/end en `state.json` |
| Tiempo por invocación de sub-agente | Timestamps antes/después de cada `Task` |
| Spawns por tipo de agente | Contador incrementado en cada `Task` |
| Iteraciones de review (agente) | Contador del loop quality-reviewer ↔ coder |
| Iteraciones de review (humano) | Contador incrementado en cada `/sdlc-resume` post-feedback |
| Issues creados / PRs creados / PRs mergeados | Contadores incrementados en cada operación |

### 8.2 Tier 2: Tokens via hooks (requiere setup)

Claude Code expone un hook `SubagentStop` que recibe `agent_transcript_path` — el archivo JSONL con el transcript del sub-agente. Un script de hook lo parsea para extraer tokens consumidos.

**Configuración** en `.claude/settings.json`:

```json
{
  "hooks": {
    "SubagentStop": [{
      "command": ".claude/hooks/track-agent-metrics.sh",
      "timeout": 5000
    }]
  }
}
```

**`.claude/hooks/track-agent-metrics.sh`**:

```bash
#!/bin/bash
# Parsea el transcript del sub-agente y extrae tokens
METRICS_FILE="metrics.jsonl"

TOKENS=$(cat "$agent_transcript_path" \
  | jq -s '[.[] | .usage // empty] | {
      input: (map(.input_tokens) | add),
      output: (map(.output_tokens) | add)
    }')

echo "{\"agent_id\": \"$agent_id\", \"agent_type\": \"$agent_type\", \"tokens\": $TOKENS, \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" \
  >> "$METRICS_FILE"
```

> **Limitación**: el formato del transcript JSONL no es una API estable de Claude Code. Puede cambiar entre versiones.

### 8.3 Tier 3: Herramientas externas

| Herramienta | Qué hace | Cuándo usarla |
|---|---|---|
| `/cost` | Costo total de la sesión interactiva | Al final del run |
| `ccusage` | Analiza logs históricos de sesiones Claude Code | Post-mortem, reportes semanales |
| OpenTelemetry | Exporta `claude_code.token.usage` y `claude_code.cost.usage` a backend | Enterprise, dashboards de monitoreo |

OpenTelemetry se habilita con:

```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_METRICS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

### 8.4 Estructura en state.json

```json
{
  "metrics": {
    "started_at": "2025-02-10T10:00:00Z",
    "phases": {
      "BOOTSTRAP": { "started_at": "...", "ended_at": "...", "duration_s": 45 },
      "DESIGN": { "started_at": "...", "ended_at": "...", "duration_s": 120 },
      "PLANNING": { "started_at": "...", "ended_at": "...", "duration_s": 90 }
    },
    "agent_invocations": [
      {
        "agent": "doc-generator",
        "model": "sonnet",
        "repo": "notification-worker",
        "started_at": "2025-02-10T10:00:30Z",
        "duration_s": 32,
        "tokens": { "input": 15200, "output": 2800 }
      },
      {
        "agent": "solution-designer",
        "model": "opus",
        "started_at": "2025-02-10T10:01:15Z",
        "duration_s": 85,
        "tokens": { "input": 42000, "output": 8500 }
      }
    ],
    "totals": {
      "duration_s": 1800,
      "agent_spawns": 12,
      "review_iterations": { "agent": 3, "human": 2 },
      "issues_created": 5,
      "prs_created": 5,
      "prs_merged": 5
    }
  }
}
```

### 8.5 Costo estimado

Se calcula post-run a partir de tokens capturados (Tier 2) y pricing por modelo:

| Modelo | Input (por 1M tokens) | Output (por 1M tokens) |
|---|---|---|
| **Opus** | $15.00 | $75.00 |
| **Sonnet** | $3.00 | $15.00 |

Fórmula: `costo = Σ (input_tokens × precio_input + output_tokens × precio_output)` por cada invocación.

> Precios sujetos a cambios. Fuente: https://docs.anthropic.com/en/docs/about-claude/pricing

### 8.6 Resumen de /sdlc-status

El skill `/sdlc-status` lee `state.json` y `metrics.jsonl` para mostrar un resumen:

```
Run: 2025-02-10-webhook-notifications
Phase: EXECUTING (3/5 issues done)
Duration: 32min

Agents:
  doc-generator    ×1   32s    18K tokens   ~$0.07
  solution-designer ×1   85s    50K tokens   ~$4.50
  task-decomposer  ×1   60s    35K tokens   ~$3.15
  coder            ×4  240s   120K tokens   ~$0.54
  quality-reviewer ×4  180s    80K tokens   ~$7.20
  feedback-writer  ×1   15s     5K tokens   ~$0.02

Total: ~$15.48 | 308K tokens | 12 agent spawns
```

### 8.7 Limitaciones de métricas

| Limitación | Impacto | Mitigación |
|---|---|---|
| Tokens del orquestador no se capturan por hook | Solo se miden tokens de sub-agentes, no de la sesión principal | `/cost` al final de sesión da el total real |
| Transcript JSONL no es API estable | Hook de parsing puede romperse en updates de Claude Code | Mantener el hook simple, validar formato |
| `/cost` solo en modo interactivo | No disponible en headless/CI | Usar OpenTelemetry para CI |
| Tokens de Tier 2 son aproximados | Parsing manual puede omitir cache tokens | Aceptable para estimación de costo |

---

## 9. MCP Servers

| MCP Server | Quién lo usa | Para qué | Limitaciones conocidas |
|---|---|---|---|
| **Linear** | Orquestador (directo) + `session-resumer` (foreground, via `mcpServers`) | CRUD de issues, blockers, estado. Resume: verificar estado real | — |
| **Notion** | Orquestador (directo) | Leer PRD, crear Scoping Doc/PERT como páginas hijas | No controla orden de páginas. Warning de posible sunsetting |
| **Context7** | coder (via prompt) | Docs de librerías populares al codear | **Solo catálogo curado** (React, Next.js, Express, etc.). Falla silenciosamente para libs no indexadas. Cobertura limitada fuera de JS/TS |

### 9.1 Qué NO usamos

**Serena**: descartada. Es una herramienta de navegación de símbolos via LSP (find_symbol, replace_symbol_body, etc.), no de análisis de codebase ni generación de documentación. Las tools nativas de Claude Code (Read, Glob, Grep) son suficientes para entender un repo.

**GitHub MCP**: innecesario. `gh` CLI via Bash cubre todo lo que necesitamos (crear PRs, ver merge status, leer reviews).

---

## 10. Estructura en Notion

```
Workspace
 └── Agentic SDLC (database)
      └── [Proyecto: Webhook Notifications]
           ├── PRD                    ← ya existe
           ├── Scoping Doc            ← generado por solution-designer
           └── PERT                   ← generado por task-decomposer
```

**Convenciones**:
- Scoping Doc y PERT se crean como páginas hijas del PRD via Notion MCP
- Properties de la database: `Status`, `PRD URL`, `Run ID`
- Limitación: no se puede controlar el orden de páginas hijas en Notion

---

## 11. Soporte multi-repo

### 11.1 manifest.yaml

```yaml
prd: https://notion.so/org/prd-webhook-notifications

repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend                    # team en Linear
    language: go                     # opcional, se autodetecta si no se pone
    skills:                          # skills adicionales para coder en este repo
      - wshobson/agents/go-concurrency-patterns

  - name: notification-worker
    path: ../notification-worker
    team: Backend

  - name: shared-events
    path: ../shared-events
    team: Platform
```

### 11.2 Detección de lenguaje

El orquestador (no un sub-agente) detecta el lenguaje directamente:

1. Si `manifest.yaml` tiene `language` → usar ese
2. Si no → verificar presencia de archivos (Glob):
   - `go.mod` → Go
   - `package.json` → TypeScript/JavaScript
   - `pyproject.toml` o `setup.py` → Python
3. Si no se detecta → notificar al usuario

### 11.3 Cross-repo

El task-decomposer produce un PERT con dependencias cross-repo:

```
shared-events:
  1. Definir evento WebhookTriggered         [no blockers]

api-gateway:
  2. Agregar endpoint POST /webhooks         [no blockers]
  3. Publicar evento WebhookTriggered        [blocked by: shared-events#1, api-gateway#2]

notification-worker:
  4. Consumir evento WebhookTriggered        [blocked by: shared-events#1]
  5. Implementar retry con backoff           [blocked by: notification-worker#4]
```

El orquestador hace topological sort y ejecuta issues respetando el orden. Cuando un PR se mergea, actualiza Linear y desbloquea issues dependientes.

---

## 12. Proyectos agent-friendly

### 12.1 CLAUDE.md

```markdown
# Proyecto X

## Stack
- Lenguaje: Go 1.22
- Framework: Chi router
- DB: PostgreSQL 15 + sqlc

## Comandos
- Build: `make build`
- Test: `make test`
- Lint: `make lint`

## Estructura
- cmd/           → entrypoints
- internal/api/  → handlers HTTP
- internal/svc/  → lógica de negocio
- internal/repo/ → acceso a datos

## Convenciones
- Errors: fmt.Errorf("contexto: %w", err)
- Tests: tabla-driven, _test.go por archivo fuente
- PRs: un feature por PR, ~300 líneas máximo
```

### 12.2 Si no existe CLAUDE.md

El sub-agente `doc-generator` (Sonnet) lo genera:
1. Lee la estructura del repo (Glob)
2. Lee archivos clave: README, go.mod/package.json, Makefile, configs
3. Lee ejemplos de código (2-3 archivos representativos)
4. Genera CLAUDE.md + ARCHITECTURE.md
5. Crea un PR en el repo con los docs

No usa Serena ni ningún MCP especial. Solo Read/Glob/Grep — las tools más baratas y confiables.

### 12.3 Preparación mínima

Requisitos reales (no generables):
- [ ] CI funcional
- [ ] Al menos un ejemplo de cada patrón (handler, service, test)

Generables por doc-generator:
- [ ] CLAUDE.md
- [ ] docs/ARCHITECTURE.md
- [ ] .claudeignore

---

## 13. Decisiones de diseño

### 13.1 Extensión de Claude Code, no CLI custom

Ventajas:
- Sin código custom para mantener — todo son archivos .md
- Hereda todo el ecosistema: MCPs, skills, sessions, hooks
- El usuario ya conoce Claude Code

Trade-off: dependemos del runtime de Claude Code. Si Anthropic cambia algo, nos afecta.

### 13.2 Hub-and-spoke plano

Los agents **no pueden anidar** (limitación de Claude Code). El orquestador spawna todos los agents directamente. No hay "agents compuestos de agents".

### 13.3 Solo 7 agents

De 19 en el diseño anterior a 7. Todo lo que no requiere razonamiento es código (Bash, tools nativas, MCP directo).

### 13.4 Dos loops de review

| Loop | Participantes | Propósito | Máx iteraciones |
|---|---|---|---|
| **Agente** | quality-reviewer ↔ coder | Convenciones, bugs obvios, tests faltantes | 3 |
| **Humano** | Developer ↔ coder | Dominio, lógica de negocio, edge cases | Sin límite |

### 13.5 Context window del orquestador

El orquestador (sesión principal de Claude Code) tiene ~200K tokens de context window. Riesgos:

- Un PRD largo + múltiples CLAUDE.md + scoping doc + PERT pueden consumir mucho contexto
- **Mitigación**: los sub-agentes tienen su propio context window (~200K cada uno). El orquestador solo mantiene resúmenes, no los artefactos completos
- **Mitigación**: Claude Code tiene auto-compaction cuando se acerca al límite
- **Mitigación**: el skill `session-handoff` permite transferir contexto entre sesiones si se agota

---

## 14. Ejemplo: PRD multi-repo

### PRD: Sistema de Webhook Notifications

**Contexto**: La plataforma permite a merchants configurar integraciones. Necesitamos webhooks para notificaciones en tiempo real de eventos.

**Repos involucrados**:
- `api-gateway` — REST API existente (Go + Chi + PostgreSQL)
- `notification-worker` — nuevo servicio worker (Go + NATS consumer)
- `shared-events` — librería compartida de eventos (Go module)

**Funcionalidades**:

1. **Registro de webhooks**: CRUD de subscriptions (URL, eventos, secret HMAC). Validación de URL (challenge GET). Máx 10 por merchant.

2. **Dispatch**: publicar eventos a bus, worker consume y POST al URL destino. Firma HMAC-SHA256. Headers: `X-Webhook-Signature`, `X-Webhook-Event`, `X-Webhook-Delivery`.

3. **Retry**: exponential backoff (1s, 5s, 30s, 2min, 10min). Desactivar webhook tras 3 failures consecutivos. Historial de deliveries (7 días).

4. **Eventos v1**: `message.created`, `conversation.status_changed`, `contact.updated`.

**Requisitos técnicos**: dispatch <5s, rate limit 1000/min por merchant, worker stateless.

**Fuera de alcance**: UI, eventos de billing, transformación de payloads.

---

## 15. Limitaciones conocidas

| Limitación | Impacto | Mitigación |
|---|---|---|
| Agents no anidan | Arquitectura forzada a hub-and-spoke plano | Diseño simplificado: 7 agents directos |
| MCP no disponible en agents **background** | Agents en background no pueden usar Notion/Linear MCP | Agents que necesitan MCP corren en foreground (`session-resumer`). El resto usa tools nativas |
| Context7 solo catálogo curado | Falla para libs nicho/internas (especialmente fuera de JS/TS) | Documentar en CLAUDE.md las libs que usa el proyecto |
| Notion MCP puede ser sunset | Riesgo de dependencia | Artefactos en Notion son convenientes pero no críticos — el sistema funciona con state.json + Linear |
| Notion no controla orden de páginas | Scoping Doc y PERT pueden aparecer desordenados | Prefijos numéricos en títulos: "1. Scoping Doc", "2. PERT" |
| Context window del orquestador (~200K) | PRDs muy grandes con muchos repos pueden agotarlo | Auto-compaction + session-handoff + agents con su propio context |
| Agent Teams (experimental) | El loop reviewer↔coder pasa por el orquestador | Aceptable para v1. Evaluar Agent Teams cuando sea estable |
| Tokens del orquestador no se capturan por hook | Métricas de Tier 2 solo cubren agents, no la sesión principal | `/cost` al final da el total real. OpenTelemetry para tracking completo |
| Transcript JSONL no es API estable | Hook de parsing de tokens puede romperse en updates | Mantener hook simple, validar formato antes de parsear |

---

## 16. Preguntas abiertas

- ¿Cómo testear el sistema end-to-end? (repos de prueba con PRDs sintéticos)
- ¿Cuándo migrar de Notion MCP a alternativa si lo sunset-ean?
- ¿Vale la pena evaluar Agent Teams para el loop de review cuando salga de experimental?
