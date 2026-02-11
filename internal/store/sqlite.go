package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yalochat/agentic-sdlc/internal/engine"
)

// SQLiteStore implements Store backed by a SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database and applies migrations.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

// ---------- Run lifecycle ----------

func (s *SQLiteStore) CreateRun(state *engine.State) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO runs (id, prd_url, phase, phase_status, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		state.RunID, state.PrdURL, string(state.Phase), string(state.PhaseStatus), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}

	if err := insertRepos(tx, state.RunID, state.Repos); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) LoadRun(runID string) (*engine.State, error) {
	return s.loadRunByQuery(`SELECT id, prd_url, phase, phase_status, created_at, updated_at FROM runs WHERE id = ?`, runID)
}

func (s *SQLiteStore) LatestRun() (*engine.State, error) {
	return s.loadRunByQuery(`SELECT id, prd_url, phase, phase_status, created_at, updated_at FROM runs ORDER BY created_at DESC LIMIT 1`)
}

func (s *SQLiteStore) loadRunByQuery(query string, args ...interface{}) (*engine.State, error) {
	st := &engine.State{
		Bootstrap: make(map[string]engine.RepoState),
		Artifacts: make(map[string]string),
		Issues:    make(map[string]engine.IssueState),
		Metrics: engine.MetricsState{
			ByAgent:      make(map[string]engine.Usage),
			PhaseTimings: make(map[string]int64),
		},
	}

	var phase, status string
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(query, args...).Scan(
		&st.RunID, &st.PrdURL, &phase, &status, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no run found")
		}
		return nil, fmt.Errorf("query run: %w", err)
	}
	st.Phase = engine.Phase(phase)
	st.PhaseStatus = engine.PhaseStatus(status)
	st.UpdatedAt = updatedAt

	// Load repos
	rows, err := s.db.Query(`SELECT name, path, team, language FROM repos WHERE run_id = ?`, st.RunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r engine.RepoConfig
		if err := rows.Scan(&r.Name, &r.Path, &r.Team, &r.Language); err != nil {
			return nil, err
		}
		st.Repos = append(st.Repos, r)
	}

	// Load bootstrap
	brows, err := s.db.Query(`SELECT repo_name, claude_md, architecture_md FROM bootstrap_state WHERE run_id = ?`, st.RunID)
	if err != nil {
		return nil, err
	}
	defer brows.Close()
	for brows.Next() {
		var name string
		var rs engine.RepoState
		if err := brows.Scan(&name, &rs.ClaudeMD, &rs.ArchitectureMD); err != nil {
			return nil, err
		}
		st.Bootstrap[name] = rs
	}

	// Load artifacts
	arows, err := s.db.Query(`SELECT key, value FROM artifacts WHERE run_id = ?`, st.RunID)
	if err != nil {
		return nil, err
	}
	defer arows.Close()
	for arows.Next() {
		var k, v string
		if err := arows.Scan(&k, &v); err != nil {
			return nil, err
		}
		st.Artifacts[k] = v
	}

	// Load issues
	irows, err := s.db.Query(`SELECT id, title, repo, status, linear_id, branch, worktree, pr_url, depends_on, iterations FROM issues WHERE run_id = ?`, st.RunID)
	if err != nil {
		return nil, err
	}
	defer irows.Close()
	for irows.Next() {
		var iss engine.IssueState
		var linearID, branch, worktree, prURL sql.NullString
		var depsJSON string
		if err := irows.Scan(&iss.ID, &iss.Title, &iss.Repo, &iss.Status, &linearID, &branch, &worktree, &prURL, &depsJSON, &iss.Iterations); err != nil {
			return nil, err
		}
		iss.LinearID = linearID.String
		iss.Branch = branch.String
		iss.Worktree = worktree.String
		iss.PRURL = prURL.String
		json.Unmarshal([]byte(depsJSON), &iss.DependsOn)
		st.Issues[iss.ID] = iss
	}

	// Load metrics aggregate
	metrics, err := s.LoadMetricsAggregate(st.RunID)
	if err == nil {
		st.Metrics = metrics
	}

	return st, nil
}

func (s *SQLiteStore) ListRuns() ([]RunSummary, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.phase, r.phase_status, r.prd_url, r.created_at, r.updated_at,
		       (SELECT COUNT(*) FROM issues i WHERE i.run_id = r.id) as issue_count
		FROM runs r ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunSummary
	for rows.Next() {
		var rs RunSummary
		if err := rows.Scan(&rs.ID, &rs.Phase, &rs.Status, &rs.PrdURL, &rs.CreatedAt, &rs.UpdatedAt, &rs.IssueCount); err != nil {
			return nil, err
		}
		runs = append(runs, rs)
	}
	return runs, nil
}

func (s *SQLiteStore) DeleteRun(runID string) error {
	_, err := s.db.Exec(`DELETE FROM runs WHERE id = ?`, runID)
	return err
}

// ---------- Partial saves ----------

func (s *SQLiteStore) SaveRunMeta(runID string, phase engine.Phase, status engine.PhaseStatus) error {
	_, err := s.db.Exec(
		`UPDATE runs SET phase = ?, phase_status = ?, updated_at = datetime('now') WHERE id = ?`,
		string(phase), string(status), runID,
	)
	return err
}

func (s *SQLiteStore) SaveRepos(runID string, repos []engine.RepoConfig) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM repos WHERE run_id = ?`, runID)
	if err != nil {
		return err
	}
	if err := insertRepos(tx, runID, repos); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) SaveBootstrap(runID string, repo string, rs engine.RepoState) error {
	_, err := s.db.Exec(
		`INSERT INTO bootstrap_state (run_id, repo_name, claude_md, architecture_md) VALUES (?,?,?,?)
		 ON CONFLICT(run_id, repo_name) DO UPDATE SET claude_md=excluded.claude_md, architecture_md=excluded.architecture_md`,
		runID, repo, rs.ClaudeMD, rs.ArchitectureMD,
	)
	return err
}

func (s *SQLiteStore) SaveArtifact(runID string, key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO artifacts (run_id, key, value) VALUES (?,?,?)
		 ON CONFLICT(run_id, key) DO UPDATE SET value=excluded.value`,
		runID, key, value,
	)
	return err
}

func (s *SQLiteStore) SaveIssue(runID string, issue engine.IssueState) error {
	depsJSON, _ := json.Marshal(issue.DependsOn)
	_, err := s.db.Exec(
		`INSERT INTO issues (run_id, id, title, repo, status, linear_id, branch, worktree, pr_url, depends_on, iterations)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(run_id, id) DO UPDATE SET
		   title=excluded.title, repo=excluded.repo, status=excluded.status,
		   linear_id=excluded.linear_id, branch=excluded.branch, worktree=excluded.worktree,
		   pr_url=excluded.pr_url, depends_on=excluded.depends_on, iterations=excluded.iterations`,
		runID, issue.ID, issue.Title, issue.Repo, string(issue.Status),
		issue.LinearID, issue.Branch, issue.Worktree, issue.PRURL,
		string(depsJSON), issue.Iterations,
	)
	return err
}

func (s *SQLiteStore) SaveIssues(runID string, issues map[string]engine.IssueState) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO issues (run_id, id, title, repo, status, linear_id, branch, worktree, pr_url, depends_on, iterations)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(run_id, id) DO UPDATE SET
		   title=excluded.title, repo=excluded.repo, status=excluded.status,
		   linear_id=excluded.linear_id, branch=excluded.branch, worktree=excluded.worktree,
		   pr_url=excluded.pr_url, depends_on=excluded.depends_on, iterations=excluded.iterations`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, iss := range issues {
		depsJSON, _ := json.Marshal(iss.DependsOn)
		_, err := stmt.Exec(
			runID, iss.ID, iss.Title, iss.Repo, string(iss.Status),
			iss.LinearID, iss.Branch, iss.Worktree, iss.PRURL,
			string(depsJSON), iss.Iterations,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------- Metrics ----------

func (s *SQLiteStore) RecordMetric(runID string, entry engine.MetricsEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO metrics_entries (run_id, timestamp, agent, model, tokens_in, tokens_out, cost, duration_ms, issue_id, phase)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		runID, entry.Timestamp, entry.Agent, entry.Model,
		entry.TokensIn, entry.TokensOut, entry.Cost, entry.Duration,
		entry.IssueID, string(entry.Phase),
	)
	return err
}

func (s *SQLiteStore) RecordPhaseTiming(runID string, phase engine.Phase, durationMs int64) error {
	_, err := s.db.Exec(
		`INSERT INTO phase_timings (run_id, phase, duration_ms) VALUES (?,?,?)
		 ON CONFLICT(run_id, phase) DO UPDATE SET duration_ms=excluded.duration_ms`,
		runID, string(phase), durationMs,
	)
	return err
}

func (s *SQLiteStore) LoadMetricsAggregate(runID string) (engine.MetricsState, error) {
	ms := engine.MetricsState{
		ByAgent:      make(map[string]engine.Usage),
		PhaseTimings: make(map[string]int64),
	}

	// Aggregate totals
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(tokens_in),0), COALESCE(SUM(tokens_out),0), COALESCE(SUM(cost),0)
		 FROM metrics_entries WHERE run_id = ?`, runID,
	).Scan(&ms.TokensIn, &ms.TokensOut, &ms.TotalCost)
	if err != nil {
		return ms, err
	}

	// By agent
	rows, err := s.db.Query(
		`SELECT agent, SUM(tokens_in), SUM(tokens_out), SUM(cost), COUNT(*)
		 FROM metrics_entries WHERE run_id = ? GROUP BY agent`, runID,
	)
	if err != nil {
		return ms, err
	}
	defer rows.Close()
	for rows.Next() {
		var agent string
		var u engine.Usage
		if err := rows.Scan(&agent, &u.TokensIn, &u.TokensOut, &u.Cost, &u.Calls); err != nil {
			return ms, err
		}
		ms.ByAgent[agent] = u
	}

	// Phase timings
	trows, err := s.db.Query(`SELECT phase, duration_ms FROM phase_timings WHERE run_id = ?`, runID)
	if err != nil {
		return ms, err
	}
	defer trows.Close()
	for trows.Next() {
		var p string
		var d int64
		if err := trows.Scan(&p, &d); err != nil {
			return ms, err
		}
		ms.PhaseTimings[p] = d
	}

	return ms, nil
}

// ---------- Executions ----------

func (s *SQLiteStore) CreateExecution(rec engine.ExecutionRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO executions (id, run_id, parent_id, type, phase, issue_id, status, session_id, tokens_in, tokens_out, error_message, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   status=excluded.status, tokens_in=excluded.tokens_in, tokens_out=excluded.tokens_out,
		   error_message=excluded.error_message, updated_at=excluded.updated_at`,
		rec.ID, rec.RunID, rec.ParentID, string(rec.Type), rec.Phase, rec.IssueID,
		string(rec.Status), rec.SessionID, rec.TokensIn, rec.TokensOut, rec.ErrorMessage,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetExecution(id string) (*engine.ExecutionRecord, error) {
	var rec engine.ExecutionRecord
	var execType, status string
	err := s.db.QueryRow(
		`SELECT id, run_id, parent_id, type, phase, issue_id, status, session_id, tokens_in, tokens_out, error_message, created_at, updated_at
		 FROM executions WHERE id = ?`, id,
	).Scan(
		&rec.ID, &rec.RunID, &rec.ParentID, &execType, &rec.Phase, &rec.IssueID,
		&status, &rec.SessionID, &rec.TokensIn, &rec.TokensOut, &rec.ErrorMessage,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Type = engine.ExecutionType(execType)
	rec.Status = engine.ExecutionStatus(status)
	return &rec, nil
}

func (s *SQLiteStore) UpdateExecutionStatus(id string, status engine.ExecutionStatus, errorMsg string) error {
	_, err := s.db.Exec(
		`UPDATE executions SET status = ?, error_message = ?, updated_at = datetime('now') WHERE id = ?`,
		string(status), errorMsg, id,
	)
	return err
}

func (s *SQLiteStore) UpdateExecutionTokens(id string, tokensIn, tokensOut int64) error {
	_, err := s.db.Exec(
		`UPDATE executions SET tokens_in = ?, tokens_out = ?, updated_at = datetime('now') WHERE id = ?`,
		tokensIn, tokensOut, id,
	)
	return err
}

func (s *SQLiteStore) ListExecutions(runID string) ([]engine.ExecutionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, parent_id, type, phase, issue_id, status, session_id, tokens_in, tokens_out, error_message, created_at, updated_at
		 FROM executions WHERE run_id = ? ORDER BY created_at ASC`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []engine.ExecutionRecord
	for rows.Next() {
		var rec engine.ExecutionRecord
		var execType, status string
		if err := rows.Scan(
			&rec.ID, &rec.RunID, &rec.ParentID, &execType, &rec.Phase, &rec.IssueID,
			&status, &rec.SessionID, &rec.TokensIn, &rec.TokensOut, &rec.ErrorMessage,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rec.Type = engine.ExecutionType(execType)
		rec.Status = engine.ExecutionStatus(status)
		records = append(records, rec)
	}
	return records, nil
}

func (s *SQLiteStore) LatestExecution(runID string, phase string) (*engine.ExecutionRecord, error) {
	var rec engine.ExecutionRecord
	var execType, status string
	err := s.db.QueryRow(
		`SELECT id, run_id, parent_id, type, phase, issue_id, status, session_id, tokens_in, tokens_out, error_message, created_at, updated_at
		 FROM executions WHERE run_id = ? AND phase = ? ORDER BY created_at DESC LIMIT 1`,
		runID, phase,
	).Scan(
		&rec.ID, &rec.RunID, &rec.ParentID, &execType, &rec.Phase, &rec.IssueID,
		&status, &rec.SessionID, &rec.TokensIn, &rec.TokensOut, &rec.ErrorMessage,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Type = engine.ExecutionType(execType)
	rec.Status = engine.ExecutionStatus(status)
	return &rec, nil
}

// ---------- Import ----------

func (s *SQLiteStore) ImportState(state *engine.State) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO runs (id, prd_url, phase, phase_status, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		state.RunID, state.PrdURL, string(state.Phase), string(state.PhaseStatus), now, now,
	)
	if err != nil {
		return fmt.Errorf("import run: %w", err)
	}

	if err := insertRepos(tx, state.RunID, state.Repos); err != nil {
		return err
	}

	for repo, rs := range state.Bootstrap {
		_, err := tx.Exec(
			`INSERT INTO bootstrap_state (run_id, repo_name, claude_md, architecture_md) VALUES (?,?,?,?)`,
			state.RunID, repo, rs.ClaudeMD, rs.ArchitectureMD,
		)
		if err != nil {
			return err
		}
	}

	for k, v := range state.Artifacts {
		_, err := tx.Exec(`INSERT INTO artifacts (run_id, key, value) VALUES (?,?,?)`, state.RunID, k, v)
		if err != nil {
			return err
		}
	}

	for _, iss := range state.Issues {
		depsJSON, _ := json.Marshal(iss.DependsOn)
		_, err := tx.Exec(
			`INSERT INTO issues (run_id, id, title, repo, status, linear_id, branch, worktree, pr_url, depends_on, iterations)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			state.RunID, iss.ID, iss.Title, iss.Repo, string(iss.Status),
			iss.LinearID, iss.Branch, iss.Worktree, iss.PRURL,
			string(depsJSON), iss.Iterations,
		)
		if err != nil {
			return err
		}
	}

	// Import phase timings
	for phase, dur := range state.Metrics.PhaseTimings {
		_, err := tx.Exec(
			`INSERT INTO phase_timings (run_id, phase, duration_ms) VALUES (?,?,?)`,
			state.RunID, phase, dur,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ---------- Helpers ----------

func insertRepos(tx *sql.Tx, runID string, repos []engine.RepoConfig) error {
	for _, r := range repos {
		_, err := tx.Exec(
			`INSERT INTO repos (run_id, name, path, team, language) VALUES (?,?,?,?,?)`,
			runID, r.Name, r.Path, r.Team, r.Language,
		)
		if err != nil {
			return fmt.Errorf("insert repo %s: %w", r.Name, err)
		}
	}
	return nil
}
