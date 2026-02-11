const PHASE_ORDER: Record<string, number> = {
  bootstrap: 1,
  design: 2,
  planning: 3,
  tracking: 4,
  executing: 5,
}

/** Returns a display label like "1- Bootstrap" for a phase name. */
export function phaseLabel(phase: string): string {
  const num = PHASE_ORDER[phase]
  const cap = phase.charAt(0).toUpperCase() + phase.slice(1)
  return num ? `${num} - ${cap}` : cap
}
