# Phase: BOOTSTRAP

This phase prepares all repositories for agentic work. It runs automatically (no approval gate).

## Objective

Ensure every repo in the manifest has:
1. A detected programming language
2. A `CLAUDE.md` file with project conventions

## Steps

### 1. Detect language for each repo

For each repo in `manifest.yaml`:

1. If `language` is specified in manifest → use it directly
2. Otherwise, check files in the repo path:
   - `go.mod` exists → `go`
   - `package.json` exists → `typescript`
   - `pyproject.toml` or `setup.py` exists → `python`
3. If no match → report to user and ask for manual input

Store in `state.json` under `bootstrap.languages`.

**Important**: This is a deterministic operation. Do it directly with Glob — do NOT spawn an agent.

### 2. Check for CLAUDE.md

For each repo, use Glob to check if `{repo_path}/CLAUDE.md` exists.

### 3. Generate missing CLAUDE.md

For each repo WITHOUT `CLAUDE.md`, spawn the `doc-generator` agent:

```
Task(
  subagent_type: "doc-generator",
  prompt: "Generate CLAUDE.md and ARCHITECTURE.md for the repository at {repo_path}. Repo name: {repo_name}.",
  description: "Generate docs for {repo_name}"
)
```

Record the result in `state.json` under `bootstrap.docs_generated`.

### 4. Update state

After all repos are processed:

```json
{
  "phase": "BOOTSTRAP",
  "phase_status": "completed",
  "bootstrap": {
    "languages": { "repo-a": "go", "repo-b": "typescript" },
    "docs_generated": { "repo-b": "org/repo-b#1" }
  }
}
```

## Transition

When BOOTSTRAP completes → automatically advance to DESIGN phase. No approval gate.

## Error Handling

- If `doc-generator` fails for a repo, log the error in state and continue with other repos
- If language detection fails, ask the user before continuing
- If a repo path doesn't exist, report error and skip that repo

## Idempotency

- Language detection is inherently idempotent (reads only)
- `doc-generator` checks if CLAUDE.md exists before generating — safe to re-run
- On resume: skip repos that already have entries in `bootstrap.languages` and `bootstrap.docs_generated`
