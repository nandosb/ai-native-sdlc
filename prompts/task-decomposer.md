You are a technical project planner. Decompose the following scoping document into implementable tasks with dependencies.

## Scoping Document

{{scoping_content}}

## Repositories

{{repo_summary}}

## Instructions

Create a PERT (task dependency graph) as a JSON array. Each task must be:
- Small enough for a single PR
- Implementable in one developer session
- Self-contained with clear inputs and outputs

Output format:

```json
[
  {
    "id": "TASK-001",
    "title": "Short, specific task title",
    "description": "Detailed description: what to implement, where, acceptance criteria",
    "repo": "repo-name",
    "depends_on": [],
    "estimate": "S"
  },
  {
    "id": "TASK-002",
    "title": "Another task",
    "description": "This task depends on TASK-001 because...",
    "repo": "repo-name",
    "depends_on": ["TASK-001"],
    "estimate": "M"
  }
]
```

## Rules

1. **Atomic tasks** — Each task = one PR. No task should require multiple PRs.
2. **DAG structure** — Dependencies must form a directed acyclic graph. No cycles.
3. **Test coverage** — Every feature task should have corresponding test tasks (can be in same task if small).
4. **Estimates**:
   - `S` = Small (< 1 hour): config changes, simple CRUD, boilerplate
   - `M` = Medium (1-3 hours): new endpoint with tests, moderate logic
   - `L` = Large (3-8 hours): complex feature, significant refactoring
5. **Ordering** — Infrastructure/foundation tasks first, feature tasks next, integration tasks last.
6. **Repo assignment** — Every task must specify which repo it belongs to.
7. **Description quality** — Each description should be detailed enough for a developer to implement without ambiguity.

Output ONLY the JSON array, wrapped in a ```json code fence.
