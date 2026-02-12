---
paths:
  - ".claude/skills/sdlc-track/**"
  - ".claude/agents/linear-issue-creator/**"
---

# Linear Conventions

- Always search by title before creating an issue — idempotent.
- Create issues in topological order (dependencies first).
- Every `blocked_by` entry MUST become a "blocks" relation in Linear.
- Direction: the dependency **blocks** the current task, not the other way.
- Estimates: S→1 point, M→2 points, L→5 points.
- Descriptions are Linear-ready markdown — only prepend/append dependency headers.
- Never merge or split tasks from the PERT.
