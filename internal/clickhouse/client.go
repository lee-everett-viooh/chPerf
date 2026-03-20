package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// QueryResult holds the result of a single query execution.
type QueryResult struct {
	RunTimeMs    float64
	RowsAffected int64
	Error        error
}

// Client executes queries against ClickHouse.
type Client struct {
	conn clickhouse.Conn
}

// NewClient creates a ClickHouse client from a DSN.
// When useTLS is true, connects over TLS with InsecureSkipVerify (trusts invalid certs).
func NewClient(ctx context.Context, dsn string, useTLS bool) (*Client, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	if useTLS {
		opts.TLS = &tls.Config{InsecureSkipVerify: true}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	return &Client{conn: conn}, nil
}

// Execute runs a query and returns timing and row count.
func (c *Client) Execute(ctx context.Context, sql string) QueryResult {
	start := time.Now()
	rows, err := c.conn.Query(ctx, sql)
	if err != nil {
		return QueryResult{
			RunTimeMs:    time.Since(start).Seconds() * 1000,
			RowsAffected: 0,
			Error:        err,
		}
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		return QueryResult{
			RunTimeMs:    time.Since(start).Seconds() * 1000,
			RowsAffected: count,
			Error:        err,
		}
	}

	return QueryResult{
		RunTimeMs:    time.Since(start).Seconds() * 1000,
		RowsAffected: count,
		Error:        nil,
	}
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
