-- connections: saved ClickHouse DSNs for reuse
CREATE TABLE IF NOT EXISTS connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 9000,
    database TEXT NOT NULL DEFAULT 'default',
    username TEXT NOT NULL DEFAULT 'default',
    password TEXT NOT NULL DEFAULT '',
    tls_enabled INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- runs: load test sessions
CREATE TABLE IF NOT EXISTS runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    clickhouse_dsn TEXT NOT NULL,
    execution_mode TEXT NOT NULL CHECK(execution_mode IN ('sequential', 'random')),
    concurrency INTEGER NOT NULL,
    iterations INTEGER NOT NULL,
    duration_sec INTEGER NOT NULL,
    wait_between_queries_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'running' CHECK(status IN ('running', 'completed', 'failed'))
);

-- queries: SQL statements per run
CREATE TABLE IF NOT EXISTS queries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sql_text TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_queries_run_id ON queries(run_id);

-- query_results: per-execution statistics
CREATE TABLE IF NOT EXISTS query_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query_id INTEGER NOT NULL REFERENCES queries(id) ON DELETE CASCADE,
    run_time_ms REAL NOT NULL,
    rows_affected INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    worker_id INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_query_results_query_id ON query_results(query_id);
CREATE INDEX IF NOT EXISTS idx_query_results_executed_at ON query_results(executed_at);
