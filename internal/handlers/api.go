package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"chperf.example.com/internal/stats"
)

// RunStatusAPI returns JSON status for a run (for polling).
func RunStatusAPI(repo *stats.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
			return
		}
		run, err := repo.GetRun(id)
		if err != nil || run == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
			return
		}
		resp := gin.H{
			"id":         run.ID,
			"name":       run.Name,
			"status":     run.Status,
			"mode":       run.ExecutionMode,
			"created":    run.CreatedAt,
			"started_at": run.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}

		progress, err := repo.GetRunProgress(id)
		if err == nil && progress != nil {
			resp["executed_count"] = progress.ExecutedCount
			resp["error_count"] = progress.ErrorCount
		}

		c.JSON(http.StatusOK, resp)
	}
}
