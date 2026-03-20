package db

import (
	_ "embed"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// InitSQLite opens the SQLite database and runs schema migrations.
func InitSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	// Migration: add tls_enabled to existing connections tables
	_, _ = db.Exec("ALTER TABLE connections ADD COLUMN tls_enabled INTEGER NOT NULL DEFAULT 0")
	// Migration: add wait_between_queries_ms to existing runs tables
	_, _ = db.Exec("ALTER TABLE runs ADD COLUMN wait_between_queries_ms INTEGER NOT NULL DEFAULT 0")

	return db, nil
}
