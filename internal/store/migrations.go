package store

const schema = `
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    prd_url TEXT,
    phase TEXT,
    phase_status TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS repos (
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    name TEXT,
    path TEXT,
    team TEXT,
    language TEXT DEFAULT '',
    PRIMARY KEY (run_id, name)
);

CREATE TABLE IF NOT EXISTS bootstrap_state (
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    repo_name TEXT,
    claude_md BOOLEAN DEFAULT 0,
    architecture_md BOOLEAN DEFAULT 0,
    PRIMARY KEY (run_id, repo_name)
);

CREATE TABLE IF NOT EXISTS artifacts (
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    key TEXT,
    value TEXT,
    PRIMARY KEY (run_id, key)
);

CREATE TABLE IF NOT EXISTS issues (
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    id TEXT,
    title TEXT,
    repo TEXT,
    status TEXT DEFAULT 'ready',
    linear_id TEXT,
    branch TEXT,
    worktree TEXT,
    pr_url TEXT,
    depends_on TEXT DEFAULT '[]',
    iterations INTEGER DEFAULT 0,
    PRIMARY KEY (run_id, id)
);

CREATE TABLE IF NOT EXISTS metrics_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    timestamp DATETIME,
    agent TEXT,
    model TEXT,
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost REAL,
    duration_ms INTEGER,
    issue_id TEXT,
    phase TEXT
);

CREATE TABLE IF NOT EXISTS phase_timings (
    run_id TEXT REFERENCES runs(id) ON DELETE CASCADE,
    phase TEXT,
    duration_ms INTEGER,
    PRIMARY KEY (run_id, phase)
);

CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    parent_id TEXT DEFAULT '',
    type TEXT NOT NULL DEFAULT 'phase',
    phase TEXT NOT NULL,
    issue_id TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'running',
    session_id TEXT DEFAULT '',
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    error_message TEXT DEFAULT '',
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_executions_run_id ON executions(run_id);
CREATE INDEX IF NOT EXISTS idx_executions_run_phase ON executions(run_id, phase);
`
