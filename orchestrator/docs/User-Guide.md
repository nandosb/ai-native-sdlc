# Guía de Usuario — Agentic SDLC

## ¿Qué es Agentic SDLC?

Es una extensión de **Claude Code** que automatiza el ciclo de desarrollo de software. Le das un PRD (Product Requirements Document) y la configuración de tus repos, y el sistema:

1. Prepara los repos (genera CLAUDE.md si falta)
2. Diseña la solución técnica (Scoping Doc)
3. Descompone en tareas con dependencias (PERT)
4. Crea issues en Linear
5. Implementa cada tarea (código + tests + PR)
6. Revisa cada PR automáticamente
7. Te pide review humano antes de mergear

Tú mantienes el control en dos puntos clave: apruebas el diseño y apruebas el plan de tareas antes de que se ejecuten.

---

## Prerequisitos

### 1. Claude Code instalado

```bash
# Verificar instalación
claude --version
```

Si no lo tienes: https://docs.anthropic.com/en/docs/claude-code

### 2. GitHub CLI (`gh`) autenticado

```bash
gh auth status
```

Los agentes usan `gh` para crear PRs y leer reviews. Necesita acceso a los repos donde vas a trabajar.

### 3. MCP Servers configurados

El sistema usa dos MCP servers opcionales:

| MCP | Para qué | ¿Obligatorio? |
|-----|----------|----------------|
| **Linear** | Crear issues, gestionar estado | Sí (para fases TRACKING y EXECUTING) |
| **Notion** | Leer PRD, guardar artefactos | No (hay fallback a archivos locales) |

Configúralos en tu Claude Code global o en el proyecto:

```json
// ~/.claude/settings.json o .claude/settings.json
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

Los repos que quieras procesar deben estar clonados en tu máquina. El sistema accede a ellos por path relativo o absoluto.

---

## Setup inicial (una sola vez)

### Paso 1: Clonar el orquestador

```bash
cd ~/go/src/yalochat   # o donde mantengas tus proyectos
git clone git@github.com:yalochat/agentic-sdlc.git
cd agentic-sdlc
```

### Paso 2: Verificar la estructura

```bash
ls .claude/skills/
# Deberías ver: sdlc-approve/  sdlc-resume/  sdlc-run/  sdlc-status/

ls .claude/agents/
# Deberías ver: coder.md  doc-generator.md  feedback-writer.md
#               quality-reviewer.md  session-resumer.md
#               solution-designer.md  task-decomposer.md
```

### Paso 3: Verificar el hook de métricas

```bash
ls -la .claude/hooks/track-agent-metrics.sh
# Debe tener permiso de ejecución (-rwxr-xr-x)

# Si no lo tiene:
chmod +x .claude/hooks/track-agent-metrics.sh
```

### Paso 4: Verificar que `jq` está instalado (para métricas)

```bash
jq --version

# Si no lo tienes:
brew install jq  # macOS
```

---

## Uso: paso a paso

### 1. Configurar el manifest

Edita `manifest.yaml` para tu proyecto. Este archivo le dice al sistema qué repos usar y dónde está el PRD:

```yaml
prd: https://notion.so/tu-org/prd-mi-feature-abc123

repos:
  - name: api-gateway
    path: ../api-gateway          # path relativo desde agentic-sdlc/
    team: Backend                 # nombre del team en Linear
    language: go                  # opcional, se autodetecta

  - name: notification-worker
    path: ../notification-worker
    team: Backend
    # language se autodetecta por go.mod

  - name: shared-events
    path: ../shared-events
    team: Platform
```

**Campos por repo:**

| Campo | Requerido | Descripción |
|-------|-----------|-------------|
| `name` | Sí | Nombre corto (usado en Linear y state.json) |
| `path` | Sí | Path al repo (relativo o absoluto) |
| `team` | Sí | Team de Linear donde crear los issues |
| `language` | No | `go`, `typescript`, `python`. Si no se pone, se detecta por `go.mod`/`package.json`/`pyproject.toml` |
| `skills` | No | Skills adicionales para inyectar al coder de este repo |

### 2. Abrir Claude Code en el directorio del orquestador

```bash
cd *AGENTIC-PATH*/agentic-sdlc
claude
```

> **Importante**: Siempre abre Claude Code desde el directorio `agentic-sdlc/`, no desde los repos individuales. El orquestador vive aquí.

### 3. Iniciar el run

```
/sdlc-run
```

O con un manifest específico:

```
/sdlc-run manifest.yaml
```

El sistema comienza automáticamente con la fase **BOOTSTRAP**:

```
Starting Agentic SDLC run: 2025-02-10-webhook-notifications
PRD: https://notion.so/...
Repos: api-gateway (go), notification-worker (go), shared-events (go)

Phase 1/6: BOOTSTRAP
✓ api-gateway — CLAUDE.md found
✓ shared-events — CLAUDE.md found
✗ notification-worker — no CLAUDE.md, generating...
  [spawning doc-generator] → CLAUDE.md generated, PR created: org/notification-worker#1

Phase 2/6: DESIGN
[Reading PRD from Notion...]
[Spawning solution-designer...]
→ Scoping Document created: https://notion.so/...

⏸ Review the Scoping Document and run /sdlc-approve to continue.
```

### 4. Revisar y aprobar el diseño

El sistema se detuvo en un **approval gate**. Revisa el Scoping Document (en Notion o en `artifacts/scoping-doc.md`) y cuando estés satisfecho:

```
/sdlc-approve
```

Esto avanza a la fase **PLANNING**:

```
Approved: DESIGN phase
Advancing to: PLANNING

[Spawning task-decomposer...]
→ PERT created: https://notion.so/...
  Tasks: 5 tasks across 3 repositories

⏸ Review the PERT and run /sdlc-approve to continue.
```

### 5. Revisar y aprobar el plan de tareas

Revisa el PERT. Verifica que las tareas y dependencias tienen sentido. Cuando estés listo:

```
/sdlc-approve
```

Esto ejecuta **TRACKING** (crea issues en Linear) y comienza **EXECUTING**:

```
Approved: PLANNING phase
Advancing to: TRACKING

Creating issues in Linear...
  ✓ LIN-101: Define WebhookTriggered event (shared-events) — ready
  ✓ LIN-102: Add POST /webhooks endpoint (api-gateway) — ready
  ✓ LIN-103: Publish WebhookTriggered event (api-gateway) — blocked by LIN-101, LIN-102
  ✓ LIN-104: Consume WebhookTriggered event (notification-worker) — blocked by LIN-101
  ✓ LIN-105: Implement retry with backoff (notification-worker) — blocked by LIN-104

EXECUTING (LIN-101): Define WebhookTriggered event
[Spawning coder in shared-events...]
→ PR created: org/shared-events#12
[Spawning quality-reviewer...]
→ APPROVE

PR ready for human review: https://github.com/org/shared-events/pull/12
Merge when ready, then run /sdlc-resume.
```

### 6. Revisar, mergear, y continuar

Ahora es tu turno:

1. **Revisa el PR** en GitHub como lo harías normalmente
2. **Mergea el PR** cuando estés satisfecho
3. **Vuelve a Claude Code** y ejecuta:

```
/sdlc-resume
```

El sistema detecta el merge, actualiza Linear, desbloquea dependencias, y continúa con el siguiente issue:

```
Resume analysis:
PR org/shared-events#12 was merged.
Updated LIN-101 → Done
Unblocked: LIN-103, LIN-104
Continuing with LIN-102: Add POST /webhooks endpoint...
```

### 7. Repetir hasta completar

El ciclo se repite para cada issue:
- El sistema implementa y revisa automáticamente
- Te presenta el PR para review humano
- Tú mergeas
- Ejecutas `/sdlc-resume`

Cuando todos los issues están `done`:

```
All 5 issues implemented and merged!

Run /sdlc-status for detailed metrics.
```

---

## Comandos disponibles

| Comando | Cuándo usarlo |
|---------|---------------|
| `/sdlc-run` | Iniciar un nuevo run desde cero |
| `/sdlc-approve` | Aprobar el diseño (después de DESIGN) o el plan (después de PLANNING) |
| `/sdlc-resume` | Después de mergear un PR, o para retomar una sesión interrumpida |
| `/sdlc-status` | En cualquier momento, para ver progreso y métricas |

---

## Escenarios comunes

### "Se me cerró Claude Code a mitad de ejecución"

No hay problema. El estado se persiste en `state.json`:

```bash
cd ~/go/src/yalochat/agentic-sdlc
claude
```

```
/sdlc-resume
```

El session-resumer analiza el estado real (GitHub + Linear) y retoma donde quedó.

### "Quiero ver en qué va el run"

```
/sdlc-status
```

Muestra: fase actual, issues por estado, métricas de agentes, y costo estimado.

### "El reviewer automático rechazó el PR 3 veces"

Después de 3 iteraciones del loop agente, el sistema escala al humano:

```
Agent review loop reached max iterations (3) for LIN-103.
PR: https://github.com/org/api-gateway/pull/47
Please review manually.
```

Revisa tú, deja comentarios en el PR, y ejecuta `/sdlc-resume`. El coder leerá tus comentarios e iterará.

### "Quiero empezar de cero"

```bash
rm state.json metrics.jsonl
```

Luego:
```
/sdlc-run
```

### "No tengo Notion configurado"

Sin problema. El sistema detecta que el MCP no está disponible y:
- Te pide que pegues el PRD como texto
- Guarda artefactos (Scoping Doc, PERT) en `artifacts/` localmente

### "Quiero agregar un repo a mitad del run"

Actualmente no se soporta. Termina el run actual, actualiza `manifest.yaml`, y empieza uno nuevo.

### "El PR necesita cambios que pidió un reviewer humano"

1. El reviewer deja comentarios en el PR en GitHub
2. Ejecutas `/sdlc-resume`
3. El session-resumer detecta los comentarios
4. Spawna al coder con el feedback del humano
5. El coder hace los cambios y actualiza el PR

---

## Ejemplo completo: de PRD a PRs mergeados

```
# 1. Setup
cd ~/go/src/yalochat/agentic-sdlc
vim manifest.yaml                  # configurar repos y PRD
claude                             # abrir Claude Code

# 2. Ejecutar
> /sdlc-run                        # inicia BOOTSTRAP → DESIGN
                                   # (espera)

> /sdlc-approve                    # aprueba diseño → PLANNING
                                   # (espera)

> /sdlc-approve                    # aprueba plan → TRACKING → EXECUTING
                                   # el sistema implementa el primer issue
                                   # (espera PR review)

# 3. Iterar por cada PR
# -- en GitHub: revisar y mergear PR --
> /sdlc-resume                     # detecta merge, continúa con siguiente issue
# -- en GitHub: revisar y mergear PR --
> /sdlc-resume                     # siguiente issue...
# ... repetir hasta terminar

# 4. Ver resultados
> /sdlc-status                     # resumen final con métricas
```

---

## Estructura de archivos generados

Durante un run, el sistema genera:

```
agentic-sdlc/
  state.json          ← estado del run (fase, issues, métricas)
  metrics.jsonl       ← tokens por invocación de agente
  artifacts/          ← solo si Notion no está disponible
    prd.md
    scoping-doc.md
    pert.md
    pert-tasks.json
```

---

## Tips

1. **Repos limpios**: Asegúrate de que tus repos estén en `main` y sin cambios locales antes de empezar.

2. **PRD detallado**: Mientras más específico sea tu PRD (endpoints, campos, comportamientos), mejor será el diseño y las tareas generadas.

3. **CLAUDE.md en tus repos**: Si tus repos ya tienen `CLAUDE.md`, el sistema los usa directamente (no spawna doc-generator). Un buen CLAUDE.md mejora significativamente la calidad del código generado.

4. **Linear teams**: Los teams en `manifest.yaml` deben coincidir exactamente con los nombres de teams en tu workspace de Linear.

5. **Métricas de costo**: Ejecuta `/sdlc-status` al final para ver cuántos tokens consumió cada agente y el costo estimado.

6. **Sesiones largas**: Para features grandes (>10 tareas), es normal que el run se extienda por varias sesiones de Claude Code. El sistema está diseñado para esto — usa `/sdlc-resume` cada vez que retomes.

7. **Context window**: Si la sesión se pone lenta, cierra y reabre Claude Code. Luego `/sdlc-resume` — el contexto se reconstruye desde state.json.
