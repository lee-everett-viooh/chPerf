package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"chperf.example.com/internal/stats"
	"chperf.example.com/web/templates"
)

// Home renders the dashboard with recent runs.
func Home(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		runs, err := repo.ListRuns(20)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to list runs: %v", err)
			return
		}
		c.HTML(http.StatusOK, "", templates.Home(runs))
	}
}
