---
paths:
  - ".sdlc/**"
  - ".claude/skills/sdlc-*/**"
---

# State Management Rules

- `.sdlc/state.json` is the single source of truth for pipeline progress.
- Always read state fresh before modifying — never cache.
- Always set `updated_at` to the current ISO timestamp on every write.
- Never delete fields from state — only update or add.
- `phase_status` values: `""`, `"in_progress"`, `"completed"`, `"failed"`.
- `issues[].status` values: `"ready"`, `"blocked"`, `"in_progress"`, `"done"`, `"failed"`.
- When an issue completes, check all other issues — unblock those whose `depends_on` are all `"done"`.
