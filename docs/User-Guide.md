# Guia de Usuario — Agentic SDLC

## Que es Agentic SDLC?

Es una extension de **Claude Code** que automatiza el ciclo de desarrollo de software. Le das un PRD (Product Requirements Document) y la configuracion de tus repos, y el sistema:

1. Prepara los repos (genera CLAUDE.md + ARCHITECTURE.md si faltan)
2. Disena la solucion tecnica (Scoping Doc)
3. Descompone en tareas con dependencias (PERT)
4. Crea issues en Linear
5. Implementa cada tarea (codigo + tests + PR)
6. Revisa cada PR automaticamente (max 3 iteraciones)
7. Te pide review humano antes de continuar

---

## Prerequisitos

### 1. Claude Code instalado

```bash
claude --version
```

Si no lo tienes: https://docs.anthropic.com/en/docs/claude-code

### 2. GitHub CLI (`gh`) autenticado

```bash
gh auth status
```

Los agentes usan `gh` para crear PRs y leer reviews.

### 3. MCP Servers configurados

| MCP | Para que | Obligatorio? |
|-----|----------|--------------|
| **Linear** | Crear issues, gestionar estado | Si (para fase track y execute) |
| **Notion** | Leer PRD desde Notion | No (puedes usar un archivo local) |

Configuralos en tu Claude Code:

```json
// ~/.claude/settings.json
{
  "mcpServers": {
    "linear": {
      "command": "npx",
      "args": ["-y", "@anthropic/linear-mcp-server"],
      "env": { "LINEAR_API_KEY": "lin_api_..." }
    },
    "notion": {
      "command": "npx",
      "args": ["-y", "@anthropic/notion-mcp-server"],
      "env": { "NOTION_API_KEY": "ntn_..." }
    }
  }
}
```

### 4. Repos accesibles localmente

Los repos que quieras procesar deben estar clonados en tu maquina.

---

## Setup inicial

### Paso 1: Clonar el orquestador

```bash
git clone git@github.com:yalochat/agentic-sdlc.git
cd agentic-sdlc
```

### Paso 2: Verificar la estructura

```bash
ls .claude/skills/
# sdlc/  sdlc-bootstrap/  sdlc-design/  sdlc-execute/
# sdlc-init/  sdlc-plan/  sdlc-status/  sdlc-track/

ls .claude/agents/
# coder/  doc-generator/  linear-issue-creator/
# quality-reviewer/  solution-designer/  task-decomposer/
```

---

## Uso: paso a paso

### 1. Abrir Claude Code en el directorio del orquestador

```bash
cd agentic-sdlc
claude
```

> **Importante**: Siempre abre Claude Code desde el directorio `agentic-sdlc/`, no desde los repos individuales.

### 2. Pipeline completo

```
/sdlc
```

Esto ejecuta todas las fases en orden, resumiendo desde el ultimo checkpoint si ya hay estado previo.

### 3. O paso a paso

```
/sdlc-init         # Configura PRD + repos → manifest.yaml
/sdlc-bootstrap    # manifest.yaml → CLAUDE.md + ARCHITECTURE.md por repo
/sdlc-design       # PRD + repo docs → scoping-doc.md
/sdlc-plan         # scoping-doc.md → pert.md
/sdlc-track        # pert.md → issues en Linear
/sdlc-execute      # issues → worktrees → PRs
/sdlc-status       # Ver estado en cualquier momento
```

Cada skill valida que sus inputs existan. Si faltan, te dice que skill ejecutar primero.

---

## Flujo tipico

```
> /sdlc-init

Claude: Cual es la URL del PRD?
Tu:     https://notion.so/org/mi-prd-abc123

Claude: Agrega un repositorio.
        Nombre: booking-app
        Path: /Users/tu/code/booking-app
        Team en Linear: Ignite-2026

Claude: manifest.yaml creado. State inicializado.

> /sdlc-bootstrap

Claude: Analizando booking-app...
        CLAUDE.md no encontrado, generando...
        [delega a doc-generator]
        CLAUDE.md + ARCHITECTURE.md generados.

> /sdlc-design

Claude: Leyendo PRD desde Notion...
        Leyendo CLAUDE.md de booking-app...
        [delega a solution-designer]
        scoping-doc.md guardado en .sdlc/artifacts/

> /sdlc-plan

Claude: Leyendo scoping-doc.md...
        [delega a task-decomposer]
        pert.md guardado en .sdlc/artifacts/
        5 tareas identificadas con dependencias.

> /sdlc-track

Claude: Parseando PERT...
        [delega a linear-issue-creator]
        Issue 1: Add data model (ready)
        Issue 2: Add API endpoints (blocked by #1)
        Issue 3: Add booking UI (blocked by #2)
        Issue 4: Integration tests (blocked by #2, #3)
        Issue 5: Deploy config (blocked by #4)

> /sdlc-execute

Claude: Issue #1 (Add data model) — ready
        Creando worktree en .sdlc/worktrees/booking-app/add-data-model/
        [delega a coder] → implementa + tests
        [delega a quality-reviewer] → approve
        PR creado: org/booking-app#1

        Issue #2 (Add API endpoints) — desbloqueado
        ...
```

---

## Comandos disponibles

| Comando | Cuando usarlo |
|---------|---------------|
| `/sdlc` | Pipeline completo (resume desde checkpoint) |
| `/sdlc-init` | Configurar PRD y repos por primera vez |
| `/sdlc-bootstrap` | Generar docs de orientacion para repos |
| `/sdlc-design` | Disenar solucion tecnica desde PRD |
| `/sdlc-plan` | Descomponer diseno en tareas |
| `/sdlc-track` | Crear issues en Linear |
| `/sdlc-execute` | Implementar issues en worktrees → PRs |
| `/sdlc-status` | Ver estado del pipeline (read-only) |

---

## Escenarios comunes

### "Se me cerro Claude Code a mitad de ejecucion"

No hay problema. El estado se persiste en `.sdlc/state.json`:

```bash
cd agentic-sdlc
claude
```

```
/sdlc
```

El pipeline resume desde el ultimo checkpoint.

### "Quiero ver en que va el pipeline"

```
/sdlc-status
```

### "Quiero empezar de cero"

```bash
rm -rf .sdlc/
```

Luego:
```
/sdlc-init
```

### "No tengo Notion configurado"

Sin problema. En `/sdlc-init` puedes apuntar a un archivo local como PRD:

```yaml
prd: ./docs/mi-prd.md
```

### "El reviewer automatico rechazo el PR 3 veces"

Despues de 3 iteraciones, el sistema escala al humano. Revisa el PR manualmente.

---

## Configuracion: manifest.yaml

```yaml
prd: https://notion.so/org/mi-prd-abc123    # o ./local-prd.md
repos:
  - name: api-gateway
    path: ../api-gateway
    team: Backend
  - name: web-app
    path: ../web-app
    team: Frontend
```

| Campo | Requerido | Descripcion |
|-------|-----------|-------------|
| `prd` | Si | URL de Notion o path local al PRD |
| `repos[].name` | Si | Nombre corto (usado en Linear y state) |
| `repos[].path` | Si | Path al repo (relativo o absoluto) |
| `repos[].team` | Si | Team de Linear donde crear los issues |

Crea interactivamente con `/sdlc-init` o copia de `manifest.example.yaml`.

---

## Tips

1. **Repos limpios**: Asegurate de que tus repos esten en `main` y sin cambios locales antes de empezar.

2. **PRD detallado**: Mientras mas especifico sea tu PRD (endpoints, campos, comportamientos), mejor sera el diseno y las tareas generadas.

3. **CLAUDE.md en tus repos**: Si tus repos ya tienen `CLAUDE.md`, el sistema los usa directamente (no genera nuevos). Un buen CLAUDE.md mejora la calidad del codigo generado.

4. **Linear teams**: Los teams en tu `manifest.yaml` deben coincidir exactamente con los nombres de teams en tu workspace de Linear.

5. **Sesiones largas**: Para features grandes (>10 tareas), es normal que el pipeline se extienda por varias sesiones. Usa `/sdlc` para resumir.
