package stats

import (
	"database/sql"
	"time"
)

// Run represents a load test run.
type Run struct {
	ID                    int64
	Name                  string
	ClickHouseDSN         string
	ExecutionMode         string
	Concurrency           int
	Iterations            int
	DurationSec           int
	WaitBetweenQueriesMs  int
	CreatedAt             time.Time
	Status                string
}

// Query represents a SQL query in a run.
type Query struct {
	ID           int64
	RunID        int64
	SQLText      string
	DisplayOrder int
}

// QueryResult represents a single query execution result.
type QueryResult struct {
	ID           int64
	QueryID      int64
	RunTimeMs    float64
	RowsAffected int64
	ErrorMessage *string
	ExecutedAt   time.Time
	WorkerID     int
}

// Percentiles holds P50, P75, P90, P99 for latency.
type Percentiles struct {
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
	P99 float64 `json:"p99"`
}

// Repository provides CRUD for runs, queries, and results.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new stats repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateRun inserts a new run and returns its ID.
func (r *Repository) CreateRun(name, dsn, mode string, concurrency, iterations, durationSec, waitBetweenQueriesMs int) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO runs (name, clickhouse_dsn, execution_mode, concurrency, iterations, duration_sec, wait_between_queries_ms, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'running')`,
		name, dsn, mode, concurrency, iterations, durationSec, waitBetweenQueriesMs,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateRunStatus sets the run status.
func (r *Repository) UpdateRunStatus(runID int64, status string) error {
	_, err := r.db.Exec(`UPDATE runs SET status = ? WHERE id = ?`, status, runID)
	return err
}

// DeleteRun removes a run and all its queries and results.
func (r *Repository) DeleteRun(runID int64) error {
	_, err := r.db.Exec(`DELETE FROM query_results WHERE query_id IN (SELECT id FROM queries WHERE run_id = ?)`, runID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`DELETE FROM queries WHERE run_id = ?`, runID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`DELETE FROM runs WHERE id = ?`, runID)
	return err
}

// CreateQuery inserts a query for a run.
func (r *Repository) CreateQuery(runID int64, sqlText string, displayOrder int) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO queries (run_id, sql_text, display_order) VALUES (?, ?, ?)`,
		runID, sqlText, displayOrder,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertQueryResult records a single query execution.
func (r *Repository) InsertQueryResult(queryID int64, runTimeMs float64, rowsAffected int64, errMsg *string, workerID int) error {
	_, err := r.db.Exec(
		`INSERT INTO query_results (query_id, run_time_ms, rows_affected, error_message, worker_id)
		 VALUES (?, ?, ?, ?, ?)`,
		queryID, runTimeMs, rowsAffected, errMsg, workerID,
	)
	return err
}

// GetRun fetches a run by ID.
func (r *Repository) GetRun(id int64) (*Run, error) {
	var run Run
	err := r.db.QueryRow(
		`SELECT id, name, clickhouse_dsn, execution_mode, concurrency, iterations, duration_sec, wait_between_queries_ms, created_at, status
		 FROM runs WHERE id = ?`, id,
	).Scan(
		&run.ID, &run.Name, &run.ClickHouseDSN, &run.ExecutionMode, &run.Concurrency,
		&run.Iterations, &run.DurationSec, &run.WaitBetweenQueriesMs, &run.CreatedAt, &run.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetQueries returns all queries for a run, ordered by display_order.
func (r *Repository) GetQueries(runID int64) ([]Query, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, sql_text, display_order FROM queries WHERE run_id = ? ORDER BY display_order, id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []Query
	for rows.Next() {
		var q Query
		if err := rows.Scan(&q.ID, &q.RunID, &q.SQLText, &q.DisplayOrder); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return queries, rows.Err()
}

// TimeseriesPoint is a single (timestamp, latency) point for charts.
type TimeseriesPoint struct {
	Timestamp string  `json:"t"`
	RunTimeMs float64 `json:"y"`
}

// GetTimeseriesForQuery returns executed_at and run_time_ms for a query, ordered by time.
func (r *Repository) GetTimeseriesForQuery(queryID int64) ([]TimeseriesPoint, error) {
	rows, err := r.db.Query(
		`SELECT executed_at, run_time_ms FROM query_results WHERE query_id = ? ORDER BY executed_at`,
		queryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TimeseriesPoint
	for rows.Next() {
		var t time.Time
		var ms float64
		if err := rows.Scan(&t, &ms); err != nil {
			return nil, err
		}
		points = append(points, TimeseriesPoint{
			Timestamp: t.Format("15:04:05.000"),
			RunTimeMs: ms,
		})
	}
	return points, rows.Err()
}

// GetRunTimeMsForQuery returns all run_time_ms values for a query (for percentile calc).
func (r *Repository) GetRunTimeMsForQuery(queryID int64) ([]float64, error) {
	rows, err := r.db.Query(
		`SELECT run_time_ms FROM query_results WHERE query_id = ? AND error_message IS NULL ORDER BY run_time_ms`,
		queryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vals []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// GetRunTimeMsForRun returns all run_time_ms for a run (overall percentiles).
func (r *Repository) GetRunTimeMsForRun(runID int64) ([]float64, error) {
	rows, err := r.db.Query(
		`SELECT qr.run_time_ms FROM query_results qr
		 JOIN queries q ON q.id = qr.query_id
		 WHERE q.run_id = ? AND qr.error_message IS NULL
		 ORDER BY qr.run_time_ms`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vals []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// RunProgress holds live progress counters for a running test.
type RunProgress struct {
	ExecutedCount int64 `json:"executed_count"`
	ErrorCount    int64 `json:"error_count"`
}

// GetRunProgress counts total and errored query results for a run.
func (r *Repository) GetRunProgress(runID int64) (*RunProgress, error) {
	var p RunProgress
	err := r.db.QueryRow(
		`SELECT COUNT(*), COUNT(qr.error_message)
		 FROM query_results qr
		 JOIN queries q ON q.id = qr.query_id
		 WHERE q.run_id = ?`, runID,
	).Scan(&p.ExecutedCount, &p.ErrorCount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListRuns returns recent runs.
func (r *Repository) ListRuns(limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, name, clickhouse_dsn, execution_mode, concurrency, iterations, duration_sec, wait_between_queries_ms, created_at, status
		 FROM runs ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(
			&run.ID, &run.Name, &run.ClickHouseDSN, &run.ExecutionMode, &run.Concurrency,
			&run.Iterations, &run.DurationSec, &run.WaitBetweenQueriesMs, &run.CreatedAt, &run.Status,
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// GetResultsForExport returns all query results for a run for CSV export.
func (r *Repository) GetResultsForExport(runID int64) ([]ExportRow, error) {
	rows, err := r.db.Query(
		`SELECT r.id, r.name, q.id, q.sql_text, qr.run_time_ms, qr.rows_affected, qr.error_message, qr.executed_at, qr.worker_id
		 FROM query_results qr
		 JOIN queries q ON q.id = qr.query_id
		 JOIN runs r ON r.id = q.run_id
		 WHERE r.id = ?
		 ORDER BY qr.executed_at`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportRow
	for rows.Next() {
		var row ExportRow
		var errMsg sql.NullString
		if err := rows.Scan(
			&row.RunID, &row.RunName, &row.QueryID, &row.SQLText,
			&row.RunTimeMs, &row.RowsAffected, &errMsg, &row.ExecutedAt, &row.WorkerID,
		); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			row.ErrorMessage = errMsg.String
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ExportRow is a single row for CSV export.
type ExportRow struct {
	RunID        int64
	RunName      string
	QueryID      int64
	SQLText      string
	RunTimeMs    float64
	RowsAffected int64
	ErrorMessage string
	ExecutedAt   time.Time
	WorkerID     int
}

// Percentile computes percentile from sorted values (linear interpolation).
func Percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (p / 100) * float64(len(sorted)-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
