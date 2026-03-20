package connections

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"
)

// Connection represents a saved ClickHouse connection.
type Connection struct {
	ID         int64
	Name       string
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
	TLSEnabled bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DSN builds the ClickHouse DSN string from connection fields.
func (c *Connection) DSN() string {
	return BuildDSN(c.Host, c.Port, c.Database, c.Username, c.Password)
}

// BuildDSN constructs a ClickHouse DSN from components.
func BuildDSN(host string, port int, database, username, password string) string {
	if database == "" {
		database = "default"
	}
	if username == "" {
		username = "default"
	}
	user := url.UserPassword(username, password)
	return fmt.Sprintf("clickhouse://%s@%s:%d/%s", user.String(), host, port, database)
}

// Repository provides CRUD for connections.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new connections repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns all connections.
func (r *Repository) List() ([]Connection, error) {
	rows, err := r.db.Query(
		`SELECT id, name, host, port, database, username, password, tls_enabled, created_at, updated_at
		 FROM connections ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []Connection
	for rows.Next() {
		var c Connection
		var tlsEnabled int
		if err := rows.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Database, &c.Username, &c.Password, &tlsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.TLSEnabled = tlsEnabled != 0
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// Get fetches a connection by ID.
func (r *Repository) Get(id int64) (*Connection, error) {
	var c Connection
	var tlsEnabled int
	err := r.db.QueryRow(
		`SELECT id, name, host, port, database, username, password, tls_enabled, created_at, updated_at
		 FROM connections WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Database, &c.Username, &c.Password, &tlsEnabled, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.TLSEnabled = tlsEnabled != 0
	return &c, nil
}

// Create inserts a new connection.
func (r *Repository) Create(name, host string, port int, database, username, password string, tlsEnabled bool) (int64, error) {
	if database == "" {
		database = "default"
	}
	if username == "" {
		username = "default"
	}
	tlsVal := 0
	if tlsEnabled {
		tlsVal = 1
	}
	res, err := r.db.Exec(
		`INSERT INTO connections (name, host, port, database, username, password, tls_enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, host, port, database, username, password, tlsVal,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update modifies an existing connection.
func (r *Repository) Update(id int64, name, host string, port int, database, username, password string, tlsEnabled bool) error {
	if database == "" {
		database = "default"
	}
	if username == "" {
		username = "default"
	}
	tlsVal := 0
	if tlsEnabled {
		tlsVal = 1
	}
	_, err := r.db.Exec(
		`UPDATE connections SET name=?, host=?, port=?, database=?, username=?, password=?, tls_enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		name, host, port, database, username, password, tlsVal, id,
	)
	return err
}

// Delete removes a connection.
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM connections WHERE id = ?`, id)
	return err
}
