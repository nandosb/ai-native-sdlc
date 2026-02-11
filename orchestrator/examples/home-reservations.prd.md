# Home de Reservaciones

## Problem Statement

Actualmente la app de reservaciones es un flujo lineal que lleva directamente a crear una nueva reserva, sin ofrecer visibilidad sobre las reservas existentes. El usuario no tiene forma de ver qué reservas se han creado ni gestionar su contexto antes de iniciar una nueva. Esto limita la utilidad de la demo y no refleja un flujo realista de una app de reservaciones.

## Evidence

- El flujo actual lleva directo a crear una reserva sin opción de ver las existentes
- Las reservas se almacenan pero no hay forma de consultarlas desde la interfaz
- Assumption - needs validation: un Home con lista de reservas mejora la percepción de la demo como producto completo

## Proposed Solution

Agregar una pantalla Home que sea el nuevo punto de entrada de la app, mostrando una lista visual (cards) de todas las reservas existentes con sus datos completos (fecha, hora, servicio, comensales), y un botón prominente para crear una nueva reserva que inicie el flujo de booking actual.

## Key Hypothesis

Creemos que una pantalla Home con listado de reservas mejorará la experiencia de la demo haciéndola sentir como un producto completo para cualquier persona que pruebe la app.
Sabremos que es correcto cuando el Home muestre las reservas existentes y el flujo de crear reserva siga funcionando sin fricciones.

## What We're NOT Building

- **Autenticación/control de usuarios** - es una demo sin usuarios, se muestran todas las reservas
- **Edición de reservas** - fuera de alcance de v1
- **Eliminación de reservas** - fuera de alcance de v1
- **Filtros o búsqueda** - no necesario para una demo
- **Paginación** - el volumen de datos de la demo no lo requiere

## Success Metrics

| Metric | Target | How Measured |
|--------|--------|--------------|
| Home muestra reservas existentes | 100% de reservas visibles | Verificación manual |
| Flujo de crear reserva funciona desde Home | Sin regresiones | Prueba manual end-to-end |
| Navegación Home <-> Booking fluida | Transiciones consistentes con diseño actual | Verificación visual |

## Open Questions

- [ ] Definir orden de las reservas en la lista (por fecha de reserva o fecha de creación)
- [ ] Decidir si mostrar un estado vacío con CTA cuando no hay reservas

---

## Users & Context

**Primary User**
- **Who**: Cualquier persona que acceda a la demo de reservaciones
- **Current behavior**: Acceden directamente al flujo de crear una reserva sin ver las existentes
- **Trigger**: Abren la app y quieren ver las reservas actuales o crear una nueva
- **Success state**: Ven una lista clara de reservas existentes y pueden iniciar el flujo de nueva reserva con un click

**Job to Be Done**
Cuando abro la app de reservaciones, quiero ver las reservas existentes y tener un botón para crear una nueva, para poder entender el estado actual y decidir si crear una reserva adicional.

**Non-Users**
No aplica - es una demo abierta sin restricciones de acceso.

---

## Solution Detail

### Core Capabilities (MoSCoW)

| Priority | Capability | Rationale |
|----------|------------|-----------|
| Must | Pantalla Home como nuevo punto de entrada de la app | El usuario necesita una vista general antes de actuar |
| Must | Lista de reservas en formato cards | Consistente con el diseño visual actual de la app |
| Must | Mostrar todos los campos de cada reserva (fecha, hora, servicio, comensales) | Visibilidad completa del estado de las reservas |
| Must | Botón prominente para crear nueva reserva | Acceso claro a la acción principal |
| Must | Navegación de regreso al Home desde el flujo de booking | El usuario debe poder volver sin completar una reserva |
| Should | Estado vacío cuando no hay reservas con invitación a crear la primera | Buena experiencia para primer uso |
| Should | La lista se actualiza al volver de crear una reserva | Reflejar inmediatamente la nueva reserva creada |
| Could | Animaciones de transición entre Home y flujo de reserva | Consistencia con las transiciones existentes de la app |
| Won't | Edición o eliminación de reservas | Fuera de alcance, se evaluará en futuras iteraciones |

### MVP Scope

- Pantalla Home que muestra cards con todas las reservas existentes
- Cada card muestra: fecha, hora, servicio y cantidad de comensales
- Botón prominente "Crear nueva reserva" que lleva al flujo actual
- Navegación de regreso al Home después de confirmar o al presionar "volver"

### User Flow

```
App Load
  → Home (lista de reservas + botón "Nueva Reserva")
    → Click "Nueva Reserva"
      → Flujo actual: Comensales → Fecha → Servicio → Horario → Confirmación
        → Post-confirmación: botón "Volver al Home"
          → Home (lista actualizada con la nueva reserva)
```

---

## Implementation Phases

<!--
  STATUS: pending | in-progress | complete
  PARALLEL: phases that can run concurrently (e.g., "with 3" or "-")
  DEPENDS: phases that must complete first (e.g., "1, 2" or "-")
  PRP: link to generated plan file once created
-->

| # | Phase | Description | Status | Parallel | Depends | PRP Plan |
|---|-------|-------------|--------|----------|---------|----------|
| 1 | Obtención de reservas | Permitir consultar la lista completa de reservas existentes | pending | - | - | - |
| 2 | Pantalla Home | Crear la vista principal con lista de reservas en cards y botón de nueva reserva | pending | - | 1 | - |
| 3 | Navegación completa | Integrar el flujo Home <-> Crear reserva con navegación de ida y vuelta | pending | - | 2 | - |
| 4 | Polish visual | Asegurar consistencia visual con el diseño actual, estado vacío, responsive | pending | - | 3 | - |

### Phase Details

**Phase 1: Obtención de reservas**
- **Goal**: Poder consultar todas las reservas existentes
- **Scope**: Exponer la lista completa de reservas ordenadas por fecha de creación (más reciente primero)
- **Success signal**: Se puede obtener la lista de reservas con todos sus campos

**Phase 2: Pantalla Home**
- **Goal**: El usuario ve todas sus reservas al abrir la app
- **Scope**: Vista con cards que muestran fecha, hora, servicio y comensales de cada reserva, más un botón de "Nueva Reserva"
- **Success signal**: Al abrir la app se ven las reservas existentes en formato card

**Phase 3: Navegación completa**
- **Goal**: Flujo completo Home → Crear reserva → Volver al Home
- **Scope**: El botón "Nueva Reserva" inicia el flujo actual, y al confirmar o cancelar se regresa al Home con la lista actualizada
- **Success signal**: El ciclo completo funciona sin errores ni pérdida de datos

**Phase 4: Polish visual**
- **Goal**: Que el Home se sienta parte natural de la app
- **Scope**: Estilos de las cards consistentes con el diseño actual, vista de estado vacío, adaptación responsive
- **Success signal**: El Home es visualmente coherente con el resto de la app

---

## Decisions Log

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Formato de lista | Cards visuales | Tabla, lista simple | Consistente con el lenguaje visual actual de la app |
| Orden de reservas | Por fecha de creación (más reciente primero) | Por fecha de la reserva | Más intuitivo para ver las últimas acciones |
| Punto de entrada | Home como primera pantalla | Mantener Welcome y agregar Home separado | Más directo, el usuario ve valor inmediato al abrir |

---

## Research Summary

**Market Context**
No aplica - es una demo interna para probar el flujo de creación de PRDs.

**Codebase Context**
- La app ya almacena reservas pero no tiene interfaz para listarlas
- El flujo actual de creación de reservas está completo y funcional
- El diseño visual usa cards, tipografía serif, y transiciones suaves
- La app soporta español e inglés (se necesitarán textos nuevos para el Home)

---

*Generated: 2026-02-10*
*Status: DRAFT - needs validation*
