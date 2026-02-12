---
paths:
  - ".claude/skills/sdlc-execute/**"
  - ".claude/agents/coder/**"
---

# Git Worktree Rules

- NEVER modify the user's main working tree. All code changes happen in worktrees.
- Worktrees live at `.sdlc/worktrees/<repo-name>/<slug>/`.
- Branch naming: `feat/<slug>` where slug is the issue title lowercased with hyphens.
- Always `git fetch origin` before creating a worktree.
- If a worktree already exists at the path, reuse it — don't fail.
- Commit messages must reference the Linear ID.
- Push with `-u origin` to set upstream tracking.
- Create PRs with `gh pr create`.
