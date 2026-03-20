package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"chperf.example.com/internal/clickhouse"
	"chperf.example.com/internal/connections"
	"chperf.example.com/web/templates"
)

// ConnectionsList renders the list of saved connections.
func ConnectionsList(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		conns, err := repo.List()
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to list connections: %v", err)
			return
		}
		c.HTML(http.StatusOK, "", templates.ConnectionsList(conns))
	}
}

// ConnectionForm renders the add connection form.
func ConnectionForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "", templates.ConnectionForm(nil))
	}
}

// ConnectionEditForm renders the edit connection form.
func ConnectionEditForm(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid connection id")
			return
		}
		conn, err := repo.Get(id)
		if err != nil || conn == nil {
			c.String(http.StatusNotFound, "connection not found")
			return
		}
		c.HTML(http.StatusOK, "", templates.ConnectionForm(conn))
	}
}

// CreateConnection handles POST /connections.
func CreateConnection(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn := parseConnectionForm(c)
		if conn.Name == "" || conn.Host == "" {
			c.String(http.StatusBadRequest, "name and host are required")
			return
		}
		if conn.Port <= 0 {
			conn.Port = 9000
		}
		_, err := repo.Create(conn.Name, conn.Host, conn.Port, conn.Database, conn.Username, conn.Password, conn.TLSEnabled)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to create connection: %v", err)
			return
		}
		c.Redirect(http.StatusFound, "/connections")
	}
}

// UpdateConnection handles POST /connections/:id (or PUT).
func UpdateConnection(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid connection id")
			return
		}
		conn := parseConnectionForm(c)
		if conn.Name == "" || conn.Host == "" {
			c.String(http.StatusBadRequest, "name and host are required")
			return
		}
		if conn.Port <= 0 {
			conn.Port = 9000
		}
		if err := repo.Update(id, conn.Name, conn.Host, conn.Port, conn.Database, conn.Username, conn.Password, conn.TLSEnabled); err != nil {
			c.String(http.StatusInternalServerError, "failed to update connection: %v", err)
			return
		}
		c.Redirect(http.StatusFound, "/connections")
	}
}

// DeleteConnection handles POST /connections/:id/delete.
func DeleteConnection(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid connection id")
			return
		}
		if err := repo.Delete(id); err != nil {
			c.String(http.StatusInternalServerError, "failed to delete connection: %v", err)
			return
		}
		c.Redirect(http.StatusFound, "/connections")
	}
}

// TestConnection runs SELECT version() and returns JSON with success/error.
func TestConnection(repo *connections.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid connection id"})
			return
		}
		conn, err := repo.Get(id)
		if err != nil || conn == nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "connection not found"})
			return
		}

		client, err := clickhouse.NewClient(c.Request.Context(), conn.DSN(), conn.TLSEnabled)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		defer client.Close()

		result := client.Execute(c.Request.Context(), "SELECT version()")
		if result.Error != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Connection successful"})
	}
}

// TestConnectionDSN tests a DSN from form data (for testing before save).
func TestConnectionDSN() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn := parseConnectionForm(c)
		if conn.Host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "host is required"})
			return
		}
		if conn.Port <= 0 {
			conn.Port = 9000
		}
		// Build DSN from form
		dsn := buildDSN(conn.Host, conn.Port, conn.Database, conn.Username, conn.Password)

		client, err := clickhouse.NewClient(c.Request.Context(), dsn, conn.TLSEnabled)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		defer client.Close()

		result := client.Execute(c.Request.Context(), "SELECT version()")
		if result.Error != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Connection successful"})
	}
}

func parseConnectionForm(c *gin.Context) *connections.Connection {
	port, _ := strconv.Atoi(c.PostForm("port"))
	if port <= 0 {
		port = 9000
	}
	tlsEnabled := c.PostForm("tls_enabled") == "on" || c.PostForm("tls_enabled") == "1"
	return &connections.Connection{
		Name:       strings.TrimSpace(c.PostForm("name")),
		Host:       strings.TrimSpace(c.PostForm("host")),
		Port:       port,
		Database:   strings.TrimSpace(c.PostForm("database")),
		Username:   strings.TrimSpace(c.PostForm("username")),
		Password:   c.PostForm("password"), // don't trim password
		TLSEnabled: tlsEnabled,
	}
}

func buildDSN(host string, port int, database, username, password string) string {
	return connections.BuildDSN(host, port, database, username, password)
}
