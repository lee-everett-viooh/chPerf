package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"chperf.example.com/config"
	"chperf.example.com/internal/connections"
	"chperf.example.com/internal/loadrunner"
	"chperf.example.com/internal/stats"
	"chperf.example.com/web/templates"
)

// NewRunForm renders the form to create a new run.
func NewRunForm(connRepo *connections.Repository, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		conns, err := connRepo.List()
		if err != nil {
			conns = nil
		}
		c.HTML(http.StatusOK, "", templates.RunForm(
			conns,
			cfg.DefaultConcurrency,
			cfg.DefaultIterations,
			cfg.DefaultDurationSec,
		))
	}
}

// CreateRun handles POST /runs - creates a run and starts the load test.
func CreateRun(repo *stats.Repository, connRepo *connections.Repository, runner *loadrunner.Runner, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if name == "" {
			name = "Load Test"
		}
		connID, err := strconv.ParseInt(c.PostForm("connection_id"), 10, 64)
		if err != nil || connID <= 0 {
			c.String(http.StatusBadRequest, "please select a connection")
			return
		}
		conn, err := connRepo.Get(connID)
		if err != nil || conn == nil {
			c.String(http.StatusBadRequest, "connection not found")
			return
		}
		dsn := conn.DSN()
		queriesText := strings.TrimSpace(c.PostForm("queries"))
		if queriesText == "" {
			c.String(http.StatusBadRequest, "queries are required")
			return
		}

		queries := parseQueries(queriesText)
		if len(queries) == 0 {
			c.String(http.StatusBadRequest, "at least one query is required")
			return
		}

		mode := c.PostForm("mode")
		if mode != "sequential" && mode != "random" {
			mode = "sequential"
		}

		concurrency, _ := strconv.Atoi(c.PostForm("concurrency"))
		if concurrency <= 0 {
			concurrency = cfg.DefaultConcurrency
		}
		if concurrency > cfg.MaxConcurrency {
			concurrency = cfg.MaxConcurrency
		}

		iterations, _ := strconv.Atoi(c.PostForm("iterations"))
		if iterations <= 0 {
			iterations = cfg.DefaultIterations
		}

		durationSec, _ := strconv.Atoi(c.PostForm("duration"))
		if durationSec < 0 {
			durationSec = 0
		}
		if durationSec == 0 && iterations <= 0 {
			iterations = cfg.DefaultIterations
		}

		waitMs, _ := strconv.Atoi(c.PostForm("wait_between"))
		if waitMs < 0 {
			waitMs = 0
		}
		// Validate against allowed values: 250, 500, 1000, 2000, 3000, 4000, 5000
		allowed := map[int]bool{250: true, 500: true, 1000: true, 2000: true, 3000: true, 4000: true, 5000: true}
		if !allowed[waitMs] {
			waitMs = 1000
		}

		runID, err := repo.CreateRun(name, dsn, mode, concurrency, iterations, durationSec, waitMs)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to create run: %v", err)
			return
		}

		var statsQueries []stats.Query
		for i, sqlText := range queries {
			qid, err := repo.CreateQuery(runID, sqlText, i)
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to create query: %v", err)
				return
			}
			statsQueries = append(statsQueries, stats.Query{ID: qid, RunID: runID, SQLText: sqlText, DisplayOrder: i})
		}

		// Use Background context - request context is cancelled when response is sent,
		// which would abort the load test immediately after redirect.
		err = runner.StartRun(context.Background(), loadrunner.RunConfig{
			RunID:               runID,
			Queries:             statsQueries,
			Mode:                mode,
			Concurrency:         concurrency,
			Iterations:          iterations,
			DurationSec:         durationSec,
			WaitBetweenQueriesMs: waitMs,
			DSN:                 dsn,
			TLSEnabled:          conn.TLSEnabled,
		})
		if err != nil {
			_ = repo.UpdateRunStatus(runID, "failed")
			c.String(http.StatusInternalServerError, "failed to start run: %v", err)
			return
		}

		c.Redirect(http.StatusFound, "/runs/"+strconv.FormatInt(runID, 10))
	}
}

// DeleteRun handles POST /runs/:id/delete - removes the run and all its data.
func DeleteRun(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid run id")
			return
		}
		if err := repo.DeleteRun(id); err != nil {
			c.String(http.StatusInternalServerError, "failed to delete run: %v", err)
			return
		}
		c.Redirect(http.StatusFound, "/")
	}
}

// RunStatus renders the run status page.
func RunStatus(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid run id")
			return
		}
		run, err := repo.GetRun(id)
		if err != nil || run == nil {
			c.String(http.StatusNotFound, "run not found")
			return
		}
		c.HTML(http.StatusOK, "", templates.RunStatus(run))
	}
}

func parseQueries(text string) []string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			out = append(out, line)
		}
	}
	return out
}
